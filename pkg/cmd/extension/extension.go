package extension

import (
	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdExtension manages external bkt extensions.
func NewCmdExtension(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extension",
		Short: "Manage bkt CLI extensions",
	}

	cmd.AddCommand(newInstallCmd(f))
	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newRemoveCmd(f))
	cmd.AddCommand(newExecCmd(f))

	return cmd
}

func newInstallCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <repository>",
		Short: "Install an extension from a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove an installed extension",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newExecCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <name> [args...]",
		Short: "Execute an installed extension",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}
