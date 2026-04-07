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
		Long: `Inspect and compare commits in a Bitbucket repository. Subcommands let you
view diffs between two commits or refs. Works with both Bitbucket Cloud and
Data Center; on Cloud the diff spec uses ".." notation, while on Data Center
the two refs are passed separately to the API.`,
		Example: `  # Show changes between two commit SHAs
  bkt commit diff abc1234 def5678

  # Compare a branch to main
  bkt commit diff feature/login main`,
	}
	cmd.AddCommand(newDiffCmd(f))
	return cmd
}
