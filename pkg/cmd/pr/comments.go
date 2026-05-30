package pr

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type commentsOptions struct {
	Workspace string
	Project   string
	Repo      string
	State     string // "all", "resolved", "unresolved", "deleted"
	Details   bool
}

type commentThreadStateResult struct {
	PullRequest int                                   `json:"pull_request" yaml:"pull_request"`
	CommentID   int                                   `json:"comment_id" yaml:"comment_id"`
	Resolved    bool                                  `json:"resolved" yaml:"resolved"`
	Resolution  *bbcloud.PullRequestCommentResolution `json:"resolution,omitempty" yaml:"resolution,omitempty"`
}

type commentDeleteResult struct {
	PullRequest int  `json:"pull_request" yaml:"pull_request"`
	CommentID   int  `json:"comment_id" yaml:"comment_id"`
	Deleted     bool `json:"deleted" yaml:"deleted"`
}

func newCommentsCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentsOptions{}
	cmd := &cobra.Command{
		Use:   "comments <id>",
		Short: "List comments on a pull request",
		Long: `List all comments on a pull request. On Cloud, use --state to filter by
resolution status (resolved, unresolved, or deleted). The --state flag is not supported
on Data Center because the DC API does not expose resolution status.

Works on both Data Center and Cloud.`,
		Example: `  # List all comments
  bkt pr comments 42

  # List only unresolved comments (Cloud only)
  bkt pr comments 42 --state unresolved

  # List resolved comments (Cloud only)
  bkt pr comments 42 --state resolved

  # List deleted comments (Cloud only)
  bkt pr comments 42 --state deleted

  # Delete a comment
  bkt pr comments delete 42 1001

  # Resolve a comment thread
  bkt pr comments resolve 42 1001

  # Reopen a resolved comment thread
  bkt pr comments reopen 42 1001`,
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
	cmd.Flags().StringVar(&opts.State, "state", "all", "Filter by state: all, resolved, unresolved, deleted (Cloud only)")
	cmd.Flags().BoolVar(&opts.Details, "details", false, "Show full comment details (file, resolved, task status)")

	cmd.AddCommand(newCommentsResolveCmd(f))
	cmd.AddCommand(newCommentsReopenCmd(f))
	cmd.AddCommand(newCommentsDeleteCmd(f))

	return cmd
}

func newCommentsResolveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentsOptions{}
	cmd := &cobra.Command{
		Use:     "resolve <id> <comment-id>",
		Short:   "Resolve a pull request comment thread",
		Example: "  bkt pr comments resolve 42 1001",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, commentID, err := parseCommentThreadArgs(args)
			if err != nil {
				return err
			}
			return runCommentThreadSetState(cmd, f, opts, prID, commentID, true)
		},
	}
	registerCommentsTargetFlags(cmd, opts)
	return cmd
}

func newCommentsReopenCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentsOptions{}
	cmd := &cobra.Command{
		Use:     "reopen <id> <comment-id>",
		Short:   "Reopen a resolved pull request comment thread",
		Example: "  bkt pr comments reopen 42 1001",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, commentID, err := parseCommentThreadArgs(args)
			if err != nil {
				return err
			}
			return runCommentThreadSetState(cmd, f, opts, prID, commentID, false)
		},
	}
	registerCommentsTargetFlags(cmd, opts)
	return cmd
}

func newCommentsDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentsOptions{}
	cmd := &cobra.Command{
		Use:     "delete <id> <comment-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a pull request comment",
		Example: "  bkt pr comments delete 42 1001",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, commentID, err := parseCommentThreadArgs(args)
			if err != nil {
				return err
			}
			return runCommentDelete(cmd, f, opts, prID, commentID)
		},
	}
	registerCommentsTargetFlags(cmd, opts)
	return cmd
}

func registerCommentsTargetFlags(cmd *cobra.Command, opts *commentsOptions) {
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
}

func parseCommentThreadArgs(args []string) (int, int, error) {
	prID, err := strconv.Atoi(args[0])
	if err != nil || prID <= 0 {
		return 0, 0, fmt.Errorf("invalid pull request id %q", args[0])
	}
	commentID, err := strconv.Atoi(args[1])
	if err != nil || commentID <= 0 {
		return 0, 0, fmt.Errorf("invalid comment id %q", args[1])
	}
	return prID, commentID, nil
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
	if state != "all" && state != "resolved" && state != "unresolved" && state != "deleted" {
		return fmt.Errorf("invalid --state value %q: must be all, resolved, unresolved, or deleted", opts.State)
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

		const maxDepth = 20
		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(comments) == 0 {
				_, err := fmt.Fprintf(ios.Out, "No comments on pull request #%d\n", id)
				return err
			}
			maxIndent := strings.Repeat("  ", maxDepth)
			var skippedDeep bool
			printSkipped := func() error {
				if skippedDeep {
					skippedDeep = false
					_, err := fmt.Fprintf(ios.Out, "%s[...]\n", maxIndent)
					if err != nil {
						return err
					}
				}
				return nil
			}
			for _, c := range comments {
				if c.Depth > maxDepth {
					skippedDeep = true
					continue
				}
				if err := printSkipped(); err != nil {
					return err
				}
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
				}
				resolved := "no"
				if c.ThreadResolved {
					resolved = "yes"
				}
				if _, err := fmt.Fprintf(ios.Out, "%sResolved: %s\n", indent, resolved); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(ios.Out, "\n%s%s\n\n", indent, c.Text); err != nil {
					return err
				}
			}
			return printSkipped()
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

		// Client-side filtering for resolved/unresolved/deleted.
		if state != "all" {
			filtered := make([]bbcloud.PullRequestComment, 0, len(comments))
			for _, c := range comments {
				if c.Deleted {
					if state == "deleted" {
						filtered = append(filtered, c)
					}
					continue
				}
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
					if c.Deleted {
						if _, err := fmt.Fprintf(ios.Out, "%d\t%s\t[deleted]\n", c.ID, author); err != nil {
							return err
						}
						continue
					}
					text := truncate(c.Content.Raw, 80)
					if _, err := fmt.Fprintf(ios.Out, "%d\t%s\t%s\n", c.ID, author, text); err != nil {
						return err
					}
					continue
				}
				if _, err := fmt.Fprintf(ios.Out, "--- Comment #%d by %s ---\n", c.ID, author); err != nil {
					return err
				}
				if c.Deleted {
					if _, err := fmt.Fprintf(ios.Out, "Deleted: yes\n\n"); err != nil {
						return err
					}
					continue
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

func runCommentThreadSetState(cmd *cobra.Command, f *cmdutil.Factory, opts *commentsOptions, prID, commentID int, resolved bool) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), timeoutWrite)
	defer cancel()

	result := commentThreadStateResult{
		PullRequest: prID,
		CommentID:   commentID,
		Resolved:    resolved,
	}
	already := false

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
		if _, err := client.SetPullRequestCommentThreadResolved(ctx, projectKey, repoSlug, prID, commentID, resolved); err != nil {
			if errors.Is(err, bbdc.ErrPullRequestCommentNotTopLevel) {
				return topLevelCommentThreadError(resolved)
			}
			return err
		}
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
		mapped, isAlready, err := inspectCommentThreadCloudState(ctx, client, workspace, repoSlug, prID, commentID, resolved)
		if err != nil {
			return err
		}
		if mapped != nil {
			return mapped
		}
		if isAlready {
			already = true
			break
		}
		resolution, err := client.SetPullRequestCommentThreadResolved(ctx, workspace, repoSlug, prID, commentID, resolved)
		if err != nil {
			mapped, isAlready, inspectErr := inspectCommentThreadCloudState(ctx, client, workspace, repoSlug, prID, commentID, resolved)
			if inspectErr == nil {
				if mapped != nil {
					return mapped
				}
				if isAlready {
					already = true
					break
				}
			}
			return err
		}
		result.Resolution = resolution
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}

	return cmdutil.WriteOutput(cmd, ios.Out, result, func() error {
		if already {
			state := "resolved"
			if !resolved {
				state = "open"
			}
			_, err := fmt.Fprintf(ios.Out, "✓ Comment thread %d on pull request #%d is already %s\n", commentID, prID, state)
			return err
		}
		verb := "Reopened"
		if resolved {
			verb = "Resolved"
		}
		_, err := fmt.Fprintf(ios.Out, "✓ %s comment thread %d on pull request #%d\n", verb, commentID, prID)
		return err
	})
}

func runCommentDelete(cmd *cobra.Command, f *cmdutil.Factory, opts *commentsOptions, prID, commentID int) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), timeoutWrite)
	defer cancel()

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
		if err := client.DeletePullRequestComment(ctx, projectKey, repoSlug, prID, commentID); err != nil {
			return err
		}
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
		if err := client.DeletePullRequestComment(ctx, workspace, repoSlug, prID, commentID); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}

	result := commentDeleteResult{
		PullRequest: prID,
		CommentID:   commentID,
		Deleted:     true,
	}
	return cmdutil.WriteOutput(cmd, ios.Out, result, func() error {
		_, err := fmt.Fprintf(ios.Out, "✓ Deleted comment %d on pull request #%d\n", commentID, prID)
		return err
	})
}

func inspectCommentThreadCloudState(ctx context.Context, client *bbcloud.Client, workspace, repoSlug string, prID, commentID int, resolved bool) (error, bool, error) {
	comment, err := client.GetPullRequestComment(ctx, workspace, repoSlug, prID, commentID)
	if err != nil {
		return nil, false, err
	}
	if comment.Deleted {
		return deletedCommentThreadError(commentID, resolved), false, nil
	}
	if comment.Parent != nil {
		return topLevelCommentThreadError(resolved), false, nil
	}
	if resolved == (comment.Resolution != nil) {
		return nil, true, nil
	}
	return nil, false, nil
}

func deletedCommentThreadError(commentID int, resolved bool) error {
	action := "resolved"
	if !resolved {
		action = "reopened"
	}
	return fmt.Errorf("Pull request comment %d has been deleted and cannot be %s.", commentID, action)
}

func topLevelCommentThreadError(resolved bool) error {
	action := "resolved"
	if !resolved {
		action = "reopened"
	}
	return fmt.Errorf("Only top-level pull request comment threads can be %s.", action)
}

// truncate shortens s to at most maxLen runes, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if maxLen <= 0 {
		return ""
	}
	if len(runes) <= maxLen {
		return string(runes)
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}
