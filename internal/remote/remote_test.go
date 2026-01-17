package remote

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestDetectCloudHTTPS(t *testing.T) {
	dir := initGitRepo(t, "https://bitbucket.org/workspace/repo.git")

	loc, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if loc.Kind != "cloud" {
		t.Fatalf("kind = %q, want %q", loc.Kind, "cloud")
	}
	if loc.Workspace != "workspace" {
		t.Fatalf("workspace = %q, want %q", loc.Workspace, "workspace")
	}
	if loc.RepoSlug != "repo" {
		t.Fatalf("repo = %q, want %q", loc.RepoSlug, "repo")
	}
	if loc.Host != "bitbucket.org" {
		t.Fatalf("host = %q, want %q", loc.Host, "bitbucket.org")
	}
}

func TestDetectCloudSSH(t *testing.T) {
	dir := initGitRepo(t, "git@bitbucket.org:workspace/repo.git")

	loc, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if loc.Kind != "cloud" || loc.Workspace != "workspace" || loc.RepoSlug != "repo" {
		t.Fatalf("locator = %+v", loc)
	}
}

func TestDetectDataCenterScm(t *testing.T) {
	dir := initGitRepo(t, "https://bitbucket.example.com/scm/PROJ/service.git")

	loc, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if loc.Kind != "dc" {
		t.Fatalf("kind = %q, want %q", loc.Kind, "dc")
	}
	if loc.ProjectKey != "PROJ" {
		t.Fatalf("project = %q, want %q", loc.ProjectKey, "PROJ")
	}
	if loc.RepoSlug != "service" {
		t.Fatalf("repo = %q, want %q", loc.RepoSlug, "service")
	}
	if loc.Host != "bitbucket.example.com" {
		t.Fatalf("host = %q, want %q", loc.Host, "bitbucket.example.com")
	}
}

func TestDetectDataCenterProjects(t *testing.T) {
	dir := initGitRepo(t, "ssh://git@bitbucket.example.com:7999/projects/PROJ/repos/service.git")

	loc, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if loc.Kind != "dc" || loc.ProjectKey != "PROJ" || loc.RepoSlug != "service" {
		t.Fatalf("locator = %+v", loc)
	}
}

func TestDetectDataCenterRootSSH(t *testing.T) {
	dir := initGitRepo(t, "ssh://git@bitbucket.example.com:7999/PLAYG/daredevil-ui.git")

	loc, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if loc.Kind != "dc" || loc.ProjectKey != "PLAYG" || loc.RepoSlug != "daredevil-ui" {
		t.Fatalf("locator = %+v", loc)
	}
}

func TestDetectNoRemote(t *testing.T) {
	dir := initGitRepo(t, "")

	_, err := Detect(dir)
	if !errors.Is(err, ErrNoGitRemote) {
		t.Fatalf("Detect() error = %v, want %v", err, ErrNoGitRemote)
	}
}

func initGitRepo(t *testing.T, remoteURL string) string {
	t.Helper()

	dir := t.TempDir()
	runGit(t, dir, "init", ".")

	if remoteURL != "" {
		runGit(t, dir, "remote", "add", "origin", remoteURL)
	}

	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmdArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
