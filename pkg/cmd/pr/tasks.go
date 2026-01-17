package pr

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/example/bitbucket-cli/pkg/cmdutil"
)

type taskOptions struct {
	Project string
	Repo    string
	ID      int
	TaskID  int
	Text    string
}

func newTaskCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage pull request tasks",
	}

	cmd.AddCommand(newTaskListCmd(f))
	cmd.AddCommand(newTaskCreateCmd(f))
	cmd.AddCommand(newTaskCompleteCmd(f))
	cmd.AddCommand(newTaskReopenCmd(f))

	return cmd
}

func newTaskListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "list <id>",
		Short: "List tasks for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runTaskList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newTaskCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "create <id>",
		Short: "Create a task on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runTaskCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Text, "text", "", "Task text")
	_ = cmd.MarkFlagRequired("text")

	return cmd
}

func newTaskCompleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "complete <id> <task-id>",
		Short: "Complete a pull request task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			taskID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid task id %q", args[1])
			}
			opts.ID = prID
			opts.TaskID = taskID
			return runTaskComplete(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newTaskReopenCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "reopen <id> <task-id>",
		Short: "Reopen a resolved task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			taskID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid task id %q", args[1])
			}
			opts.ID = prID
			opts.TaskID = taskID
			return runTaskReopen(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runTaskList(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("task list currently supports Data Center contexts only")
	}

	projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	tasks, err := client.ListPullRequestTasks(ctx, projectKey, repoSlug, opts.ID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project": projectKey,
		"repo":    repoSlug,
		"tasks":   tasks,
	}

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
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("task create currently supports Data Center contexts only")
	}

	projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	task, err := client.CreatePullRequestTask(ctx, projectKey, repoSlug, opts.ID, opts.Text)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Created task %d\n", task.ID); err != nil {
		return err
	}
	return nil
}

func runTaskComplete(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	return toggleTaskState(cmd, f, opts, true)
}

func runTaskReopen(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	return toggleTaskState(cmd, f, opts, false)
}

func toggleTaskState(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions, resolve bool) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("task management currently supports Data Center contexts only")
	}

	projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if resolve {
		if err := client.CompletePullRequestTask(ctx, projectKey, repoSlug, opts.ID, opts.TaskID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(ios.Out, "✓ Completed task %d\n", opts.TaskID); err != nil {
			return err
		}
		return nil
	}

	if err := client.ReopenPullRequestTask(ctx, projectKey, repoSlug, opts.ID, opts.TaskID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(ios.Out, "✓ Reopened task %d\n", opts.TaskID); err != nil {
		return err
	}
	return nil
}
