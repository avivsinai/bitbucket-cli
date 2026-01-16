package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type autoMergeOptions struct {
	Project     string
	Repo        string
	ID          int
	Strategy    string
	Message     string
	CloseSource bool
}

func newAutoMergeCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-merge",
		Short: "Manage pull request auto-merge",
	}

	cmd.AddCommand(newAutoMergeEnableCmd(f))
	cmd.AddCommand(newAutoMergeDisableCmd(f))
	cmd.AddCommand(newAutoMergeStatusCmd(f))

	return cmd
}

func newAutoMergeEnableCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &autoMergeOptions{CloseSource: true}
	cmd := &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable auto-merge for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runAutoMergeEnable(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Strategy, "strategy", "", "Merge strategy ID (leave empty for default)")
	cmd.Flags().StringVar(&opts.Message, "message", "", "Custom merge commit message")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", opts.CloseSource, "Close source branch when auto-merge completes")

	return cmd
}

func newAutoMergeDisableCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &autoMergeOptions{}
	cmd := &cobra.Command{
		Use:   "disable <id>",
		Short: "Disable auto-merge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runAutoMergeDisable(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newAutoMergeStatusCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &autoMergeOptions{}
	cmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Show auto-merge configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runAutoMergeStatus(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runAutoMergeEnable(cmd *cobra.Command, f *cmdutil.Factory, opts *autoMergeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("auto-merge enable currently supports Data Center contexts only")
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

	settings := bbdc.AutoMergeSettings{
		StrategyID:    opts.Strategy,
		CommitMessage: opts.Message,
		CloseSource:   opts.CloseSource,
	}

	if err := client.EnableAutoMerge(ctx, projectKey, repoSlug, opts.ID, settings); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Auto-merge enabled for pull request #%d\n", opts.ID); err != nil {
		return err
	}
	return nil
}

func runAutoMergeDisable(cmd *cobra.Command, f *cmdutil.Factory, opts *autoMergeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("auto-merge disable currently supports Data Center contexts only")
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

	if err := client.DisableAutoMerge(ctx, projectKey, repoSlug, opts.ID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Auto-merge disabled for pull request #%d\n", opts.ID); err != nil {
		return err
	}
	return nil
}

func runAutoMergeStatus(cmd *cobra.Command, f *cmdutil.Factory, opts *autoMergeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("auto-merge status currently supports Data Center contexts only")
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

	settings, err := client.GetAutoMerge(ctx, projectKey, repoSlug, opts.ID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":      projectKey,
		"repo":         repoSlug,
		"pull_request": opts.ID,
		"auto_merge":   settings,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if settings == nil || !settings.Enabled {
			_, err := fmt.Fprintf(ios.Out, "Auto-merge disabled for pull request #%d\n", opts.ID)
			return err
		}
		if _, err := fmt.Fprintf(ios.Out, "Auto-merge enabled using strategy %s\n", settings.StrategyID); err != nil {
			return err
		}
		if settings.CommitMessage != "" {
			if _, err := fmt.Fprintf(ios.Out, "Message: %s\n", settings.CommitMessage); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(ios.Out, "Close source branch: %t\n", settings.CloseSource); err != nil {
			return err
		}
		return nil
	})
}
