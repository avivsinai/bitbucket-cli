package cmdutil

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// ErrSilent mirrors gh's sentinel used to suppress error printing.
	ErrSilent = errors.New("silent")
)

// ExitError wraps an exit code and optional message.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string {
	return e.Msg
}

// NotImplemented returns a helpful placeholder error for unfinished commands.
func NotImplemented(cmd *cobra.Command) error {
	return fmt.Errorf("%s not yet implemented", cmd.CommandPath())
}
