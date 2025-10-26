package api

import (
	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdAPI exposes a raw REST escape hatch akin to gh api.
func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "Make raw Bitbucket API requests",
		Long:  "Call Bitbucket REST APIs directly. Helpful for experimentation and unsupported endpoints.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.NotImplemented(cmd)
		},
	}
	return cmd
}
