package pr

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

const taskRequestTimeout = 10 * time.Second

// errCommentIDOnlyLegacy is returned when --comment-id is supplied in a context
// where it carries no meaning (Cloud tasks and DC blocker comments are not
// anchored to a comment).
var errCommentIDOnlyLegacy = fmt.Errorf("--comment-id only applies to legacy Data Center tasks (--task-api legacy)")

type taskOptions struct {
	Project   string
	Repo      string
	Workspace string
	TaskAPI   string
	ID        int
	TaskID    int
	CommentID int
	Text      string
}

// taskView is the normalized, host-agnostic shape the CLI prints. Data Center
// states are OPEN/RESOLVED; Cloud's UNRESOLVED is normalized to OPEN.
type taskView struct {
	ID    int    `json:"id"`
	State string `json:"state"`
	Text  string `json:"text"`
}

func dcTaskViews(tasks []bbdc.PullRequestTask) []taskView {
	views := make([]taskView, 0, len(tasks))
	for _, t := range tasks {
		views = append(views, taskView{ID: t.ID, State: t.State, Text: t.Text})
	}
	return views
}

func cloudTaskView(t bbcloud.PullRequestTask) taskView {
	state := t.State
	if state == bbcloud.TaskStateUnresolved {
		state = "OPEN"
	}
	return taskView{ID: t.ID, State: state, Text: t.Content.Raw}
}

func newTaskCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage pull request tasks (DC and Cloud)",
		Long: `List, create, complete, or reopen tasks on a pull request.

Bitbucket Cloud exposes tasks as a first-class pull request resource. Bitbucket
Data Center 7.2+ models them as blocker comments; older servers use the legacy
/tasks API. The --task-api flag selects the Data Center model:

  auto             detect the server version and pick automatically (default)
  blocker-comments force the Data Center 7.2+ blocker-comments model
  legacy           force the pre-7.2 /tasks model (create requires --comment-id)

The flag is ignored on Cloud.`,
		Example: `  # List tasks on a pull request
  bkt pr task list 42

  # Create a task
  bkt pr task create 42 --text "Update the changelog"

  # Create a legacy DC task anchored to a comment
  bkt pr task create 42 --text "Fix this" --task-api legacy --comment-id 1001

  # Complete / reopen a task
  bkt pr task complete 42 99
  bkt pr task reopen 42 99`,
	}

	cmd.PersistentFlags().StringVar(&opts.TaskAPI, "task-api", taskAPIAuto, "Data Center task model: auto, blocker-comments, or legacy")

	cmd.AddCommand(newTaskListCmd(f, opts))
	cmd.AddCommand(newTaskCreateCmd(f, opts))
	cmd.AddCommand(newTaskCompleteCmd(f, opts))
	cmd.AddCommand(newTaskReopenCmd(f, opts))

	return cmd
}

func registerTaskTargetFlags(cmd *cobra.Command, opts *taskOptions) {
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override (DC)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
}

func newTaskListCmd(f *cmdutil.Factory, opts *taskOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <id>",
		Short:   "List tasks for a pull request",
		Example: "  bkt pr task list 42",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			opts.ID = id
			return runTaskList(cmd, f, opts)
		},
	}
	registerTaskTargetFlags(cmd, opts)
	return cmd
}

func newTaskCreateCmd(f *cmdutil.Factory, opts *taskOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create <id>",
		Short:   "Create a task on a pull request",
		Example: `  bkt pr task create 42 --text "Add unit tests"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parsePRID(args[0])
			if err != nil {
				return err
			}
			opts.ID = id
			return runTaskCreate(cmd, f, opts)
		},
	}
	registerTaskTargetFlags(cmd, opts)
	cmd.Flags().StringVar(&opts.Text, "text", "", "Task text")
	cmd.Flags().IntVar(&opts.CommentID, "comment-id", 0, "Comment to anchor the task to (required for --task-api legacy)")
	_ = cmd.MarkFlagRequired("text")
	return cmd
}

func newTaskCompleteCmd(f *cmdutil.Factory, opts *taskOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "complete <id> <task-id>",
		Short:   "Complete (resolve) a pull request task",
		Example: "  bkt pr task complete 42 99",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskToggle(cmd, f, opts, args, true)
		},
	}
	registerTaskTargetFlags(cmd, opts)
	return cmd
}

func newTaskReopenCmd(f *cmdutil.Factory, opts *taskOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reopen <id> <task-id>",
		Short:   "Reopen a resolved pull request task",
		Example: "  bkt pr task reopen 42 99",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskToggle(cmd, f, opts, args, false)
		},
	}
	registerTaskTargetFlags(cmd, opts)
	return cmd
}

func parsePRID(arg string) (int, error) {
	id, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("invalid pull request id %q", arg)
	}
	return id, nil
}

func runTaskToggle(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions, args []string, resolve bool) error {
	prID, err := parsePRID(args[0])
	if err != nil {
		return err
	}
	taskID, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid task id %q", args[1])
	}
	opts.ID = prID
	opts.TaskID = taskID
	return runTaskSetState(cmd, f, opts, resolve)
}

// dcContext resolves the project/repo for a Data Center task command.
func dcContext(opts *taskOptions, ctxProject, ctxRepo string) (projectKey, repoSlug string, err error) {
	projectKey = cmdutil.FirstNonEmpty(opts.Project, ctxProject)
	repoSlug = cmdutil.FirstNonEmpty(opts.Repo, ctxRepo)
	if projectKey == "" || repoSlug == "" {
		return "", "", fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}
	return projectKey, repoSlug, nil
}

// cloudContext resolves the workspace/repo for a Cloud task command.
func cloudContext(opts *taskOptions, ctxWorkspace, ctxRepo string) (workspace, repoSlug string, err error) {
	workspace = cmdutil.FirstNonEmpty(opts.Workspace, ctxWorkspace)
	repoSlug = cmdutil.FirstNonEmpty(opts.Repo, ctxRepo)
	if workspace == "" || repoSlug == "" {
		return "", "", fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
	}
	return workspace, repoSlug, nil
}

func runTaskList(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	if err := validateTaskAPIMode(opts.TaskAPI); err != nil {
		return err
	}
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), taskRequestTimeout)
	defer cancel()

	var tasks []taskView
	payload := map[string]any{}

	switch host.Kind {
	case "dc":
		projectKey, repoSlug, err := dcContext(opts, ctxCfg.ProjectKey, ctxCfg.DefaultRepo)
		if err != nil {
			return err
		}
		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}
		mode, err := resolveDCTaskMode(opts.TaskAPI, func() (string, error) { return client.ServerVersion(ctx) }, false)
		if err != nil {
			return err
		}
		var dcTasks []bbdc.PullRequestTask
		if mode == taskAPILegacy {
			dcTasks, err = client.ListPullRequestTasks(ctx, projectKey, repoSlug, opts.ID)
		} else {
			dcTasks, err = client.ListBlockerComments(ctx, projectKey, repoSlug, opts.ID)
		}
		if err != nil {
			return err
		}
		tasks = dcTaskViews(dcTasks)
		payload["project"] = projectKey
		payload["repo"] = repoSlug
	case "cloud":
		workspace, repoSlug, err := cloudContext(opts, ctxCfg.Workspace, ctxCfg.DefaultRepo)
		if err != nil {
			return err
		}
		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}
		cloudTasks, err := client.ListPullRequestTasks(ctx, workspace, repoSlug, opts.ID, 0)
		if err != nil {
			return err
		}
		for _, t := range cloudTasks {
			tasks = append(tasks, cloudTaskView(t))
		}
		payload["workspace"] = workspace
		payload["repo"] = repoSlug
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}

	payload["tasks"] = tasks

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(tasks) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No tasks on pull request #%d\n", opts.ID)
			return err
		}
		for _, task := range tasks {
			if _, err := fmt.Fprintf(ios.Out, "[%s] %d %s\n", strings.ToUpper(task.State), task.ID, task.Text); err != nil {
				return err
			}
		}
		return nil
	})
}

func runTaskCreate(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	if err := validateTaskAPIMode(opts.TaskAPI); err != nil {
		return err
	}
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.Text) == "" {
		return fmt.Errorf("task text is required")
	}
	commentIDSet := cmd.Flags().Changed("comment-id")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), taskRequestTimeout)
	defer cancel()

	var created int

	switch host.Kind {
	case "dc":
		projectKey, repoSlug, err := dcContext(opts, ctxCfg.ProjectKey, ctxCfg.DefaultRepo)
		if err != nil {
			return err
		}
		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}
		mode, err := resolveDCTaskMode(opts.TaskAPI, func() (string, error) { return client.ServerVersion(ctx) }, true)
		if err != nil {
			return err
		}
		var task *bbdc.PullRequestTask
		if mode == taskAPILegacy {
			if opts.CommentID <= 0 {
				return fmt.Errorf("legacy Data Center tasks must anchor to a comment; pass --comment-id <id>")
			}
			task, err = client.CreateLegacyTask(ctx, projectKey, repoSlug, opts.ID, opts.CommentID, opts.Text)
		} else {
			if commentIDSet {
				return errCommentIDOnlyLegacy
			}
			task, err = client.CreateBlockerComment(ctx, projectKey, repoSlug, opts.ID, opts.Text)
		}
		if err != nil {
			return err
		}
		created = task.ID
	case "cloud":
		if commentIDSet {
			return errCommentIDOnlyLegacy
		}
		workspace, repoSlug, err := cloudContext(opts, ctxCfg.Workspace, ctxCfg.DefaultRepo)
		if err != nil {
			return err
		}
		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}
		task, err := client.CreatePullRequestTask(ctx, workspace, repoSlug, opts.ID, opts.Text)
		if err != nil {
			return err
		}
		created = task.ID
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}

	_, err = fmt.Fprintf(ios.Out, "✓ Created task %d\n", created)
	return err
}

func runTaskSetState(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions, resolve bool) error {
	if err := validateTaskAPIMode(opts.TaskAPI); err != nil {
		return err
	}
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), taskRequestTimeout)
	defer cancel()

	switch host.Kind {
	case "dc":
		projectKey, repoSlug, err := dcContext(opts, ctxCfg.ProjectKey, ctxCfg.DefaultRepo)
		if err != nil {
			return err
		}
		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}
		mode, err := resolveDCTaskMode(opts.TaskAPI, func() (string, error) { return client.ServerVersion(ctx) }, true)
		if err != nil {
			return err
		}
		if mode == taskAPILegacy {
			_, err = client.SetLegacyTaskState(ctx, opts.TaskID, resolve)
		} else {
			_, err = client.SetBlockerCommentState(ctx, projectKey, repoSlug, opts.ID, opts.TaskID, resolve)
		}
		if err != nil {
			return err
		}
	case "cloud":
		workspace, repoSlug, err := cloudContext(opts, ctxCfg.Workspace, ctxCfg.DefaultRepo)
		if err != nil {
			return err
		}
		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}
		if _, err := client.SetPullRequestTaskState(ctx, workspace, repoSlug, opts.ID, opts.TaskID, resolve); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}

	verb := "Reopened"
	if resolve {
		verb = "Completed"
	}
	_, err = fmt.Fprintf(ios.Out, "✓ %s task %d\n", verb, opts.TaskID)
	return err
}
