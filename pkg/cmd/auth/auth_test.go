package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestCloudTokenURLIsAtlassian(t *testing.T) {
	// Verify the actual CloudTokenURL constant points to Atlassian's account management.
	// This test ensures we don't regress to the old bitbucket.org URL.
	if !strings.Contains(CloudTokenURL, "id.atlassian.com") {
		t.Fatalf("CloudTokenURL should use id.atlassian.com, got: %s", CloudTokenURL)
	}
	if !strings.Contains(CloudTokenURL, "api-tokens") {
		t.Fatalf("CloudTokenURL should point to api-tokens page, got: %s", CloudTokenURL)
	}
}

func TestLoginFlagHelpTextNoAppPassword(t *testing.T) {
	// Create the login command and verify help text doesn't mention "app password"
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}

	var stdout, stderr strings.Builder
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    &stdout,
			ErrOut: &stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := newLoginCmd(f)

	// Check --token flag usage
	tokenFlag := cmd.Flag("token")
	if tokenFlag == nil {
		t.Fatal("expected --token flag")
	}
	if strings.Contains(strings.ToLower(tokenFlag.Usage), "app password") {
		t.Fatalf("--token flag should not mention app password, got: %s", tokenFlag.Usage)
	}
}

func TestLoginFlagHelpTextWarnsAboutTokenExposure(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	cmd := newLoginCmd(f)

	tokenFlag := cmd.Flag("token")
	if tokenFlag == nil {
		t.Fatal("expected --token flag")
	}
	if !strings.Contains(tokenFlag.Usage, "process list") {
		t.Fatalf("--token flag should warn about process-list exposure, got: %s", tokenFlag.Usage)
	}

	if cmd.Flag("allow-http") == nil {
		t.Fatal("expected --allow-http flag")
	}
}

func TestCloudLoginPromptsNoAppPassword(t *testing.T) {
	// Verify that the cloud login prompt constants don't mention "app password".
	// This ensures users aren't confused by old terminology since Bitbucket Cloud
	// uses API tokens, not app passwords.
	prompts := []struct {
		name  string
		value string
	}{
		{"CloudEmailPrompt", CloudEmailPrompt},
		{"CloudTokenPrompt", CloudTokenPrompt},
	}

	for _, p := range prompts {
		if strings.Contains(strings.ToLower(p.value), "app password") {
			t.Errorf("%s should not mention 'app password', got: %s", p.name, p.value)
		}
	}
}

func TestRunLoginBlockedWhenEnvTokenSet(t *testing.T) {
	t.Setenv(secret.EnvToken, "env-token")

	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{})
	if err == nil {
		t.Fatal("expected error when BKT_TOKEN is set")
	}

	want := "BKT_TOKEN environment variable is set; token is externally managed. Unset BKT_TOKEN to use auth login"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRunLogoutBlockedWhenEnvTokenSet(t *testing.T) {
	t.Setenv(secret.EnvToken, "env-token")

	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
			},
		},
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogout(&cobra.Command{}, f, &logoutOptions{Host: "bitbucket.example.com"})
	if err == nil {
		t.Fatal("expected error when BKT_TOKEN is set")
	}

	want := "BKT_TOKEN environment variable is set; token is externally managed. Unset BKT_TOKEN to use auth logout"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRunLoginRejectsHTTPWithoutAllowHTTP(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{Host: "http://bitbucket.example.com"})
	if err == nil {
		t.Fatal("expected error")
	}

	want := "http:// URLs are not allowed by default; rerun with --allow-http if you understand the credentials will be sent in plaintext"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRunLoginWarnsOnTokenFlag(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, stderr := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:  "https://bitbucket.example.com",
		Token: "secret-token",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "username is required when not running in a TTY" {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stderr.String(), "WARNING: --token is visible in process listings and shell history") {
		t.Fatalf("expected token warning, got stderr:\n%s", stderr.String())
	}
}

func TestRunLoginWarnsOnAllowHTTP(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, stderr := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:      "http://bitbucket.example.com",
		AllowHTTP: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "username is required when not running in a TTY" {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stderr.String(), "WARNING: using http:// will send credentials in plaintext") {
		t.Fatalf("expected http warning, got stderr:\n%s", stderr.String())
	}
}

func TestRunStatusShowsEnvTokenSource(t *testing.T) {
	t.Setenv(secret.EnvToken, "env-token")

	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:     "dc",
				BaseURL:  "https://bitbucket.example.com",
				Username: "admin",
			},
		},
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	if err := runStatus(&cobra.Command{}, f); err != nil {
		t.Fatalf("runStatus returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "token source: BKT_TOKEN") {
		t.Fatalf("expected token source in output, got:\n%s", output)
	}
}

func TestRunStatusShowsKeyringTokenSource(t *testing.T) {
	t.Setenv(secret.EnvToken, "")

	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:     "dc",
				BaseURL:  "https://bitbucket.example.com",
				Username: "admin",
			},
		},
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	if err := runStatus(&cobra.Command{}, f); err != nil {
		t.Fatalf("runStatus returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "token source: keyring") {
		t.Fatalf("expected keyring token source in output, got:\n%s", output)
	}
}

func TestRunLoginRejectsWebOnDC(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host: "https://bitbucket.example.com",
		Kind: "dc",
		Web:  true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "only supported for Bitbucket Cloud") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLoginRejectsWebAndWebToken(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:     "https://bitbucket.org",
		Kind:     "cloud",
		Web:      true,
		WebToken: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLoginRejectsWebWithoutOAuthCreds(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_ID", "")
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "")

	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host: "https://bitbucket.org",
		Kind: "cloud",
		Web:  true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "OAuth credentials were not embedded") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLoginRejectsAuthMethodOnCloudNonWeb(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:       "https://bitbucket.org",
		Kind:       "cloud",
		AuthMethod: "bearer",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth-method is only supported for Data Center") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLoginRejectsUnsupportedAuthMethod(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:       "https://bitbucket.example.com",
		Kind:       "dc",
		AuthMethod: "digest",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported auth method") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLoginRejectsUnsupportedKind(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:     "https://bitbucket.example.com",
		Kind:     "server",
		Username: "admin",
		Token:    "tok",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported deployment kind") {
		t.Errorf("error = %q", err)
	}
}

func TestRunStatusShowsOAuthMethod(t *testing.T) {
	t.Setenv(secret.EnvToken, "")

	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"api.bitbucket.org": {
				Kind:       "cloud",
				BaseURL:    "https://api.bitbucket.org/2.0",
				Username:   "erank_ai21",
				AuthMethod: "oauth",
			},
		},
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	if err := runStatus(&cobra.Command{}, f); err != nil {
		t.Fatalf("runStatus error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "auth: oauth") {
		t.Errorf("expected 'auth: oauth' in output, got:\n%s", output)
	}
}

func TestRunStatusHidesOAuthExpiryWithBKTToken(t *testing.T) {
	t.Setenv(secret.EnvToken, "override-token")

	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"api.bitbucket.org": {
				Kind:       "cloud",
				BaseURL:    "https://api.bitbucket.org/2.0",
				Username:   "erank_ai21",
				AuthMethod: "oauth",
			},
		},
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	if err := runStatus(&cobra.Command{}, f); err != nil {
		t.Fatalf("runStatus error: %v", err)
	}

	output := stdout.String()
	if strings.Contains(output, "expires:") {
		t.Errorf("should not show expires when BKT_TOKEN is set, got:\n%s", output)
	}
	if !strings.Contains(output, "token source: BKT_TOKEN") {
		t.Errorf("expected BKT_TOKEN source, got:\n%s", output)
	}
}

func TestRunStatusShowsNoHostsMessage(t *testing.T) {
	t.Setenv(secret.EnvToken, "")

	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	if err := runStatus(&cobra.Command{}, f); err != nil {
		t.Fatalf("runStatus error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No hosts configured") {
		t.Errorf("expected no-hosts message, got:\n%s", output)
	}
}

func TestRunLogoutRequiresHost(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogout(&cobra.Command{}, f, &logoutOptions{Host: ""})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "host is required") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLogoutUnknownHost(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogout(&cobra.Command{}, f, &logoutOptions{Host: "unknown.example.com"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q", err)
	}
}

func TestNewCmdAuthHasSubcommands(t *testing.T) {
	f, _, _ := newAuthTestFactory(&config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	})

	cmd := NewCmdAuth(f)
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"login", "status", "logout"} {
		if !names[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestRunLoginCloudNonWebRequiresUsername(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host: "https://bitbucket.org",
		Kind: "cloud",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "username is required") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLoginCloudNonWebRequiresToken(t *testing.T) {
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:     "https://bitbucket.org",
		Kind:     "cloud",
		Username: "user@example.com",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "token is required") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLoginCloudWebTokenNonTTYSkipsBrowserPrompt(t *testing.T) {
	// --web-token in non-TTY: browser scope output is skipped, falls through
	// to username prompt which errors in non-TTY.
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:     "https://bitbucket.org",
		Kind:     "cloud",
		WebToken: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "username is required") {
		t.Errorf("error = %q", err)
	}
	// In non-TTY, the browser scope output should NOT be printed.
	if strings.Contains(stdout.String(), "Opening Atlassian") {
		t.Error("should not print browser prompt in non-TTY")
	}
}

func TestRunLoginDCWebTokenNonTTY(t *testing.T) {
	// --web-token on DC in non-TTY: browser output skipped, falls to username prompt.
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:     "https://bitbucket.example.com",
		Kind:     "dc",
		WebToken: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "username is required") {
		t.Errorf("error = %q", err)
	}
	if strings.Contains(stdout.String(), "Opening") {
		t.Error("should not print browser prompt in non-TTY")
	}
}

func TestRunLoginDCBearerSkipsUsernamePrompt(t *testing.T) {
	// DC with bearer auth skips username prompt, goes straight to token prompt.
	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(&cobra.Command{}, f, &loginOptions{
		Host:       "https://bitbucket.example.com",
		Kind:       "dc",
		AuthMethod: "bearer",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// Should ask for token, not username.
	if !strings.Contains(err.Error(), "token is required") {
		t.Errorf("error = %q, want 'token is required'", err)
	}
}

func TestRunStatusShowsContexts(t *testing.T) {
	t.Setenv(secret.EnvToken, "")

	cfg := &config.Config{
		ActiveContext: "dev",
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
			},
		},
		Contexts: map[string]*config.Context{
			"dev": {
				Host:       "bitbucket.example.com",
				ProjectKey: "PROJ",
				Workspace:  "ws",
			},
		},
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	if err := runStatus(&cobra.Command{}, f); err != nil {
		t.Fatalf("runStatus error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "* dev") {
		t.Errorf("expected active context marker, got:\n%s", output)
	}
	if !strings.Contains(output, "project: PROJ") {
		t.Errorf("expected project in output, got:\n%s", output)
	}
}

func TestLoginCmdHasWebAndWebTokenFlags(t *testing.T) {
	f, _, _ := newAuthTestFactory(&config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	})

	cmd := newLoginCmd(f)
	if cmd.Flag("web") == nil {
		t.Error("missing --web flag")
	}
	if cmd.Flag("web-token") == nil {
		t.Error("missing --web-token flag")
	}
}

// setupFileKeyring configures env vars for file-backed keyring in a temp dir.
func setupFileKeyring(t *testing.T) {
	t.Helper()
	t.Setenv("BKT_ALLOW_INSECURE_STORE", "1")
	t.Setenv("BKT_KEYRING_PASSPHRASE", "test-pass")
	t.Setenv("KEYRING_BACKEND", "file")
	t.Setenv("KEYRING_FILE_DIR", t.TempDir())
}

func TestRunLoginDCFullFlow(t *testing.T) {
	// Mock DC server: /rest/api/1.0/users/admin returns user info.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/rest/api/1.0/users/admin") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":        "admin",
				"displayName": "Admin User",
				"slug":        "admin",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	setupFileKeyring(t)
	t.Setenv(secret.EnvToken, "")

	tmpDir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", tmpDir)

	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	err := runLogin(newTestCmd(), f, &loginOptions{
		Host:               srv.URL,
		Kind:               "dc",
		Username:           "admin",
		Token:              "test-pat",
		AllowHTTP:          true,
		AllowInsecureStore: true,
	})
	if err != nil {
		t.Fatalf("runLogin: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Logged in") {
		t.Errorf("expected success message, got:\n%s", output)
	}
	if !strings.Contains(output, "Admin User") {
		t.Errorf("expected display name in output, got:\n%s", output)
	}
}

func TestRunLoginDCBearerFullFlow(t *testing.T) {
	// Mock DC server: /rest/api/1.0/users?limit=1 returns user list.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/rest/api/1.0/users") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"name": "admin", "displayName": "Admin User"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	setupFileKeyring(t)
	t.Setenv(secret.EnvToken, "")

	tmpDir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", tmpDir)

	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	err := runLogin(newTestCmd(), f, &loginOptions{
		Host:               srv.URL,
		Kind:               "dc",
		AuthMethod:         "bearer",
		Token:              "test-pat",
		AllowHTTP:          true,
		AllowInsecureStore: true,
	})
	if err != nil {
		t.Fatalf("runLogin: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Logged in") {
		t.Errorf("expected success message, got:\n%s", output)
	}
	if !strings.Contains(output, "bearer token") {
		t.Errorf("expected 'bearer token' display name, got:\n%s", output)
	}
}

func TestRunLoginCloudAPITokenFullFlow(t *testing.T) {
	// Mock Cloud server: /user returns user info.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/user" || strings.HasSuffix(r.URL.Path, "/user") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"username":     "erank",
				"display_name": "Eran K",
				"account_id":   "123:abc",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	setupFileKeyring(t)
	t.Setenv(secret.EnvToken, "")

	tmpDir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", tmpDir)

	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	err := runLogin(newTestCmd(), f, &loginOptions{
		Host:               srv.URL,
		Kind:               "cloud",
		Username:           "erank@example.com",
		Token:              "api-token",
		AllowHTTP:          true,
		AllowInsecureStore: true,
	})
	if err != nil {
		t.Fatalf("runLogin: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Logged in to Bitbucket Cloud") {
		t.Errorf("expected success message, got:\n%s", output)
	}
	if !strings.Contains(output, "erank") {
		t.Errorf("expected username in output, got:\n%s", output)
	}
}

func TestRunLoginDCVerifyFails(t *testing.T) {
	// Mock DC server that returns 401.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(newTestCmd(), f, &loginOptions{
		Host:      srv.URL,
		Kind:      "dc",
		Username:  "admin",
		Token:     "bad-token",
		AllowHTTP: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "verify credentials") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLoginCloudVerifyFails(t *testing.T) {
	// Mock Cloud server that returns 401.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := &config.Config{
		Hosts:    make(map[string]*config.Host),
		Contexts: make(map[string]*config.Context),
	}
	f, _, _ := newAuthTestFactory(cfg)

	err := runLogin(newTestCmd(), f, &loginOptions{
		Host:      srv.URL,
		Kind:      "cloud",
		Username:  "user@example.com",
		Token:     "bad-token",
		AllowHTTP: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "verify credentials") {
		t.Errorf("error = %q", err)
	}
}

func TestRunLogoutFullFlow(t *testing.T) {
	setupFileKeyring(t)
	t.Setenv(secret.EnvToken, "")

	tmpDir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", tmpDir)

	// Pre-store a token so delete succeeds.
	store, storeErr := secret.Open(secret.WithAllowFileFallback(true))
	if storeErr != nil {
		t.Fatalf("open store: %v", storeErr)
	}
	hostKey := "bitbucket.example.com"
	if err := store.Set(secret.TokenKey(hostKey), "test-token"); err != nil {
		t.Fatalf("store token: %v", err)
	}

	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			hostKey: {
				Kind:               "dc",
				BaseURL:            "https://bitbucket.example.com",
				Username:           "admin",
				AllowInsecureStore: true,
			},
		},
		Contexts: map[string]*config.Context{
			"dev": {Host: hostKey},
		},
		ActiveContext: "dev",
	}
	f, stdout, _ := newAuthTestFactory(cfg)

	err := runLogout(&cobra.Command{}, f, &logoutOptions{Host: hostKey})
	if err != nil {
		t.Fatalf("runLogout: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Removed credentials") {
		t.Errorf("expected removal message, got:\n%s", output)
	}
}

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

func newAuthTestFactory(cfg *config.Config) (*cmdutil.Factory, *strings.Builder, *strings.Builder) {
	var stdout, stderr strings.Builder
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    &stdout,
			ErrOut: &stderr,
			In:     io.NopCloser(strings.NewReader("")),
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}
	return f, &stdout, &stderr
}
