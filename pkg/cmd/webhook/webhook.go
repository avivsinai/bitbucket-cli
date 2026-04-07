package webhook

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCommand returns the webhook command.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage Bitbucket webhooks",
		Long: `Create, list, delete, and test webhooks on Bitbucket repositories.

Webhooks notify external services when events occur in a repository (e.g. push,
pull-request creation). Both Data Center and Cloud are supported, though the
identifier format differs: Data Center uses numeric IDs while Cloud uses UUIDs.

The test subcommand is available for Data Center only.`,
		Example: `  # List all webhooks on the current repository
  bkt webhook list

  # Create a webhook for push events
  bkt webhook create --name ci-trigger --url https://ci.example.com/hook --event repo:refs_changed

  # Delete a webhook by ID (Data Center) or UUID (Cloud)
  bkt webhook delete 42`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newTestCmd(f))

	return cmd
}

type listOptions struct {
	Project   string
	Workspace string
	Repo      string
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured webhooks",
		Long: `List all webhooks configured on a Bitbucket repository.

For Data Center, the output includes the numeric webhook ID, status, name, and
callback URL. For Cloud, the output includes the webhook UUID, status, and URL.

Use --project/--repo for Data Center or --workspace/--repo for Cloud to override
the values from the current context.`,
		Example: `  # List webhooks using the current context
  bkt webhook list

  # List webhooks for a specific Data Center repository
  bkt webhook list --project MYPROJ --repo my-repo

  # List webhooks for a specific Cloud repository
  bkt webhook list --workspace myteam --repo my-repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override (Data Center)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
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

	switch host.Kind {
	case "dc":
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

		hooks, err := client.ListWebhooks(ctx, projectKey, repoSlug)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":  projectKey,
			"repo":     repoSlug,
			"webhooks": hooks,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(hooks) == 0 {
				_, err := fmt.Fprintln(ios.Out, "No webhooks configured.")
				return err
			}

			for _, hook := range hooks {
				status := "disabled"
				if hook.Active {
					status = "active"
				}
				if _, err := fmt.Fprintf(ios.Out, "%d\t%s\t%s (%s)\n", hook.ID, status, hook.Name, hook.URL); err != nil {
					return err
				}
			}
			return nil
		})
	case "cloud":
		workspace := cmdutil.FirstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		hooks, err := client.ListWebhooks(ctx, workspace, repoSlug)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace": workspace,
			"repo":      repoSlug,
			"webhooks":  hooks,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(hooks) == 0 {
				_, err := fmt.Fprintln(ios.Out, "No webhooks configured.")
				return err
			}
			for _, hook := range hooks {
				status := "disabled"
				if hook.Active {
					status = "active"
				}
				if _, err := fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", hook.UUID, status, hook.URL); err != nil {
					return err
				}
			}
			return nil
		})
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type createOptions struct {
	Project   string
	Workspace string
	Repo      string
	Name      string
	URL       string
	Events    []string
	Active    bool
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{Active: true}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new webhook",
		Long: `Create a new webhook on a Bitbucket repository.

You must specify a name, a callback URL, and at least one event to subscribe to.
The webhook is created active by default; pass --active=false to create it in a
disabled state.

Event names differ between Data Center and Cloud. For Data Center, common events
include repo:refs_changed and pr:opened. For Cloud, events follow the pattern
repo:push, pullrequest:created, and similar.`,
		Example: `  # Create a webhook for push events (Data Center)
  bkt webhook create --name ci-trigger --url https://ci.example.com/hook --event repo:refs_changed

  # Create a webhook for multiple events (Cloud)
  bkt webhook create --name slack-notify --url https://hooks.slack.com/abc --event repo:push --event pullrequest:created

  # Create a webhook in a disabled state
  bkt webhook create --name staging-hook --url https://staging.example.com/hook --event repo:push --active=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override (Data Center)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
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

	switch host.Kind {
	case "dc":
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

		hook, err := client.CreateWebhook(ctx, projectKey, repoSlug, bbdc.CreateWebhookInput{
			Name:   opts.Name,
			URL:    opts.URL,
			Events: opts.Events,
			Active: opts.Active,
		})
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Created webhook #%d (%s)\n", hook.ID, hook.Name); err != nil {
			return err
		}
		return nil
	case "cloud":
		workspace := cmdutil.FirstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		hook, err := client.CreateWebhook(ctx, workspace, repoSlug, bbcloud.WebhookInput{
			Description: opts.Name,
			URL:         opts.URL,
			Events:      opts.Events,
			Active:      opts.Active,
		})
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Created webhook %s\n", hook.UUID); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type deleteOptions struct {
	Project    string
	Workspace  string
	Repo       string
	Identifier string
}

type testOptions struct {
	Project string
	Repo    string
	ID      string
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}
	cmd := &cobra.Command{
		Use:     "delete <id|uuid>",
		Aliases: []string{"rm"},
		Short:   "Delete a webhook",
		Long: `Delete a webhook from a Bitbucket repository.

For Data Center, pass the numeric webhook ID (shown by "bkt webhook list").
For Cloud, pass the webhook UUID. The webhook is removed immediately and cannot
be recovered.`,
		Example: `  # Delete a webhook by numeric ID (Data Center)
  bkt webhook delete 42

  # Delete a webhook by UUID (Cloud)
  bkt webhook delete {a1b2c3d4-e5f6-7890-abcd-ef1234567890}

  # Delete a webhook from a specific repository
  bkt webhook delete 7 --project MYPROJ --repo my-repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Identifier = args[0]
			return runDelete(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override (Data Center)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
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

	switch host.Kind {
	case "dc":
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		id, err := strconv.Atoi(opts.Identifier)
		if err != nil {
			return fmt.Errorf("invalid webhook id %q", opts.Identifier)
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		if err := client.DeleteWebhook(ctx, projectKey, repoSlug, id); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Deleted webhook #%d\n", id); err != nil {
			return err
		}
		return nil
	case "cloud":
		workspace := cmdutil.FirstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		if err := client.DeleteWebhook(ctx, workspace, repoSlug, opts.Identifier); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Deleted webhook %s\n", opts.Identifier); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func newTestCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &testOptions{}
	cmd := &cobra.Command{
		Use:   "test <id>",
		Short: "Trigger a webhook test delivery",
		Long: `Send a test payload to a webhook's callback URL to verify connectivity.

This command is supported for Data Center only. It triggers a diagnostic POST
request from the Bitbucket server to the webhook's configured URL and reports
whether the delivery succeeded.`,
		Example: `  # Test a webhook by ID
  bkt webhook test 42

  # Test a webhook in a specific repository
  bkt webhook test 7 --project MYPROJ --repo my-repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ID = args[0]
			return runTest(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runTest(cmd *cobra.Command, f *cmdutil.Factory, opts *testOptions) error {
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
		return fmt.Errorf("webhook test is supported for Data Center contexts only")
	}

	projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	id, err := strconv.Atoi(opts.ID)
	if err != nil {
		return fmt.Errorf("invalid webhook id %q", opts.ID)
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.TestWebhook(ctx, projectKey, repoSlug, id); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Triggered test delivery for webhook #%d\n", id); err != nil {
		return err
	}
	return nil
}
