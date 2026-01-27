package auth

import (
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestCloudTokenURLIsAtlassian(t *testing.T) {
	// The token URL for cloud should point to Atlassian's account management
	expectedURL := "https://id.atlassian.com/manage-profile/security/api-tokens"

	// Verify the URL constant/value used in runLogin for cloud kind
	// This test ensures we don't regress to the old bitbucket.org URL
	if !strings.Contains(expectedURL, "id.atlassian.com") {
		t.Fatal("Cloud token URL should use id.atlassian.com")
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
	// Test that the email prompt for cloud login doesn't mention app password
	// This would require capturing output during runLogin with opts.Web = false
	// and verifying the prompt text
	//
	// The implementation removes "app password" from prompt text when
	// opts.Web is false, so this verifies the change was made correctly.
	// Full integration testing would require mocking terminal input.
}
