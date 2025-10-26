package pr

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

func newReviewerGroupCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reviewer-group",
		Short: "Manage default reviewer groups",
	}

	cmd.AddCommand(newReviewerGroupListCmd(f))
	cmd.AddCommand(newReviewerGroupAddCmd(f))
	cmd.AddCommand(newReviewerGroupRemoveCmd(f))

	return cmd
}

type reviewerGroupOptions struct {
	Project string
	Repo    string
	Name    string
}

func newReviewerGroupListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reviewerGroupOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List default reviewer groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReviewerGroupList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newReviewerGroupAddCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reviewerGroupOptions{}
	cmd := &cobra.Command{
		Use:   "add <group>",
		Short: "Add a default reviewer group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return runReviewerGroupAdd(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newReviewerGroupRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reviewerGroupOptions{}
	cmd := &cobra.Command{
		Use:   "remove <group>",
		Short: "Remove a default reviewer group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return runReviewerGroupRemove(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runReviewerGroupList(cmd *cobra.Command, f *cmdutil.Factory, opts *reviewerGroupOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reviewer-group list currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	groups, err := client.ListReviewerGroups(ctx, projectKey, repoSlug)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":         projectKey,
		"repo":            repoSlug,
		"reviewer_groups": groups,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(groups) == 0 {
			fmt.Fprintln(ios.Out, "No reviewer groups configured.")
			return nil
		}
		for _, g := range groups {
			fmt.Fprintf(ios.Out, "%s\n", g.Name)
		}
		return nil
	})
}

func runReviewerGroupAdd(cmd *cobra.Command, f *cmdutil.Factory, opts *reviewerGroupOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reviewer-group add currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.AddReviewerGroup(ctx, projectKey, repoSlug, opts.Name); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Added reviewer group %s\n", opts.Name)
	return nil
}

func runReviewerGroupRemove(cmd *cobra.Command, f *cmdutil.Factory, opts *reviewerGroupOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reviewer-group remove currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.RemoveReviewerGroup(ctx, projectKey, repoSlug, opts.Name); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Removed reviewer group %s\n", opts.Name)
	return nil
}
