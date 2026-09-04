package skill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/discovery"
	"github.com/avivsinai/bitbucket-cli/internal/skills/frontmatter"
	"github.com/avivsinai/bitbucket-cli/internal/skills/installer"
	"github.com/avivsinai/bitbucket-cli/internal/skills/registry"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

type installOptions struct {
	SkillSource     string // WORKSPACE/REPO, PROJECT/REPO, URL, or a local path with --from-local
	SkillName       string // possibly with @version suffix
	Agent           string
	Scope           string
	Pin             string
	Dir             string
	All             bool
	Force           bool
	FromLocal       bool
	AllowHiddenDirs bool
	Upstream        bool

	version string // parsed from SkillName@version or --pin
}

func newInstallCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &installOptions{}
	cmd := &cobra.Command{
		Use:     "install <repository> [<skill[@version]>]",
		Aliases: []string{"add"},
		Short:   "Install agent skills from a Bitbucket repository",
		Long: `Install agent skills from a Bitbucket repository or local directory into
your local environment. Skills are placed in a host-specific directory at
either project scope (inside the current git repository) or user scope (in
your home directory, available everywhere).

Supported --agent values:

` + registry.AgentHelpList() + `

Use --agent and --scope to control placement, or --dir for a custom
directory. The default scope is "project" and the default agent is
"universal", which installs into the shared .agents/skills directory read by
GitHub Copilot, Cursor, Codex, Gemini CLI, Antigravity, Amp, Cline, OpenCode,
Warp and others. Agents that resolve to the same destination share one copy.

The first argument is a Bitbucket repository: WORKSPACE/REPO on Bitbucket
Cloud, PROJECT/REPO on Bitbucket Data Center, or a full repository URL. The
active context (or --context) supplies the host and credentials. Use
--from-local to install from a local directory instead; files are copied
(not symlinked) with local-path tracking metadata injected into frontmatter.

Skills are discovered using the skills/*/SKILL.md convention from the Agent
Skills specification (https://agentskills.io/specification), including
namespaced skills/{author}/*/SKILL.md, plugins/*/skills/*/SKILL.md, root-level
*/SKILL.md, and a skills/ directory nested under a prefix.

The skill argument can be a name, a namespaced name (author/skill), or an
exact path within the repository (skills/author/skill,
packages/agent-skills/code-review, or any .../SKILL.md path). Passing an
exact path skips the full repository listing, which is faster on large
repositories.

When a skill name is given without a version, the newest tag is used,
falling back to the default branch when the repository has no tags. To pin
a version, append @VERSION to the skill name or use --pin; the version is
resolved as a branch, tag, or commit SHA.

Installed skills carry source metadata (metadata.bitbucket-*) in their
SKILL.md frontmatter so "bkt skill update" can detect changes.

Use --all to install every discovered skill. Without a skill name or --all,
the available skills are listed (tab-separated when piped) so you can browse
or grep them before re-running with a specific skill.`,
		Example: `  # List the skills available in a repository
  bkt skill install myteam/agent-skills

  # Install a specific skill into .agents/skills of the current repository
  bkt skill install myteam/agent-skills code-review

  # Install all skills from a repository
  bkt skill install myteam/agent-skills --all

  # Install a specific version
  bkt skill install myteam/agent-skills code-review@v1.2.0

  # Install by exact path (skips the full repository listing)
  bkt skill install myteam/agent-skills skills/monalisa/code-review

  # Install for Claude Code at user scope
  bkt skill install myteam/agent-skills code-review --agent claude-code --scope user

  # Install from a Data Center repository by project key
  bkt skill install PROJ/agent-skills code-review

  # Install from a local directory
  bkt skill install ./my-skills-repo code-review --from-local

  # Include skills kept in hidden directories such as .claude/skills/
  bkt skill install myteam/agent-skills --allow-hidden-dirs`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				opts.SkillSource = args[0]
			}
			if len(args) >= 2 {
				opts.SkillName = args[1]
			}
			if opts.All && opts.SkillName != "" {
				return fmt.Errorf("cannot use --all with a skill argument")
			}
			if opts.FromLocal && opts.SkillSource == "" {
				return fmt.Errorf("--from-local requires a directory path argument")
			}
			if opts.FromLocal && opts.Pin != "" {
				return fmt.Errorf("--from-local and --pin cannot be used together")
			}
			if opts.FromLocal && opts.Upstream {
				return fmt.Errorf("--from-local and --upstream cannot be used together")
			}
			if opts.Pin != "" && strings.Contains(opts.SkillName, "@") {
				return fmt.Errorf("cannot use --pin with an inline @version in the skill name")
			}
			if opts.Scope != string(registry.ScopeProject) && opts.Scope != string(registry.ScopeUser) {
				return fmt.Errorf("--scope must be %q or %q", registry.ScopeProject, registry.ScopeUser)
			}
			if opts.Agent != "" {
				if _, err := registry.FindByID(opts.Agent); err != nil {
					return err
				}
			}
			return runInstall(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Agent, "agent", "", "Target agent (see supported values above)")
	cmd.Flags().StringVar(&opts.Scope, "scope", string(registry.ScopeProject), "Installation scope: project or user")
	cmd.Flags().StringVar(&opts.Pin, "pin", "", "Pin to a specific branch, tag, or commit SHA")
	cmd.Flags().StringVar(&opts.Dir, "dir", "", "Install to a custom directory (overrides --agent and --scope)")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Install all skills in the repository")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Overwrite existing skills without prompting")
	cmd.Flags().BoolVar(&opts.FromLocal, "from-local", false, "Treat the argument as a local directory path instead of a repository")
	cmd.Flags().BoolVar(&opts.AllowHiddenDirs, "allow-hidden-dirs", false, "Include skills in hidden directories (e.g. .claude/skills/, .agents/skills/)")
	cmd.Flags().BoolVar(&opts.Upstream, "upstream", false, "Install from the upstream source when a re-published skill is detected")

	return cmd
}

func runInstall(cmd *cobra.Command, f *cmdutil.Factory, opts *installOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	if opts.FromLocal {
		return runLocalInstall(cmd, f, ios, opts)
	}

	if opts.SkillSource == "" {
		if !ios.CanPrompt() {
			return fmt.Errorf("must specify a repository to install from")
		}
		input, err := f.Prompt().Input("Repository (workspace/repo or project/repo):", "")
		if err != nil {
			return err
		}
		opts.SkillSource = strings.TrimSpace(input)
		if opts.SkillSource == "" {
			return fmt.Errorf("must specify a repository to install from")
		}
	}

	parseSkillVersion(opts)

	repo, err := openRepositoryFunc(cmd, f, opts.SkillSource)
	if err != nil {
		return err
	}
	return installFromRepository(cmd, f, ios, opts, repo)
}

func installFromRepository(cmd *cobra.Command, f *cmdutil.Factory, ios *iostreams.IOStreams, opts *installOptions, repo source.Repository) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	resolved, err := resolveVersion(ctx, f, ios, repo, opts.version)
	if err != nil {
		return err
	}

	var selected []discovery.Skill
	if discovery.IsSkillPath(opts.SkillName) {
		stop := startProgress(f, ios, "Looking up skill")
		skill, err := discovery.DiscoverSkillByPath(ctx, repo, resolved.SHA, opts.SkillName, discovery.DiscoverSkillByPathOptions{})
		stop()
		if err != nil {
			return err
		}
		selected = []discovery.Skill{*skill}
	} else {
		skills, err := discoverRepositorySkills(ctx, f, ios, repo, resolved, opts.AllowHiddenDirs)
		if err != nil {
			return err
		}
		selected, err = selectSkills(cmd, ios, opts, skills, skillSelector{
			sourceHint: repo.FullName(),
			match: func(name string) ([]discovery.Skill, error) {
				return matchSkillByName(skills, name, repo.FullName())
			},
			fetchDescriptions: func() {
				stop := startProgress(f, ios, "Fetching skill info")
				discovery.FetchDescriptions(ctx, repo, resolved.SHA, skills, nil)
				stop()
			},
		})
		if err != nil {
			if errors.Is(err, errSkillsListed) {
				return nil
			}
			return err
		}
	}

	// A re-published skill carries bitbucket-repo metadata pointing at the
	// original repository; offer to install from there instead.
	if len(selected) == 1 {
		upstreamArg, err := checkUpstreamProvenance(ctx, f, ios, opts, repo, selected[0], resolved.SHA)
		if err != nil {
			return err
		}
		if upstreamArg != "" {
			upstreamRepo, err := openRepositoryFunc(cmd, f, upstreamArg)
			if err != nil {
				return fmt.Errorf("could not open upstream repository %s: %w", upstreamArg, err)
			}
			opts.SkillSource = upstreamArg
			opts.version = ""
			opts.Pin = ""
			opts.Upstream = false
			return installFromRepository(cmd, f, ios, opts, upstreamRepo)
		}
	}

	printPreInstallDisclaimer(ios.ErrOut)

	hosts, err := resolveHosts(opts)
	if err != nil {
		return err
	}
	scope := registry.Scope(opts.Scope)
	gitRoot := installer.ResolveGitRoot(ctx)
	homeDir := installer.ResolveHomeDir()

	plans, err := buildInstallPlans(f, ios, opts, selected, hosts, scope, gitRoot, homeDir)
	if err != nil {
		return err
	}

	for _, plan := range plans {
		if len(plans) > 1 {
			fmt.Fprintf(ios.ErrOut, "\nInstalling to %s for %s...\n", friendlyDir(plan.dir), formatPlanHosts(plan.hosts))
		}

		stopDownload := func() {}
		result, err := installer.Install(ctx, &installer.Options{
			Repo:      repo,
			Ref:       resolved,
			PinnedRef: opts.Pin,
			Skills:    plan.skills,
			Dir:       plan.dir,
			HomeDir:   homeDir,
			OnProgress: func(done, total int) {
				if done == 0 {
					stopDownload = startProgress(f, ios, "Downloading skill files")
				} else if done >= total {
					stopDownload()
				}
			},
		})
		stopDownload()

		if result != nil {
			for _, w := range result.Warnings {
				fmt.Fprintf(ios.ErrOut, "! %s\n", w)
			}
			for _, name := range result.Installed {
				fmt.Fprintf(ios.Out, "✓ Installed %s (from %s@%s) in %s\n", name, repo.FullName(), source.ShortRef(resolved.Ref), friendlyDir(result.Dir))
			}
			printInstalledTree(ios.ErrOut, result.Dir, result.Installed)
			printReviewHint(ios.ErrOut, repo.FullName(), resolved.SHA, result.Installed, opts.AllowHiddenDirs)
			printHostHints(ios.ErrOut, plan.hosts, result.Installed, result.Dir, gitRoot)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func runLocalInstall(cmd *cobra.Command, f *cmdutil.Factory, ios *iostreams.IOStreams, opts *installOptions) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	sourcePath := opts.SkillSource
	if sourcePath == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			sourcePath = home
		}
	} else if after, ok := strings.CutPrefix(sourcePath, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			sourcePath = filepath.Join(home, after)
		}
	}
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("could not resolve path: %w", err)
	}

	allSkills, err := discovery.DiscoverAllLocalSkills(absSource)
	if err != nil {
		return err
	}
	skills, err := filterHiddenDirSkills(ios.ErrOut, opts.AllowHiddenDirs, allSkills, false)
	if err != nil {
		return err
	}
	if ios.CanPrompt() {
		fmt.Fprintf(ios.ErrOut, "Found %s\n", pluralize(len(skills), "skill"))
	}

	selected, err := selectSkills(cmd, ios, opts, skills, skillSelector{
		sourceHint: absSource,
		match:      func(name string) ([]discovery.Skill, error) { return matchLocalSkillByName(skills, name) },
	})
	if err != nil {
		if errors.Is(err, errSkillsListed) {
			return nil
		}
		return err
	}

	printPreInstallDisclaimer(ios.ErrOut)

	hosts, err := resolveHosts(opts)
	if err != nil {
		return err
	}
	scope := registry.Scope(opts.Scope)
	gitRoot := installer.ResolveGitRoot(ctx)
	homeDir := installer.ResolveHomeDir()

	plans, err := buildInstallPlans(f, ios, opts, selected, hosts, scope, gitRoot, homeDir)
	if err != nil {
		return err
	}

	for _, plan := range plans {
		if len(plans) > 1 {
			fmt.Fprintf(ios.ErrOut, "\nInstalling to %s for %s...\n", friendlyDir(plan.dir), formatPlanHosts(plan.hosts))
		}
		result, err := installer.InstallLocal(&installer.LocalOptions{
			SourceDir: absSource,
			Skills:    plan.skills,
			Dir:       plan.dir,
		})
		if err != nil {
			return err
		}
		for _, name := range result.Installed {
			fmt.Fprintf(ios.Out, "Installed %s (from %s) in %s\n", name, opts.SkillSource, friendlyDir(result.Dir))
		}
		printInstalledTree(ios.ErrOut, result.Dir, result.Installed)
		printReviewHint(ios.ErrOut, "", "", result.Installed, false)
		printHostHints(ios.ErrOut, plan.hosts, result.Installed, result.Dir, gitRoot)
	}

	return nil
}

// parseSkillVersion splits "name@version" and applies --pin. Both forms pin
// the skill, so "bkt skill update" leaves it alone until --unpin is passed.
func parseSkillVersion(opts *installOptions) {
	if opts.SkillName != "" {
		if i := strings.LastIndex(opts.SkillName, "@"); i > 0 {
			opts.version = opts.SkillName[i+1:]
			opts.SkillName = opts.SkillName[:i]
			opts.Pin = opts.version
			return
		}
	}
	if opts.Pin != "" {
		opts.version = opts.Pin
	}
}

func resolveVersion(ctx context.Context, f *cmdutil.Factory, ios *iostreams.IOStreams, repo source.Repository, version string) (source.Ref, error) {
	stop := startProgress(f, ios, "Resolving version")
	resolved, err := source.ResolveRef(ctx, repo, version)
	stop()
	if err != nil {
		return source.Ref{}, fmt.Errorf("could not resolve version: %w", err)
	}
	fmt.Fprintf(ios.ErrOut, "Using ref %s (%s)\n", source.ShortRef(resolved.Ref), shortSHA(resolved.SHA))
	return resolved, nil
}

func discoverRepositorySkills(ctx context.Context, f *cmdutil.Factory, ios *iostreams.IOStreams, repo source.Repository, resolved source.Ref, allowHidden bool) ([]discovery.Skill, error) {
	stop := startProgress(f, ios, "Discovering skills")
	allSkills, err := discovery.DiscoverAllSkills(ctx, repo, resolved.SHA)
	stop()
	if err != nil {
		return nil, err
	}
	skills, err := filterHiddenDirSkills(ios.ErrOut, allowHidden, allSkills, false)
	if err != nil {
		return nil, err
	}
	logConventions(ios.ErrOut, skills)
	for _, s := range skills {
		if !discovery.IsSpecCompliant(s.Name) {
			fmt.Fprintf(ios.ErrOut, "Warning: skill %q does not follow the agentskills.io naming convention\n", s.DisplayName())
		}
	}
	return skills, nil
}

func logConventions(w io.Writer, skills []discovery.Skill) {
	conventions := make(map[string]int)
	for _, s := range skills {
		conventions[s.Convention]++
	}
	if n, ok := conventions["skills-namespaced"]; ok {
		fmt.Fprintf(w, "Note: found %d namespaced skill(s) in skills/{author}/ directories\n", n)
	}
	if n, ok := conventions["plugins"]; ok {
		fmt.Fprintf(w, "Note: found %d skill(s) using the plugins/ convention\n", n)
	}
	if n, ok := conventions["root"]; ok {
		fmt.Fprintf(w, "Note: found %d skill(s) at the repository root\n", n)
	}
}

// filterHiddenDirSkills applies --allow-hidden-dirs. When set, every skill is
// returned with a warning; otherwise hidden-dir skills are dropped, with an
// error when nothing else remains and, if noteExcluded is set, a note when
// some were dropped.
func filterHiddenDirSkills(w io.Writer, allowHidden bool, allSkills []discovery.Skill, noteExcluded bool) ([]discovery.Skill, error) {
	if allowHidden {
		if discovery.HasHiddenDirSkills(allSkills) {
			fmt.Fprint(w, "! Skills in hidden directories (e.g. .claude/, .agents/) may be installed\n"+
				"  copies from another publisher. Verify the skill's origin and check for a\n"+
				"  canonical source.\n")
		}
		return allSkills, nil
	}

	r := discovery.PartitionHiddenDirSkills(allSkills)
	if r.HiddenCount > 0 {
		if len(r.Standard) == 0 {
			return nil, fmt.Errorf("no standard skills found, but %d skill(s) exist in hidden directories\n  Use --allow-hidden-dirs to include them", r.HiddenCount)
		}
		if noteExcluded {
			fmt.Fprintf(w, "! %d skill(s) in hidden directories were excluded, use --allow-hidden-dirs to include them\n", r.HiddenCount)
		}
	}
	return r.Standard, nil
}

// skillSelector holds the callbacks that differ between remote and local selection.
type skillSelector struct {
	sourceHint        string
	match             func(name string) ([]discovery.Skill, error)
	fetchDescriptions func()
}

// errSkillsListed signals that the available skills were printed instead of
// installed because no skill was named.
var errSkillsListed = errors.New("skills listed")

func selectSkills(cmd *cobra.Command, ios *iostreams.IOStreams, opts *installOptions, skills []discovery.Skill, sel skillSelector) ([]discovery.Skill, error) {
	checkCollisions := func(ss []discovery.Skill) error {
		if err := collisionError(ss); err != nil {
			fmt.Fprintf(ios.ErrOut, "Hint: install individually using the full name: bkt skill install %s namespace/skill-name\n", sel.sourceHint)
			return err
		}
		return nil
	}

	if opts.All {
		if err := checkCollisions(skills); err != nil {
			return nil, err
		}
		return skills, nil
	}
	if opts.SkillName != "" {
		return sel.match(opts.SkillName)
	}
	if err := listAvailableSkills(cmd, ios, skills, sel); err != nil {
		return nil, err
	}
	return nil, errSkillsListed
}

// listAvailableSkills prints the discovered skills so they can be browsed or
// piped into grep before re-running with a skill name.
// availableSkill is one row of the skill catalogue and its --json shape.
type availableSkill struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

func listAvailableSkills(cmd *cobra.Command, ios *iostreams.IOStreams, skills []discovery.Skill, sel skillSelector) error {
	if len(skills) == 0 {
		return fmt.Errorf("no skills found in %s", sel.sourceHint)
	}
	if sel.fetchDescriptions != nil {
		sel.fetchDescriptions()
	}

	rows := make([]availableSkill, 0, len(skills))
	for _, s := range skills {
		rows = append(rows, availableSkill{
			Name:        s.DisplayName(),
			Namespace:   s.Namespace,
			Path:        s.Path,
			Description: s.Description,
		})
	}

	return cmdutil.WriteOutput(cmd, ios.Out, rows, func() error {
		if ios.IsStdoutTTY() {
			fmt.Fprintf(ios.ErrOut, "Showing %s from %s. Re-run with a skill name to install.\n\n", pluralize(len(skills), "skill"), sel.sourceHint)
			tw := tabwriter.NewWriter(ios.Out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "SKILL\tDESCRIPTION")
			for _, s := range skills {
				fmt.Fprintf(tw, "%s\t%s\n", sanitizeForTerminal(s.DisplayName()), truncate(sanitizeForTerminal(s.Description), 60))
			}
			return tw.Flush()
		}
		for _, s := range skills {
			if _, err := fmt.Fprintf(ios.Out, "%s\t%s\n", sanitizeForTerminal(s.DisplayName()), sanitizeForTerminal(s.Description)); err != nil {
				return err
			}
		}
		return nil
	})
}

func truncate(s string, width int) string {
	if width <= 3 || len(s) <= width {
		return s
	}
	return s[:width-3] + "..."
}

func matchSkillByName(skills []discovery.Skill, name, sourceHint string) ([]discovery.Skill, error) {
	for _, s := range skills {
		if s.DisplayName() == name {
			return []discovery.Skill{s}, nil
		}
	}

	var matches []discovery.Skill
	for _, s := range skills {
		if s.Name == name {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("skill %q not found in %s", name, sourceHint)
	case 1:
		return matches, nil
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.DisplayName()
		}
		return nil, fmt.Errorf(
			"skill name %q is ambiguous, multiple matches found:\n  %s\n  Specify the full name (e.g. %s) to disambiguate",
			name, strings.Join(names, "\n  "), names[0],
		)
	}
}

func matchLocalSkillByName(skills []discovery.Skill, name string) ([]discovery.Skill, error) {
	for _, s := range skills {
		if s.DisplayName() == name || s.Name == name {
			return []discovery.Skill{s}, nil
		}
	}
	return nil, fmt.Errorf("skill %q not found in local directory", name)
}

func collisionError(ss []discovery.Skill) error {
	collisions := discovery.FindNameCollisions(ss)
	if len(collisions) == 0 {
		return nil
	}
	return fmt.Errorf("cannot install skills with conflicting names; they would overwrite each other:\n  %s", discovery.FormatCollisions(collisions))
}

// checkUpstreamProvenance reads the selected skill's SKILL.md and returns the
// upstream repository argument to redirect to when the skill was re-published
// from another Bitbucket repository and the user chose (or asked with
// --upstream) to install from there.
func checkUpstreamProvenance(ctx context.Context, f *cmdutil.Factory, ios *iostreams.IOStreams, opts *installOptions, repo source.Repository, skill discovery.Skill, sha string) (string, error) {
	content, err := repo.ReadFile(ctx, sha, skill.Path+"/SKILL.md")
	if err != nil {
		return "", nil //nolint:nilerr // best-effort check; failing to fetch is not fatal
	}
	result, err := frontmatter.Parse(string(content))
	if err != nil || result.Metadata.Meta == nil {
		return "", nil //nolint:nilerr // unparsable frontmatter means no upstream to detect
	}
	existingRepo, _ := result.Metadata.Meta[frontmatter.KeyRepo].(string)
	if existingRepo == "" || existingRepo == repo.WebURL() {
		return "", nil
	}
	upstream, err := source.ParseRepoArg(existingRepo)
	if err != nil {
		return "", nil //nolint:nilerr // an unparsable URL means we cannot redirect; install normally
	}
	upstreamLabel := sanitizeForTerminal(upstream.Owner + "/" + upstream.Slug)

	fmt.Fprintf(ios.ErrOut, "! This skill was originally published in %s\n", upstreamLabel)
	if opts.Upstream {
		fmt.Fprintf(ios.ErrOut, "Redirecting install to %s...\n", upstreamLabel)
		return existingRepo, nil
	}
	if !ios.CanPrompt() {
		fmt.Fprintf(ios.ErrOut, "  Installing from %s (use --upstream or interactive mode to choose upstream)\n", repo.FullName())
		return "", nil
	}
	redirect, err := f.Prompt().Confirm(fmt.Sprintf("Install from upstream %s instead of re-publisher %s?", upstreamLabel, repo.FullName()), false)
	if err != nil {
		return "", err
	}
	if redirect {
		fmt.Fprintf(ios.ErrOut, "Redirecting install to %s...\n", upstreamLabel)
		return existingRepo, nil
	}
	return "", nil
}

func printPreInstallDisclaimer(w io.Writer) {
	fmt.Fprint(w, "\n! Skills are not verified by bkt or Bitbucket and may contain prompt injections, hidden instructions, or malicious scripts. Always review skill contents before use.\n\n")
}

func resolveHosts(opts *installOptions) ([]*registry.AgentHost, error) {
	id := opts.Agent
	if id == "" {
		// --dir fixes the destination regardless of agent; the default host
		// then only affects post-install hints.
		id = registry.DefaultAgentID
	}
	h, err := registry.FindByID(id)
	if err != nil {
		return nil, err
	}
	return []*registry.AgentHost{h}, nil
}

type installPlan struct {
	dir    string
	hosts  []*registry.AgentHost
	skills []discovery.Skill
}

func buildInstallPlans(f *cmdutil.Factory, ios *iostreams.IOStreams, opts *installOptions, selected []discovery.Skill, hosts []*registry.AgentHost, scope registry.Scope, gitRoot, homeDir string) ([]installPlan, error) {
	byDir := make(map[string]*installPlan)
	var orderedDirs []string
	for _, host := range hosts {
		targetDir := opts.Dir
		if targetDir == "" {
			var err error
			targetDir, err = host.InstallDir(scope, gitRoot, homeDir)
			if err != nil {
				return nil, err
			}
		}
		plan, ok := byDir[targetDir]
		if !ok {
			plan = &installPlan{dir: targetDir}
			byDir[targetDir] = plan
			orderedDirs = append(orderedDirs, targetDir)
		}
		plan.hosts = append(plan.hosts, host)
	}

	plans := make([]installPlan, 0, len(orderedDirs))
	for _, dir := range orderedDirs {
		plan := byDir[dir]
		skills, err := checkOverwrite(f, ios, opts, selected, plan.dir)
		if err != nil {
			return nil, err
		}
		if len(skills) == 0 {
			fmt.Fprintf(ios.ErrOut, "No skills to install in %s for %s.\n", friendlyDir(plan.dir), formatPlanHosts(plan.hosts))
			continue
		}
		plan.skills = skills
		plans = append(plans, *plan)
	}
	return plans, nil
}

func formatPlanHosts(hosts []*registry.AgentHost) string {
	names := make([]string, len(hosts))
	for i, host := range hosts {
		names[i] = host.Name
	}
	return strings.Join(names, ", ")
}

func checkOverwrite(f *cmdutil.Factory, ios *iostreams.IOStreams, opts *installOptions, skills []discovery.Skill, targetDir string) ([]discovery.Skill, error) {
	var existing, fresh []discovery.Skill
	for _, s := range skills {
		if _, err := os.Stat(filepath.Join(targetDir, s.Name)); err == nil {
			existing = append(existing, s)
		} else {
			fresh = append(fresh, s)
		}
	}
	if len(existing) == 0 || opts.Force {
		return skills, nil
	}
	if !ios.CanPrompt() {
		names := make([]string, len(existing))
		for i, s := range existing {
			names[i] = s.DisplayName()
		}
		return nil, fmt.Errorf("skills already installed: %s (use --force to overwrite)", strings.Join(names, ", "))
	}

	var confirmed []discovery.Skill
	for _, s := range existing {
		ok, err := f.Prompt().Confirm(existingSkillPrompt(targetDir, s), false)
		if err != nil {
			return nil, err
		}
		if ok {
			confirmed = append(confirmed, s)
		} else {
			fmt.Fprintf(ios.ErrOut, "Skipping %s\n", s.DisplayName())
		}
	}
	return append(fresh, confirmed...), nil
}

func existingSkillPrompt(targetDir string, incoming discovery.Skill) string {
	fallback := fmt.Sprintf("Skill %q already exists. Overwrite?", incoming.DisplayName())
	data, err := os.ReadFile(filepath.Join(targetDir, incoming.Name, "SKILL.md"))
	if err != nil {
		return fallback
	}
	result, err := frontmatter.Parse(string(data))
	if err != nil || result.Metadata.Meta == nil {
		return fallback
	}
	repoURL, _ := result.Metadata.Meta[frontmatter.KeyRepo].(string)
	if repoURL == "" {
		return fallback
	}
	ref, err := source.ParseRepoArg(repoURL)
	if err != nil {
		return fallback
	}
	sourceName := sanitizeForTerminal(ref.Owner + "/" + ref.Slug)
	if installedRef, _ := result.Metadata.Meta[frontmatter.KeyRef].(string); installedRef != "" {
		sourceName += "@" + sanitizeForTerminal(installedRef)
	}
	return fmt.Sprintf("Skill %q already installed from %s. Overwrite?", incoming.DisplayName(), sourceName)
}

// printReviewHint reminds the user to inspect what was installed, suggesting
// preview commands pinned to the exact commit when the source is a repository.
func printReviewHint(w io.Writer, repoName, sha string, skillNames []string, allowHiddenDirs bool) {
	if len(skillNames) == 0 {
		return
	}
	fmt.Fprint(w, "\n! Skills may contain prompt injections or malicious scripts.\n")
	if repoName == "" {
		fmt.Fprintln(w, "  Review the installed files before use.")
		return
	}
	fmt.Fprintln(w, "  Review installed content before use:")
	fmt.Fprintln(w)
	hiddenFlag := ""
	if allowHiddenDirs {
		hiddenFlag = " --allow-hidden-dirs"
	}
	for _, name := range skillNames {
		fmt.Fprintf(w, "    bkt skill preview %s %s@%s%s\n", repoName, name, sha, hiddenFlag)
	}
	fmt.Fprintln(w)
}

// printHostHints prints agent-specific post-install guidance. Only Kiro CLI
// needs an extra registration step.
func printHostHints(w io.Writer, hosts []*registry.AgentHost, installed []string, installDir, gitRoot string) {
	if len(installed) == 0 {
		return
	}
	for _, h := range hosts {
		if h.ID != "kiro-cli" {
			continue
		}
		fmt.Fprintf(w, "\n! Kiro CLI: register these skills on a custom agent by adding them to\n"+
			"  .kiro/agents/<agent>.json under \"resources\", for example:\n\n"+
			"    {\n      \"resources\": [\"skill://%s/**/SKILL.md\"]\n    }\n\n", kiroResourcePath(installDir, gitRoot))
		return
	}
}

func kiroResourcePath(installDir, gitRoot string) string {
	if gitRoot != "" && installDir != "" {
		if rel, err := filepath.Rel(gitRoot, installDir); err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(installDir)
}
