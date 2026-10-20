package remote

import (
	"errors"
	"os"
	"os/exec"
	"strings"
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
	dir := initGitRepo(t, "ssh://git@bitbucket.example.com:7999/TEAM/sample-app.git")

	loc, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if loc.Kind != "dc" || loc.ProjectKey != "TEAM" || loc.RepoSlug != "sample-app" {
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

func TestDetectNotGitRepoPreservesUnderlyingError(t *testing.T) {
	dir := t.TempDir()

	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrNoGitRemote) {
		t.Fatalf("Detect() error = %v, did not want %v", err, ErrNoGitRemote)
	}
	if !strings.Contains(err.Error(), "git remote -v:") {
		t.Fatalf("error = %q, want git remote context", err)
	}
}

func TestDetectResolvesSSHHostAlias(t *testing.T) {
	stubSSHHostnames(t, map[string]string{
		"work.bitbucket.org": "bitbucket.org",
		"bbdc":               "bitbucket.example.com",
	})

	tests := []struct {
		name   string
		remote string
		want   Locator
	}{
		{
			name:   "cloud scp-like alias",
			remote: "git@work.bitbucket.org:workspace/repo.git",
			want:   Locator{Host: "bitbucket.org", Kind: "cloud", Workspace: "workspace", RepoSlug: "repo", RemoteHost: "work.bitbucket.org"},
		},
		{
			name:   "cloud ssh scheme alias",
			remote: "ssh://git@work.bitbucket.org/workspace/repo.git",
			want:   Locator{Host: "bitbucket.org", Kind: "cloud", Workspace: "workspace", RepoSlug: "repo", RemoteHost: "work.bitbucket.org"},
		},
		{
			name:   "data center alias with port",
			remote: "ssh://git@bbdc:7999/proj/service.git",
			want:   Locator{Host: "bitbucket.example.com", Kind: "dc", ProjectKey: "PROJ", RepoSlug: "service", RemoteHost: "bbdc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initGitRepo(t, tt.remote)

			loc, err := Detect(dir)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}
			if loc != tt.want {
				t.Fatalf("locator = %+v, want %+v", loc, tt.want)
			}
		})
	}
}

func TestParseRemoteLocatorCloudSSHHostname(t *testing.T) {
	// #357: Bitbucket Cloud moves SSH to ssh.bitbucket.org on 12 Nov 2026.
	stubSSHHostnames(t, map[string]string{"bb-work": "ssh.bitbucket.org"})

	tests := []struct {
		remote string
		want   Locator
	}{
		{"git@ssh.bitbucket.org:myteam/my-repo.git", Locator{Host: "bitbucket.org", Kind: "cloud", Workspace: "myteam", RepoSlug: "my-repo"}},
		{"git@bb-work:myteam/my-repo.git", Locator{Host: "bitbucket.org", Kind: "cloud", Workspace: "myteam", RepoSlug: "my-repo", RemoteHost: "bb-work"}},
	}
	for _, tt := range tests {
		loc, err := ParseRemoteLocator(tt.remote)
		if err != nil {
			t.Fatalf("ParseRemoteLocator(%q) error = %v", tt.remote, err)
		}
		if loc != tt.want {
			t.Fatalf("ParseRemoteLocator(%q) = %+v, want %+v", tt.remote, loc, tt.want)
		}
	}
}

func TestParseRemoteLocatorKeepsHostWhenLookupFails(t *testing.T) {
	stubSSHHostnames(t, nil)

	loc, err := ParseRemoteLocator("git@work.bitbucket.org:workspace/repo.git")
	if err != nil {
		t.Fatalf("ParseRemoteLocator() error = %v", err)
	}
	if loc.Host != "work.bitbucket.org" || loc.Kind != "dc" {
		t.Fatalf("locator = %+v, want unresolved host parsed as before", loc)
	}
}

func TestParseRemoteLocatorSkipsLookup(t *testing.T) {
	tests := []struct {
		name   string
		remote string
	}{
		{name: "https remote", remote: "https://work.bitbucket.org/workspace/repo.git"},
		{name: "canonical cloud host", remote: "git@bitbucket.org:workspace/repo.git"},
		{name: "option-like host", remote: "-oProxyCommand=x:workspace/repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := stubSSHHostnames(t, map[string]string{"work.bitbucket.org": "bitbucket.org"})

			if _, err := ParseRemoteLocator(tt.remote); err != nil {
				t.Fatalf("ParseRemoteLocator() error = %v", err)
			}
			if *calls != 0 {
				t.Fatalf("ssh lookups = %d, want 0", *calls)
			}
		})
	}
}

func TestParseLocatorDoesNotResolveSSHHostAlias(t *testing.T) {
	calls := stubSSHHostnames(t, map[string]string{"work.bitbucket.org": "bitbucket.org"})

	loc, err := ParseLocator("git@work.bitbucket.org:workspace/repo.git")
	if err != nil {
		t.Fatalf("ParseLocator() error = %v", err)
	}
	if loc.Host != "work.bitbucket.org" || *calls != 0 {
		t.Fatalf("locator = %+v, lookups = %d; want alias untouched and no lookups", loc, *calls)
	}
}

func TestResolveSSHHostCachesLookups(t *testing.T) {
	calls := stubSSHHostnames(t, map[string]string{"work.bitbucket.org": "BitBucket.org"})

	for i := 0; i < 3; i++ {
		if got := resolveSSHHost("work.bitbucket.org"); got != "bitbucket.org" {
			t.Fatalf("resolveSSHHost() = %q, want %q", got, "bitbucket.org")
		}
	}
	if *calls != 1 {
		t.Fatalf("ssh lookups = %d, want 1", *calls)
	}
}

// stubSSHHostnames replaces the ssh -G lookup with a fixed alias table and
// returns a pointer to the number of lookups performed. Unknown hosts fail.
func stubSSHHostnames(t *testing.T, aliases map[string]string) *int {
	t.Helper()

	calls := 0
	original := sshHostnameLookup
	sshHostnameLookup = func(host string) (string, error) {
		calls++
		if name, ok := aliases[host]; ok {
			return name, nil
		}
		return "", errors.New("unknown host")
	}
	sshHostCache.Clear()
	t.Cleanup(func() {
		sshHostnameLookup = original
		sshHostCache.Clear()
	})
	return &calls
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
