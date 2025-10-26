package webhook

import (
	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdWebhook returns the webhook command.
func NewCmdWebhook(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage Bitbucket webhooks",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newTestCmd(f))

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured webhooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a webhook",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newTestCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <id>",
		Short: "Trigger a webhook test delivery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}
