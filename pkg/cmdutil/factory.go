package cmdutil

import (
	"sync"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// Factory wires together shared services used by Cobra commands.
type Factory struct {
	AppVersion     string
	ExecutableName string

	IOStreams *iostreams.IOStreams

	Config func() (*config.Config, error)

	once struct {
		cfg sync.Once
	}
	cfg    *config.Config
	cfgErr error
	ioOnce sync.Once
	ios    *iostreams.IOStreams
}

// ResolveConfig loads configuration, caching the result.
func (f *Factory) ResolveConfig() (*config.Config, error) {
	f.once.cfg.Do(func() {
		if f.Config == nil {
			f.cfg, f.cfgErr = config.Load()
			return
		}
		f.cfg, f.cfgErr = f.Config()
	})
	return f.cfg, f.cfgErr
}

// Streams returns process IO streams, initialising them lazily.
func (f *Factory) Streams() (*iostreams.IOStreams, error) {
	f.ioOnce.Do(func() {
		if f.IOStreams != nil {
			f.ios = f.IOStreams
			return
		}
		f.ios = iostreams.System()
	})
	return f.ios, nil
}
