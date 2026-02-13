package cmdutil

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
)

func newTestFactory(cfg *config.Config) *Factory {
	return &Factory{
		ExecutableName: "bkt",
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}
}

func TestResolveHostWithHostKey(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	key, host, err := ResolveHost(f, "", "bitbucket.example.com")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Fatalf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Fatalf("unexpected host: %#v", host)
	}
}

func TestResolveHostWithHostURL(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	key, host, err := ResolveHost(f, "", "https://bitbucket.example.com")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Fatalf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Fatalf("unexpected host: %#v", host)
	}
}

func TestResolveHostWithContext(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "dev",
		Contexts: map[string]*config.Context{
			"dev": {
				Host: "bitbucket.example.com",
			},
		},
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	key, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Fatalf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Fatalf("unexpected host: %#v", host)
	}
}

func TestResolveHostSingleHostFallback(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	key, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Fatalf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Fatalf("unexpected host: %#v", host)
	}
}

func TestResolveHostMultipleHostsError(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"one.example.com": {
				Kind:    "dc",
				BaseURL: "https://one.example.com",
				Token:   "test-token",
			},
			"two.example.com": {
				Kind:    "dc",
				BaseURL: "https://two.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	_, _, err := ResolveHost(f, "", "")
	if err == nil {
		t.Fatalf("expected error for multiple hosts")
	}
	if !strings.Contains(err.Error(), "multiple hosts") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveHostNoHostsError(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{},
	}
	f := newTestFactory(cfg)

	_, _, err := ResolveHost(f, "", "")
	if err == nil {
		t.Fatalf("expected error when no hosts configured")
	}
	if !strings.Contains(err.Error(), "no hosts configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveContextOverridesProjectFromRemoteSSH(t *testing.T) {
	repoDir := initGitRepo(t, "ssh://git@bitbucket.example.com:7999/TEAM/sample-app.git")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	cfg := &config.Config{
		ActiveContext: "dev",
		Contexts: map[string]*config.Context{
			"dev": {
				Host:       "bitbucket.example.com",
				ProjectKey: "DEV",
			},
		},
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	_, ctx, _, err := ResolveContext(f, nil, "")
	if err != nil {
		t.Fatalf("ResolveContext error: %v", err)
	}
	if ctx.ProjectKey != "TEAM" {
		t.Fatalf("project = %q, want %q", ctx.ProjectKey, "TEAM")
	}
	if ctx.DefaultRepo != "sample-app" {
		t.Fatalf("repo = %q, want %q", ctx.DefaultRepo, "sample-app")
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
