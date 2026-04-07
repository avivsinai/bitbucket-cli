package pr

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type commentsOptions struct {
	Workspace string
	Project   string
	Repo      string
	State   string // "all", "resolved", "unresolved"
	Details bool
}

func newCommentsCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentsOptions{}
	cmd := &cobra.Command{
		Use:   "comments <id>",
		Short: "List comments on a pull request",
		Long: `List all comments on a pull request. On Cloud, use --state to filter by
resolution status (resolved or unresolved). The --state flag is not supported
on Data Center because the DC API does not expose resolution status.

Works on both Data Center and Cloud.`,
		Example: `  # List all comments
  bkt pr comments 42

  # List only unresolved comments (Cloud only)
  bkt pr comments 42 --state unresolved

  # List resolved comments (Cloud only)
  bkt pr comments 42 --state resolved`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runComments(cmd, f, id, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.State, "state", "all", "Filter by state: all, resolved, unresolved (Cloud only)")
	cmd.Flags().BoolVar(&opts.Details, "details", false, "Show full comment details (file, resolved, task status)")

	return cmd
}

func runComments(cmd *cobra.Command, f *cmdutil.Factory, id int, opts *commentsOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	state := strings.ToLower(strings.TrimSpace(opts.State))
	if state != "all" && state != "resolved" && state != "unresolved" {
		return fmt.Errorf("invalid --state value %q: must be all, resolved, or unresolved", opts.State)
	}

	switch host.Kind {
	case "dc":
		if state != "all" {
			return fmt.Errorf("--state filtering is only supported on Cloud contexts (Data Center does not expose resolved status)")
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

		ctx, cancel := context.WithTimeout(cmd.Context(), timeoutRead)
		defer cancel()

		comments, err := client.ListPullRequestComments(ctx, projectKey, repoSlug, id)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":  projectKey,
			"repo":     repoSlug,
			"comments": comments,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(comments) == 0 {
				_, err := fmt.Fprintf(ios.Out, "No comments on pull request #%d\n", id)
				return err
			}
			for _, c := range comments {
				author := c.Author.Name
				if author == "" {
					author = c.Author.FullName
				}
				if !opts.Details {
					indent := strings.Repeat("  ", c.Depth)
					text := truncate(c.Text, 80-2*c.Depth)
					if _, err := fmt.Fprintf(ios.Out, "%d\t%s\t%s%s\n", c.ID, author, indent, text); err != nil {
						return err
					}
					continue
				}
				indent := strings.Repeat("  ", c.Depth)
				kind := "Comment"
				if strings.EqualFold(c.Severity, "BLOCKER") {
					kind = "Task"
				}
				if _, err := fmt.Fprintf(ios.Out, "%s--- %s #%d by %s ---\n", indent, kind, c.ID, author); err != nil {
					return err
				}
				if c.Anchor != nil {
					if c.Anchor.Line > 0 {
						if _, err := fmt.Fprintf(ios.Out, "%sFile: %s:%d\n", indent, c.Anchor.Path, c.Anchor.Line); err != nil {
							return err
						}
					} else {
						if _, err := fmt.Fprintf(ios.Out, "%sFile: %s\n", indent, c.Anchor.Path); err != nil {
							return err
						}
					}
				}
				if kind == "Task" {
					complete := "no"
					if strings.EqualFold(c.State, "RESOLVED") {
						complete = "yes"
					}
					if _, err := fmt.Fprintf(ios.Out, "%sComplete: %s\n", indent, complete); err != nil {
						return err
					}
				} else {
					resolved := "no"
					if c.ThreadResolved {
						resolved = "yes"
					}
					if _, err := fmt.Fprintf(ios.Out, "%sResolved: %s\n", indent, resolved); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintf(ios.Out, "\n%s%s\n\n", indent, c.Text); err != nil {
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

		ctx, cancel := context.WithTimeout(cmd.Context(), timeoutRead)
		defer cancel()

		comments, err := client.ListPullRequestComments(ctx, workspace, repoSlug, id, 0)
		if err != nil {
			return err
		}

		// Client-side filtering for resolved/unresolved
		if state != "all" {
			filtered := make([]bbcloud.PullRequestComment, 0, len(comments))
			for _, c := range comments {
				switch state {
				case "resolved":
					if c.Resolution != nil {
						filtered = append(filtered, c)
					}
				case "unresolved":
					if c.Resolution == nil {
						filtered = append(filtered, c)
					}
				}
			}
			comments = filtered
		}

		payload := map[string]any{
			"workspace": workspace,
			"repo":      repoSlug,
			"comments":  comments,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(comments) == 0 {
				_, err := fmt.Fprintf(ios.Out, "No comments on pull request #%d\n", id)
				return err
			}
			for _, c := range comments {
				author := "unknown"
				if c.User != nil {
					author = c.User.DisplayName
					if author == "" {
						author = c.User.Nickname
					}
				}
				if !opts.Details {
					text := truncate(c.Content.Raw, 80)
					if _, err := fmt.Fprintf(ios.Out, "%d\t%s\t%s\n", c.ID, author, text); err != nil {
						return err
					}
					continue
				}
				if _, err := fmt.Fprintf(ios.Out, "--- Comment #%d by %s ---\n", c.ID, author); err != nil {
					return err
				}
				if c.Inline != nil {
					line := ""
					if c.Inline.To != nil {
						line = fmt.Sprintf(":%d", *c.Inline.To)
					} else if c.Inline.From != nil {
						line = fmt.Sprintf(":%d", *c.Inline.From)
					}
					if _, err := fmt.Fprintf(ios.Out, "File: %s%s\n", c.Inline.Path, line); err != nil {
						return err
					}
				}
				resolved := "no"
				if c.Resolution != nil {
					resolved = "yes"
				}
				if _, err := fmt.Fprintf(ios.Out, "Resolved: %s\n", resolved); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(ios.Out, "\n%s\n\n", c.Content.Raw); err != nil {
					return err
				}
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

// truncate shortens s to at most maxLen runes, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
