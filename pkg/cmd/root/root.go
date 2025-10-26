package root

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmd/admin"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/api"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/auth"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/branch"
	contextcmd "github.com/avivsinai/bitbucket-cli/pkg/cmd/context"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/extension"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/perms"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/pipeline"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/pr"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/repo"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/status"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/webhook"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdRoot assembles the root Cobra command using shared dependencies.
func NewCmdRoot(f *cmdutil.Factory) (*cobra.Command, error) {
	ios, err := f.Streams()
	if err != nil {
		return nil, err
	}

	root := &cobra.Command{
		Use:   f.ExecutableName,
		Short: "Bitbucket CLI with gh-style ergonomics.",
		Long: `Work seamlessly with Bitbucket Data Center and Cloud from the command line.

Common flows:
  bkt auth login https://bitbucket.example.com
  bkt pr list --mine
  bkt status pr 123 --json`,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	root.SetContext(context.Background())

	root.PersistentFlags().StringP("context", "c", "", "Active Bitbucket context name")
	root.PersistentFlags().Bool("json", false, "Output in JSON format when supported")
	root.PersistentFlags().Bool("yaml", false, "Output in YAML format when supported")
	root.PersistentFlags().String("jq", "", "Apply a jq expression to JSON output (requires --json)")
	root.PersistentFlags().String("template", "", "Render output using Go templates")

	root.AddCommand(
		admin.NewCmdAdmin(f),
		auth.NewCmdAuth(f),
		contextcmd.NewCmdContext(f),
		repo.NewCmdRepo(f),
		pr.NewCmdPR(f),
		branch.NewCmdBranch(f),
		perms.NewCommand(f),
		webhook.NewCommand(f),
		status.NewCmdStatus(f),
		pipeline.NewCmdPipeline(f),
		api.NewCmdAPI(f),
		extension.NewCmdExtension(f),
	)

	root.Version = f.AppVersion
	root.SetIn(ios.In)
	root.SetOut(ios.Out)
	root.SetErr(ios.ErrOut)

	return root, nil
}
