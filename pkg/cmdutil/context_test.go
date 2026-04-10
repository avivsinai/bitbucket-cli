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

func TestHostFromEnv(t *testing.T) {
	type hostAssert struct {
		key        string
		kind       string
		baseURL    string
		username   string
		authMethod string
		token      string
	}
	tests := []struct {
		name       string
		rawURL     string
		token      string
		username   string
		authMethod string
		// wantNil: expect ("", nil, nil) — no host synthesised
		wantNil bool
		// wantErr: expect an error whose message contains this string
		wantErr string
		// wantHost: fields to assert when a host is returned
		wantHost *hostAssert
	}{
		{
			name:    "DC host synthesis",
			rawURL:  "https://bitbucket.example.com",
			token:   "test-token",
			wantHost: &hostAssert{key: "bitbucket.example.com", kind: "dc", baseURL: "https://bitbucket.example.com", authMethod: "bearer", token: "test-token"},
		},
		{
			name:     "Cloud auto-detect from bitbucket.org",
			rawURL:   "https://bitbucket.org",
			token:    "cloud-token",
			username: "test@example.com",
			wantHost: &hostAssert{key: "api.bitbucket.org", kind: "cloud", baseURL: "https://api.bitbucket.org/2.0", authMethod: "basic"},
		},
		{
			name:    "DC host with bitbucket.org in name not rewritten",
			rawURL:  "https://bitbucket.org.example.com",
			token:   "test-token",
			wantHost: &hostAssert{key: "bitbucket.org.example.com", kind: "dc", baseURL: "https://bitbucket.org.example.com"},
		},
		{
			name:     "Cloud bare api.bitbucket.org canonicalised",
			rawURL:   "api.bitbucket.org",
			token:    "cloud-token",
			username: "test@example.com",
			wantHost: &hostAssert{key: "api.bitbucket.org", kind: "cloud", baseURL: "https://api.bitbucket.org/2.0"},
		},
		{
			name:     "Cloud https://api.bitbucket.org/2.0 passthrough",
			rawURL:   "https://api.bitbucket.org/2.0",
			token:    "cloud-token",
			username: "test@example.com",
			wantHost: &hostAssert{key: "api.bitbucket.org", kind: "cloud"},
		},
		{
			name:    "no token returns nil",
			rawURL:  "https://bitbucket.example.com",
			token:   "",
			wantNil: true,
		},
		{
			name:    "no URL returns nil",
			rawURL:  "",
			token:   "test-token",
			wantNil: true,
		},
		{
			name:       "explicit bearer auth method honoured",
			rawURL:     "https://bitbucket.example.com",
			token:      "bearer-token",
			authMethod: "bearer",
			wantHost:   &hostAssert{authMethod: "bearer"},
		},
		{
			name:     "username is mapped",
			rawURL:   "https://bitbucket.example.com",
			token:    "test-token",
			username: "admin",
			wantHost: &hostAssert{username: "admin"},
		},
		{
			name:    "bare hostname gets https scheme",
			rawURL:  "bitbucket.example.com",
			token:   "test-token",
			wantHost: &hostAssert{key: "bitbucket.example.com", baseURL: "https://bitbucket.example.com"},
		},
		{
			name:    "DC defaults to bearer when no username",
			rawURL:  "https://bitbucket.example.com",
			token:   "pat-token",
			wantHost: &hostAssert{authMethod: "bearer"},
		},
		{
			name:       "DC basic without username is an error",
			rawURL:     "https://bitbucket.example.com",
			token:      "pat-token",
			authMethod: "basic",
			wantErr:    "BKT_USERNAME",
		},
		{
			name:    "Cloud without username is an error",
			rawURL:  "https://bitbucket.org",
			token:   "cloud-token",
			wantErr: "BKT_USERNAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(secret.EnvToken, tt.token)
			t.Setenv(secret.EnvUsername, tt.username)
			t.Setenv(secret.EnvAuthMethod, tt.authMethod)

			key, host, err := hostFromEnv(tt.rawURL)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %v, want to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if key != "" || host != nil {
					t.Errorf("expected (empty, nil), got (%q, %v)", key, host)
				}
				return
			}
			if host == nil {
				t.Fatal("expected host, got nil")
			}
			a := tt.wantHost
			if a.key != "" && key != a.key {
				t.Errorf("key = %q, want %q", key, a.key)
			}
			if a.kind != "" && host.Kind != a.kind {
				t.Errorf("kind = %q, want %q", host.Kind, a.kind)
			}
			if a.baseURL != "" && host.BaseURL != a.baseURL {
				t.Errorf("baseURL = %q, want %q", host.BaseURL, a.baseURL)
			}
			if a.username != "" && host.Username != a.username {
				t.Errorf("username = %q, want %q", host.Username, a.username)
			}
			if a.authMethod != "" && host.AuthMethod != a.authMethod {
				t.Errorf("authMethod = %q, want %q", host.AuthMethod, a.authMethod)
			}
			if a.token != "" && host.Token != a.token {
				t.Errorf("token = %q, want %q", host.Token, a.token)
			}
		})
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
	// Neither BKT_HOST nor BKT_TOKEN — generic "no hosts configured" error.
	t.Setenv(secret.EnvToken, "")
	t.Setenv(secret.EnvHost, "")

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

func TestResolveHostHostSetButNoToken(t *testing.T) {
	// BKT_HOST set but BKT_TOKEN absent — actionable hint error.
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")
	t.Setenv(secret.EnvToken, "")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{},
	})

	_, _, err := ResolveHost(f, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "BKT_TOKEN") {
		t.Errorf("expected BKT_TOKEN mention in error, got: %v", err)
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

func TestResolveContextEnvOverridesSavedHostAuthFields(t *testing.T) {
	// When BKT_USERNAME or BKT_AUTH_METHOD are set, they must override the
	// values stored in the config so headless runs use the requested credentials.
	t.Setenv(secret.EnvToken, "env-token")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")
	t.Setenv(secret.EnvUsername, "bob")
	t.Setenv(secret.EnvAuthMethod, "basic")

	f := newTestFactory(&config.Config{
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
	if host.Username != "bob" {
		t.Errorf("Username = %q, want bob", host.Username)
	}
	if host.AuthMethod != "basic" {
		t.Errorf("AuthMethod = %q, want basic", host.AuthMethod)
	}
}

func TestResolveHostEnvOverridesSavedHostAuthFields(t *testing.T) {
	// BKT_USERNAME and BKT_AUTH_METHOD must override saved host fields in all
	// ResolveHost paths that reuse a saved config entry.
	t.Setenv(secret.EnvToken, "env-token")
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")
	t.Setenv(secret.EnvUsername, "ci-user")
	t.Setenv(secret.EnvAuthMethod, "bearer")

	f := newTestFactory(&config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:       "dc",
				BaseURL:    "https://bitbucket.example.com",
				Username:   "dev-user",
				AuthMethod: "basic",
			},
			"other.example.com": {
				Kind:    "dc",
				BaseURL: "https://other.example.com",
				Token:   "other-token",
			},
		},
	})

	_, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if host.Username != "ci-user" {
		t.Errorf("Username = %q, want ci-user", host.Username)
	}
	if host.AuthMethod != "bearer" {
		t.Errorf("AuthMethod = %q, want bearer", host.AuthMethod)
	}
}

func TestResolveContextEnvFallbackNotTriggeredWithoutToken(t *testing.T) {
	// Neither BKT_HOST nor BKT_TOKEN — generic "no active context" error.
	t.Setenv(secret.EnvToken, "")
	t.Setenv(secret.EnvHost, "")

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

func TestResolveContextHostSetButNoToken(t *testing.T) {
	// BKT_HOST set but BKT_TOKEN absent — actionable hint error.
	t.Setenv(secret.EnvHost, "https://bitbucket.example.com")
	t.Setenv(secret.EnvToken, "")

	f := newTestFactory(&config.Config{
		Contexts: map[string]*config.Context{},
		Hosts:    map[string]*config.Host{},
	})

	_, _, _, err := ResolveContext(f, nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "BKT_TOKEN") {
		t.Errorf("expected BKT_TOKEN mention in error, got: %v", err)
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
