package branch

import (
	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdBranch exposes branch operations.
func NewCmdBranch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Inspect and manage branches",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newSetDefaultCmd(f))
	cmd.AddCommand(newProtectCmd(f))

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <branch>",
		Short: "Create a new branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <branch>",
		Aliases: []string{"rm"},
		Short:   "Delete a branch",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newSetDefaultCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-default <branch>",
		Short: "Set the default branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newProtectCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protect <branch>",
		Short: "Configure branch protection rules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}
