package skill

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/sourcetest"
)

// writeSkill creates a skill directory with the given SKILL.md content.
func writeSkill(t *testing.T, root, relDir, content string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(relDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// validSkill is a SKILL.md that passes every check, including the recommendations.
func validSkill(name string) string {
	return "---\nname: " + name + "\ndescription: A skill that does something useful.\nlicense: MIT\n---\n# Body\n"
}

func TestPublishValidatesCleanRepository(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))
	writeSkill(t, root, "skills/acme/git-commit", validSkill("git-commit"))

	if err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run"); err != nil {
		t.Fatalf("publish --dry-run: %v (stderr=%s)", err, stderr)
	}
	if !strings.Contains(stdout.String(), "✓ 2 skills validated successfully") {
		t.Errorf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr.String(), "Dry run complete. Re-run with --tag <version> to publish.") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestPublishValidationFindings(t *testing.T) {
	tests := []struct {
		name     string
		relDir   string
		content  string
		severity string
		want     string
	}{
		{
			name:     "name does not match the directory",
			relDir:   "skills/code-review",
			content:  "---\nname: reviewer\ndescription: d\nlicense: MIT\n---\n",
			severity: "error",
			want:     `name "reviewer" does not match directory name "code-review"`,
		},
		{
			name:     "name breaks the spec naming rules",
			relDir:   "skills/Code_Review",
			content:  "---\nname: Code_Review\ndescription: d\nlicense: MIT\n---\n",
			severity: "error",
			want:     "does not follow the agentskills.io naming convention",
		},
		{
			name:     "missing name",
			relDir:   "skills/code-review",
			content:  "---\ndescription: d\nlicense: MIT\n---\n",
			severity: "error",
			want:     "missing required field: name",
		},
		{
			name:     "missing description",
			relDir:   "skills/code-review",
			content:  "---\nname: code-review\nlicense: MIT\n---\n",
			severity: "error",
			want:     "missing required field: description",
		},
		{
			name:     "allowed-tools as a list",
			relDir:   "skills/code-review",
			content:  "---\nname: code-review\ndescription: d\nlicense: MIT\nallowed-tools:\n  - Read\n  - Bash\n---\n",
			severity: "error",
			want:     "allowed-tools must be a string (space-delimited), not a list",
		},
		{
			name:     "committed install metadata",
			relDir:   "skills/code-review",
			content:  "---\nname: code-review\ndescription: d\nlicense: MIT\nmetadata:\n    bitbucket-repo: https://bitbucket.org/other/skills\n    bitbucket-ref: refs/tags/v1\n---\n",
			severity: "error",
			want:     "contains install metadata that must be stripped: bitbucket-ref, bitbucket-repo (use --fix)",
		},
		{
			name:     "invalid frontmatter",
			relDir:   "skills/code-review",
			content:  "---\n: bad [[\n---\n",
			severity: "error",
			want:     "invalid frontmatter YAML",
		},
		{
			name:     "missing license is only a warning",
			relDir:   "skills/code-review",
			content:  "---\nname: code-review\ndescription: d\n---\n",
			severity: "warning",
			want:     "recommended field missing: license",
		},
		{
			name:     "over-long description is only a warning",
			relDir:   "skills/code-review",
			content:  "---\nname: code-review\nlicense: MIT\ndescription: " + strings.Repeat("x", 1100) + "\n---\n",
			severity: "warning",
			want:     "description is 1100 chars (recommended max: 1024)",
		},
		{
			name:     "over-long body is only a warning",
			relDir:   "skills/code-review",
			content:  "---\nname: code-review\ndescription: d\nlicense: MIT\n---\n" + strings.Repeat("line\n", 600),
			severity: "warning",
			want:     "skill body is 601 lines (recommended max: 500 for efficient context)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, stdout, stderr := newTestFactory(t)
			root := t.TempDir()
			writeSkill(t, root, tt.relDir, tt.content)

			err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run")
			out := stdout.String()

			if !strings.Contains(out, tt.want) {
				t.Fatalf("output missing %q:\n%s", tt.want, out)
			}
			if !strings.Contains(out, tt.severity) {
				t.Errorf("output should mark this as a %s:\n%s", tt.severity, out)
			}
			if tt.severity == "error" {
				if err == nil || !strings.Contains(err.Error(), "validation failed with") {
					t.Fatalf("error = %v, want a validation failure", err)
				}
			} else if err != nil {
				t.Fatalf("a warning must not fail validation: %v", err)
			}
		})
	}
}

func TestPublishFixStripsInstallMetadata(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review",
		"---\nname: code-review\ndescription: d\nlicense: MIT\nmetadata:\n    bitbucket-repo: https://bitbucket.org/other/skills\n    bitbucket-commit: abc\n    author: monalisa\n---\n# Body\n")

	if err := runSkill(t, f, stdout, stderr, "publish", root, "--fix"); err != nil {
		t.Fatalf("publish --fix: %v (stderr=%s)", err, stderr)
	}
	if !strings.Contains(stdout.String(), "stripped install metadata: bitbucket-commit, bitbucket-repo") {
		t.Errorf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr.String(), "Fixed 1 file(s)") {
		t.Errorf("stderr = %q", stderr)
	}

	fixed := readFile(t, filepath.Join(root, "skills", "code-review", "SKILL.md"))
	for _, gone := range []string{"bitbucket-repo", "bitbucket-commit"} {
		if strings.Contains(fixed, gone) {
			t.Errorf("%s survived --fix:\n%s", gone, fixed)
		}
	}
	// Unrelated metadata and the body are the author's, so they stay.
	for _, kept := range []string{"author: monalisa", "# Body", "name: code-review"} {
		if !strings.Contains(fixed, kept) {
			t.Errorf("--fix removed %q:\n%s", kept, fixed)
		}
	}

	// The repository now validates.
	if err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run"); err != nil {
		t.Fatalf("after --fix the repository should validate: %v (stdout=%s)", err, stdout)
	}
}

func TestPublishFixWithNothingToFix(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))

	if err := runSkill(t, f, stdout, stderr, "publish", root, "--fix"); err != nil {
		t.Fatalf("publish --fix: %v", err)
	}
	if !strings.Contains(stderr.String(), "No issues to fix.") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestPublishWarnsAboutInstallDirectoriesThatAreNotIgnored(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))
	// An install directory git is not ignoring would be published alongside
	// the repository's own skills.
	writeSkill(t, root, ".claude/skills/someone-elses", validSkill("someone-elses"))

	runGit(t, root, "init")

	if err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run"); err != nil {
		t.Fatalf("publish --dry-run: %v (stdout=%s)", err, stdout)
	}
	if !strings.Contains(stdout.String(), ".claude/skills/ contains installed skills and should be added to .gitignore") {
		t.Fatalf("expected a warning about the tracked install directory:\n%s", stdout)
	}

	// Ignoring it clears the warning.
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".claude/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run"); err != nil {
		t.Fatalf("publish --dry-run: %v", err)
	}
	if strings.Contains(stdout.String(), "should be added to .gitignore") {
		t.Errorf("the warning should be gone once the directory is ignored:\n%s", stdout)
	}
}

func TestPublishIgnoresHiddenDirSkills(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))
	// An installed copy would fail the name check; publish must not validate it.
	writeSkill(t, root, ".agents/skills/mirrored", "---\nname: WRONG\n---\n")

	if err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run"); err != nil {
		t.Fatalf("publish --dry-run: %v (stdout=%s)", err, stdout)
	}
	if strings.Contains(stdout.String(), "mirrored") {
		t.Errorf("installed copies must not be validated:\n%s", stdout)
	}
}

func TestPublishJSONOutput(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", "---\nname: code-review\ndescription: d\n---\n")

	cmd := NewCmdSkill(f)
	rootCmd := &cobra.Command{Use: "bkt"}
	rootCmd.PersistentFlags().Bool("json", false, "")
	rootCmd.PersistentFlags().Bool("yaml", false, "")
	rootCmd.PersistentFlags().String("format", "", "")
	rootCmd.PersistentFlags().String("jq", "", "")
	rootCmd.PersistentFlags().String("template", "", "")
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{"skill", "publish", root, "--dry-run", "--json"})
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetContext(context.Background())
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("publish --json: %v (stderr=%s)", err, stderr)
	}

	var diagnostics []map[string]any
	if err := json.Unmarshal([]byte(stdout.String()), &diagnostics); err != nil {
		t.Fatalf("parse JSON %q: %v", stdout, err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v, want the missing-license warning", diagnostics)
	}
	if diagnostics[0]["severity"] != "warning" || diagnostics[0]["skill"] != "code-review" {
		t.Fatalf("diagnostic = %+v", diagnostics[0])
	}
}

func TestPublishFlagConflicts(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "fix and tag", args: []string{"publish", "--fix", "--tag", "v1.0.0"}, wantErr: "--fix and --tag cannot be used together"},
		{name: "dry-run and tag", args: []string{"publish", "--dry-run", "--tag", "v1.0.0"}, wantErr: "--dry-run and --tag cannot be used together"},
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

func TestPublishWithoutTagStopsAfterValidation(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))

	if err := runSkill(t, f, stdout, stderr, "publish", root); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !strings.Contains(stderr.String(), "Validation passed. Re-run with --tag <version> to publish.") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestPublishNoSkills(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	err := runSkill(t, f, stdout, stderr, "publish", t.TempDir(), "--dry-run")
	if err == nil || !strings.Contains(err.Error(), "no skills found in") {
		t.Fatalf("error = %v", err)
	}
}

func TestSuggestNextTag(t *testing.T) {
	tests := []struct{ in, want string }{
		{"v1.2.3", "v1.2.4"},
		{"1.2.3", "1.2.4"},
		{"v0.0.9", "v0.0.10"},
		{"v1.2", ""},
		{"release-2024", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := suggestNextTag(tt.in); got != tt.want {
				t.Fatalf("suggestNextTag(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// runGit runs git in root with a throwaway identity and configuration, so the
// developer's own git config (commit signing, hooks) cannot break these tests.
func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_CONFIG_GLOBAL="+filepath.Join(t.TempDir(), "gitconfig"),
		"GIT_CONFIG_NOSYSTEM=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// setupTagRepo creates a git repository with one skill and a Bitbucket Cloud
// remote, returning the repository root and its HEAD commit.
func setupTagRepo(t *testing.T) (root, head string) {
	t.Helper()
	return setupTagRepoWithRemote(t, "https://bitbucket.org/myteam/agent-skills.git")
}

// setupTagRepoWithRemote is setupTagRepo with an explicit remote URL, so both
// Bitbucket platforms can be exercised.
func setupTagRepoWithRemote(t *testing.T, remoteURL string) (root, head string) {
	t.Helper()
	root = t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))

	runGit(t, root, "init")
	runGit(t, root, "remote", "add", "origin", remoteURL)
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-m", "add skill")
	return root, runGit(t, root, "rev-parse", "HEAD")
}

func TestPublishCreatesTag(t *testing.T) {
	root, head := setupTagRepo(t)

	repo := sourcetest.New("myteam/agent-skills", nil)
	repo.Branches = map[string]string{"main": head}
	args := stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0"); err != nil {
		t.Fatalf("publish --tag: %v (stderr=%s)", err, stderr)
	}

	// The repository to publish to is derived from the git remote.
	if len(*args) != 1 || (*args)[0] != "myteam/agent-skills" {
		t.Fatalf("repository arguments = %v", *args)
	}
	if repo.CreatedTags["v1.0.0"] != head {
		t.Fatalf("created tags = %v, want v1.0.0 at %s", repo.CreatedTags, head)
	}
	if len(repo.CreatedTagMessages) != 1 || repo.CreatedTagMessages[0] != "v1.0.0" {
		t.Errorf("tag message = %v, want the tag name by default", repo.CreatedTagMessages)
	}
	for _, want := range []string{"✓ Published v1.0.0", "Install with: bkt skill install myteam/agent-skills", "--pin v1.0.0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if !strings.Contains(stderr.String(), "Next version would be v1.0.1.") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestPublishTagCustomMessage(t *testing.T) {
	root, head := setupTagRepo(t)
	repo := sourcetest.New("myteam/agent-skills", nil)
	repo.Branches = map[string]string{"main": head}
	stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0", "--message", "First release"); err != nil {
		t.Fatalf("publish --tag: %v (stderr=%s)", err, stderr)
	}
	if len(repo.CreatedTagMessages) != 1 || repo.CreatedTagMessages[0] != "First release" {
		t.Fatalf("tag message = %v", repo.CreatedTagMessages)
	}
}

func TestPublishTagErrors(t *testing.T) {
	t.Run("tag already exists", func(t *testing.T) {
		root, head := setupTagRepo(t)
		repo := sourcetest.New("myteam/agent-skills", nil)
		repo.Branches = map[string]string{"main": head}
		repo.Tags = map[string]string{"v1.0.0": "oldercommit"}
		stubRepository(t, repo)

		f, stdout, stderr := newTestFactory(t)
		err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0")
		if err == nil || !strings.Contains(err.Error(), "tag v1.0.0 already exists in myteam/agent-skills") {
			t.Fatalf("error = %v", err)
		}
		if len(repo.CreatedTags) != 0 {
			t.Errorf("no tag should have been created, got %v", repo.CreatedTags)
		}
	})

	t.Run("commit not pushed", func(t *testing.T) {
		root, _ := setupTagRepo(t)
		// The remote knows nothing about the local commit.
		repo := sourcetest.New("myteam/agent-skills", nil)
		repo.Branches = map[string]string{"main": "someothercommit"}
		stubRepository(t, repo)

		f, stdout, stderr := newTestFactory(t)
		err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0")
		if err == nil || !strings.Contains(err.Error(), "cannot tag a commit the remote does not have") {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(err.Error(), "git push origin") {
			t.Errorf("the error should say how to fix it: %v", err)
		}
		if len(repo.CreatedTags) != 0 {
			t.Errorf("no tag should have been created, got %v", repo.CreatedTags)
		}
	})

	t.Run("no bitbucket remote", func(t *testing.T) {
		root := t.TempDir()
		writeSkill(t, root, "skills/code-review", validSkill("code-review"))
		runGit(t, root, "init")

		f, stdout, stderr := newTestFactory(t)
		err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0")
		if err == nil || !strings.Contains(err.Error(), "could not determine the Bitbucket repository to publish to") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("validation errors block tagging", func(t *testing.T) {
		root, _ := setupTagRepo(t)
		writeSkill(t, root, "skills/broken", "---\nname: WRONG\ndescription: d\n---\n")
		repo := sourcetest.New("myteam/agent-skills", nil)
		stubRepository(t, repo)

		f, stdout, stderr := newTestFactory(t)
		err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0")
		if err == nil || !strings.Contains(err.Error(), "validation failed with") {
			t.Fatalf("error = %v", err)
		}
		if len(repo.CreatedTags) != 0 {
			t.Errorf("no tag should have been created, got %v", repo.CreatedTags)
		}
	})
}
