package progress

import (
	"fmt"
	"sync"
	"time"

	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// Spinner renders a simple textual indicator while a background task runs.
// Commands can use the spinner to provide user feedback in long-running
// operations without taking a dependency on external packages.
type Spinner interface {
	Start(msg string)
	Stop(msg string)
	Fail(msg string)
}

type noopSpinner struct {
	ios *iostreams.IOStreams
}

// NewSpinner constructs a terminal spinner when stderr is a TTY. Otherwise a
// newline-based fallback is returned.
func NewSpinner(ios *iostreams.IOStreams) Spinner {
	if ios != nil && ios.IsStderrTTY() {
		return newTTYSpinner(ios)
	}
	return &noopSpinner{ios: ios}
}

func (s *noopSpinner) Start(msg string) { s.write(msg) }
func (s *noopSpinner) Stop(msg string)  { s.write(msg) }
func (s *noopSpinner) Fail(msg string)  { s.write(msg) }

func (s *noopSpinner) write(msg string) {
	if s.ios == nil || msg == "" {
		return
	}
	fmt.Fprintln(s.ios.ErrOut, msg)
}

type ttySpinner struct {
	ios    *iostreams.IOStreams
	stopCh chan struct{}
	mux    sync.Mutex
}

func newTTYSpinner(ios *iostreams.IOStreams) Spinner {
	return &ttySpinner{ios: ios}
}

func (s *ttySpinner) Start(msg string) {
	s.mux.Lock()
	if s.stopCh != nil {
		close(s.stopCh)
	}
	s.stopCh = make(chan struct{})
	stop := s.stopCh
	s.mux.Unlock()

	frames := []rune{'|', '/', '-', '\\'}

	go func() {
		idx := 0
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fmt.Fprintf(s.ios.ErrOut, "\r%c %s", frames[idx], msg)
				idx = (idx + 1) % len(frames)
			}
		}
	}()
}

func (s *ttySpinner) Stop(msg string) {
	s.endWithPrefix("[OK]", msg)
}

func (s *ttySpinner) Fail(msg string) {
	s.endWithPrefix("[ERR]", msg)
}

func (s *ttySpinner) endWithPrefix(prefix, msg string) {
	s.mux.Lock()
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
	s.mux.Unlock()

	if msg == "" {
		return
	}
	fmt.Fprintf(s.ios.ErrOut, "\r%s %s\n", prefix, msg)
}
