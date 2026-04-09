package issue

import (
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestDeprecationWarning(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantWarning bool
	}{
		{
			name: "cloud context prints warning",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"cloud": {Kind: "cloud", BaseURL: "https://api.bitbucket.org/2.0", Token: "t"},
				},
			},
			wantWarning: true,
		},
		{
			name: "dc context does not print warning",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "mydc", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"mydc": {Kind: "dc", BaseURL: "https://bitbucket.example.com", Token: "t"},
				},
			},
			wantWarning: false,
		},
		{
			name: "no context does not crash",
			cfg: &config.Config{
				Contexts: map[string]*config.Context{},
				Hosts:    map[string]*config.Host{},
			},
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr strings.Builder
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams: &iostreams.IOStreams{
					Out:    &strings.Builder{},
					ErrOut: &stderr,
				},
				Config: func() (*config.Config, error) {
					return tt.cfg, nil
				},
			}

			cmd := NewCmdIssue(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"list"})
			_ = cmd.Execute() // subcommand fails (no API) but PersistentPreRunE runs first

			got := strings.Contains(stderr.String(), "WARNING: Bitbucket Cloud is removing native Issues")
			if got != tt.wantWarning {
				t.Errorf("warning present = %v, want %v; stderr = %q", got, tt.wantWarning, stderr.String())
			}
		})
	}
}
