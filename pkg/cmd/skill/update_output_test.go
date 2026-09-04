package skill

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/discovery"
	"github.com/avivsinai/bitbucket-cli/internal/skills/lockfile"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

func isolateUpdateHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func runUpdateJSON(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	isolateUpdateHome(t)
	f, stdout, stderr := newTestFactory(t)
	root := &cobra.Command{Use: "bkt"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("yaml", false, "")
	root.PersistentFlags().String("format", "", "")
	root.PersistentFlags().String("jq", "", "")
	root.PersistentFlags().String("template", "", "")
	root.AddCommand(NewCmdSkill(f))
	root.SetArgs(append([]string{"skill", "update"}, args...))
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SilenceUsage = true
	root.SilenceErrors = true
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func TestUpdateJSONIsSingleDocument(t *testing.T) {
	repo := newSkillsRepo()
	stubRepository(t, repo)
	dir := t.TempDir()
	installedSkillDir(t, dir, "alpha", "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: old\n    bitbucket-path: skills/alpha\n")

	stdout, stderr, err := runUpdateJSON(t, "--dir", dir, "--all", "--json")
	if err != nil {
		t.Fatalf("update --json: %v (stderr=%s)", err, stderr)
	}
	var rows []availableUpdate
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("stdout is not one JSON document: %q: %v", stdout, err)
	}
	if len(rows) != 1 || rows[0].Name != "alpha" {
		t.Fatalf("rows = %+v", rows)
	}
	if strings.Contains(stdout, "Updated") {
		t.Fatalf("human update text appended to JSON: %q", stdout)
	}
}

func TestUpdateJSONEmptyResultsAreArray(t *testing.T) {
	stubRepository(t, newSkillsRepo())
	dir := t.TempDir()
	installedSkillDir(t, dir, "alpha", "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: alpha-commit\n    bitbucket-path: skills/alpha\n")

	stdout, stderr, err := runUpdateJSON(t, "--dir", dir, "--all", "--json")
	if err != nil {
		t.Fatalf("update --json: %v (stderr=%s)", err, stderr)
	}
	if strings.TrimSpace(stdout) != "[]" {
		t.Fatalf("stdout = %q, want []", stdout)
	}
}

func TestUpdateRuntimeFailureIsNotUpToDateSuccess(t *testing.T) {
	isolateUpdateHome(t)
	repo := newSkillsRepo()
	repo.Err = errors.New("service unavailable")
	stubRepository(t, repo)
	f, stdout, stderr := newTestFactory(t)
	dir := t.TempDir()
	installedSkillDir(t, dir, "alpha", "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: old\n    bitbucket-path: skills/alpha\n")

	err := runSkill(t, f, stdout, stderr, "update", "--dir", filepath.Clean(dir), "--all")
	if err == nil || !strings.Contains(err.Error(), "could not check all skills") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(stderr.String(), "All skills are up to date") {
		t.Fatalf("runtime failure reported success: %s", stderr)
	}
}

func TestUpdateMixedRepositoriesProcessesHealthyAndReturnsFailure(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		name := "apply"
		if dryRun {
			name = "dry run"
		}
		t.Run(name, func(t *testing.T) {
			isolateUpdateHome(t)
			healthy := newSkillsRepo()
			original := openRepositoryFunc
			openRepositoryFunc = func(_ *cobra.Command, _ *cmdutil.Factory, arg string) (source.Repository, error) {
				if strings.Contains(arg, "unavailable") {
					return nil, errors.New("service unavailable")
				}
				return healthy, nil
			}
			t.Cleanup(func() { openRepositoryFunc = original })

			f, stdout, stderr := newTestFactory(t)
			dir := t.TempDir()
			installedSkillDir(t, dir, "alpha", "    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: old\n    bitbucket-path: skills/alpha\n")
			installedSkillDir(t, dir, "beta", "    bitbucket-repo: https://bitbucket.org/bad/unavailable\n    bitbucket-commit: old\n    bitbucket-path: skills/beta\n")
			args := []string{"update", "--dir", dir, "--all"}
			if dryRun {
				args = append(args, "--dry-run")
			}

			err := runSkill(t, f, stdout, stderr, args...)
			if err == nil || !strings.Contains(err.Error(), "could not check all skills") {
				t.Fatalf("error = %v", err)
			}
			if !strings.Contains(stdout.String(), "alpha (myteam/agent-skills)") {
				t.Fatalf("healthy update is not visible: %s", stdout)
			}
			alpha := readFile(t, filepath.Join(dir, "alpha", "SKILL.md"))
			if dryRun && !strings.Contains(alpha, "# Old body") {
				t.Fatal("dry run changed healthy skill")
			}
			if !dryRun && (!strings.Contains(stdout.String(), "✓ Updated alpha") || strings.Contains(alpha, "# Old body")) {
				t.Fatalf("healthy update was not applied: stdout=%s body=%s", stdout, alpha)
			}
		})
	}
}

func TestUpdateSwapFailureDoesNotAdvanceLockfile(t *testing.T) {
	home := t.TempDir()
	if err := lockfile.RecordInstall(home, "alpha", lockfile.Entry{
		Source:          "myteam/agent-skills",
		SkillFolderHash: "old-commit",
	}); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "alpha")
	if err := os.WriteFile(destination, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := newSkillsRepo()
	u := pendingUpdate{
		local:     installedSkill{name: "alpha", dir: destination},
		repo:      repo,
		resolved:  source.Ref{Ref: "refs/heads/main", SHA: "sha-main"},
		skill:     discovery.Skill{Name: "alpha", Path: "skills/alpha"},
		newCommit: "alpha-commit",
	}

	if _, err := updateSkillInPlace(context.Background(), u, home); err == nil {
		t.Fatal("updateSkillInPlace succeeded with a file as its destination directory")
	}
	data, err := os.ReadFile(lockfile.Path(home))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Skills map[string]lockfile.Entry `json:"skills"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if got := document.Skills["alpha"].SkillFolderHash; got != "old-commit" {
		t.Fatalf("lockfile advanced to %q after failed swap", got)
	}
}

func TestRestoreBackupFailurePreservesRecoveryDirectory(t *testing.T) {
	parent := t.TempDir()
	dest := filepath.Join(parent, "alpha")
	backup := filepath.Join(parent, "backup")
	if err := os.MkdirAll(filepath.Join(dest, "keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(backup, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backup, "keep"), []byte("previous"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := restoreBackup(dest, backup, []string{"keep"}, nil)
	if err == nil || !strings.Contains(err.Error(), backup) {
		t.Fatalf("error = %v, want recovery path %s", err, backup)
	}
	data, readErr := os.ReadFile(filepath.Join(backup, "keep"))
	if readErr != nil || string(data) != "previous" {
		t.Fatalf("backup was not preserved: data=%q err=%v", data, readErr)
	}
}
