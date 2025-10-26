package factory

import (
	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// New constructs a command factory following gh/jk idioms.
func New(appVersion string) (*cmdutil.Factory, error) {
	ios := iostreams.System()

	f := &cmdutil.Factory{
		AppVersion:     appVersion,
		ExecutableName: "bkt",
		IOStreams:      ios,
	}

	f.Config = func() (*config.Config, error) {
		return config.Load()
	}

	return f, nil
}
