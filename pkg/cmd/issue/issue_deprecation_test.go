package issue

import (
	"errors"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// Bitbucket Cloud answers 410 Gone (CHANGE-3071) for every issues endpoint
// since Atlassian removed the tracker; bkt must say so instead of calling it.
func TestIssueCommandsReportRemoval(t *testing.T) {
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config: func() (*config.Config, error) {
			return &config.Config{
				ActiveContext: "default",
				Contexts:      map[string]*config.Context{"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"}},
				Hosts:         map[string]*config.Host{"cloud": {Kind: "cloud", BaseURL: "https://api.bitbucket.org/2.0", Token: "t"}},
			}, nil
		},
	}

	cmd := NewCmdIssue(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); !errors.Is(err, errIssuesRemoved) {
		t.Fatalf("issue list error = %v, want %v", err, errIssuesRemoved)
	}
}
