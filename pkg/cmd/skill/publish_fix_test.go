package skill

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/internal/skills/sourcetest"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// These cover the behaviour that keeps publish from damaging a repository:
// validation never writes, and --fix only runs when it can finish the job.

func TestPublishDryRunNeverWritesFiles(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	content := "---\nname: code-review\ndescription: d\nlicense: MIT\nmetadata:\n    bitbucket-repo: r\n---\n"
	writeSkill(t, root, "skills/code-review", content)
	skillFile := filepath.Join(root, "skills", "code-review", "SKILL.md")

	// Combining the flags would otherwise rewrite files while reporting a dry run.
	err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run", "--fix")
	if err == nil || !strings.Contains(err.Error(), "--dry-run and --fix cannot be used together") {
		t.Fatalf("error = %v", err)
	}
	if got := readFile(t, skillFile); got != content {
		t.Fatalf("the file was modified:\n%s", got)
	}

	if err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run"); err == nil {
		t.Fatal("expected validation to fail")
	}
	if got := readFile(t, skillFile); got != content {
		t.Fatalf("--dry-run modified the file:\n%s", got)
	}
}

func TestPublishFixLeavesFilesAloneWhenOtherErrorsExist(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	// The name is wrong too, so --fix cannot make the repository valid and must
	// not mix its rewrite into changes the user still has to make by hand.
	content := "---\nname: wrong-name\ndescription: d\nlicense: MIT\nmetadata:\n    bitbucket-repo: r\n---\n"
	writeSkill(t, root, "skills/code-review", content)
	skillFile := filepath.Join(root, "skills", "code-review", "SKILL.md")

	err := runSkill(t, f, stdout, stderr, "publish", root, "--fix")
	if err == nil || !strings.Contains(err.Error(), "validation failed with") {
		t.Fatalf("error = %v", err)
	}
	if got := readFile(t, skillFile); got != content {
		t.Fatalf("--fix rewrote a file it could not make valid:\n%s", got)
	}
	if !strings.Contains(stdout.String(), "(use --fix)") {
		t.Errorf("the metadata error should still be reported:\n%s", stdout)
	}
}

func TestPublishReportsUnrecognisedSkillFiles(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))
	// Discovery silently skips a directory name it cannot use; a validation
	// command must not stay quiet about it.
	writeSkill(t, root, "skills/Bad Name!", "---\nname: whatever\n---\n")

	err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run")
	if err == nil || !strings.Contains(err.Error(), "validation failed with") {
		t.Fatalf("error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "is not a recognised skill") {
		t.Fatalf("the skipped file should be reported:\n%s", out)
	}
	if !strings.Contains(out, "Bad Name!/SKILL.md") {
		t.Errorf("the report should name the file:\n%s", out)
	}
}

func TestPublishOutsideGitRepository(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))
	writeSkill(t, root, ".claude/skills/borrowed", validSkill("borrowed"))

	if err := runSkill(t, f, stdout, stderr, "publish", root, "--dry-run"); err != nil {
		t.Fatalf("publish outside a repository: %v (stdout=%s)", err, stdout)
	}
	// With no work tree there is nothing to check, so no raw git exit status.
	for _, unwanted := range []string{"exit status", "could not verify"} {
		if strings.Contains(stdout.String(), unwanted) {
			t.Fatalf("output should not mention %q:\n%s", unwanted, stdout)
		}
	}
}

func TestPublishEmptyJSONIsAnArray(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	root := t.TempDir()
	writeSkill(t, root, "skills/code-review", validSkill("code-review"))

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
		t.Fatalf("publish --json: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "[]" {
		t.Fatalf("stdout = %q, want an empty array rather than null", got)
	}
}

func TestWriteDiagnostics(t *testing.T) {
	diagnostics := []diagnostic{
		{Skill: "code-review", Severity: severityError, Message: "name does not match"},
		{Skill: "code-review", Severity: severityWarning, Message: "recommended field missing: license"},
		{Severity: severityWarning, Message: ".claude/skills/ contains installed skills"},
	}

	t.Run("terminal", func(t *testing.T) {
		var buf strings.Builder
		if err := writeDiagnostics(&buf, true, diagnostics, 1, 1, 2); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		for _, want := range []string{"✗ code-review: name does not match", "! code-review: recommended", "! .claude/skills/", "1 error(s), 2 warning(s)"} {
			if !strings.Contains(out, want) {
				t.Errorf("terminal output missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("warnings only", func(t *testing.T) {
		var buf strings.Builder
		if err := writeDiagnostics(&buf, true, diagnostics[1:], 1, 0, 2); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "2 warning(s)") || strings.Contains(buf.String(), "error(s)") {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("piped", func(t *testing.T) {
		var buf strings.Builder
		if err := writeDiagnostics(&buf, false, diagnostics, 1, 1, 2); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if strings.ContainsAny(out, "✗!") && !strings.Contains(out, "error") {
			t.Errorf("piped output should not use icons:\n%s", out)
		}
		if !strings.Contains(out, "error") || !strings.Contains(out, "warning") {
			t.Errorf("piped output should be severity-tagged rows:\n%s", out)
		}
	})

	t.Run("nothing to report", func(t *testing.T) {
		var buf strings.Builder
		if err := writeDiagnostics(&buf, true, nil, 3, 0, 0); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "✓ 3 skills validated successfully") {
			t.Fatalf("output = %q", buf.String())
		}
	})
}

func TestPublishTagOnDataCenterRemote(t *testing.T) {
	root, head := setupTagRepoWithRemote(t, "https://bitbucket.example.com/scm/proj/agent-skills.git")

	repo := sourcetest.New("PROJ/agent-skills", nil)
	repo.Branches = map[string]string{"main": head}
	args := stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0"); err != nil {
		t.Fatalf("publish --tag: %v (stderr=%s)", err, stderr)
	}
	// A Data Center remote yields PROJECT/REPO, not workspace/repo.
	if len(*args) != 1 || (*args)[0] != "https://bitbucket.example.com/scm/PROJ/agent-skills.git" {
		t.Fatalf("repository arguments = %v, want the detected host and repository", *args)
	}
	if repo.CreatedTags["v1.0.0"] != head {
		t.Fatalf("created tags = %v", repo.CreatedTags)
	}
}

func TestPublishTagOnDataCenterSSHRemoteWithPort(t *testing.T) {
	root, head := setupTagRepoWithRemote(t, "ssh://git@bitbucket.example.com:7999/proj/agent-skills.git")
	repo := sourcetest.New("PROJ/agent-skills", nil)
	repo.Branches = map[string]string{"main": head}
	args := stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0"); err != nil {
		t.Fatalf("publish --tag: %v (stderr=%s)", err, stderr)
	}
	if len(*args) != 1 || (*args)[0] != "https://bitbucket.example.com/scm/PROJ/agent-skills.git" {
		t.Fatalf("repository arguments = %v, want a Data Center URL without the SSH port", *args)
	}
}

func TestPublishTagRejectsIgnoredUntrackedSkill(t *testing.T) {
	root, _ := setupTagRepo(t)
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("skills/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "rm", "-r", "--cached", "skills")
	runGit(t, root, "add", ".gitignore")
	runGit(t, root, "commit", "-m", "ignore skills")
	head := runGit(t, root, "rev-parse", "HEAD")

	repo := sourcetest.New("myteam/agent-skills", nil)
	repo.Branches = map[string]string{"main": head}
	stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0")
	if err == nil || !strings.Contains(err.Error(), "because it is not committed") {
		t.Fatalf("error = %v, want ignored untracked skill to block the tag", err)
	}
	if len(repo.CreatedTags) != 0 {
		t.Fatalf("created tags = %v, want none", repo.CreatedTags)
	}
}

func TestPublishTagAllowsChangesOutsideSelectedDirectory(t *testing.T) {
	root, head := setupTagRepo(t)
	selected := filepath.Join(root, "skills")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("not part of this publish\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := sourcetest.New("myteam/agent-skills", nil)
	repo.Branches = map[string]string{"main": head}
	stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	if err := runSkill(t, f, stdout, stderr, "publish", selected, "--tag", "v1.0.0"); err != nil {
		t.Fatalf("publish selected directory: %v (stderr=%s)", err, stderr)
	}
	if repo.CreatedTags["v1.0.0"] != head {
		t.Fatalf("created tags = %v, want selected committed files to publish", repo.CreatedTags)
	}
}

// readOnlyRepo is a Repository that cannot create tags.
type readOnlyRepo struct{ source.Repository }

func TestPublishRepositoryWithoutTagSupport(t *testing.T) {
	root, head := setupTagRepo(t)
	inner := sourcetest.New("myteam/agent-skills", nil)
	inner.Branches = map[string]string{"main": head}

	original := openRepositoryFunc
	openRepositoryFunc = func(_ *cobra.Command, _ *cmdutil.Factory, _ string) (source.Repository, error) {
		return readOnlyRepo{Repository: inner}, nil
	}
	t.Cleanup(func() { openRepositoryFunc = original })

	f, stdout, stderr := newTestFactory(t)
	err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0")
	if err == nil || !strings.Contains(err.Error(), "does not support creating tags") {
		t.Fatalf("error = %v", err)
	}
}

func TestPublishTagCreationFailure(t *testing.T) {
	root, head := setupTagRepo(t)
	repo := sourcetest.New("myteam/agent-skills", nil)
	repo.Branches = map[string]string{"main": head}
	repo.CreateTagErr = errors.New("403 Forbidden")
	stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0")
	if err == nil || !strings.Contains(err.Error(), "could not create tag v1.0.0 in myteam/agent-skills: 403 Forbidden") {
		t.Fatalf("error = %v", err)
	}
}

func TestPublishSurfacesLookupFailures(t *testing.T) {
	// A transport or permission failure must not be read as "does not exist":
	// that would tag over an existing version or blame the user for not pushing.
	tests := []struct {
		name    string
		setup   func(*sourcetest.Repo)
		wantErr string
	}{
		{
			name:    "tag lookup fails",
			setup:   func(r *sourcetest.Repo) { r.TagErr = errors.New("500 Internal Server Error") },
			wantErr: "could not check whether tag v1.0.0 exists",
		},
		{
			name:    "commit lookup fails",
			setup:   func(r *sourcetest.Repo) { r.CommitErr = errors.New("401 Unauthorized") },
			wantErr: "could not check whether myteam/agent-skills has commit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, head := setupTagRepo(t)
			repo := sourcetest.New("myteam/agent-skills", nil)
			repo.Branches = map[string]string{"main": head}
			tt.setup(repo)
			stubRepository(t, repo)

			f, stdout, stderr := newTestFactory(t)
			err := runSkill(t, f, stdout, stderr, "publish", root, "--tag", "v1.0.0")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
			if len(repo.CreatedTags) != 0 {
				t.Errorf("a failed lookup must not lead to a tag: %v", repo.CreatedTags)
			}
		})
	}
}

func TestPublishMessageRequiresTag(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	err := runSkill(t, f, stdout, stderr, "publish", "--message", "notes")
	if err == nil || !strings.Contains(err.Error(), "--message only applies with --tag") {
		t.Fatalf("error = %v", err)
	}
}
