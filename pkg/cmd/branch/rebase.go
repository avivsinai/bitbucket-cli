package branch

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/example/bitbucket-cli/pkg/cmdutil"
)

type rebaseOptions struct {
	Onto        string
	Interactive bool
	NoFetch     bool
}

func newRebaseCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &rebaseOptions{}
	cmd := &cobra.Command{
		Use:   "rebase <branch>",
		Short: "Rebase the current branch onto another branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Onto = args[0]
			return runRebase(cmd, f, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Interactive, "interactive", false, "Run rebase in interactive mode")
	cmd.Flags().BoolVar(&opts.NoFetch, "no-fetch", false, "Skip fetching before rebase")

	return cmd
}

func runRebase(cmd *cobra.Command, f *cmdutil.Factory, opts *rebaseOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	if !opts.NoFetch {
		fetch := exec.CommandContext(cmd.Context(), "git", "fetch", "--all")
		fetch.Stdout = ios.Out
		fetch.Stderr = ios.ErrOut
		fetch.Stdin = ios.In
		if err := fetch.Run(); err != nil {
			return fmt.Errorf("git fetch: %w", err)
		}
	}

	args := []string{"rebase"}
	if opts.Interactive {
		args = append(args, "-i")
	}
	args = append(args, opts.Onto)

	rebase := exec.CommandContext(cmd.Context(), "git", args...)
	rebase.Stdout = ios.Out
	rebase.Stderr = ios.ErrOut
	rebase.Stdin = ios.In
	if err := rebase.Run(); err != nil {
		return fmt.Errorf("git rebase: %w", err)
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Rebasing onto %s complete\n", opts.Onto); err != nil {
		return err
	}
	return nil
}
