package perms

import (
	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdPerms manages project and repository permissions.
func NewCmdPerms(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perms",
		Short: "Manage Bitbucket permissions",
	}

	cmd.AddCommand(newProjectCmd(f))
	cmd.AddCommand(newRepoCmd(f))

	return cmd
}

func newProjectCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage project-level permissions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List project permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "grant",
		Short: "Grant project permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "revoke",
		Short: "Revoke project permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	})

	return cmd
}

func newRepoCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repository-level permissions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List repository permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "grant",
		Short: "Grant repository permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "revoke",
		Short: "Revoke repository permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	})

	return cmd
}
