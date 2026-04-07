package status

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdStatus exposes commit and PR status commands.
func NewCmdStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect commit and pull request statuses",
		Long: `Inspect build and CI statuses attached to commits and pull requests. Subcommands
cover Data Center commit statuses, pull request head-commit statuses, Cloud
pipeline runs, and API rate-limit telemetry.`,
		Example: `  # Show build statuses for a commit (Data Center)
  bkt status commit abc1234

  # Show build statuses for a pull request (Data Center)
  bkt status pr 42

  # Show a Cloud pipeline run
  bkt status pipeline {pipeline-uuid}

  # Check API rate limits for the active context
  bkt status rate-limit`,
	}

	cmd.AddCommand(newCommitCmd(f))
	cmd.AddCommand(newPullRequestCmd(f))
	cmd.AddCommand(newCloudPipelineCmd(f))
	cmd.AddCommand(newRateLimitCmd(f))

	return cmd
}

func newCommitCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit <sha>",
		Short: "Show the build statuses for a commit (DC only)",
		Long: `Display the CI/build statuses reported against a specific commit SHA. Each
status includes the state (SUCCESSFUL, FAILED, INPROGRESS), the build key,
name, optional description, and a link to the build.

Currently supports Data Center contexts only. The commit does not need to
belong to any particular branch or pull request.`,
		Example: `  # Show statuses for a commit
  bkt status commit abc1234def5678

  # Show statuses using a full 40-character SHA
  bkt status commit 6f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a

  # Output as JSON
  bkt status commit abc1234 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommit(cmd, f, args[0])
		},
	}
	return cmd
}

func runCommit(cmd *cobra.Command, f *cmdutil.Factory, sha string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "dc" {
		return fmt.Errorf("status commit currently supports Data Center contexts only")
	}

	client, err := bbdc.New(bbdc.Options{
		BaseURL:    host.BaseURL,
		Username:   host.Username,
		Token:      host.Token,
		AuthMethod: host.AuthMethod,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	statuses, err := client.CommitStatuses(ctx, sha)
	if err != nil {
		return err
	}

	return renderStatuses(cmd, f, ios.Out, sha, statuses, nil)
}

type prOptions struct {
	Project string
	Repo    string
}

func newPullRequestCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &prOptions{}
	cmd := &cobra.Command{
		Use:   "pr <id>",
		Short: "Show the build statuses for a pull request head commit (DC only)",
		Long: `Look up the head (latest) commit of a pull request and display all CI/build
statuses attached to it. The output includes the pull request title and the
resolved commit SHA alongside the status details.

The project and repository are resolved from the active context or can be
overridden with --project and --repo. Currently supports Data Center
contexts only.`,
		Example: `  # Show statuses for pull request #42
  bkt status pr 42

  # Specify project and repo explicitly
  bkt status pr 42 --project MYPROJ --repo my-service

  # Output as JSON
  bkt status pr 42 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runPullRequest(cmd, f, id, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runPullRequest(cmd *cobra.Command, f *cmdutil.Factory, prID int, opts *prOptions) error {
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
		return fmt.Errorf("status pr currently supports Data Center contexts only")
	}

	projectKey := strings.TrimSpace(opts.Project)
	if projectKey == "" {
		projectKey = ctxCfg.ProjectKey
	}
	if projectKey == "" {
		return fmt.Errorf("project key required; set with --project or configure the context default")
	}
	projectKey = strings.ToUpper(projectKey)

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; pass --repo or set the context default")
	}

	client, err := bbdc.New(bbdc.Options{
		BaseURL:    host.BaseURL,
		Username:   host.Username,
		Token:      host.Token,
		AuthMethod: host.AuthMethod,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, prID)
	if err != nil {
		return err
	}

	statuses, err := client.CommitStatuses(ctx, pr.FromRef.LatestCommit)
	if err != nil {
		return err
	}

	info := map[string]any{
		"pull_request": map[string]any{
			"id":    pr.ID,
			"title": pr.Title,
		},
		"context": map[string]any{
			"project": projectKey,
			"repo":    repoSlug,
		},
		"commit": pr.FromRef.LatestCommit,
	}

	return renderStatuses(cmd, f, ios.Out, pr.FromRef.LatestCommit, statuses, info)
}

func renderStatuses(cmd *cobra.Command, f *cmdutil.Factory, out io.Writer, commit string, statuses []bbdc.CommitStatus, metadata map[string]any) error {
	type statusSummary struct {
		State       string `json:"state"`
		Key         string `json:"key"`
		Name        string `json:"name"`
		URL         string `json:"url,omitempty"`
		Description string `json:"description,omitempty"`
	}

	var summaries []statusSummary
	for _, s := range statuses {
		summaries = append(summaries, statusSummary{
			State:       s.State,
			Key:         s.Key,
			Name:        s.Name,
			URL:         s.URL,
			Description: s.Description,
		})
	}

	payload := map[string]any{
		"commit":   commit,
		"statuses": summaries,
	}
	for k, v := range metadata {
		payload[k] = v
	}

	return cmdutil.WriteOutput(cmd, out, payload, func() error {
		if metadata != nil {
			if pr, ok := metadata["pull_request"].(map[string]any); ok {
				if _, err := fmt.Fprintf(out, "Pull request #%d: %s\n", pr["id"], pr["title"]); err != nil {
					return err
				}
			}
			if ctx, ok := metadata["context"].(map[string]any); ok {
				if _, err := fmt.Fprintf(out, "Project %s / Repo %s\n", ctx["project"], ctx["repo"]); err != nil {
					return err
				}
			}
		}

		if _, err := fmt.Fprintf(out, "Commit %s\n", commit); err != nil {
			return err
		}
		if len(summaries) == 0 {
			_, err := fmt.Fprintln(out, "No statuses reported.")
			return err
		}

		for _, s := range summaries {
			line := fmt.Sprintf("%-10s %-20s %s", s.State, s.Key, s.Name)
			if s.Description != "" {
				line = fmt.Sprintf("%s — %s", line, s.Description)
			}
			if _, err := fmt.Fprintln(out, line); err != nil {
				return err
			}
			if s.URL != "" {
				if _, err := fmt.Fprintf(out, "    %s\n", s.URL); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
