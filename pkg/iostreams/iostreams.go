package iostreams

import (
	"io"
	"os"
)

// IOStreams collects input and output streams for command execution.
type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer
}

// System returns IOStreams bound to the current process standard streams.
func System() *IOStreams {
	return &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}
