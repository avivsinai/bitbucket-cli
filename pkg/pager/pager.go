package pager

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// Manager coordinates optional pager processes (e.g. less) used for long
// command output.
type Manager interface {
	Enabled() bool
	Start() (io.WriteCloser, error)
	Stop() error
}

type system struct {
	ios    *iostreams.IOStreams
	cmd    *exec.Cmd
	writer io.WriteCloser
}

// NewSystem returns a pager manager backed by the user's $PAGER when stdout is
// a TTY. When stdout is redirected a no-op manager is returned instead.
func NewSystem(ios *iostreams.IOStreams) Manager {
	if ios == nil || !ios.IsStdoutTTY() {
		return noop{}
	}
	return &system{ios: ios}
}

func (p *system) Enabled() bool { return true }

func (p *system) Start() (io.WriteCloser, error) {
	if p.writer != nil {
		return p.writer, nil
	}

	pagerCmd := strings.Fields(resolvePager())
	cmd := exec.Command(pagerCmd[0], pagerCmd[1:]...)
	cmd.Stdout = p.ios.Out
	cmd.Stderr = p.ios.ErrOut

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		_ = in.Close()
		return nil, err
	}

	p.cmd = cmd
	p.writer = in
	return in, nil
}

func (p *system) Stop() error {
	if p.writer != nil {
		_ = p.writer.Close()
		p.writer = nil
	}
	if p.cmd != nil {
		err := p.cmd.Wait()
		p.cmd = nil
		return err
	}
	return nil
}

type noop struct{}

func (noop) Enabled() bool { return false }
func (noop) Start() (io.WriteCloser, error) {
	return nopWriteCloser{Writer: os.Stdout}, nil
}
func (noop) Stop() error { return nil }

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func resolvePager() string {
	if cmd := os.Getenv("BKT_PAGER"); cmd != "" {
		return cmd
	}
	if cmd := os.Getenv("PAGER"); cmd != "" {
		return cmd
	}
	return "less -R"
}
