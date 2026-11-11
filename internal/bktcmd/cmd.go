package bktcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/avivsinai/bitbucket-cli/internal/build"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/factory"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/root"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// Main initialises CLI dependencies and executes the root command.
func Main() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	f, err := factory.New(build.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise factory: %v\n", err)
		return 1
	}

	ios, err := f.Streams()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to configure IO: %v\n", err)
		return 1
	}

	rootCmd, err := root.NewCmdRoot(f)
	if err != nil {
		_, _ = fmt.Fprintf(ios.ErrOut, "failed to create root command: %v\n", err)
		return 1
	}
	rootCmd.SetContext(ctx)

	return exitCode(ctx, rootCmd.ExecuteContext(ctx), ios.ErrOut)
}

// exitCode maps the command result to a process exit code. A cancelled root
// context means a signal (Ctrl-C, SIGTERM) interrupted the run, whatever the
// command returned: exit 130 and print nothing.
func exitCode(ctx context.Context, err error, errOut io.Writer) int {
	if ctx.Err() != nil {
		return 130
	}
	if err == nil {
		return 0
	}
	var exitErr *cmdutil.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.Msg != "" {
			_, _ = fmt.Fprintln(errOut, exitErr.Msg)
		}
		return exitErr.Code
	}
	// ErrPending: checks still pending (e.g., timeout hit) - exit code 8
	if errors.Is(err, cmdutil.ErrPending) {
		return 8
	}
	// ErrSilent: failure without message - exit code 1
	if errors.Is(err, cmdutil.ErrSilent) {
		return 1
	}
	_, _ = fmt.Fprintf(errOut, "Error: %v\n", err)
	return 1
}
