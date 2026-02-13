package factory

import (
	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/browser"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/pager"
	"github.com/avivsinai/bitbucket-cli/pkg/progress"
	"github.com/avivsinai/bitbucket-cli/pkg/prompter"
)

// New constructs a command factory following gh/jk idioms.
func New(appVersion string) (*cmdutil.Factory, error) {
	ios := iostreams.System()

	f := &cmdutil.Factory{
		AppVersion:     appVersion,
		ExecutableName: "bkt",
		IOStreams:      ios,
	}

	f.Browser = browser.NewSystem()
	f.Pager = pager.NewSystem(ios)
	f.Prompter = prompter.New(ios)
	f.Spinner = progress.NewSpinner(ios)

	f.Config = config.Load

	return f, nil
}
