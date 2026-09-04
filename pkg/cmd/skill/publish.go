package skill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/remote"
	"github.com/avivsinai/bitbucket-cli/internal/skills/discovery"
	"github.com/avivsinai/bitbucket-cli/internal/skills/frontmatter"
	"github.com/avivsinai/bitbucket-cli/internal/skills/installer"
	"github.com/avivsinai/bitbucket-cli/internal/skills/registry"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

const (
	// maxDescriptionChars and maxBodyLines are the agentskills.io recommendations.
	// Exceeding them is a warning, not an error: they are about how much context
	// a skill costs an agent, not about validity.
	maxDescriptionChars = 1024
	maxBodyLines        = 500
)

type severity string

const (
	severityError   severity = "error"
	severityWarning severity = "warning"
	severityFixed   severity = "fixed"
)

// diagnostic is one validation result for a skill or for the repository.
type diagnostic struct {
	Skill    string   `json:"skill,omitempty"`
	Path     string   `json:"path,omitempty"` // skill directory within the repository
	Severity severity `json:"severity"`
	Message  string   `json:"message"`
}

type publishOptions struct {
	Directory string
	DryRun    bool
	Fix       bool
	Tag       string
	Message   string
}

type publishResult struct {
	Diagnostics    []diagnostic `json:"diagnostics" yaml:"diagnostics"`
	Tag            string       `json:"tag" yaml:"tag"`
	Commit         string       `json:"commit" yaml:"commit"`
	Repository     string       `json:"repository" yaml:"repository"`
	InstallCommand string       `json:"installCommand" yaml:"installCommand"`
	PinCommand     string       `json:"pinCommand" yaml:"pinCommand"`
}

func newPublishCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &publishOptions{}
	cmd := &cobra.Command{
		Use:   "publish [<directory>]",
		Short: "Validate skills and tag a release",
		Long: `Validate a repository's skills against the Agent Skills specification and,
with --tag, mark the current commit as a released version.

Skills are discovered with the same conventions as install: skills/*/SKILL.md,
skills/{author}/*/SKILL.md, plugins/*/skills/*/SKILL.md, root-level
*/SKILL.md, and a skills/ directory nested under a prefix. Skills in hidden
directories are ignored: those are copies installed by an agent, not the
repository's own work.

Validation checks:

  - the skill name follows the agentskills.io naming rules
  - the skill name matches its directory name
  - the required frontmatter fields (name, description) are present
  - allowed-tools is a space-delimited string, not a list
  - no install metadata (metadata.bitbucket-*, metadata.github-*, local-path)
    is committed, since that records where a copy came from

Recommendations are reported as warnings: a license field, a description
under 1024 characters, and a body under 500 lines.

Use --dry-run to validate without tagging, and --fix to strip install
metadata from the committed files so you can review and commit the result.

Bitbucket has no releases, so publishing a version means creating a tag on the
current commit. "bkt skill install" resolves an unversioned request to the
newest tag, so tagging is what makes a version installable.`,
		Example: `  # Validate the skills in the current repository
  bkt skill publish --dry-run

  # Strip committed install metadata, then review the changes
  bkt skill publish --fix

  # Validate and tag the current commit as v1.2.0
  bkt skill publish --tag v1.2.0

  # Validate a repository checked out elsewhere
  bkt skill publish ./agent-skills --dry-run`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Directory = args[0]
			}
			if opts.Fix && opts.Tag != "" {
				return fmt.Errorf("--fix and --tag cannot be used together; review and commit the fixes first")
			}
			if opts.DryRun && opts.Tag != "" {
				return fmt.Errorf("--dry-run and --tag cannot be used together")
			}
			if opts.DryRun && opts.Fix {
				return fmt.Errorf("--dry-run and --fix cannot be used together; --fix rewrites files")
			}
			if opts.Message != "" && opts.Tag == "" {
				return fmt.Errorf("--message only applies with --tag")
			}
			return runPublish(cmd, f, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Validate without tagging")
	cmd.Flags().BoolVar(&opts.Fix, "fix", false, "Strip committed install metadata without tagging")
	cmd.Flags().StringVar(&opts.Tag, "tag", "", "Tag the current commit with this version (e.g. v1.2.0)")
	cmd.Flags().StringVar(&opts.Message, "message", "", "Annotation for the tag (defaults to the tag name)")

	return cmd
}

func runPublish(cmd *cobra.Command, f *cmdutil.Factory, opts *publishOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	settings, err := cmdutil.ResolveOutputSettings(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	dir := opts.Directory
	if dir == "" {
		// Validate the whole repository, not just the current subdirectory.
		if dir = installer.ResolveGitRoot(ctx); dir == "" {
			if dir, err = os.Getwd(); err != nil {
				return fmt.Errorf("could not determine working directory: %w", err)
			}
		}
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("could not resolve path: %w", err)
	}

	skills, err := discovery.DiscoverLocalSkills(absDir)
	if err != nil {
		return err
	}

	// Validate before writing anything: --fix must not touch a repository whose
	// skills have other problems, since the user would have to fix those by
	// hand anyway and the rewrite would be mixed into their own edits.
	diagnostics := validateSkills(absDir, skills)
	diagnostics = append(diagnostics, checkUnrecognisedSkillFiles(absDir, skills)...)
	diagnostics = append(diagnostics, checkInstalledSkillDirs(ctx, absDir)...)

	fixed := 0
	if opts.Fix && countSeverity(diagnostics, severityError) == countFixableErrors(diagnostics) {
		diagnostics, fixed = applyFixes(absDir, skills, diagnostics)
	}

	errCount := countSeverity(diagnostics, severityError)
	warnCount := countSeverity(diagnostics, severityWarning)

	structuredTag := opts.Tag != "" && (settings.Format != "" || settings.JQ != "" || settings.Template != "")
	if !structuredTag || errCount > 0 {
		if err := renderDiagnostics(cmd, ios, diagnostics, len(skills), errCount, warnCount); err != nil {
			return err
		}
	}
	if errCount > 0 {
		return fmt.Errorf("validation failed with %d error(s)", errCount)
	}

	if opts.Fix {
		if fixed > 0 {
			fmt.Fprintf(ios.ErrOut, "\nFixed %d file(s). Review and commit the changes, then run `bkt skill publish --tag <version>`.\n", fixed)
		} else {
			fmt.Fprintln(ios.ErrOut, "\nNo issues to fix.")
		}
		return nil
	}
	if opts.DryRun {
		fmt.Fprintln(ios.ErrOut, "\nDry run complete. Re-run with --tag <version> to publish.")
		return nil
	}
	if opts.Tag == "" {
		fmt.Fprintln(ios.ErrOut, "\nValidation passed. Re-run with --tag <version> to publish.")
		return nil
	}

	return publishTag(cmd, f, ios, absDir, skills, diagnostics, opts)
}

// fixableMessage marks the one error --fix can resolve, so the fix pass can
// tell "only committed install metadata is wrong" from "something else is too".
const fixableMessage = "contains install metadata that must be stripped"

// countFixableErrors counts the errors --fix knows how to resolve.
func countFixableErrors(diagnostics []diagnostic) int {
	n := 0
	for _, d := range diagnostics {
		if d.Severity == severityError && strings.HasPrefix(d.Message, fixableMessage) {
			n++
		}
	}
	return n
}

// validateSkills checks every skill without touching any file.
func validateSkills(root string, skills []discovery.Skill) []diagnostic {
	diagnostics := make([]diagnostic, 0)

	for _, skill := range skills {
		name := skill.DisplayName()
		add := func(sev severity, format string, args ...any) {
			diagnostics = append(diagnostics, diagnostic{Skill: name, Path: skill.Path, Severity: sev, Message: fmt.Sprintf(format, args...)})
		}

		skillFile := filepath.Join(root, filepath.FromSlash(skill.Path), "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			add(severityError, "missing SKILL.md file")
			continue
		}

		result, err := frontmatter.Parse(string(data))
		if err != nil {
			add(severityError, "%v", err)
			continue
		}

		switch {
		case result.Metadata.Name == "":
			add(severityError, "missing required field: name")
		case !discovery.IsSpecCompliant(result.Metadata.Name):
			add(severityError, "name %q does not follow the agentskills.io naming convention (lowercase alphanumeric and hyphens)", result.Metadata.Name)
		}
		// A single-skill directory has no directory name to match against.
		if dirName := path.Base(skill.Path); skill.Path != "." && result.Metadata.Name != "" && result.Metadata.Name != dirName {
			add(severityError, "name %q does not match directory name %q", result.Metadata.Name, dirName)
		}

		if result.Metadata.Description == "" {
			add(severityError, "missing required field: description")
		} else if n := len(result.Metadata.Description); n > maxDescriptionChars {
			add(severityWarning, "description is %d chars (recommended max: %d)", n, maxDescriptionChars)
		}

		// The spec defines allowed-tools as a space-delimited string; a YAML
		// list parses but is not portable across agents.
		if _, isList := result.RawYAML["allowed-tools"].([]any); isList {
			add(severityError, "allowed-tools must be a string (space-delimited), not a list")
		}

		if keys := frontmatter.FindInstallMetadata(result); len(keys) > 0 {
			add(severityError, "%s: %s (use --fix)", fixableMessage, strings.Join(keys, ", "))
		}

		if _, ok := result.RawYAML["license"]; !ok {
			add(severityWarning, "recommended field missing: license")
		}
		if lines := strings.Count(result.Body, "\n") + 1; lines > maxBodyLines {
			add(severityWarning, "skill body is %d lines (recommended max: %d for efficient context)", lines, maxBodyLines)
		}
	}

	return diagnostics
}

// applyFixes strips committed install metadata and replaces each corresponding
// error with the result. It runs only when that is the sole error, so a repair
// is never mixed into a repository the user still has to fix by hand.
func applyFixes(root string, skills []discovery.Skill, diagnostics []diagnostic) ([]diagnostic, int) {
	byPath := make(map[string]discovery.Skill, len(skills))
	for _, skill := range skills {
		byPath[skill.Path] = skill
	}

	fixed := 0
	out := make([]diagnostic, 0, len(diagnostics))
	for _, d := range diagnostics {
		if d.Severity != severityError || !strings.HasPrefix(d.Message, fixableMessage) {
			out = append(out, d)
			continue
		}

		skillFile := filepath.Join(root, filepath.FromSlash(byPath[d.Path].Path), "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			d.Message = fmt.Sprintf("could not read SKILL.md to fix it: %v", err)
			out = append(out, d)
			continue
		}
		stripped, err := frontmatter.StripInstallMetadata(string(data))
		if err != nil {
			d.Message = fmt.Sprintf("could not strip install metadata: %v", err)
			out = append(out, d)
			continue
		}
		if err := os.WriteFile(skillFile, []byte(stripped), 0o644); err != nil {
			d.Message = fmt.Sprintf("could not write fixed SKILL.md: %v", err)
			out = append(out, d)
			continue
		}

		keys := strings.TrimSuffix(strings.TrimPrefix(d.Message, fixableMessage+": "), " (use --fix)")
		d.Severity = severityFixed
		d.Message = "stripped install metadata: " + keys
		out = append(out, d)
		fixed++
	}
	return out, fixed
}

// checkUnrecognisedSkillFiles reports SKILL.md files that discovery did not
// pick up. Discovery silently skips a skill whose directory or frontmatter name
// is not usable, which for a validation command would be the worst thing to
// stay quiet about.
func checkUnrecognisedSkillFiles(root string, skills []discovery.Skill) []diagnostic {
	found := make(map[string]bool, len(skills))
	for _, skill := range skills {
		found[path.Join(skill.Path, "SKILL.md")] = true
	}

	var diagnostics []diagnostic
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // an unreadable directory is not this command's problem
		}
		name := d.Name()
		if d.IsDir() {
			// Hidden directories hold installed copies, not authored skills.
			if p != root && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if name != "SKILL.md" {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil //nolint:nilerr // skip anything we cannot name
		}
		slashed := filepath.ToSlash(rel)
		if found[slashed] {
			return nil
		}
		diagnostics = append(diagnostics, diagnostic{
			Path:     path.Dir(slashed),
			Severity: severityError,
			Message: fmt.Sprintf("%s is not a recognised skill: its directory name must be lowercase alphanumeric with hyphens, "+
				"and it must sit in skills/, skills/{author}/, plugins/{name}/skills/, or at the repository root", slashed),
		})
		return nil
	})
	return diagnostics
}

// checkInstalledSkillDirs warns when a directory an agent installs into is not
// gitignored: publishing it would ship other authors' skills. Outside a git
// work tree there is nothing to check.
func checkInstalledSkillDirs(ctx context.Context, root string) []diagnostic {
	if !insideWorkTree(ctx, root) {
		return nil
	}

	var diagnostics []diagnostic
	for _, dir := range registry.UniqueProjectDirs() {
		if !strings.HasPrefix(dir, ".") {
			// The repository's own skills/ directory is what publish is for.
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(dir))
		if _, err := os.Stat(full); err != nil {
			continue
		}
		ignored, err := gitIgnores(ctx, root, dir)
		switch {
		case err != nil:
			diagnostics = append(diagnostics, diagnostic{
				Severity: severityWarning,
				Message:  fmt.Sprintf("%s/ may contain installed skills that are not gitignored (could not verify: %v)", dir, err),
			})
		case !ignored:
			diagnostics = append(diagnostics, diagnostic{
				Severity: severityWarning,
				Message:  fmt.Sprintf("%s/ contains installed skills and should be added to .gitignore to avoid publishing other authors' content", dir),
			})
		}
	}
	return diagnostics
}

func insideWorkTree(ctx context.Context, root string) bool {
	out, err := gitOutput(ctx, root, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// gitIgnores reports whether git ignores the given repository-relative path.
func gitIgnores(ctx context.Context, root, relPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "check-ignore", "-q", "--", relPath)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	// git check-ignore exits 1 when the path is not ignored, and 128 on error.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, gitError(err, stderr.String())
}

// gitError folds git's stderr into the error, so callers do not report a bare
// exit status to the user.
func gitError(err error, stderr string) error {
	if msg := strings.TrimSpace(stderr); msg != "" {
		return fmt.Errorf("%s", msg)
	}
	return err
}

func countSeverity(diagnostics []diagnostic, sev severity) int {
	n := 0
	for _, d := range diagnostics {
		if d.Severity == sev {
			n++
		}
	}
	return n
}

// renderDiagnostics reports the validation result, honouring the global output flags.
func renderDiagnostics(cmd *cobra.Command, ios *iostreams.IOStreams, diagnostics []diagnostic, skillCount, errCount, warnCount int) error {
	sort.SliceStable(diagnostics, func(i, j int) bool { return diagnostics[i].Skill < diagnostics[j].Skill })

	return cmdutil.WriteOutput(cmd, ios.Out, diagnostics, func() error {
		return writeDiagnostics(ios.Out, ios.IsStdoutTTY(), diagnostics, skillCount, errCount, warnCount)
	})
}

// writeDiagnostics renders the human-readable report: a tab-separated table
// when the output is piped, icons and a summary when it is a terminal.
func writeDiagnostics(w io.Writer, tty bool, diagnostics []diagnostic, skillCount, errCount, warnCount int) error {
	if len(diagnostics) == 0 {
		_, err := fmt.Fprintf(w, "✓ %s validated successfully\n", pluralize(skillCount, "skill"))
		return err
	}

	if !tty {
		tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		for _, d := range diagnostics {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", d.Severity, sanitizeForTerminal(d.Skill), sanitizeForTerminal(d.Message))
		}
		return tw.Flush()
	}

	icons := map[severity]string{severityError: "✗", severityWarning: "!", severityFixed: "✓"}
	for _, d := range diagnostics {
		if d.Skill == "" {
			fmt.Fprintf(w, "%s %s\n", icons[d.Severity], sanitizeForTerminal(d.Message))
		} else {
			fmt.Fprintf(w, "%s %s: %s\n", icons[d.Severity], sanitizeForTerminal(d.Skill), sanitizeForTerminal(d.Message))
		}
	}
	fmt.Fprintln(w)
	switch {
	case errCount > 0:
		fmt.Fprintf(w, "%d error(s), %d warning(s)\n", errCount, warnCount)
	case warnCount > 0:
		fmt.Fprintf(w, "%d warning(s)\n", warnCount)
	}
	return nil
}

// publishTag creates the requested tag on the current commit.
func publishTag(cmd *cobra.Command, f *cmdutil.Factory, ios *iostreams.IOStreams, root string, skills []discovery.Skill, diagnostics []diagnostic, opts *publishOptions) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	locator, err := remote.Detect(root)
	if err != nil {
		return fmt.Errorf("could not determine the Bitbucket repository to publish to: %w", err)
	}
	if err := ensurePublishedFilesCommitted(ctx, root, skills); err != nil {
		return err
	}
	owner := locator.Workspace
	if owner == "" {
		owner = locator.ProjectKey
	}
	repoArg := publishRepositoryURL(locator, owner)

	repo, err := openRepositoryFunc(cmd, f, repoArg)
	if err != nil {
		return err
	}
	tagger, ok := repo.(source.TagCreator)
	if !ok {
		return fmt.Errorf("this Bitbucket host does not support creating tags")
	}

	commit, err := gitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("could not determine the current commit: %w", err)
	}

	// A tag on a commit the remote does not have would be unusable.
	if err := ensurePushed(ctx, ios, root, commit, repo); err != nil {
		return err
	}

	switch existing, err := repo.Tag(ctx, opts.Tag); {
	case err == nil:
		return fmt.Errorf("tag %s already exists in %s (at %s); choose a different version", opts.Tag, repo.FullName(), shortSHA(existing))
	case !errors.Is(err, source.ErrNotFound):
		return fmt.Errorf("could not check whether tag %s exists in %s: %w", opts.Tag, repo.FullName(), err)
	}

	message := opts.Message
	if message == "" {
		message = opts.Tag
	}
	if err := tagger.CreateTag(ctx, opts.Tag, commit, message); err != nil {
		return fmt.Errorf("could not create tag %s in %s: %w", opts.Tag, repo.FullName(), err)
	}

	result := publishResult{
		Diagnostics:    diagnostics,
		Tag:            opts.Tag,
		Commit:         commit,
		Repository:     repo.FullName(),
		InstallCommand: fmt.Sprintf("bkt skill install %s", repo.FullName()),
		PinCommand:     fmt.Sprintf("bkt skill install %s <skill> --pin %s", repo.FullName(), opts.Tag),
	}
	if err := cmdutil.WriteOutput(cmd, ios.Out, result, func() error {
		fmt.Fprintf(ios.Out, "✓ Published %s (%s) in %s\n", opts.Tag, shortSHA(commit), repo.FullName())
		fmt.Fprintf(ios.Out, "  Install with: %s\n", result.InstallCommand)
		fmt.Fprintf(ios.Out, "  Pin with:     %s\n", result.PinCommand)
		return nil
	}); err != nil {
		return err
	}
	if next := suggestNextTag(opts.Tag); next != "" {
		fmt.Fprintf(ios.ErrOut, "\nNext version would be %s.\n", next)
	}
	return nil
}

// ensurePublishedFilesCommitted makes validation and tagging refer to the same
// bytes. The directory can be a repository root or a selected subtree, so only
// changes inside that directory block publishing.
func ensurePublishedFilesCommitted(ctx context.Context, root string, skills []discovery.Skill) error {
	status, err := gitOutput(ctx, root, "status", "--porcelain", "--untracked-files=all", "--", ".")
	if err != nil {
		return fmt.Errorf("could not verify that the published files are committed: %w", err)
	}
	if status != "" {
		return fmt.Errorf("cannot publish with uncommitted changes in %s; commit or discard them, then retry", root)
	}
	for _, skill := range skills {
		skillFile := filepath.Join(filepath.FromSlash(skill.Path), "SKILL.md")
		if _, err := gitOutput(ctx, root, "ls-files", "--error-unmatch", "--", skillFile); err != nil {
			return fmt.Errorf("cannot publish %s because it is not committed; add and commit it, then retry", filepath.Join(root, skillFile))
		}
	}
	return nil
}

func publishRepositoryURL(locator remote.Locator, owner string) string {
	if locator.Kind == "dc" {
		return fmt.Sprintf("https://%s/scm/%s/%s.git", locator.Host, owner, locator.RepoSlug)
	}
	return fmt.Sprintf("https://%s/%s/%s.git", locator.Host, owner, locator.RepoSlug)
}

// ensurePushed refuses to tag a commit the remote does not have yet.
func ensurePushed(ctx context.Context, ios *iostreams.IOStreams, root, commit string, repo source.Repository) error {
	_, err := repo.Commit(ctx, commit)
	if err == nil {
		return nil
	}
	if !errors.Is(err, source.ErrNotFound) {
		return fmt.Errorf("could not check whether %s has commit %s: %w", repo.FullName(), shortSHA(commit), err)
	}
	branch, branchErr := gitOutput(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
	hint := "push the current branch first"
	if branchErr == nil && branch != "HEAD" {
		hint = fmt.Sprintf("run `git push origin %s` first", branch)
	}
	fmt.Fprintf(ios.ErrOut, "! Commit %s is not on %s.\n", shortSHA(commit), repo.FullName())
	return fmt.Errorf("cannot tag a commit the remote does not have; %s", hint)
}

func gitOutput(ctx context.Context, root string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", gitError(err, stderr.String())
	}
	return strings.TrimSpace(string(out)), nil
}

var semverTag = regexp.MustCompile(`^(v?)(\d+)\.(\d+)\.(\d+)$`)

// suggestNextTag bumps the patch component of a semver tag, keeping any v prefix.
// It returns "" for tags that are not semver.
func suggestNextTag(tag string) string {
	m := semverTag.FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	patch, err := strconv.Atoi(m[4])
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s%s.%s.%d", m[1], m[2], m[3], patch+1)
}
