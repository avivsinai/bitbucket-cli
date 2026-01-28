package auth

import (
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
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
