package skill

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/discovery"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

const (
	previewMaxFiles      = 20
	previewMaxTotalBytes = 512 * 1024
)

type previewOptions struct {
	RepoArg         string
	SkillName       string
	Version         string
	AllowHiddenDirs bool
}

func newPreviewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &previewOptions{}
	cmd := &cobra.Command{
		Use:     "preview <repository> [<skill[@version]>]",
		Aliases: []string{"show"},
		Short:   "Preview a skill from a Bitbucket repository",
		Long: `Show a skill's files and SKILL.md content in the terminal without installing
anything. The output goes through the configured pager when stdout is a
terminal.

A file tree is shown first, followed by SKILL.md and the skill's other files
(up to 20 files or 512 KiB).

When run with only a repository argument, the available skills are listed.

The skill argument can be a name, a namespaced name (author/skill), or an
exact path within the repository (skills/author/skill,
packages/agent-skills/code-review, or any .../SKILL.md path).

To preview a specific version, append @VERSION to the skill name. The version
is resolved as a branch, tag, or commit SHA.`,
		Example: `  # Preview a specific skill
  bkt skill preview myteam/agent-skills code-review

  # Preview a skill at a specific version
  bkt skill preview myteam/agent-skills code-review@v1.2.0

  # Preview exactly what an install wrote, by commit SHA
  bkt skill preview myteam/agent-skills code-review@abc123def456

  # Preview from a non-standard nested path
  bkt skill preview myteam/skills-repo packages/agent-skills/code-review

  # List the skills available in a repository
  bkt skill preview myteam/agent-skills`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RepoArg = args[0]
			if len(args) == 2 {
				opts.SkillName = args[1]
			}
			if i := strings.LastIndex(opts.SkillName, "@"); i > 0 {
				opts.Version = opts.SkillName[i+1:]
				opts.SkillName = opts.SkillName[:i]
			}
			return runPreview(cmd, f, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.AllowHiddenDirs, "allow-hidden-dirs", false, "Include skills in hidden directories (e.g. .claude/skills/, .agents/skills/)")

	return cmd
}

func runPreview(cmd *cobra.Command, f *cmdutil.Factory, opts *previewOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	repo, err := openRepositoryFunc(cmd, f, opts.RepoArg)
	if err != nil {
		return err
	}

	stop := startProgress(f, ios, fmt.Sprintf("Resolving %s", repo.FullName()))
	resolved, err := source.ResolveRef(ctx, repo, opts.Version)
	stop()
	if err != nil {
		return fmt.Errorf("could not resolve version: %w", err)
	}

	var skill discovery.Skill
	if discovery.IsSkillPath(opts.SkillName) {
		stop := startProgress(f, ios, "Looking up skill")
		found, err := discovery.DiscoverSkillByPath(ctx, repo, resolved.SHA, opts.SkillName, discovery.DiscoverSkillByPathOptions{SkipDescription: true})
		stop()
		if err != nil {
			return err
		}
		skill = *found
	} else {
		stop := startProgress(f, ios, "Discovering skills")
		allSkills, err := discovery.DiscoverAllSkills(ctx, repo, resolved.SHA)
		stop()
		if err != nil {
			return err
		}
		skills, err := filterHiddenDirSkills(ios.ErrOut, opts.AllowHiddenDirs, allSkills, true)
		if err != nil {
			return err
		}
		if opts.SkillName == "" {
			return listAvailableSkills(cmd, ios, skills, skillSelector{
				sourceHint: repo.FullName(),
				fetchDescriptions: func() {
					stop := startProgress(f, ios, "Fetching skill info")
					discovery.FetchDescriptions(ctx, repo, resolved.SHA, skills, nil)
					stop()
				},
			})
		}
		skill, err = selectPreviewSkill(skills, opts.SkillName, repo.FullName())
		if err != nil {
			return err
		}
	}

	stop = startProgress(f, ios, "Fetching skill content")
	files, err := discovery.SkillFiles(ctx, repo, resolved.SHA, skill.Path)
	if err != nil {
		fmt.Fprintf(ios.ErrOut, "warning: %v\n", err)
		files = nil
	}
	content, err := repo.ReadFile(ctx, resolved.SHA, skill.Path+"/SKILL.md")
	stop()
	if err != nil {
		return fmt.Errorf("could not fetch SKILL.md: %w", err)
	}

	out := ios.Out
	if ios.IsStdoutTTY() {
		pager := f.PagerManager()
		if w, err := pager.Start(); err == nil {
			out = w
			defer func() { _ = pager.Stop() }()
		} else {
			fmt.Fprintf(ios.ErrOut, "starting pager failed: %v\n", err)
		}
	}

	renderPreview(ctx, out, repo, resolved.SHA, skill, files, content)
	return nil
}

func selectPreviewSkill(skills []discovery.Skill, name, sourceHint string) (discovery.Skill, error) {
	for _, s := range skills {
		if s.DisplayName() == name || s.Name == name {
			return s, nil
		}
	}
	// Accept the namespaced identifiers printed by the post-install hint.
	for _, s := range skills {
		if s.InstallName() == name {
			return s, nil
		}
	}
	return discovery.Skill{}, fmt.Errorf("skill %q not found in %s", name, sourceHint)
}

// renderPreview writes the file tree, SKILL.md, and the remaining files.
func renderPreview(ctx context.Context, out io.Writer, repo source.Repository, sha string, skill discovery.Skill, files []source.File, skillMD []byte) {
	if len(files) > 0 {
		fmt.Fprintf(out, "%s/\n", sanitizeForTerminal(skill.DisplayName()))
		printFileTree(out, files)
		fmt.Fprintln(out)
	}

	fmt.Fprint(out, "── SKILL.md ──\n\n")
	writeSanitized(out, skillMD)

	fetched, totalBytes := 0, 0
	for _, f := range files {
		if f.Path == "SKILL.md" {
			continue
		}
		if fetched >= previewMaxFiles {
			fmt.Fprintf(out, "\n(skipped remaining files, showing first %d)\n", previewMaxFiles)
			break
		}
		if totalBytes+int(f.Size) > previewMaxTotalBytes {
			fmt.Fprint(out, "\n(skipped remaining files, size limit reached)\n")
			break
		}
		content, err := repo.ReadFile(ctx, sha, skill.Path+"/"+f.Path)
		// File paths come from the repository, so they are sanitized like any
		// other untrusted string before reaching the terminal.
		heading := sanitizeForTerminal(f.Path)
		if err != nil {
			fmt.Fprintf(out, "\n── %s ──\n\n(could not fetch file)\n", heading)
			continue
		}
		fetched++
		totalBytes += len(content)
		fmt.Fprintf(out, "\n── %s ──\n\n", heading)
		writeSanitized(out, content)
	}
}

// writeSanitized prints file content with terminal control sequences
// neutralised, keeping newlines and tabs, and ends with a newline.
func writeSanitized(w io.Writer, content []byte) {
	text := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || r == '\r' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, string(content))
	fmt.Fprint(w, text)
	if !strings.HasSuffix(text, "\n") {
		fmt.Fprintln(w)
	}
}
