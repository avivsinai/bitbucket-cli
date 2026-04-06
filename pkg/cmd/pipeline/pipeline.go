package pipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
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
	cmd.AddCommand(newLogsCmd(f))

	return cmd
}

type baseOptions struct {
	Workspace string
	Repo      string
}

type runOptions struct {
	baseOptions
	Ref       string
	Variables []string
}

type listOptions struct {
	baseOptions
	Limit int
}

type viewOptions struct {
	baseOptions
	Identifier string // UUID or build number
}

type logsOptions struct {
	baseOptions
	Identifier string // UUID or build number
	Step       string
}

func newRunCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &runOptions{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a new pipeline run",
		Long: `Trigger a new pipeline run on Bitbucket Cloud for the current repository.

The pipeline runs against the specified Git ref (branch, tag, or commit). You can
pass custom pipeline variables using the --var flag, which accepts KEY=VALUE pairs
and can be repeated. This command is available for Bitbucket Cloud contexts only.`,
		Example: `  # Run the pipeline on the default branch
  bkt pipeline run

  # Run the pipeline on a specific branch
  bkt pipeline run --ref feature/my-branch

  # Run with custom pipeline variables
  bkt pipeline run --ref main --var ENV=staging --var DEBUG=true

  # Run against a specific repository
  bkt pipeline run --workspace myteam --repo backend-api --ref develop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipelineRun(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Ref, "ref", "main", "Git ref to run the pipeline on")
	cmd.Flags().StringSliceVar(&opts.Variables, "var", nil, "Pipeline variable in KEY=VALUE form (repeatable)")

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{Limit: 20}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List recent pipeline runs",
		Long: `List recent pipeline runs for a Bitbucket Cloud repository.

Displays build number, UUID, state, result, target branch, and creation time for
each pipeline. By default the most recent 20 runs are shown; use --limit to adjust.
This command is available for Bitbucket Cloud contexts only.`,
		Example: `  # List the 20 most recent pipeline runs
  bkt pipeline list

  # List the last 5 pipeline runs
  bkt pipeline list --limit 5

  # List pipelines for a specific repository
  bkt pipeline list --workspace myteam --repo backend-api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipelineList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum pipelines to display")

	return cmd
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "Show details for a pipeline run",
		Long: `Show details for a pipeline run on Bitbucket Cloud.

Displays the pipeline state, result, and a breakdown of each step with its UUID,
status, and name. The <id> argument accepts either a build number (e.g., 10) or a
pipeline UUID. This command is available for Bitbucket Cloud contexts only.`,
		Example: `  # View pipeline run by build number
  bkt pipeline view 42

  # View pipeline run by UUID
  bkt pipeline view '{a1b2c3d4-e5f6-7890-abcd-ef1234567890}'

  # View a pipeline in a specific repository
  bkt pipeline view 10 --workspace myteam --repo backend-api`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Identifier = args[0]
			return runPipelineView(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func newLogsCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &logsOptions{}
	cmd := &cobra.Command{
		Use:   "logs <id>",
		Short: "Fetch logs for a pipeline run",
		Long: `Fetch logs for a pipeline run on Bitbucket Cloud.

Prints the log output for a pipeline step. By default the last step is selected;
use --step to target a specific step UUID. The <id> argument accepts either a build
number (e.g., 10) or a pipeline UUID. This command is available for Bitbucket Cloud
contexts only.`,
		Example: `  # Fetch logs for the latest step of pipeline #42
  bkt pipeline logs 42

  # Fetch logs for a specific step
  bkt pipeline logs 42 --step '{step-uuid-here}'

  # Fetch logs using a pipeline UUID
  bkt pipeline logs '{a1b2c3d4-e5f6-7890-abcd-ef1234567890}'

  # Fetch logs for a pipeline in a specific repository
  bkt pipeline logs 10 --workspace myteam --repo backend-api`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Identifier = args[0]
			return runPipelineLogs(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Step, "step", "", "Specific step UUID to fetch logs for")

	return cmd
}

func runPipelineRun(cmd *cobra.Command, f *cmdutil.Factory, opts *runOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudRepo(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	vars := make(map[string]string)
	for _, v := range opts.Variables {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid variable %q, expected KEY=VALUE", v)
		}
		vars[strings.TrimSpace(parts[0])] = parts[1]
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	pipeline, err := client.TriggerPipeline(ctx, workspace, repo, bbcloud.TriggerPipelineInput{
		Ref:       opts.Ref,
		Variables: vars,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Triggered pipeline %s on %s/%s (%s)\n", pipeline.UUID, workspace, repo, pipeline.State.Name); err != nil {
		return err
	}
	return nil
}

func runPipelineList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudRepo(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	pipelines, err := client.ListPipelines(ctx, workspace, repo, opts.Limit)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"workspace": workspace,
		"repo":      repo,
		"pipelines": pipelines,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(pipelines) == 0 {
			_, err := fmt.Fprintln(ios.Out, "No pipelines found.")
			return err
		}
		for _, p := range pipelines {
			created := ""
			if p.CreatedOn != "" {
				if t, err := time.Parse(time.RFC3339Nano, p.CreatedOn); err == nil {
					created = t.Local().Format("2006-01-02 15:04")
				}
			}
			if _, err := fmt.Fprintf(ios.Out, "#%-4d %s\t%-12s\t%-10s\t%s\t%s\n",
				p.BuildNumber, p.UUID, p.State.Name, p.State.Result.Name, p.Target.Ref.Name, created); err != nil {
				return err
			}
		}
		return nil
	})
}

// resolvePipeline fetches a pipeline by build number or UUID.
// If the identifier looks like a number, tries build number first, then falls back to UUID.
func resolvePipeline(ctx context.Context, client *bbcloud.Client, workspace, repo, identifier string) (*bbcloud.Pipeline, error) {
	// Try parsing as build number first
	if buildNum, err := strconv.Atoi(strings.TrimPrefix(identifier, "#")); err == nil {
		pipeline, err := client.GetPipelineByBuildNumber(ctx, workspace, repo, buildNum)
		if err == nil {
			return pipeline, nil
		}
		// If build number lookup failed, fall back to treating as UUID
		// (in case the numeric string is actually part of a UUID)
	}
	// Treat as UUID
	return client.GetPipeline(ctx, workspace, repo, identifier)
}

func runPipelineView(cmd *cobra.Command, f *cmdutil.Factory, opts *viewOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudRepo(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	pipeline, err := resolvePipeline(ctx, client, workspace, repo, opts.Identifier)
	if err != nil {
		return err
	}

	steps, err := client.ListPipelineSteps(ctx, workspace, repo, pipeline.UUID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"pipeline": pipeline,
		"steps":    steps,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if _, err := fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", pipeline.UUID, pipeline.State.Name, pipeline.State.Result.Name); err != nil {
			return err
		}
		if len(steps) > 0 {
			if _, err := fmt.Fprintln(ios.Out, "Steps:"); err != nil {
				return err
			}
			for _, step := range steps {
				if _, err := fmt.Fprintf(ios.Out, "  %s\t%-24s\t%s\n", step.UUID, step.Status(), step.Name); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func runPipelineLogs(cmd *cobra.Command, f *cmdutil.Factory, opts *logsOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudRepo(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// Resolve build number or UUID to pipeline
	pipeline, err := resolvePipeline(ctx, client, workspace, repo, opts.Identifier)
	if err != nil {
		return err
	}

	stepID := opts.Step
	if stepID == "" {
		steps, err := client.ListPipelineSteps(ctx, workspace, repo, pipeline.UUID)
		if err != nil {
			return err
		}
		if len(steps) == 0 {
			return fmt.Errorf("pipeline #%d has no steps yet", pipeline.BuildNumber)
		}
		stepID = steps[len(steps)-1].UUID
	}

	logs, err := client.GetPipelineLogs(ctx, workspace, repo, pipeline.UUID, stepID)
	if err != nil {
		return err
	}

	if _, err := ios.Out.Write(logs); err != nil {
		return err
	}
	return nil
}

func resolveCloudRepo(cmd *cobra.Command, f *cmdutil.Factory, workspaceOverride, repoOverride string) (string, string, *config.Host, error) {
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return "", "", nil, err
	}
	if host.Kind != "cloud" {
		return "", "", nil, fmt.Errorf("command supports Bitbucket Cloud contexts only")
	}

	workspace := cmdutil.FirstNonEmpty(workspaceOverride, ctxCfg.Workspace)
	repo := cmdutil.FirstNonEmpty(repoOverride, ctxCfg.DefaultRepo)
	if workspace == "" || repo == "" {
		return "", "", nil, fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
	}

	return workspace, repo, host, nil
}
