package cmdutil

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// ErrSilent mirrors gh's sentinel used to suppress error printing.
	// Returns exit code 1.
	ErrSilent = errors.New("silent")

	// ErrPending signals that checks are still pending (e.g., timeout hit).
	// Returns exit code 8, matching gh pr checks behavior.
	ErrPending = errors.New("pending")
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
