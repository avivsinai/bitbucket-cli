package pr

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

func newMergeabilityCmd(f *cmdutil.Factory) *cobra.Command {
	var workspace, repo string
	cmd := &cobra.Command{
		Use:   "mergeability <id>",
		Short: "Check whether a pull request can be merged (Cloud only)",
		Long: `Simulate Bitbucket Cloud's merge checks without attempting a merge.
Mergeable is MERGEABLE, CONFLICTING, or UNKNOWN. Merge state is CLEAN,
DIRTY (conflicts), BLOCKED (a blocking check), or UNKNOWN (incomplete checks).
Structured output includes the complete checks and their blocking flags.

CLEAN describes the checks available now; Bitbucket cannot evaluate every
condition without an actual merge attempt. API failures print UNKNOWN and
exit nonzero. Data Center is not supported.`,
		Example: `  bkt pr mergeability 42
  bkt pr mergeability 42 --json
  bkt pr mergeability 42 --json --jq '{mergeable, mergeStateStatus}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			ios, err := f.Streams()
			if err != nil {
				return err
			}
			_, cfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
			if err != nil {
				return err
			}
			if host.Kind != "cloud" {
				return fmt.Errorf("pr mergeability is supported only on Bitbucket Cloud")
			}
			workspace = cmdutil.FirstNonEmpty(workspace, cfg.Workspace)
			repo = cmdutil.FirstNonEmpty(repo, cfg.DefaultRepo)
			if workspace == "" || repo == "" {
				return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
			}
			client, err := f.CloudClient(host)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeoutRead)
			defer cancel()
			result, fetchErr := client.GetPullRequestMergeability(ctx, workspace, repo, id)
			if err := cmdutil.WriteOutput(cmd, ios.Out, result, func() error {
				_, err := fmt.Fprintf(ios.Out, "Mergeable: %s\nMerge state: %s\n", result.Mergeable, result.MergeStateStatus)
				return err
			}); err != nil {
				return err
			}
			return fetchErr
		},
	}
	cmd.Flags().StringVar(&workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository slug override")
	return cmd
}
