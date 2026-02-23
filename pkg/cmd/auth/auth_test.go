package auth

import (
	"io"
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
