package commit

import (
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdCommit returns the commit command tree.
func NewCmdCommit(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Work with commits",
	}
	cmd.AddCommand(newDiffCmd(f))
	return cmd
}
