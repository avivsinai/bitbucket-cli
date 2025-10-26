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
	"github.com/avivsinai/bitbucket-cli/pkg/format"
)

// NewCmdStatus exposes commit and PR status commands.
func NewCmdStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect commit and pull request statuses",
	}

	cmd.AddCommand(newCommitCmd(f))
	cmd.AddCommand(newPullRequestCmd(f))

	return cmd
}

func newCommitCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit <sha>",
		Short: "Show the build statuses for a commit",
		Args:  cobra.ExactArgs(1),
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
		BaseURL:  host.BaseURL,
		Username: host.Username,
		Token:    host.Token,
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
		Short: "Show the build statuses for a pull request head commit",
		Args:  cobra.ExactArgs(1),
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
		BaseURL:  host.BaseURL,
		Username: host.Username,
		Token:    host.Token,
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
	formatOpt, err := cmdutil.OutputFormat(cmd)
	if err != nil {
		return err
	}

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

	if formatOpt != "" {
		payload := map[string]any{
			"commit":   commit,
			"statuses": summaries,
		}
		for k, v := range metadata {
			payload[k] = v
		}
		return format.Write(out, formatOpt, payload, nil)
	}

	if metadata != nil {
		if pr, ok := metadata["pull_request"].(map[string]any); ok {
			fmt.Fprintf(out, "Pull request #%d: %s\n", pr["id"], pr["title"])
		}
		if ctx, ok := metadata["context"].(map[string]any); ok {
			fmt.Fprintf(out, "Project %s / Repo %s\n", ctx["project"], ctx["repo"])
		}
	}

	fmt.Fprintf(out, "Commit %s\n", commit)
	if len(summaries) == 0 {
		fmt.Fprintln(out, "No statuses reported.")
		return nil
	}

	for _, s := range summaries {
		line := fmt.Sprintf("%-10s %-20s %s", s.State, s.Key, s.Name)
		if s.Description != "" {
			line = fmt.Sprintf("%s — %s", line, s.Description)
		}
		fmt.Fprintln(out, line)
		if s.URL != "" {
			fmt.Fprintf(out, "    %s\n", s.URL)
		}
	}
	return nil
}
