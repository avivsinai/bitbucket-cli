package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/discovery"
	"github.com/avivsinai/bitbucket-cli/internal/skills/frontmatter"
	"github.com/avivsinai/bitbucket-cli/internal/skills/installer"
	"github.com/avivsinai/bitbucket-cli/internal/skills/registry"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type updateOptions struct {
	Skills []string
	All    bool
	Force  bool
	DryRun bool
	Unpin  bool
	Dir    string
}

// installedSkill is a locally installed skill parsed from its SKILL.md frontmatter.
type installedSkill struct {
	name        string
	repoURL     string // bitbucket-repo metadata (canonical web URL)
	commit      string // bitbucket-commit at install time
	pinned      string // bitbucket-pinned (empty = unpinned)
	sourcePath  string // bitbucket-path in the source repository
	dir         string // local directory
	installedBy string // "gh" when the skill carries github-* metadata instead
	metadataErr error
}

// pendingUpdate is a skill whose remote commit differs from the local one.
type pendingUpdate struct {
	local     installedSkill
	repo      source.Repository
	resolved  source.Ref
	skill     discovery.Skill
	newCommit string
}

func newUpdateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &updateOptions{}
	cmd := &cobra.Command{
		Use:   "update [<skill>...]",
		Short: "Update installed skills to their latest versions",
		Long: `Check installed skills for updates by comparing the commit recorded in each
SKILL.md frontmatter (metadata.bitbucket-commit) against the latest commit
that touched the skill in its source repository, and reinstall the ones that
changed.

Scans every known agent host directory in both project and user scope, or a
single directory with --dir. Without arguments, checks all installed skills;
with skill names, checks only those.

Pinned skills (installed with --pin or @version) are skipped with a notice.
Use --unpin to clear the pin and include them in the update.

Skills without Bitbucket metadata (installed manually or by another tool) are
prompted for their source repository in interactive mode. With --all or in
non-interactive mode they are skipped with a notice. Skills installed by
gh are listed with a reminder to update them with gh.

With --force, skills are re-downloaded even when already up to date. This
overwrites locally modified files with their original content but does not
remove extra files added locally.

In interactive mode, the available updates are shown and confirmed before
proceeding. With --all, updates are applied without prompting. With
--dry-run, updates are reported without modifying any files.`,
		Example: `  # Check and update all skills interactively
  bkt skill update

  # Update specific skills
  bkt skill update code-review git-commit

  # Update all without prompting
  bkt skill update --all

  # Re-download all skills (restore locally modified files)
  bkt skill update --force --all

  # Check for updates without applying
  bkt skill update --dry-run

  # Unpin skills and update them to latest
  bkt skill update --unpin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Skills = args
			return runUpdate(cmd, f, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "Update all skills without prompting")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Re-download even if already up to date")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Report available updates without modifying files")
	cmd.Flags().BoolVar(&opts.Unpin, "unpin", false, "Clear pinned version and include pinned skills in update")
	cmd.Flags().StringVar(&opts.Dir, "dir", "", "Scan a custom directory for installed skills")

	return cmd
}

func runUpdate(cmd *cobra.Command, f *cmdutil.Factory, opts *updateOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	canPrompt := ios.CanPrompt()

	gitRoot := installer.ResolveGitRoot(ctx)
	homeDir := installer.ResolveHomeDir()

	var installed []installedSkill
	if opts.Dir != "" {
		installed, err = scanInstalledForUpdate(opts.Dir)
		if err != nil {
			return fmt.Errorf("could not scan directory: %w", err)
		}
	} else {
		installed = scanAllAgentsForUpdate(gitRoot, homeDir)
	}
	if len(installed) == 0 {
		fmt.Fprintln(ios.ErrOut, "No installed skills found.")
		return nil
	}

	if len(opts.Skills) > 0 {
		requested := make(map[string]bool, len(opts.Skills))
		for _, name := range opts.Skills {
			requested[name] = true
		}
		var filtered []installedSkill
		for _, s := range installed {
			if requested[s.name] {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("none of the specified skills are installed")
		}
		installed = filtered
	}

	// One corrupt skill must not prevent updating the others.
	var valid []installedSkill
	for _, s := range installed {
		if s.metadataErr != nil {
			fmt.Fprintf(ios.ErrOut, "! Skipping %s: invalid repository metadata: %s\n", s.name, s.metadataErr)
			continue
		}
		valid = append(valid, s)
	}
	installed = valid
	if len(installed) == 0 {
		fmt.Fprintln(ios.ErrOut, "No updatable skills found.")
		return nil
	}

	var noMeta, byGH []string
	for i := range installed {
		s := &installed[i]
		if s.repoURL != "" {
			continue
		}
		if s.installedBy == "gh" {
			byGH = append(byGH, s.name)
			continue
		}
		if !canPrompt || opts.All {
			noMeta = append(noMeta, s.name)
			continue
		}
		fmt.Fprintf(ios.ErrOut, "! %s has no Bitbucket metadata\n", s.name)
		input, promptErr := f.Prompt().Input(fmt.Sprintf("Repository for %s (workspace/repo or project/repo):", s.name), "")
		if promptErr != nil {
			return promptErr
		}
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Fprintf(ios.ErrOut, "  Skipping %s\n", s.name)
			continue
		}
		if _, parseErr := source.ParseRepoArg(input); parseErr != nil {
			fmt.Fprintf(ios.ErrOut, "  ! %v\n  Skipping %s\n", parseErr, s.name)
			continue
		}
		s.repoURL = input
	}

	stop := startProgress(f, ios, fmt.Sprintf("Checking %d installed skill(s) for updates", len(installed)))
	var updates []pendingUpdate
	var pinned []installedSkill

	type repoState struct {
		repo     source.Repository
		resolved source.Ref
		skills   []discovery.Skill
		failed   bool
	}
	repos := make(map[string]*repoState)

	for _, s := range installed {
		if s.repoURL == "" {
			continue
		}
		if s.pinned != "" && !opts.Unpin {
			pinned = append(pinned, s)
			continue
		}

		state, ok := repos[s.repoURL]
		if !ok {
			state = &repoState{}
			repos[s.repoURL] = state
			repo, openErr := openRepositoryFunc(cmd, f, s.repoURL)
			if openErr != nil {
				state.failed = true
				fmt.Fprintf(ios.ErrOut, "! Skipping %s: could not open %s: %v\n", s.name, s.repoURL, openErr)
			} else if resolved, resolveErr := source.ResolveRef(ctx, repo, ""); resolveErr != nil {
				state.failed = true
				fmt.Fprintf(ios.ErrOut, "! Skipping %s: could not resolve %s: %v\n", s.name, repo.FullName(), resolveErr)
			} else if skills, discoverErr := discovery.DiscoverAllSkills(ctx, repo, resolved.SHA); discoverErr != nil {
				state.failed = true
				fmt.Fprintf(ios.ErrOut, "! Skipping %s: %v\n", s.name, discoverErr)
			} else {
				state.repo, state.resolved, state.skills = repo, resolved, skills
			}
		}
		if state.failed {
			continue
		}

		for _, remote := range state.skills {
			matched := false
			if s.sourcePath != "" {
				matched = remote.Path == s.sourcePath
			} else {
				matched = remote.InstallName() == s.name || remote.Name == s.name
			}
			if !matched {
				continue
			}
			newCommit, commitErr := state.repo.LatestCommit(ctx, state.resolved.SHA, remote.Path)
			if commitErr != nil {
				fmt.Fprintf(ios.ErrOut, "! Skipping %s: could not determine latest commit: %v\n", s.name, commitErr)
				break
			}
			if newCommit != s.commit || opts.Force {
				updates = append(updates, pendingUpdate{local: s, repo: state.repo, resolved: state.resolved, skill: remote, newCommit: newCommit})
			}
			break
		}
	}
	stop()

	for _, s := range pinned {
		fmt.Fprintf(ios.ErrOut, "⊘ %s is pinned to %s (skipped)\n", s.name, s.pinned)
	}
	for _, name := range byGH {
		fmt.Fprintf(ios.ErrOut, "! %s was installed by gh; run `gh skill update %s` to update it\n", name, name)
	}
	for _, name := range noMeta {
		fmt.Fprintf(ios.ErrOut, "! %s has no Bitbucket metadata. Run `bkt skill update %s` interactively to add metadata, or reinstall to enable updates\n", name, name)
	}

	if len(updates) == 0 {
		if opts.Force && opts.DryRun {
			fmt.Fprintln(ios.ErrOut, "All skills are up to date. Use --force without --dry-run to re-download anyway.")
		} else {
			fmt.Fprintln(ios.ErrOut, "All skills are up to date.")
		}
		return nil
	}

	fmt.Fprintf(ios.ErrOut, "\n%d update(s) available:\n", len(updates))
	for _, u := range updates {
		if u.local.commit == u.newCommit {
			fmt.Fprintf(ios.Out, "  • %s (%s) %s (reinstall) [%s]\n", u.local.name, u.repo.FullName(), shortSHA(u.newCommit), source.ShortRef(u.resolved.Ref))
		} else {
			fmt.Fprintf(ios.Out, "  • %s (%s) %s > %s [%s]\n", u.local.name, u.repo.FullName(), shortSHA(u.local.commit), shortSHA(u.newCommit), source.ShortRef(u.resolved.Ref))
		}
	}
	fmt.Fprintln(ios.ErrOut)

	if opts.DryRun {
		return nil
	}

	if !opts.All {
		if !canPrompt {
			return fmt.Errorf("updates available; re-run with --all to apply, or run interactively to confirm")
		}
		confirmed, confirmErr := f.Prompt().Confirm(fmt.Sprintf("Update %d skill(s)?", len(updates)), true)
		if confirmErr != nil {
			return confirmErr
		}
		if !confirmed {
			fmt.Fprintln(ios.ErrOut, "Update cancelled.")
			return nil
		}
	}

	var failed bool
	for _, u := range updates {
		if err := updateSkillInPlace(ctx, u, gitRoot, homeDir); err != nil {
			fmt.Fprintf(ios.ErrOut, "✗ Failed to update %s: %v\n", u.local.name, err)
			failed = true
			continue
		}
		fmt.Fprintf(ios.Out, "✓ Updated %s\n", u.local.name)
	}
	if failed {
		return fmt.Errorf("some skills failed to update")
	}
	return nil
}

// updateSkillInPlace installs the update into a staging directory next to the
// existing skill and swaps the contents in with same-filesystem renames, so
// the skill directory itself is preserved, stale files are removed, and any
// failure leaves the existing skill untouched.
func updateSkillInPlace(ctx context.Context, u pendingUpdate, gitRoot, homeDir string) error {
	if u.local.dir == "" {
		return fmt.Errorf("cannot update %s: no install location recorded", u.local.name)
	}
	parent := filepath.Dir(u.local.dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("could not ensure parent directory %s: %w", parent, err)
	}

	staging, err := os.MkdirTemp(parent, "."+u.skill.Name+".bkt-skill-update-")
	if err != nil {
		return fmt.Errorf("could not create staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if _, err := installer.Install(ctx, &installer.Options{
		Repo:    u.repo,
		Ref:     u.resolved,
		Skills:  []discovery.Skill{u.skill},
		Dir:     staging,
		GitRoot: gitRoot,
		HomeDir: homeDir,
	}); err != nil {
		return err
	}

	staged := filepath.Join(staging, u.skill.Name)
	if _, err := os.Stat(staged); err != nil {
		return fmt.Errorf("installer did not produce %s: %w", staged, err)
	}
	if err := os.MkdirAll(u.local.dir, 0o755); err != nil {
		return fmt.Errorf("could not ensure skill directory %s: %w", u.local.dir, err)
	}
	return swapDirectoryContents(u.local.dir, staged)
}

// swapDirectoryContents replaces the entries inside dest with the entries
// inside src, preserving dest itself. Existing entries are moved to a backup
// first and restored if any step fails.
func swapDirectoryContents(dest, src string) error {
	backup, err := os.MkdirTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".bkt-skill-backup-")
	if err != nil {
		return fmt.Errorf("could not create backup directory: %w", err)
	}

	existing, err := os.ReadDir(dest)
	if err != nil {
		_ = os.RemoveAll(backup)
		return fmt.Errorf("could not read skill directory %s: %w", dest, err)
	}
	var movedOut []string
	for _, entry := range existing {
		if err := os.Rename(filepath.Join(dest, entry.Name()), filepath.Join(backup, entry.Name())); err != nil {
			restoreBackup(dest, backup, movedOut, nil)
			return fmt.Errorf("could not move %s aside: %w", entry.Name(), err)
		}
		movedOut = append(movedOut, entry.Name())
	}

	staged, err := os.ReadDir(src)
	if err != nil {
		restoreBackup(dest, backup, movedOut, nil)
		return fmt.Errorf("could not read staged skill directory %s: %w", src, err)
	}
	var movedIn []string
	for _, entry := range staged {
		if err := os.Rename(filepath.Join(src, entry.Name()), filepath.Join(dest, entry.Name())); err != nil {
			restoreBackup(dest, backup, movedOut, movedIn)
			return fmt.Errorf("could not move %s into place: %w", entry.Name(), err)
		}
		movedIn = append(movedIn, entry.Name())
	}

	_ = os.RemoveAll(backup)
	return nil
}

func restoreBackup(dest, backup string, movedOut, movedIn []string) {
	for _, name := range movedIn {
		_ = os.RemoveAll(filepath.Join(dest, name))
	}
	for _, name := range movedOut {
		_ = os.Rename(filepath.Join(backup, name), filepath.Join(dest, name))
	}
	_ = os.RemoveAll(backup)
}

// scanAllAgentsForUpdate collects installed skills from every registered
// agent directory, scanning shared directories only once.
func scanAllAgentsForUpdate(gitRoot, homeDir string) []installedSkill {
	scanned := make(map[string]bool)
	var all []installedSkill
	for i := range registry.Agents {
		host := &registry.Agents[i]
		for _, scope := range []registry.Scope{registry.ScopeProject, registry.ScopeUser} {
			dir, err := host.InstallDir(scope, gitRoot, homeDir)
			if err != nil || scanned[dir] {
				continue
			}
			scanned[dir] = true
			skills, err := scanInstalledForUpdate(dir)
			if err != nil {
				continue
			}
			all = append(all, skills...)
		}
	}
	return all
}

// scanInstalledForUpdate reads SKILL.md files in flat ({dir}/{name}) and
// namespaced ({dir}/{ns}/{name}) layouts.
func scanInstalledForUpdate(skillsDir string) ([]installedSkill, error) {
	entries, err := os.ReadDir(skillsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read skills directory: %w", err)
	}

	var skills []installedSkill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(skillsDir, e.Name())
		if data, readErr := os.ReadFile(filepath.Join(dir, "SKILL.md")); readErr == nil {
			skills = append(skills, parseInstalledSkill(data, e.Name(), dir))
			continue
		}
		subEntries, subErr := os.ReadDir(dir)
		if subErr != nil {
			continue
		}
		for _, sub := range subEntries {
			if !sub.IsDir() {
				continue
			}
			subDir := filepath.Join(dir, sub.Name())
			if data, readErr := os.ReadFile(filepath.Join(subDir, "SKILL.md")); readErr == nil {
				skills = append(skills, parseInstalledSkill(data, e.Name()+"/"+sub.Name(), subDir))
			}
		}
	}
	return skills, nil
}

func parseInstalledSkill(data []byte, name, dir string) installedSkill {
	s := installedSkill{name: name, dir: dir}
	result, err := frontmatter.Parse(string(data))
	if err != nil {
		s.metadataErr = fmt.Errorf("invalid SKILL.md: %w", err)
		return s
	}
	meta := result.Metadata.Meta
	if meta == nil {
		return s
	}
	str := func(key string) string {
		v, _ := meta[key].(string)
		return v
	}
	if repoURL := str(frontmatter.KeyRepo); repoURL != "" {
		if _, parseErr := source.ParseRepoArg(repoURL); parseErr != nil {
			s.metadataErr = parseErr
		} else {
			s.repoURL = repoURL
		}
	} else if str("github-repo") != "" {
		s.installedBy = "gh"
	}
	s.commit = str(frontmatter.KeyCommit)
	s.pinned = str(frontmatter.KeyPinned)
	s.sourcePath = str(frontmatter.KeyPath)
	return s
}
