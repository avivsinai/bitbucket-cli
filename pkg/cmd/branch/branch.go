package branch

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdBranch exposes branch operations.
func NewCmdBranch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Inspect and manage branches",
		Long: `Inspect and manage branches in a Bitbucket repository.

Supports listing, creating, deleting, rebasing, and setting the default branch.
Branch protection rules are available through the "protect" subcommand.

Listing works on both Bitbucket Data Center and Cloud. Create, delete,
set-default, and protect subcommands currently support Data Center only.`,
		Example: `  # List branches in the current context
  bkt branch list

  # Create a branch from main (Data Center)
  bkt branch create feature/login --from main

  # Delete a stale branch (Data Center)
  bkt branch delete feature/old-experiment`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newSetDefaultCmd(f))
	cmd.AddCommand(newProtectCmd(f))
	cmd.AddCommand(newRebaseCmd(f))

	return cmd
}

type listOptions struct {
	Project   string
	Workspace string
	Repo      string
	Filter    string
	Limit     int
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{Limit: 50}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List branches",
		Long: `List branches in a Bitbucket repository.

Works on both Bitbucket Data Center and Cloud. On Data Center, uses --project
and --repo to identify the repository. On Cloud, uses --workspace and --repo.
The default branch is marked with an asterisk (*).`,
		Example: `  # List branches in the current context
  bkt branch list

  # Filter branches by name
  bkt branch list --filter feature/

  # List up to 10 branches
  bkt branch list --limit 10

  # List branches in a specific Cloud workspace and repo
  bkt branch list --workspace myteam --repo backend`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Filter, "filter", "", "Filter branches by text")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum branches to list (0 for all)")

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

		branches, err := client.ListBranches(ctx, projectKey, repoSlug, bbdc.BranchListOptions{Filter: opts.Filter, Limit: opts.Limit})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":  projectKey,
			"repo":     repoSlug,
			"branches": branches,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(branches) == 0 {
				_, err := fmt.Fprintln(ios.Out, "No branches found.")
				return err
			}

			for _, branch := range branches {
				marker := " "
				if branch.IsDefault {
					marker = "*"
				}
				if _, err := fmt.Fprintf(ios.Out, "%s %s\t%s\n", marker, branch.DisplayID, branch.LatestCommit); err != nil {
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

		branches, err := client.ListBranches(ctx, workspace, repoSlug, bbcloud.BranchListOptions{Filter: opts.Filter, Limit: opts.Limit})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace": workspace,
			"repo":      repoSlug,
			"branches":  branches,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(branches) == 0 {
				_, err := fmt.Fprintln(ios.Out, "No branches found.")
				return err
			}

			for _, branch := range branches {
				marker := " "
				if branch.IsDefault {
					marker = "*"
				}
				hash := branch.Target.Hash
				if len(hash) > 12 {
					hash = hash[:12]
				}
				if _, err := fmt.Fprintf(ios.Out, "%s %s\t%s\n", marker, branch.Name, hash); err != nil {
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
	Project string
	Repo    string
	Source  string
	Message string
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create <branch>",
		Short: "Create a new branch",
		Long: `Create a new branch in a Bitbucket Data Center repository.

The --from flag is required and specifies the branch or commit to use as the
starting point. An optional --message flag lets you attach a creation message.

This command currently supports Data Center contexts only.`,
		Example: `  # Create a feature branch from main
  bkt branch create feature/user-auth --from main

  # Create a branch from a specific commit
  bkt branch create hotfix/login --from abc1234

  # Create a branch with a message
  bkt branch create release/v2.0 --from main --message "Release candidate"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Source, "from", "", "Branch or commit to start from (required)")
	cmd.Flags().StringVar(&opts.Message, "message", "", "Optional branch creation message")
	_ = cmd.MarkFlagRequired("from")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, name string, opts *createOptions) error {
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
		return fmt.Errorf("branch create currently supports Data Center contexts only")
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

	branch, err := client.CreateBranch(ctx, projectKey, repoSlug, bbdc.CreateBranchInput{
		Name:       name,
		StartPoint: opts.Source,
		Message:    opts.Message,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Created branch %s (%s)\n", branch.DisplayID, branch.LatestCommit); err != nil {
		return err
	}
	return nil
}

type deleteOptions struct {
	Project string
	Repo    string
	DryRun  bool
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}
	cmd := &cobra.Command{
		Use:     "delete <branch>",
		Aliases: []string{"rm"},
		Short:   "Delete a branch",
		Long: `Delete a branch from a Bitbucket Data Center repository.

Use --dry-run to validate that the branch can be deleted without actually
removing it. This is useful for confirming permissions and branch existence.

This command currently supports Data Center contexts only.`,
		Example: `  # Delete a branch
  bkt branch delete feature/old-experiment

  # Dry-run to verify before deleting
  bkt branch delete feature/old-experiment --dry-run

  # Delete a branch in a specific project and repo
  bkt branch delete bugfix/stale --project MYPROJ --repo backend`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd, f, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Perform a dry run without deleting")

	return cmd
}

func runDelete(cmd *cobra.Command, f *cmdutil.Factory, name string, opts *deleteOptions) error {
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
		return fmt.Errorf("branch delete currently supports Data Center contexts only")
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

	if err := client.DeleteBranch(ctx, projectKey, repoSlug, name, opts.DryRun); err != nil {
		return err
	}

	action := "Deleted"
	if opts.DryRun {
		action = "Validated"
	}
	if _, err := fmt.Fprintf(ios.Out, "✓ %s branch %s\n", action, name); err != nil {
		return err
	}
	return nil
}

func newSetDefaultCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-default <branch>",
		Short: "Set the default branch",
		Long: `Set the default branch for a Bitbucket Data Center repository.

The default branch is the one shown by default when browsing the repository
and is used as the base for new pull requests.

This command currently supports Data Center contexts only.`,
		Example: `  # Set main as the default branch
  bkt branch set-default main

  # Switch the default branch to develop
  bkt branch set-default develop`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetDefault(cmd, f, args[0])
		},
	}
	return cmd
}

func runSetDefault(cmd *cobra.Command, f *cmdutil.Factory, name string) error {
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
		return fmt.Errorf("branch set-default currently supports Data Center contexts only")
	}

	projectKey := ctxCfg.ProjectKey
	repoSlug := ctxCfg.DefaultRepo
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.SetDefaultBranch(ctx, projectKey, repoSlug, name); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Set default branch to %s\n", name); err != nil {
		return err
	}
	return nil
}
