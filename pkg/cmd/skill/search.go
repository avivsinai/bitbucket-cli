package skill

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type searchOptions struct {
	Workspace string
	Limit     int
}

type searchResult struct {
	Repository string `json:"repository,omitempty"`
	Path       string `json:"path"`
	Matches    int    `json:"matches"`
	URL        string `json:"url,omitempty"`
}

func newSearchCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &searchOptions{Limit: 30}
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for skills in Bitbucket Cloud code",
		Long: `Search for skill files across repositories in a Bitbucket Cloud workspace.

The query uses Bitbucket Cloud code-search syntax. Scope it further with terms
such as repo:agent-skills or path:skills. The workspace comes from --workspace
or the active context. Results are restricted to SKILL.md files. Bitbucket Data
Center is not supported because it has no public workspace code-search API.

Note: Atlassian has announced that the Cloud code-search REST endpoint will be
deprecated on November 1, 2026.`,
		Example: `  # Search the active context's workspace
  bkt skill search "code review"

  # Search one repository and emit structured output
  bkt skill search "review repo:agent-skills" --workspace myteam --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd, f, opts, args[0])
		},
	}
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace (defaults to context workspace)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", opts.Limit, "Maximum matches to return")
	return cmd
}

func runSearch(cmd *cobra.Command, f *cmdutil.Factory, opts *searchOptions, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("search query cannot be empty")
	}
	if opts.Limit < 1 {
		return fmt.Errorf("--limit must be at least 1")
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "cloud" {
		return fmt.Errorf("skill search is not supported for Bitbucket Data Center; use a Bitbucket Cloud context")
	}
	workspace := strings.TrimSpace(opts.Workspace)
	if cmd.Flags().Changed("workspace") && workspace == "" {
		return fmt.Errorf("--workspace cannot be blank")
	}
	if workspace == "" {
		workspace = strings.TrimSpace(ctxCfg.Workspace)
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	client, err := f.CloudClient(host)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	matches, err := client.SearchWorkspaceSkills(ctx, workspace, query, opts.Limit)
	if err != nil {
		return fmt.Errorf("search workspace %q: %w", workspace, err)
	}

	results := make([]searchResult, 0, len(matches))
	for _, match := range matches {
		results = append(results, summarizeSearchResult(match))
	}
	payload := struct {
		Workspace string         `json:"workspace"`
		Query     string         `json:"query"`
		Results   []searchResult `json:"results"`
	}{Workspace: workspace, Query: query, Results: results}

	ios, err := f.Streams()
	if err != nil {
		return err
	}
	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(results) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No skills found in workspace %s.\n", sanitizeForTerminal(workspace))
			return err
		}
		for _, result := range results {
			repository := result.Repository
			if repository == "" {
				repository = workspace
			}
			if _, err := fmt.Fprintf(ios.Out, "%s\t%s\t%d match(es)\n", sanitizeForTerminal(repository), sanitizeForTerminal(result.Path), result.Matches); err != nil {
				return err
			}
		}
		return nil
	})
}

func summarizeSearchResult(result bbcloud.CodeSearchResult) searchResult {
	return searchResult{
		Repository: result.File.Commit.Repository.FullName,
		Path:       result.File.Path,
		Matches:    result.ContentMatchCount,
		URL:        result.File.Links.Self.Href,
	}
}
