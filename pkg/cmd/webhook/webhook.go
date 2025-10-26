package webhook

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/format"
)

// NewCommand returns the webhook command.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage Bitbucket webhooks",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newTestCmd(f))

	return cmd
}

type listOptions struct {
	Project string
	Repo    string
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured webhooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("webhook list currently supports Data Center contexts only")
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

	hooks, err := client.ListWebhooks(ctx, projectKey, repoSlug)
	if err != nil {
		return err
	}

	formatOpt, err := cmdutil.OutputFormat(cmd)
	if err != nil {
		return err
	}

	if formatOpt != "" {
		payload := map[string]any{
			"project":  projectKey,
			"repo":     repoSlug,
			"webhooks": hooks,
		}
		return format.Write(ios.Out, formatOpt, payload, nil)
	}

	if len(hooks) == 0 {
		fmt.Fprintln(ios.Out, "No webhooks configured.")
		return nil
	}

	for _, hook := range hooks {
		status := "disabled"
		if hook.Active {
			status = "active"
		}
		fmt.Fprintf(ios.Out, "%d\t%s\t%s (%s)\n", hook.ID, status, hook.Name, hook.URL)
	}
	return nil
}

type createOptions struct {
	Project string
	Repo    string
	Name    string
	URL     string
	Events  []string
	Active  bool
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{Active: true}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Webhook name (required)")
	cmd.Flags().StringVar(&opts.URL, "url", "", "Webhook callback URL (required)")
	cmd.Flags().StringSliceVar(&opts.Events, "event", nil, "Events to subscribe to (repeatable)")
	cmd.Flags().BoolVar(&opts.Active, "active", opts.Active, "Whether the webhook starts active")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("event")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, opts *createOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("webhook create currently supports Data Center contexts only")
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

	hook, err := client.CreateWebhook(ctx, projectKey, repoSlug, bbdc.CreateWebhookInput{
		Name:   opts.Name,
		URL:    opts.URL,
		Events: opts.Events,
		Active: opts.Active,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Created webhook #%d (%s)\n", hook.ID, hook.Name)
	return nil
}

type deleteOptions struct {
	Project string
	Repo    string
	ID      int
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}
	cmd := &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a webhook",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid webhook id %q", args[0])
			}
			opts.ID = id
			return runDelete(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runDelete(cmd *cobra.Command, f *cmdutil.Factory, opts *deleteOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("webhook delete currently supports Data Center contexts only")
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

	if err := client.DeleteWebhook(ctx, projectKey, repoSlug, opts.ID); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Deleted webhook #%d\n", opts.ID)
	return nil
}

func newTestCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <id>",
		Short: "Trigger a webhook test delivery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("webhook test is not yet implemented")
		},
	}
	return cmd
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
