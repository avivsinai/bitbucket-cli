package skill

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/internal/skills/sourcetest"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// newTestFactory returns a factory writing to buffers, with an isolated config
// directory so the developer's real ~/.config/bkt is never touched.
func newTestFactory(t *testing.T) (*cmdutil.Factory, *strings.Builder, *strings.Builder) {
	t.Helper()
	t.Setenv("BKT_CONFIG_DIR", t.TempDir())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	var stdout, stderr strings.Builder
	f := &cmdutil.Factory{
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    &stdout,
			ErrOut: &stderr,
			In:     io.NopCloser(strings.NewReader("")),
		},
		Config: func() (*config.Config, error) { return cfg, nil },
	}
	return f, &stdout, &stderr
}

// stubRepository makes every repository argument resolve to repo, and records
// the arguments the command asked for.
func stubRepository(t *testing.T, repo source.Repository) *[]string {
	t.Helper()
	var args []string
	original := openRepositoryFunc
	openRepositoryFunc = func(_ *cobra.Command, _ *cmdutil.Factory, arg string) (source.Repository, error) {
		args = append(args, arg)
		return repo, nil
	}
	t.Cleanup(func() { openRepositoryFunc = original })
	return &args
}

// runSkill executes "bkt skill ..." against the factory and returns stdout/stderr.
func runSkill(t *testing.T, f *cmdutil.Factory, stdout, stderr *strings.Builder, args ...string) error {
	t.Helper()
	stdout.Reset()
	stderr.Reset()

	cmd := NewCmdSkill(f)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetContext(context.Background())
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return cmd.Execute()
}

func newSkillsRepo() *sourcetest.Repo {
	r := sourcetest.New("myteam/agent-skills", map[string]string{
		"README.md":                       "# skills",
		"skills/alpha/SKILL.md":           "---\nname: alpha\ndescription: Alpha skill\n---\n# Alpha\n",
		"skills/alpha/scripts/run.sh":     "#!/bin/sh\necho hi\n",
		"skills/beta/SKILL.md":            "---\nname: beta\ndescription: Beta skill\n---\n",
		"skills/monalisa/triage/SKILL.md": "---\nname: triage\ndescription: Namespaced skill\n---\n",
	})
	r.Tags = map[string]string{"v1.0.0": "tagsha"}
	r.TagOrder = []string{"v1.0.0"}
	r.Commits = map[string]string{
		"skills/alpha":           "alpha-commit",
		"skills/beta":            "beta-commit",
		"skills/monalisa/triage": "triage-commit",
	}
	return r
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestInstallListsSkillsWhenNoneNamed(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	stubRepository(t, newSkillsRepo())

	if err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills"); err != nil {
		t.Fatalf("install: %v (stderr=%s)", err, stderr)
	}

	// Non-interactive output is tab separated so it can be piped into grep.
	got := stdout.String()
	for _, want := range []string{"alpha\tAlpha skill\n", "beta\tBeta skill\n", "monalisa/triage\tNamespaced skill\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("stdout missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "SKILL\tDESCRIPTION") {
		t.Error("non-TTY listing must not print a header row")
	}
	if !strings.Contains(stderr.String(), "Using ref v1.0.0 (tagsha)") {
		t.Errorf("stderr should report the resolved ref:\n%s", stderr)
	}
}

func TestInstallSkillByName(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	repo := newSkillsRepo()
	stubRepository(t, repo)
	target := t.TempDir()

	if err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "alpha", "--dir", target); err != nil {
		t.Fatalf("install: %v (stderr=%s)", err, stderr)
	}

	skillMD := readFile(t, filepath.Join(target, "alpha", "SKILL.md"))
	for _, want := range []string{
		"bitbucket-repo: https://bitbucket.org/myteam/agent-skills",
		"bitbucket-ref: refs/tags/v1.0.0",
		"bitbucket-commit: alpha-commit",
		"bitbucket-path: skills/alpha",
		"# Alpha",
	} {
		if !strings.Contains(skillMD, want) {
			t.Errorf("installed SKILL.md missing %q:\n%s", want, skillMD)
		}
	}
	if strings.Contains(skillMD, "bitbucket-pinned") {
		t.Error("an unpinned install must not record a pin")
	}
	if got := readFile(t, filepath.Join(target, "alpha", "scripts", "run.sh")); got != "#!/bin/sh\necho hi\n" {
		t.Errorf("nested file = %q", got)
	}

	if !strings.Contains(stdout.String(), "✓ Installed alpha (from myteam/agent-skills@v1.0.0)") {
		t.Errorf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr.String(), "Skills are not verified by bkt or Bitbucket") {
		t.Errorf("the pre-install disclaimer must be shown:\n%s", stderr)
	}
	if !strings.Contains(stderr.String(), "bkt skill preview myteam/agent-skills alpha@tagsha") {
		t.Errorf("review hint should pin the exact commit:\n%s", stderr)
	}
}

func TestInstallPinnedVersionRecordsPin(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	repo := newSkillsRepo()
	repo.Branches["main"] = "sha-main"
	stubRepository(t, repo)
	target := t.TempDir()

	if err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "alpha@v1.0.0", "--dir", target); err != nil {
		t.Fatalf("install: %v (stderr=%s)", err, stderr)
	}
	if !strings.Contains(readFile(t, filepath.Join(target, "alpha", "SKILL.md")), "bitbucket-pinned: v1.0.0") {
		t.Error("an @version install must record the pin")
	}
}

func TestInstallByExactPathSkipsDiscovery(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	repo := newSkillsRepo()
	// Make full discovery impossible: only the by-path lookup can succeed.
	repo.Files["skills/broken/SKILL.md"] = "ok"
	stubRepository(t, repo)
	target := t.TempDir()

	if err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "skills/monalisa/triage", "--dir", target); err != nil {
		t.Fatalf("install: %v (stderr=%s)", err, stderr)
	}
	// Namespaced skills are installed flat by their base name.
	if _, err := os.Stat(filepath.Join(target, "triage", "SKILL.md")); err != nil {
		t.Fatalf("skill not installed flat: %v", err)
	}
	if !strings.Contains(stdout.String(), "Installed monalisa/triage") {
		t.Errorf("stdout should use the namespaced install name: %q", stdout)
	}
}

func TestInstallErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(t *testing.T, repo *sourcetest.Repo) []string
		wantErr string
	}{
		{
			name:    "unknown skill",
			args:    []string{"install", "myteam/agent-skills", "nope"},
			wantErr: `skill "nope" not found in myteam/agent-skills`,
		},
		{
			name:    "--all with a skill argument",
			args:    []string{"install", "myteam/agent-skills", "alpha", "--all"},
			wantErr: "cannot use --all with a skill argument",
		},
		{
			name:    "--pin with inline version",
			args:    []string{"install", "myteam/agent-skills", "alpha@v1", "--pin", "v2"},
			wantErr: "cannot use --pin with an inline @version",
		},
		{
			name:    "--from-local without a path",
			args:    []string{"install", "--from-local"},
			wantErr: "--from-local requires a directory path argument",
		},
		{
			name:    "unknown agent",
			args:    []string{"install", "myteam/agent-skills", "alpha", "--agent", "nope"},
			wantErr: `unknown agent "nope"`,
		},
		{
			name:    "invalid scope",
			args:    []string{"install", "myteam/agent-skills", "alpha", "--scope", "global"},
			wantErr: `--scope must be "project" or "user"`,
		},
		{
			name:    "no repository non-interactively",
			args:    []string{"install"},
			wantErr: "must specify a repository to install from",
		},
		{
			// Two namespaced skills share a base name and neither display name
			// matches exactly, so the user must disambiguate.
			name: "ambiguous skill name",
			args: []string{"install", "myteam/agent-skills", "review"},
			setup: func(t *testing.T, repo *sourcetest.Repo) []string {
				repo.Files["skills/one/review/SKILL.md"] = "---\nname: review\n---\n"
				repo.Files["skills/two/review/SKILL.md"] = "---\nname: review\n---\n"
				return nil
			},
			wantErr: `skill name "review" is ambiguous`,
		},
		{
			name: "colliding names with --all",
			args: []string{"install", "myteam/agent-skills", "--all"},
			setup: func(t *testing.T, repo *sourcetest.Repo) []string {
				repo.Files["skills/other/alpha/SKILL.md"] = "---\nname: alpha\n---\n"
				return nil
			},
			wantErr: "cannot install skills with conflicting names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, stdout, stderr := newTestFactory(t)
			repo := newSkillsRepo()
			if tt.setup != nil {
				tt.setup(t, repo)
			}
			stubRepository(t, repo)

			args := append([]string{}, tt.args...)
			if tt.args[0] == "install" && len(tt.args) > 1 && !strings.Contains(strings.Join(tt.args, " "), "--dir") {
				args = append(args, "--dir", t.TempDir())
			}
			err := runSkill(t, f, stdout, stderr, args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q (stderr=%s)", err, tt.wantErr, stderr)
			}
		})
	}
}

func TestInstallExistingSkillRequiresForce(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	stubRepository(t, newSkillsRepo())
	target := t.TempDir()

	if err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "alpha", "--dir", target); err != nil {
		t.Fatalf("first install: %v", err)
	}

	err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "alpha", "--dir", target)
	if err == nil || !strings.Contains(err.Error(), "skills already installed: alpha (use --force to overwrite)") {
		t.Fatalf("error = %v, want the already-installed error", err)
	}

	if err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "alpha", "--dir", target, "--force"); err != nil {
		t.Fatalf("forced reinstall: %v (stderr=%s)", err, stderr)
	}
}

func TestInstallHiddenDirSkillsNeedFlag(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	repo := sourcetest.New("myteam/agent-skills", map[string]string{
		".claude/skills/hidden/SKILL.md": "---\nname: hidden\n---\n",
	})
	repo.TagOrder = nil
	stubRepository(t, repo)
	target := t.TempDir()

	err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "hidden", "--dir", target)
	if err == nil || !strings.Contains(err.Error(), "no standard skills found, but 1 skill(s) exist in hidden directories") {
		t.Fatalf("error = %v, want discovery to exclude hidden dirs and point at --allow-hidden-dirs", err)
	}

	if err := runSkill(t, f, stdout, stderr, "install", "myteam/agent-skills", "hidden", "--dir", target, "--allow-hidden-dirs"); err != nil {
		t.Fatalf("install with --allow-hidden-dirs: %v (stderr=%s)", err, stderr)
	}
	if _, statErr := os.Stat(filepath.Join(target, "hidden", "SKILL.md")); statErr != nil {
		t.Fatalf("hidden skill not installed: %v", statErr)
	}
	if !strings.Contains(stderr.String(), "may be installed") {
		t.Errorf("expected a provenance warning for hidden-dir skills:\n%s", stderr)
	}
}

func TestInstallFromLocalCopiesAndRecordsLocalPath(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	src := t.TempDir()
	skillDir := filepath.Join(src, "skills", "alpha")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: alpha\n---\n# Local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()

	if err := runSkill(t, f, stdout, stderr, "install", src, "alpha", "--from-local", "--dir", target); err != nil {
		t.Fatalf("install --from-local: %v (stderr=%s)", err, stderr)
	}
	installed := readFile(t, filepath.Join(target, "alpha", "SKILL.md"))
	if !strings.Contains(installed, "local-path:") || !strings.Contains(installed, "# Local") {
		t.Fatalf("installed SKILL.md = %q", installed)
	}
	if strings.Contains(installed, "bitbucket-repo") {
		t.Error("local installs must not record a repository")
	}
	if !strings.Contains(stdout.String(), "Installed alpha (from "+src+")") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestListReportsInstalledSkills(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	dir := t.TempDir()

	write := func(name, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("alpha", "---\nname: alpha\nmetadata:\n    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-ref: refs/tags/v1.0.0\n    bitbucket-commit: c1\n    bitbucket-path: skills/alpha\n    bitbucket-pinned: v1.0.0\n---\n")
	write("triage", "---\nname: triage\nmetadata:\n    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-ref: refs/heads/main\n    bitbucket-path: skills/monalisa/triage\n---\n")
	write("from-gh", "---\nname: from-gh\nmetadata:\n    github-repo: https://github.com/monalisa/skills\n    github-ref: refs/tags/v2\n    github-path: skills/from-gh\n---\n")
	write("authored", "---\nname: authored\n---\n")

	if err := runSkill(t, f, stdout, stderr, "list", "--dir", dir); err != nil {
		t.Fatalf("list: %v (stderr=%s)", err, stderr)
	}
	// Columns are name, agent, scope, source. A --dir scan has no agent host.
	got := stdout.String()
	for _, want := range []string{
		"alpha\t-\tcustom\tmyteam/agent-skills",
		"monalisa/triage\t-\tcustom\tmyteam/agent-skills",
		"from-gh\t-\tcustom\tgithub.com/monalisa/skills",
		"authored\t-\tcustom\t-",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("list output missing %q:\n%s", want, got)
		}
	}
	// The namespace is recovered from the recorded source path, since skills
	// are installed flat on disk.
	if strings.Contains(got, "triage\tcustom") && !strings.Contains(got, "monalisa/triage") {
		t.Error("namespaced skill should be listed as monalisa/triage")
	}
}

func TestListJSONOutput(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alpha", "SKILL.md"),
		[]byte("---\nname: alpha\nmetadata:\n    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-ref: refs/tags/v1.0.0\n    bitbucket-pinned: v1.0.0\n    bitbucket-path: skills/alpha\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewCmdSkill(f)
	// The --json flag is inherited from the root command in the real CLI; add
	// the same persistent flags here so WriteOutput sees them.
	root := &cobra.Command{Use: "bkt"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("yaml", false, "")
	root.PersistentFlags().String("format", "", "")
	root.PersistentFlags().String("jq", "", "")
	root.PersistentFlags().String("template", "", "")
	root.AddCommand(cmd)
	root.SetArgs([]string{"skill", "list", "--dir", dir, "--json"})
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetContext(context.Background())
	if err := root.Execute(); err != nil {
		t.Fatalf("list --json: %v (stderr=%s)", err, stderr)
	}

	var skills []map[string]any
	if err := json.Unmarshal([]byte(stdout.String()), &skills); err != nil {
		t.Fatalf("parse JSON %q: %v", stdout, err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills = %+v, want 1", skills)
	}
	got := skills[0]
	if got["skillName"] != "alpha" || got["version"] != "v1.0.0" || got["pinned"] != true {
		t.Fatalf("json = %+v", got)
	}
	if got["sourceURL"] != "https://bitbucket.org/myteam/agent-skills" {
		t.Errorf("sourceURL = %v", got["sourceURL"])
	}
}

func TestListEmptyDirectory(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "list", "--dir", t.TempDir()); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(stdout.String(), "No skills installed.") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestListFlagConflicts(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "dir and agent", args: []string{"list", "--dir", ".", "--agent", "claude-code"}, wantErr: "--dir and --agent cannot be used together"},
		{name: "dir and scope", args: []string{"list", "--dir", ".", "--scope", "user"}, wantErr: "--dir and --scope cannot be used together"},
		{name: "bad scope", args: []string{"list", "--scope", "global"}, wantErr: `--scope must be "project" or "user"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, stdout, stderr := newTestFactory(t)
			err := runSkill(t, f, stdout, stderr, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestPreviewRendersTreeAndContent(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	stubRepository(t, newSkillsRepo())

	if err := runSkill(t, f, stdout, stderr, "preview", "myteam/agent-skills", "alpha"); err != nil {
		t.Fatalf("preview: %v (stderr=%s)", err, stderr)
	}
	// Directories sort before files in the tree, matching gh skill preview.
	got := stdout.String()
	for _, want := range []string{"alpha/", "├── scripts/", "│   └── run.sh", "└── SKILL.md", "── SKILL.md ──", "# Alpha", "── scripts/run.sh ──", "echo hi"} {
		if !strings.Contains(got, want) {
			t.Errorf("preview output missing %q:\n%s", want, got)
		}
	}
}

func TestPreviewListsSkillsWithoutName(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	stubRepository(t, newSkillsRepo())

	if err := runSkill(t, f, stdout, stderr, "preview", "myteam/agent-skills"); err != nil {
		t.Fatalf("preview: %v (stderr=%s)", err, stderr)
	}
	if !strings.Contains(stdout.String(), "alpha\tAlpha skill") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestPreviewUnknownSkill(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	stubRepository(t, newSkillsRepo())

	err := runSkill(t, f, stdout, stderr, "preview", "myteam/agent-skills", "nope")
	if err == nil || !strings.Contains(err.Error(), `skill "nope" not found in myteam/agent-skills`) {
		t.Fatalf("error = %v", err)
	}
}

// installedSkillDir writes an installed skill with the given metadata block.
func installedSkillDir(t *testing.T, dir, name, metadata string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\nmetadata:\n" + metadata + "---\n# Old body\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "stale.txt"), []byte("remove me"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAppliesChangedSkills(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	repo := newSkillsRepo()
	stubRepository(t, repo)
	dir := t.TempDir()
	installedSkillDir(t, dir, "alpha", "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-ref: refs/tags/v1.0.0\n    bitbucket-commit: old-commit\n    bitbucket-path: skills/alpha\n")

	if err := runSkill(t, f, stdout, stderr, "update", "--dir", dir, "--all"); err != nil {
		t.Fatalf("update: %v (stderr=%s)", err, stderr)
	}
	if !strings.Contains(stdout.String(), "✓ Updated alpha") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout.String(), "old-com > alpha-c") {
		t.Errorf("expected the commit transition in the summary:\n%s", stdout)
	}

	updated := readFile(t, filepath.Join(dir, "alpha", "SKILL.md"))
	if !strings.Contains(updated, "bitbucket-commit: alpha-commit") || !strings.Contains(updated, "# Alpha") {
		t.Fatalf("updated SKILL.md = %q", updated)
	}
	// The staged swap must drop files that are no longer in the source.
	if _, err := os.Stat(filepath.Join(dir, "alpha", "stale.txt")); !os.IsNotExist(err) {
		t.Error("stale file survived the update")
	}
	if _, err := os.Stat(filepath.Join(dir, "alpha", "scripts", "run.sh")); err != nil {
		t.Errorf("new file missing after update: %v", err)
	}
}

func TestUpdateSkipsAndReports(t *testing.T) {
	tests := []struct {
		name       string
		metadata   string
		args       []string
		wantStderr string
		wantStdout string
		wantChange bool
	}{
		{
			name:       "up to date",
			metadata:   "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: alpha-commit\n    bitbucket-path: skills/alpha\n",
			args:       []string{"--all"},
			wantStderr: "All skills are up to date.",
		},
		{
			name:       "pinned is skipped",
			metadata:   "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: old\n    bitbucket-pinned: v1.0.0\n    bitbucket-path: skills/alpha\n",
			args:       []string{"--all"},
			wantStderr: "alpha is pinned to v1.0.0 (skipped)",
		},
		{
			name:       "unpin includes pinned skills",
			metadata:   "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: old\n    bitbucket-pinned: v1.0.0\n    bitbucket-path: skills/alpha\n",
			args:       []string{"--all", "--unpin"},
			wantStdout: "✓ Updated alpha",
			wantChange: true,
		},
		{
			name:       "dry run reports without writing",
			metadata:   "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: old\n    bitbucket-path: skills/alpha\n",
			args:       []string{"--dry-run"},
			wantStdout: "• alpha (myteam/agent-skills)",
		},
		{
			name:       "no metadata is skipped non-interactively",
			metadata:   "    author: monalisa\n",
			args:       []string{"--all"},
			wantStderr: "alpha has no Bitbucket metadata",
		},
		{
			name:       "gh-installed skills point at gh",
			metadata:   "    github-repo: https://github.com/monalisa/skills\n    github-tree-sha: abc\n",
			args:       []string{"--all"},
			wantStderr: "alpha was installed by gh; run `gh skill update alpha`",
		},
		{
			name:       "force re-downloads an up-to-date skill",
			metadata:   "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: alpha-commit\n    bitbucket-path: skills/alpha\n",
			args:       []string{"--all", "--force"},
			wantStdout: "✓ Updated alpha",
			wantChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, stdout, stderr := newTestFactory(t)
			stubRepository(t, newSkillsRepo())
			dir := t.TempDir()
			installedSkillDir(t, dir, "alpha", tt.metadata)

			args := append([]string{"update", "--dir", dir}, tt.args...)
			if err := runSkill(t, f, stdout, stderr, args...); err != nil {
				t.Fatalf("update: %v (stderr=%s)", err, stderr)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr missing %q:\n%s", tt.wantStderr, stderr)
			}
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout missing %q:\n%s", tt.wantStdout, stdout)
			}

			body := readFile(t, filepath.Join(dir, "alpha", "SKILL.md"))
			if tt.wantChange && strings.Contains(body, "# Old body") {
				t.Error("skill should have been rewritten")
			}
			if !tt.wantChange && !strings.Contains(body, "# Old body") {
				t.Error("skill should not have been rewritten")
			}
		})
	}
}

func TestUpdateNamedSkillNotInstalled(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	stubRepository(t, newSkillsRepo())
	dir := t.TempDir()
	installedSkillDir(t, dir, "alpha", "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: old\n")

	err := runSkill(t, f, stdout, stderr, "update", "--dir", dir, "nope")
	if err == nil || !strings.Contains(err.Error(), "none of the specified skills are installed") {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateNoSkillsInstalled(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "update", "--dir", t.TempDir()); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !strings.Contains(stderr.String(), "No installed skills found.") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestSkillNameFromSourcePath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"skills/alpha", "alpha"},
		{"skills/alpha/SKILL.md", "alpha"},
		{"skills/monalisa/triage", "monalisa/triage"},
		{"plugins/hubot/skills/pr-summary", "hubot/pr-summary"},
		{"a/b/c/skills/deep", "deep"},
		{"alpha", "alpha"},
		{"", ""},
		{"skills", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := skillNameFromSourcePath(tt.path); got != tt.want {
				t.Fatalf("skillNameFromSourcePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFriendlyDirAndTruncate(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if got := friendlyDir(filepath.Join(cwd, "sub", "dir")); got != filepath.Join("sub", "dir") {
		t.Errorf("friendlyDir inside cwd = %q", got)
	}
	if got := friendlyDir(cwd); got != filepath.Base(cwd) {
		t.Errorf("friendlyDir of cwd = %q", got)
	}
	if got := truncate("abcdefghij", 8); got != "abcde..." {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("abc", 8); got != "abc" {
		t.Errorf("truncate short string = %q", got)
	}
}

func TestSanitizeForTerminalRemovesEscapes(t *testing.T) {
	got := sanitizeForTerminal("safe\x1b[31mred\x07\ttab")
	if strings.ContainsRune(got, 0x1b) || strings.ContainsRune(got, 0x07) {
		t.Fatalf("control characters survived: %q", got)
	}
	if !strings.Contains(got, "safe") || !strings.Contains(got, "tab") {
		t.Fatalf("printable text was lost: %q", got)
	}
}

func TestPreviewSanitizesRepositoryControlledPaths(t *testing.T) {
	// A repository can name a file to inject terminal escapes; the tree and the
	// per-file headings must neutralise them like file contents already are.
	esc := "\x1b]0;pwned\x07"
	repo := sourcetest.New("myteam/agent-skills", map[string]string{
		"skills/alpha/SKILL.md":           "---\nname: alpha\n---\n# Alpha\n",
		"skills/alpha/evil" + esc + ".md": "content" + esc + "\n",
	})
	stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "preview", "myteam/agent-skills", "alpha"); err != nil {
		t.Fatalf("preview: %v (stderr=%s)", err, stderr)
	}
	out := stdout.String()
	if strings.ContainsRune(out, 0x1b) || strings.ContainsRune(out, 0x07) {
		t.Fatalf("terminal escapes reached stdout:\n%q", out)
	}
	if !strings.Contains(out, "evil") {
		t.Errorf("the file should still be listed by its printable name:\n%s", out)
	}
}

func TestInstalledTreeSanitizesNames(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "alpha")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Control characters are not legal in Windows filenames, so the escape is
	// applied to the skill name the caller passes in.
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	printInstalledTree(&buf, dir, []string{"alpha\x1b]0;pwned\x07"})
	if strings.ContainsRune(buf.String(), 0x1b) || strings.ContainsRune(buf.String(), 0x07) {
		t.Fatalf("terminal escapes reached the tree output:\n%q", buf.String())
	}
}
