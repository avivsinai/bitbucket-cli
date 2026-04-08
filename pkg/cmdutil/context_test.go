package cmdutil

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
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

func TestLoadHostTokenBypassesKeyringWhenEnvTokenSet(t *testing.T) {
	host := &config.Host{
		Kind:               "dc",
		BaseURL:            "https://bitbucket.example.com",
		AllowInsecureStore: true,
	}

	t.Setenv(secret.EnvToken, "env-token")

	// Make keyring usage fail in headless file-backend mode.
	t.Setenv("KEYRING_BACKEND", "file")
	t.Setenv("SSH_CONNECTION", "1")
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("BKT_KEYRING_PASSPHRASE", "")
	t.Setenv("KEYRING_FILE_PASSWORD", "")
	t.Setenv("KEYRING_PASSWORD", "")

	if err := loadHostToken("bkt", "bitbucket.example.com", host); err != nil {
		t.Fatalf("loadHostToken returned error: %v", err)
	}
	if host.Token != "env-token" {
		t.Fatalf("token = %q, want %q", host.Token, "env-token")
	}
}

func TestLoadHostTokenEnvTokenTakesPrecedence(t *testing.T) {
	host := &config.Host{
		Kind:    "dc",
		BaseURL: "https://bitbucket.example.com",
		Token:   "stored-token",
	}

	t.Setenv(secret.EnvToken, "env-token")

	if err := loadHostToken("bkt", "bitbucket.example.com", host); err != nil {
		t.Fatalf("loadHostToken returned error: %v", err)
	}
	if host.Token != "env-token" {
		t.Fatalf("token = %q, want %q", host.Token, "env-token")
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

// --- hostFromEnv tests ---

func TestHostFromEnvDCHost(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")

	key, host, err := hostFromEnv("https://bitbucket.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host == nil {
		t.Fatal("expected host, got nil")
	}
	if key != "bitbucket.example.com" {
		t.Errorf("key = %q, want bitbucket.example.com", key)
	}
	if host.Kind != "dc" {
		t.Errorf("kind = %q, want dc", host.Kind)
	}
	if host.BaseURL != "https://bitbucket.example.com" {
		t.Errorf("baseURL = %q, want https://bitbucket.example.com", host.BaseURL)
	}
	if host.Token != "test-token" {
		t.Errorf("token = %q, want test-token", host.Token)
	}
}

func TestHostFromEnvCloudAutoDetect(t *testing.T) {
	t.Setenv(secret.EnvToken, "cloud-token")

	// BKT_HOST=https://bitbucket.org is canonicalised to the API origin the
	// same way bkt auth login does, so the Cloud client is routed correctly.
	key, host, err := hostFromEnv("https://bitbucket.org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host == nil {
		t.Fatal("expected host, got nil")
	}
	if key != "api.bitbucket.org" {
		t.Errorf("key = %q, want api.bitbucket.org", key)
	}
	if host.Kind != "cloud" {
		t.Errorf("kind = %q, want cloud", host.Kind)
	}
	if host.BaseURL != "https://api.bitbucket.org/2.0" {
		t.Errorf("baseURL = %q, want https://api.bitbucket.org/2.0", host.BaseURL)
	}
}

func TestHostFromEnvDCHostWithBitbucketOrgInName(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")

	// A DC host whose name contains "bitbucket.org" must NOT be rewritten to
	// the Cloud API origin, and must not be classified as Cloud.
	key, host, err := hostFromEnv("https://bitbucket.org.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host == nil {
		t.Fatal("expected host, got nil")
	}
	if key != "bitbucket.org.example.com" {
		t.Errorf("key = %q, want bitbucket.org.example.com", key)
	}
	if host.Kind != "dc" {
		t.Errorf("kind = %q, want dc", host.Kind)
	}
	if host.BaseURL != "https://bitbucket.org.example.com" {
		t.Errorf("baseURL = %q, want https://bitbucket.org.example.com", host.BaseURL)
	}
}

func TestHostFromEnvCloudBareAPIHost(t *testing.T) {
	t.Setenv(secret.EnvToken, "cloud-token")

	// api.bitbucket.org without the /2.0 path must be canonicalised so that
	// NewCloudClient routes to the correct API base URL.
	key, host, err := hostFromEnv("api.bitbucket.org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host == nil {
		t.Fatal("expected host, got nil")
	}
	if key != "api.bitbucket.org" {
		t.Errorf("key = %q, want api.bitbucket.org", key)
	}
	if host.Kind != "cloud" {
		t.Errorf("kind = %q, want cloud", host.Kind)
	}
	if host.BaseURL != "https://api.bitbucket.org/2.0" {
		t.Errorf("baseURL = %q, want https://api.bitbucket.org/2.0", host.BaseURL)
	}
}

func TestHostFromEnvCloudAPIURLPassthrough(t *testing.T) {
	t.Setenv(secret.EnvToken, "cloud-token")

	// BKT_HOST=https://api.bitbucket.org/2.0 must also be classified as Cloud.
	key, host, err := hostFromEnv("https://api.bitbucket.org/2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host == nil {
		t.Fatal("expected host, got nil")
	}
	if key != "api.bitbucket.org" {
		t.Errorf("key = %q, want api.bitbucket.org", key)
	}
	if host.Kind != "cloud" {
		t.Errorf("kind = %q, want cloud", host.Kind)
	}
}

func TestHostFromEnvNoToken(t *testing.T) {
	t.Setenv(secret.EnvToken, "")

	key, host, err := hostFromEnv("https://bitbucket.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "" || host != nil {
		t.Errorf("expected (empty, nil), got (%q, %v)", key, host)
	}
}

func TestHostFromEnvNoURL(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")

	key, host, err := hostFromEnv("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "" || host != nil {
		t.Errorf("expected (empty, nil), got (%q, %v)", key, host)
	}
}

func TestHostFromEnvAuthMethodBearer(t *testing.T) {
	t.Setenv(secret.EnvToken, "bearer-token")
	t.Setenv(secret.EnvAuthMethod, "bearer")

	_, host, err := hostFromEnv("https://bitbucket.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host.AuthMethod != "bearer" {
		t.Errorf("authMethod = %q, want bearer", host.AuthMethod)
	}
}

func TestHostFromEnvUsername(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")
	t.Setenv(secret.EnvUsername, "admin")

	_, host, err := hostFromEnv("https://bitbucket.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host.Username != "admin" {
		t.Errorf("username = %q, want admin", host.Username)
	}
}

func TestHostFromEnvHostWithoutScheme(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")

	key, host, err := hostFromEnv("bitbucket.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host == nil {
		t.Fatal("expected host, got nil")
	}
	if key != "bitbucket.example.com" {
		t.Errorf("key = %q, want bitbucket.example.com", key)
	}
	if host.BaseURL != "https://bitbucket.example.com" {
		t.Errorf("baseURL = %q, want https://bitbucket.example.com", host.BaseURL)
	}
}

// --- contextFromEnv tests ---

func TestContextFromEnvDC(t *testing.T) {
	t.Setenv(secret.EnvProject, "PROJ")
	t.Setenv(secret.EnvRepo, "my-repo")
	t.Setenv(secret.EnvWorkspace, "")

	ctx := contextFromEnv()
	if ctx.ProjectKey != "PROJ" {
		t.Errorf("ProjectKey = %q, want PROJ", ctx.ProjectKey)
	}
	if ctx.DefaultRepo != "my-repo" {
		t.Errorf("DefaultRepo = %q, want my-repo", ctx.DefaultRepo)
	}
	if ctx.Workspace != "" {
		t.Errorf("Workspace = %q, want empty", ctx.Workspace)
	}
}

func TestContextFromEnvCloud(t *testing.T) {
	t.Setenv(secret.EnvWorkspace, "my-workspace")
	t.Setenv(secret.EnvRepo, "my-repo")
	t.Setenv(secret.EnvProject, "")

	ctx := contextFromEnv()
	if ctx.Workspace != "my-workspace" {
		t.Errorf("Workspace = %q, want my-workspace", ctx.Workspace)
	}
	if ctx.DefaultRepo != "my-repo" {
		t.Errorf("DefaultRepo = %q, want my-repo", ctx.DefaultRepo)
	}
	if ctx.ProjectKey != "" {
		t.Errorf("ProjectKey = %q, want empty", ctx.ProjectKey)
	}
}

// --- ResolveHost env-var fallback tests ---

func TestResolveHostEnvFallbackNoHostsConfigured(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{},
	})

	key, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Errorf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.Kind != "dc" {
		t.Errorf("unexpected host: %#v", host)
	}
	if host.Token != "test-token" {
		t.Errorf("token = %q, want test-token", host.Token)
	}
}

func TestResolveHostEnvFallbackHostOverrideNotFound(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{},
	})

	key, host, err := ResolveHost(f, "", "https://bitbucket.example.com")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Errorf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Errorf("unexpected host: %#v", host)
	}
}

func TestResolveHostEnvFallbackNotTriggeredWithoutToken(t *testing.T) {
	t.Setenv(secret.EnvToken, "")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{},
	})

	_, _, err := ResolveHost(f, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no hosts configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveHostExistingConfigNotOverridden(t *testing.T) {
	t.Setenv(secret.EnvToken, "env-token")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "config-token",
			},
		},
	})

	_, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	// BKT_TOKEN overrides stored token (existing loadHostToken behaviour), but
	// the host entry itself must come from config, not be synthesised from env.
	if host == nil {
		t.Fatal("expected host, got nil")
	}
	if host.BaseURL != "https://bitbucket.example.com" {
		t.Errorf("baseURL = %q, want https://bitbucket.example.com", host.BaseURL)
	}
}

// --- ResolveContext env-var fallback tests ---

func TestResolveContextEnvFallback(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")
	t.Setenv(secret.EnvProject, "MYPROJ")
	t.Setenv(secret.EnvRepo, "my-repo")

	f := newTestFactory(&config.Config{
		Contexts: map[string]*config.Context{},
		Hosts:    map[string]*config.Host{},
	})

	_, ctx, host, err := ResolveContext(f, nil, "")
	if err != nil {
		t.Fatalf("ResolveContext returned error: %v", err)
	}
	if host == nil || host.Kind != "dc" {
		t.Errorf("unexpected host: %#v", host)
	}
	if host.Token != "test-token" {
		t.Errorf("token = %q, want test-token", host.Token)
	}
	if ctx == nil {
		t.Fatal("expected context, got nil")
	}
	if ctx.ProjectKey != "MYPROJ" {
		t.Errorf("ProjectKey = %q, want MYPROJ", ctx.ProjectKey)
	}
	if ctx.DefaultRepo != "my-repo" {
		t.Errorf("DefaultRepo = %q, want my-repo", ctx.DefaultRepo)
	}
	if ctx.Host != "bitbucket.example.com" {
		t.Errorf("Host = %q, want bitbucket.example.com", ctx.Host)
	}
}

func TestResolveHostTokenSetButHostMissing(t *testing.T) {
	t.Setenv(secret.EnvToken, "test-token")
	t.Setenv(secret.EnvHost, "")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{},
	})

	_, _, err := ResolveHost(f, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "BKT_HOST") {
		t.Errorf("expected BKT_HOST mention in error, got: %v", err)
	}
}

func TestResolveContextPrefersSavedHostWhenEnvMatches(t *testing.T) {
	t.Setenv(secret.EnvToken, "env-token")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")

	f := newTestFactory(&config.Config{
		// No active context, but a host entry exists with persisted fields.
		Contexts: map[string]*config.Context{},
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:       "dc",
				BaseURL:    "https://bitbucket.example.com",
				Username:   "alice",
				AuthMethod: "bearer",
			},
		},
	})

	_, _, host, err := ResolveContext(f, nil, "")
	if err != nil {
		t.Fatalf("ResolveContext returned error: %v", err)
	}
	// Must use the saved host, preserving Username and AuthMethod.
	if host.Username != "alice" {
		t.Errorf("Username = %q, want alice", host.Username)
	}
	if host.AuthMethod != "bearer" {
		t.Errorf("AuthMethod = %q, want bearer", host.AuthMethod)
	}
	// BKT_TOKEN is still applied via loadHostToken.
	if host.Token != "env-token" {
		t.Errorf("Token = %q, want env-token", host.Token)
	}
}

func TestResolveContextEnvFallbackNotTriggeredWithoutToken(t *testing.T) {
	t.Setenv(secret.EnvToken, "")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")

	f := newTestFactory(&config.Config{
		Contexts: map[string]*config.Context{},
		Hosts:    map[string]*config.Host{},
	})

	_, _, _, err := ResolveContext(f, nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no active context") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveHostEnvFallbackWithMultipleHostsMatchesSaved(t *testing.T) {
	t.Setenv(secret.EnvToken, "env-token")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:       "dc",
				BaseURL:    "https://bitbucket.example.com",
				Username:   "alice",
				AuthMethod: "bearer",
			},
			"other.example.com": {
				Kind:    "dc",
				BaseURL: "https://other.example.com",
				Token:   "other-token",
			},
		},
	})

	key, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Errorf("key = %q, want bitbucket.example.com", key)
	}
	// Saved fields must be preserved.
	if host.Username != "alice" {
		t.Errorf("Username = %q, want alice", host.Username)
	}
	if host.AuthMethod != "bearer" {
		t.Errorf("AuthMethod = %q, want bearer", host.AuthMethod)
	}
	if host.Token != "env-token" {
		t.Errorf("Token = %q, want env-token", host.Token)
	}
}

func TestResolveHostEnvFallbackWithMultipleHostsNoMatch(t *testing.T) {
	t.Setenv(secret.EnvToken, "env-token")
	t.Setenv(secret.EnvHost, "https://new.example.com")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{
			"one.example.com": {Kind: "dc", BaseURL: "https://one.example.com", Token: "t1"},
			"two.example.com": {Kind: "dc", BaseURL: "https://two.example.com", Token: "t2"},
		},
	})

	key, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "new.example.com" {
		t.Errorf("key = %q, want new.example.com", key)
	}
	if host.Token != "env-token" {
		t.Errorf("Token = %q, want env-token", host.Token)
	}
}

func TestResolveHostEnvFallbackWithSingleHostOverride(t *testing.T) {
	t.Setenv(secret.EnvToken, "env-token")
	t.Setenv(secret.EnvHost, "https://other.example.com")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{
			"saved.example.com": {
				Kind:    "dc",
				BaseURL: "https://saved.example.com",
				Token:   "saved-token",
			},
		},
	})

	// BKT_HOST points to a different server than the one in config; the env
	// target should win (consistent with ResolveContext and default behaviour).
	key, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "other.example.com" {
		t.Errorf("key = %q, want other.example.com", key)
	}
	if host.BaseURL != "https://other.example.com" {
		t.Errorf("baseURL = %q, want https://other.example.com", host.BaseURL)
	}
	if host.Token != "env-token" {
		t.Errorf("Token = %q, want env-token", host.Token)
	}
}

func TestResolveHostMultipleHostsErrorWhenNoEnvHost(t *testing.T) {
	t.Setenv(secret.EnvToken, "")
	t.Setenv(secret.EnvHost, "")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{
			"one.example.com": {Kind: "dc", BaseURL: "https://one.example.com", Token: "t1"},
			"two.example.com": {Kind: "dc", BaseURL: "https://two.example.com", Token: "t2"},
		},
	})

	_, _, err := ResolveHost(f, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "multiple hosts") {
		t.Errorf("unexpected error: %v", err)
	}
}
