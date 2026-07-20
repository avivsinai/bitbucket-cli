package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// NewCmdPipeline interacts with Bitbucket Cloud pipelines.
func NewCmdPipeline(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Run and inspect Bitbucket Cloud pipelines (Cloud only)",
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

// waitOptions bundles the --wait polling flags shared by run and view.
type waitOptions struct {
	Wait        bool
	Interval    time.Duration
	MaxInterval time.Duration
	Timeout     time.Duration
	waitForPoll func(context.Context, time.Duration) error
}

type runOptions struct {
	baseOptions
	waitOptions
	Ref       string
	Variables []string
}

type listOptions struct {
	baseOptions
	Limit int
}

type viewOptions struct {
	baseOptions
	waitOptions
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
		Short: "Trigger a new pipeline run (Cloud only)",
		Long: `Trigger a new pipeline run on Bitbucket Cloud for the current repository.

The pipeline runs against the specified Git ref (branch, tag, or commit). You can
pass custom pipeline variables using the --var flag, which accepts KEY=VALUE pairs
and can be repeated. This command is available for Bitbucket Cloud contexts only.

Use --wait to poll the triggered pipeline until it completes, with exponential
backoff and jitter. Exit codes in --wait mode: 0 = pipeline succeeded,
1 = pipeline completed unsuccessfully, 8 = timed out while still running.`,
		Example: `  # Run the pipeline on the default branch
  bkt pipeline run

  # Run the pipeline on a specific branch
  bkt pipeline run --ref feature/my-branch

  # Run with custom pipeline variables
  bkt pipeline run --ref main --var ENV=staging --var DEBUG=true

  # Run against a specific repository
  bkt pipeline run --workspace myteam --repo backend-api --ref develop

  # Trigger and wait for the result
  bkt pipeline run --ref main --wait`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateWaitFlags(cmd, &opts.waitOptions); err != nil {
				return err
			}
			return runPipelineRun(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Ref, "ref", "main", "Git ref to run the pipeline on")
	cmd.Flags().StringSliceVar(&opts.Variables, "var", nil, "Pipeline variable in KEY=VALUE form (repeatable)")
	addWaitFlags(cmd, &opts.waitOptions, "Wait for the triggered pipeline to complete")

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{Limit: 20}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List recent pipeline runs (Cloud only)",
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
		Short: "Show details for a pipeline run (Cloud only)",
		Long: `Show details for a pipeline run on Bitbucket Cloud.

Displays the pipeline state, result, and a breakdown of each step with its UUID,
status, and name. The <id> argument accepts either a build number (e.g., 10) or a
pipeline UUID. This command is available for Bitbucket Cloud contexts only.

Use --wait to poll until the pipeline completes, with exponential backoff and
jitter. Exit codes in --wait mode: 0 = pipeline succeeded, 1 = pipeline
completed unsuccessfully, 8 = timed out while still running.`,
		Example: `  # View pipeline run by build number
  bkt pipeline view 42

  # View pipeline run by UUID
  bkt pipeline view '{a1b2c3d4-e5f6-7890-abcd-ef1234567890}'

  # View a pipeline in a specific repository
  bkt pipeline view 10 --workspace myteam --repo backend-api

  # Wait for a running pipeline to finish
  bkt pipeline view 42 --wait`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Identifier = args[0]
			if err := validateWaitFlags(cmd, &opts.waitOptions); err != nil {
				return err
			}
			return runPipelineView(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	addWaitFlags(cmd, &opts.waitOptions, "Wait for the pipeline to complete")

	return cmd
}

func newLogsCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &logsOptions{}
	cmd := &cobra.Command{
		Use:   "logs <id>",
		Short: "Fetch logs for a pipeline run (Cloud only)",
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

	if !opts.Wait {
		_, err := fmt.Fprintf(ios.Out, "✓ Triggered pipeline %s on %s/%s (%s)\n", pipeline.UUID, workspace, repo, pipeline.State.Name)
		return err
	}

	quietPoll, err := quietPollRequested(cmd)
	if err != nil {
		return err
	}
	if !quietPoll {
		if _, err := fmt.Fprintf(ios.Out, "✓ Triggered pipeline %s on %s/%s (%s)\n", pipeline.UUID, workspace, repo, pipeline.State.Name); err != nil {
			return err
		}
	}

	pipeline, timedOut, err := waitForPipeline(cmd, ios, &opts.waitOptions, pipeline, func(c context.Context) (*bbcloud.Pipeline, error) {
		return client.GetPipeline(c, workspace, repo, pipeline.UUID)
	})
	if err != nil || pipeline == nil {
		return err
	}

	payload := map[string]any{
		"workspace": workspace,
		"repo":      repo,
		"pipeline":  pipeline,
	}
	if err := cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		// Without a TTY the poll loop already printed the terminal status;
		// match pr checks and skip the duplicate final print.
		if !ios.IsStdoutTTY() {
			return nil
		}
		_, err := fmt.Fprintf(ios.Out, "Pipeline #%d (%s): %s\n", pipeline.BuildNumber, pipeline.Target.Ref.Name, pipelineStatus(pipeline))
		return err
	}); err != nil {
		return err
	}
	return waitExitError(pipeline, timedOut)
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
func resolvePipeline(ctx context.Context, client *bbcloud.Client, workspace, repo, identifier string) (*bbcloud.Pipeline, error) {
	if buildNum, err := strconv.Atoi(strings.TrimPrefix(identifier, "#")); err == nil {
		return client.GetPipelineByBuildNumber(ctx, workspace, repo, buildNum)
	}
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

	var timedOut bool
	if opts.Wait {
		pipeline, timedOut, err = waitForPipeline(cmd, ios, &opts.waitOptions, pipeline, func(c context.Context) (*bbcloud.Pipeline, error) {
			return client.GetPipeline(c, workspace, repo, pipeline.UUID)
		})
		if err != nil || pipeline == nil {
			return err
		}
	}

	// The wait path derives its own timeout/signal context, so cmd.Context()
	// is still live here; give the steps fetch its own short deadline while
	// preserving caller cancellation.
	stepsCtx, stepsCancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer stepsCancel()

	steps, err := client.ListPipelineSteps(stepsCtx, workspace, repo, pipeline.UUID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"pipeline": pipeline,
		"steps":    steps,
	}

	if err := cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
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
	}); err != nil {
		return err
	}

	if opts.Wait {
		return waitExitError(pipeline, timedOut)
	}
	return nil
}

// addWaitFlags registers the --wait polling flags shared by run and view.
func addWaitFlags(cmd *cobra.Command, w *waitOptions, waitHelp string) {
	cmd.Flags().BoolVar(&w.Wait, "wait", false, waitHelp)
	cmd.Flags().DurationVar(&w.Interval, "interval", 10*time.Second, "Initial polling interval when using --wait")
	cmd.Flags().DurationVar(&w.MaxInterval, "max-interval", 2*time.Minute, "Maximum polling interval (backoff cap)")
	cmd.Flags().DurationVar(&w.Timeout, "timeout", 30*time.Minute, "Maximum time to wait for the pipeline (0 for no timeout)")
}

// validateWaitFlags enforces the same flag rules as `bkt pr checks`:
// polling flags require --wait, and intervals must be sane.
func validateWaitFlags(cmd *cobra.Command, w *waitOptions) error {
	if !w.Wait {
		for _, name := range []string{"interval", "max-interval", "timeout"} {
			if cmd.Flags().Changed(name) {
				return fmt.Errorf("--%s requires --wait", name)
			}
		}
		return nil
	}
	if w.Interval <= 0 {
		return fmt.Errorf("--interval must be positive")
	}
	if w.MaxInterval <= 0 {
		return fmt.Errorf("--max-interval must be positive")
	}
	if w.MaxInterval < w.Interval {
		return fmt.Errorf("--max-interval must be >= --interval")
	}
	return nil
}

// quietPollRequested reports whether structured output (--json/--yaml/
// --template/--jq) is active, in which case poll progress must not be printed.
func quietPollRequested(cmd *cobra.Command) (bool, error) {
	settings, err := cmdutil.ResolveOutputSettings(cmd)
	if err != nil {
		return false, err
	}
	return settings.Format != "" || settings.Template != "" || settings.JQ != "", nil
}

// waitForPipeline polls until the pipeline completes, handling signal
// cancellation, --timeout, and progress output. It returns the last known
// pipeline and whether the wait timed out while the pipeline was still
// running. A nil pipeline with nil error means the user cancelled.
func waitForPipeline(cmd *cobra.Command, ios *iostreams.IOStreams, w *waitOptions, initial *bbcloud.Pipeline, fetch func(context.Context) (*bbcloud.Pipeline, error)) (*bbcloud.Pipeline, bool, error) {
	if pipelineCompleted(initial) {
		return initial, false, nil
	}

	quietPoll, err := quietPollRequested(cmd)
	if err != nil {
		return nil, false, err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if w.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.Timeout)
		defer cancel()
	}

	if !quietPoll {
		ios.StartAlternateScreenBuffer()
	}
	pipeline, pollErr := pollPipelineUntilDone(ctx, ios, w, quietPoll, initial, fetch)
	if !quietPoll {
		ios.StopAlternateScreenBuffer()
	}

	if errors.Is(pollErr, context.Canceled) {
		_, _ = fmt.Fprintln(ios.ErrOut, "\nOperation cancelled")
		return nil, false, nil
	}
	if errors.Is(pollErr, context.DeadlineExceeded) {
		_, _ = fmt.Fprintln(ios.ErrOut, "\nTimeout waiting for pipeline to complete")
		return pipeline, !pipelineCompleted(pipeline), nil
	}
	if pollErr != nil {
		return nil, false, pollErr
	}
	return pipeline, false, nil
}

// pollPipelineUntilDone polls fetch until the pipeline reaches a terminal
// state, the context ends, or fetch fails repeatedly. The last successfully
// fetched pipeline is always returned so callers can report partial state.
func pollPipelineUntilDone(ctx context.Context, ios *iostreams.IOStreams, w *waitOptions, quietPoll bool, initial *bbcloud.Pipeline, fetch func(context.Context) (*bbcloud.Pipeline, error)) (*bbcloud.Pipeline, error) {
	last := initial
	iteration := 0
	consecutiveErrors := 0
	const maxConsecutiveErrors = 3

	for {
		if !quietPoll {
			if iteration > 0 {
				ios.ClearScreen()
			}
			if _, err := fmt.Fprintf(ios.Out, "Pipeline #%d (%s): %s\n", last.BuildNumber, last.Target.Ref.Name, pipelineStatus(last)); err != nil {
				return last, err
			}
		}

		if pipelineCompleted(last) {
			return last, nil
		}

		next := cmdutil.PollInterval(w.Interval, w.MaxInterval, iteration+consecutiveErrors)
		if !quietPoll {
			if _, err := fmt.Fprintf(ios.Out, "\n  Waiting for pipeline to complete... (next poll in %s, Ctrl-C to cancel)\n", next.Round(time.Second)); err != nil {
				return last, err
			}
		}
		iteration++

		if err := waitPipelinePoll(ctx, w, next); err != nil {
			return last, err
		}

		p, err := fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return last, ctx.Err()
			}
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				return last, fmt.Errorf("fetch pipeline failed after %d attempts: %w", consecutiveErrors, err)
			}
			_, _ = fmt.Fprintf(ios.ErrOut, "  Warning: error fetching pipeline (attempt %d/%d): %v\n", consecutiveErrors, maxConsecutiveErrors, err)
			continue
		}
		consecutiveErrors = 0
		last = p
	}
}

func waitPipelinePoll(ctx context.Context, w *waitOptions, d time.Duration) error {
	if w.waitForPoll != nil {
		return w.waitForPoll(ctx, d)
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// pipelineStatus renders a human-readable status such as
// "IN_PROGRESS (RUNNING)" or "COMPLETED SUCCESSFUL".
func pipelineStatus(p *bbcloud.Pipeline) string {
	s := p.State.Name
	if p.State.Result.Name != "" {
		return s + " " + p.State.Result.Name
	}
	if p.State.Stage.Name != "" {
		return s + " (" + p.State.Stage.Name + ")"
	}
	return s
}

// pipelineCompleted reports whether the pipeline reached a terminal state.
func pipelineCompleted(p *bbcloud.Pipeline) bool {
	return p != nil && strings.EqualFold(p.State.Name, "COMPLETED")
}

// pipelineSucceeded reports whether a completed pipeline finished successfully.
func pipelineSucceeded(p *bbcloud.Pipeline) bool {
	return p != nil && strings.EqualFold(p.State.Result.Name, "SUCCESSFUL")
}

// waitExitError maps the final wait outcome to the documented exit codes:
// 0 = succeeded, 1 = completed unsuccessfully, 8 = timed out still running.
func waitExitError(p *bbcloud.Pipeline, timedOut bool) error {
	if timedOut {
		return cmdutil.ErrPending
	}
	if !pipelineSucceeded(p) {
		return cmdutil.ErrSilent
	}
	return nil
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
		var ok bool
		stepID, ok = defaultPipelineLogStepID(steps)
		if !ok {
			return fmt.Errorf("pipeline #%d has no steps yet", pipeline.BuildNumber)
		}
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

func defaultPipelineLogStepID(steps []bbcloud.PipelineStep) (string, bool) {
	if len(steps) == 0 {
		return "", false
	}
	return steps[len(steps)-1].UUID, true
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
