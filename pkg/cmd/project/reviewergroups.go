package project

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

func newReviewerGroupsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reviewer-groups",
		Aliases: []string{"reviewer-group"},
		Short:   "Work with project reviewer groups (DC only)",
		Long: `List the reviewer groups defined in a project's settings.

Reviewer groups are named sets of users that can be added as default reviewers
on repositories within the project. Data Center only.`,
		Example: `  # List reviewer groups for the active context project
  bkt project reviewer-groups list

  # List reviewer groups for a specific project
  bkt project reviewer-groups list --project PLATFORM`,
	}

	cmd.AddCommand(newReviewerGroupsListCmd(f))

	return cmd
}

type reviewerGroupsListOptions struct {
	Project string
	Limit   int
}

func newReviewerGroupsListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reviewerGroupsListOptions{
		Limit: 30,
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List project reviewer groups (DC only)",
		Long: `List the reviewer groups defined in a Bitbucket Data Center project's
settings, including each group's members. The project is resolved from the
active context unless overridden with --project. Use --limit to control the
number of results returned.

This command is only available for Data Center hosts. Attempting to run it
against a Cloud context will return an error.`,
		Example: `  # List reviewer groups for the active context project (default limit of 30)
  bkt project reviewer-groups list

  # List all reviewer groups without a limit
  bkt project reviewer-groups ls --limit 0

  # List reviewer groups for a specific project
  bkt project reviewer-groups list --project PLATFORM

  # List reviewer groups in JSON format
  bkt project reviewer-groups list --project PLATFORM --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReviewerGroupsList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum reviewer groups to display (0 for all)")

	return cmd
}

func runReviewerGroupsList(cmd *cobra.Command, f *cmdutil.Factory, opts *reviewerGroupsListOptions) error {
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
		return fmt.Errorf("project reviewer groups are only supported for Bitbucket Data Center hosts")
	}

	projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	if projectKey == "" {
		return fmt.Errorf("context must supply a project; use --project if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	groups, err := client.ListProjectReviewerGroups(ctx, projectKey, opts.Limit)
	if err != nil {
		return err
	}

	type memberSummary struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
		ID          int    `json:"id"`
	}

	type groupSummary struct {
		ID          int             `json:"id"`
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Members     []memberSummary `json:"members"`
	}

	var summaries []groupSummary
	for _, g := range groups {
		var members []memberSummary
		for _, u := range g.Users {
			members = append(members, memberSummary{
				DisplayName: u.FullName,
				Username:    u.Name,
				ID:          u.ID,
			})
		}
		summaries = append(summaries, groupSummary{
			ID:          g.ID,
			Name:        g.Name,
			Description: strings.TrimSpace(g.Description),
			Members:     members,
		})
	}

	payload := struct {
		Project        string         `json:"project"`
		ReviewerGroups []groupSummary `json:"reviewer_groups"`
	}{
		Project:        projectKey,
		ReviewerGroups: summaries,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(summaries) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No reviewer groups defined for project %s.\n", projectKey)
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "Reviewer groups for project %s:\n", projectKey); err != nil {
			return err
		}
		for _, g := range summaries {
			if _, err := fmt.Fprintf(ios.Out, "%s\t(id: %d, members: %d)\n", g.Name, g.ID, len(g.Members)); err != nil {
				return err
			}
			if g.Description != "" {
				if _, err := fmt.Fprintf(ios.Out, "    desc: %s\n", g.Description); err != nil {
					return err
				}
			}
			for _, m := range g.Members {
				if _, err := fmt.Fprintf(ios.Out, "    member: %s (%s)\n", m.DisplayName, m.Username); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
