package iostreams

import (
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// IOStreams collects input and output streams for command execution.
//
// The structure mirrors gh/jk ergonomics by exposing terminal metadata and
// lazy colour profile detection. Commands can inspect the terminal
// capabilities to decide when to render ANSI colours, tables, or spinner
// widgets.
type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer

	isStdinTTY  bool
	isStdoutTTY bool
	isStderrTTY bool

	colorEnabled bool
	once         sync.Once
}

// System returns IOStreams bound to the current process standard streams and
// captures terminal metadata so downstream components can make ergonomic
// decisions (colours, paging, prompts, etc.).
func System() *IOStreams {
	isTTY := func(f *os.File) bool {
		if f == nil {
			return false
		}
		return term.IsTerminal(int(f.Fd()))
	}

	return &IOStreams{
		In:          os.Stdin,
		Out:         os.Stdout,
		ErrOut:      os.Stderr,
		isStdinTTY:  isTTY(os.Stdin),
		isStdoutTTY: isTTY(os.Stdout),
		isStderrTTY: isTTY(os.Stderr),
	}
}

// CanPrompt reports whether stdin is a TTY and therefore suitable for
// interactive prompts.
func (s *IOStreams) CanPrompt() bool {
	return s != nil && s.isStdinTTY
}

// ColorEnabled returns true when ANSI colour output should be rendered. The
// decision is cached so repeated checks are inexpensive.
func (s *IOStreams) ColorEnabled() bool {
	if s == nil {
		return false
	}
	s.once.Do(func() {
		s.colorEnabled = s.isStdoutTTY
	})
	return s.colorEnabled
}

// SetColorEnabled allows callers (e.g. tests) to force colour behaviour.
func (s *IOStreams) SetColorEnabled(enabled bool) {
	if s == nil {
		return
	}
	s.once.Do(func() {})
	s.colorEnabled = enabled
}

// IsStdoutTTY reports whether stdout is attached to a terminal.
func (s *IOStreams) IsStdoutTTY() bool {
	return s != nil && s.isStdoutTTY
}

// IsStderrTTY reports whether stderr is attached to a terminal.
func (s *IOStreams) IsStderrTTY() bool {
	return s != nil && s.isStderrTTY
}

// ANSI escape sequences for alternate screen buffer
const (
	enterAltScreen = "\x1b[?1049h"
	exitAltScreen  = "\x1b[?1049l"
	clearScreen    = "\x1b[2J\x1b[H" // Clear screen and move cursor to top-left
)

// StartAlternateScreenBuffer switches to the alternate screen buffer.
// This keeps the terminal clean during watch mode, showing only the final
// result on the original screen when done. No-op if not a TTY.
func (s *IOStreams) StartAlternateScreenBuffer() {
	if s == nil || !s.isStdoutTTY {
		return
	}
	_, _ = io.WriteString(s.Out, enterAltScreen)
}

// StopAlternateScreenBuffer switches back to the main screen buffer.
// No-op if not a TTY.
func (s *IOStreams) StopAlternateScreenBuffer() {
	if s == nil || !s.isStdoutTTY {
		return
	}
	_, _ = io.WriteString(s.Out, exitAltScreen)
}

// ClearScreen clears the screen and moves cursor to top-left.
// Useful for refreshing watch mode output. No-op if not a TTY.
func (s *IOStreams) ClearScreen() {
	if s == nil || !s.isStdoutTTY {
		return
	}
	_, _ = io.WriteString(s.Out, clearScreen)
}
