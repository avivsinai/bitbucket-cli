package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type defaultReviewersOptions struct {
	Workspace string
	Project   string
	Repo      string
	Source    string
	Target    string
}

func newDefaultReviewersCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default-reviewers",
		Short: "List effective default reviewers for a repository",
		Long: `Manage default reviewers configured for a repository.

On Cloud, returns the effective default reviewers (merged from workspace and
repository-level settings). On Data Center, returns the effective default
reviewers for a pull request from --source to --target.`,
	}
	cmd.AddCommand(newDefaultReviewersListCmd(f))
	return cmd
}

func newDefaultReviewersListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &defaultReviewersOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List default reviewers",
		Long: `List the default reviewers configured for a repository.

On Cloud, this returns the effective default reviewers that would be automatically
added to new pull requests, including reviewers inherited from workspace-level
settings. On Data Center, this returns the effective default reviewers that would
be added to a pull request from --source to --target. Data Center requires
source and target branch or tag names.

The workspace/project and repository are resolved from the active context unless
overridden with flags.`,
		Example: `  # List default reviewers for the active context repository
  bkt repo default-reviewers list

  # List default reviewers for a specific Cloud repository
  bkt repo default-reviewers list --workspace my-team --repo api-service

  # List default reviewers for a Data Center repository
  bkt repo default-reviewers list --project PLATFORM --repo backend --source feature/auth --target main`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefaultReviewersList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Source, "source", "", "Data Center source branch or tag")
	cmd.Flags().StringVar(&opts.Target, "target", "", "Data Center target branch or tag")
	return cmd
}

func runDefaultReviewersList(cmd *cobra.Command, f *cmdutil.Factory, opts *defaultReviewersOptions) error {
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
	case "cloud":
		if cmd.Flags().Changed("source") || cmd.Flags().Changed("target") {
			return fmt.Errorf("--source and --target are only supported for Data Center default reviewer lookup")
		}
		workspace := cmdutil.FirstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		users, err := client.GetEffectiveDefaultReviewers(ctx, workspace, repoSlug)
		if err != nil {
			return err
		}

		type reviewerSummary struct {
			DisplayName string `json:"display_name"`
			Username    string `json:"username"`
			UUID        string `json:"uuid"`
		}

		var summaries []reviewerSummary
		for _, u := range users {
			summaries = append(summaries, reviewerSummary{
				DisplayName: u.Display,
				Username:    u.Username,
				UUID:        u.UUID,
			})
		}

		payload := struct {
			Workspace string            `json:"workspace"`
			Repo      string            `json:"repo"`
			Reviewers []reviewerSummary `json:"reviewers"`
		}{
			Workspace: workspace,
			Repo:      repoSlug,
			Reviewers: summaries,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(summaries) == 0 {
				_, err := fmt.Fprintf(ios.Out, "No default reviewers configured for %s/%s.\n", workspace, repoSlug)
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "%-30s %-20s %s\n", "DISPLAY NAME", "USERNAME", "UUID"); err != nil {
				return err
			}
			for _, r := range summaries {
				if _, err := fmt.Fprintf(ios.Out, "%-30s %-20s %s\n", r.DisplayName, r.Username, r.UUID); err != nil {
					return err
				}
			}
			return nil
		})

	case "dc":
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}
		sourceRef := strings.TrimSpace(opts.Source)
		targetRef := strings.TrimSpace(opts.Target)
		if sourceRef == "" || targetRef == "" {
			return fmt.Errorf("data center default reviewers require --source and --target refs")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		users, err := client.GetDefaultReviewers(ctx, projectKey, repoSlug, sourceRef, targetRef)
		if err != nil {
			return err
		}

		type reviewerSummary struct {
			DisplayName string `json:"display_name"`
			Username    string `json:"username"`
			ID          int    `json:"id"`
		}

		var summaries []reviewerSummary
		for _, u := range users {
			summaries = append(summaries, reviewerSummary{
				DisplayName: u.FullName,
				Username:    u.Name,
				ID:          u.ID,
			})
		}

		payload := struct {
			Project   string            `json:"project"`
			Repo      string            `json:"repo"`
			Source    string            `json:"source"`
			Target    string            `json:"target"`
			Reviewers []reviewerSummary `json:"reviewers"`
		}{
			Project:   projectKey,
			Repo:      repoSlug,
			Source:    sourceRef,
			Target:    targetRef,
			Reviewers: summaries,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(summaries) == 0 {
				_, err := fmt.Fprintf(ios.Out, "No default reviewers configured for %s/%s.\n", projectKey, repoSlug)
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "%-30s %-20s %s\n", "DISPLAY NAME", "USERNAME", "ID"); err != nil {
				return err
			}
			for _, r := range summaries {
				if _, err := fmt.Fprintf(ios.Out, "%-30s %-20s %d\n", r.DisplayName, r.Username, r.ID); err != nil {
					return err
				}
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}
