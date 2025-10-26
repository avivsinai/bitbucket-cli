package pipeline

import (
	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdPipeline interacts with Bitbucket Cloud pipelines.
func NewCmdPipeline(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Run and inspect Bitbucket Cloud pipelines",
		Long:  "Interact with Bitbucket Cloud Pipelines. Commands are no-ops for Data Center contexts.",
	}

	cmd.AddCommand(newRunCmd(f))
	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))

	return cmd
}

func newRunCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a new pipeline run",
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
		Short:   "List recent pipeline runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <uuid>",
		Short: "Show details for a pipeline run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}
