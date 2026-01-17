package cmdutil

import (
	"sync"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/browser"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/pager"
	"github.com/avivsinai/bitbucket-cli/pkg/progress"
	"github.com/avivsinai/bitbucket-cli/pkg/prompter"
)

// Factory wires together shared services used by Cobra commands.
type Factory struct {
	AppVersion     string
	ExecutableName string

	IOStreams *iostreams.IOStreams

	Config func() (*config.Config, error)

	// Lazy-initialised platform helpers.
	Browser  browser.Browser
	Pager    pager.Manager
	Prompter prompter.Interface
	Spinner  progress.Spinner

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

// BrowserOpener returns a Browser, initialising the default system implementation
// when necessary.
func (f *Factory) BrowserOpener() browser.Browser {
	if f.Browser == nil {
		f.Browser = browser.NewSystem()
	}
	return f.Browser
}

// PagerManager returns the pager manager, defaulting to a system-backed
// instance bound to the factory streams.
func (f *Factory) PagerManager() pager.Manager {
	if f.Pager == nil {
		ios, _ := f.Streams()
		f.Pager = pager.NewSystem(ios)
	}
	return f.Pager
}

// Prompt returns the prompter helper for interactive input.
func (f *Factory) Prompt() prompter.Interface {
	if f.Prompter == nil {
		ios, _ := f.Streams()
		f.Prompter = prompter.New(ios)
	}
	return f.Prompter
}

// ProgressSpinner exposes a spinner helper for long-running operations.
func (f *Factory) ProgressSpinner() progress.Spinner {
	if f.Spinner == nil {
		ios, _ := f.Streams()
		f.Spinner = progress.NewSpinner(ios)
	}
	return f.Spinner
}
