package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type cloudStatusOptions struct {
	Workspace string
	Repo      string
	UUID      string
}

func newCloudPipelineCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &cloudStatusOptions{}
	cmd := &cobra.Command{
		Use:   "pipeline <uuid>",
		Short: "Show Bitbucket Cloud pipeline status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.UUID = args[0]
			return runCloudPipelineStatus(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runCloudPipelineStatus(cmd *cobra.Command, f *cmdutil.Factory, opts *cloudStatusOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudStatusContext(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	pipeline, err := client.GetPipeline(ctx, workspace, repo, opts.UUID)
	if err != nil {
		return err
	}

	steps, err := client.ListPipelineSteps(ctx, workspace, repo, opts.UUID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"pipeline": pipeline,
		"steps":    steps,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", pipeline.UUID, pipeline.State.Name, pipeline.State.Result.Name)
		fmt.Fprintf(ios.Out, "Ref: %s\n", pipeline.Target.Ref.Name)
		if pipeline.CreatedOn != "" {
			fmt.Fprintf(ios.Out, "Created: %s\n", pipeline.CreatedOn)
		}
		if pipeline.CompletedOn != "" {
			fmt.Fprintf(ios.Out, "Completed: %s\n", pipeline.CompletedOn)
		}
		if len(steps) > 0 {
			fmt.Fprintln(ios.Out, "Steps:")
			for _, step := range steps {
				fmt.Fprintf(ios.Out, "  %s\t%s\t%s\n", step.UUID, step.Name, step.Result.Name)
			}
		}
		return nil
	})
}

func resolveCloudStatusContext(cmd *cobra.Command, f *cmdutil.Factory, workspaceOverride, repoOverride string) (string, string, *config.Host, error) {
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return "", "", nil, err
	}
	if host.Kind != "cloud" {
		return "", "", nil, fmt.Errorf("command supports Bitbucket Cloud contexts only")
	}

	workspace := firstNonEmpty(workspaceOverride, ctxCfg.Workspace)
	repo := firstNonEmpty(repoOverride, ctxCfg.DefaultRepo)
	if workspace == "" || repo == "" {
		return "", "", nil, fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
	}

	return workspace, repo, host, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
