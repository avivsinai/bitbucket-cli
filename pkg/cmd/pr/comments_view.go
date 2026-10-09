package pr

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// maxCommentDepth caps reply indentation in text output.
const maxCommentDepth = 20

func newCommentsViewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentsOptions{}
	cmd := &cobra.Command{
		Use:   "view <id> <comment-id>",
		Short: "View one pull request comment thread",
		Long: `Show the comment thread that contains a comment: the root comment, its
replies, and the inline anchor (file, line, orphaned). The comment-id can be the
root comment or any reply; the requested comment is marked in the output.

On Data Center, the output also includes a few lines of diff around the
anchored line when Bitbucket returns them. --json returns the root comment as
"thread" with replies nested under "comments" and the diff lines as
"diff_context". On Cloud, --json returns the thread as a flat "comments" list,
root first.`,
		Example: `  # View the thread that contains comment 1001 on pull request 42
  bkt pr comments view 42 1001

  # Get the thread as JSON
  bkt pr comments view 42 1001 --json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, commentID, err := parseCommentThreadArgs(args)
			if err != nil {
				return err
			}
			return runCommentView(cmd, f, opts, prID, commentID)
		},
	}
	registerCommentsTargetFlags(cmd, opts)
	return cmd
}

func runCommentView(cmd *cobra.Command, f *cmdutil.Factory, opts *commentsOptions, prID, commentID int) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), timeoutRead)
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
		thread, err := client.GetPullRequestCommentThread(ctx, projectKey, repoSlug, prID, commentID)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":      projectKey,
			"repo":         repoSlug,
			"pull_request": prID,
			"comment_id":   commentID,
			"thread":       thread.Root,
		}
		if len(thread.DiffContext) > 0 {
			payload["diff_context"] = thread.DiffContext
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if _, err := fmt.Fprintf(ios.Out, "Thread #%d on pull request #%d\n\n", thread.Root.ID, prID); err != nil {
				return err
			}
			if len(thread.DiffContext) > 0 {
				if err := printDiffContext(ios.Out, thread.DiffContext); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(ios.Out); err != nil {
					return err
				}
			}
			for _, c := range thread.Comments() {
				if err := printDCCommentDetails(ios.Out, c, c.ID == commentID); err != nil {
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
		all, err := client.ListPullRequestComments(ctx, workspace, repoSlug, prID, 0)
		if err != nil {
			return err
		}
		thread, depths := cloudCommentThread(all, commentID)
		if len(thread) == 0 {
			return fmt.Errorf("comment %d not found on pull request #%d", commentID, prID)
		}

		payload := map[string]any{
			"workspace":    workspace,
			"repo":         repoSlug,
			"pull_request": prID,
			"comment_id":   commentID,
			"comments":     thread,
		}
		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if _, err := fmt.Fprintf(ios.Out, "Thread #%d on pull request #%d\n\n", thread[0].ID, prID); err != nil {
				return err
			}
			for i, c := range thread {
				if err := printCloudCommentDetails(ios.Out, c, depths[i], c.ID == commentID); err != nil {
					return err
				}
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

// cloudCommentThread returns the thread that contains commentID in display
// order, root first, with each comment's reply depth.
func cloudCommentThread(comments []bbcloud.PullRequestComment, commentID int) ([]bbcloud.PullRequestComment, []int) {
	byID := make(map[int]bbcloud.PullRequestComment, len(comments))
	children := make(map[int][]bbcloud.PullRequestComment)
	for _, c := range comments {
		byID[c.ID] = c
		if c.Parent != nil {
			children[c.Parent.ID] = append(children[c.Parent.ID], c)
		}
	}
	root, ok := byID[commentID]
	if !ok {
		return nil, nil
	}
	seen := map[int]bool{root.ID: true}
	for root.Parent != nil {
		parent, ok := byID[root.Parent.ID]
		if !ok || seen[parent.ID] {
			break
		}
		seen[parent.ID] = true
		root = parent
	}

	var (
		thread []bbcloud.PullRequestComment
		depths []int
		visit  func(c bbcloud.PullRequestComment, depth int)
	)
	visited := map[int]bool{}
	visit = func(c bbcloud.PullRequestComment, depth int) {
		if visited[c.ID] {
			return
		}
		visited[c.ID] = true
		thread = append(thread, c)
		depths = append(depths, depth)
		for _, child := range children[c.ID] {
			visit(child, depth+1)
		}
	}
	visit(root, 0)
	return thread, depths
}

func dcCommentAuthor(c bbdc.PullRequestComment) string {
	if c.Author.Name != "" {
		return c.Author.Name
	}
	return c.Author.FullName
}

func cloudCommentAuthor(c bbcloud.PullRequestComment) string {
	if c.User == nil {
		return "unknown"
	}
	if c.User.DisplayName != "" {
		return c.User.DisplayName
	}
	return c.User.Nickname
}

func commentHeader(indent, kind string, id int, author string, target bool) string {
	header := fmt.Sprintf("%s--- %s #%d by %s ---", indent, kind, id, author)
	if target {
		header += " (requested)"
	}
	return header + "\n"
}

// printDCCommentDetails writes one Data Center comment in the --details
// layout, indented by its reply depth.
func printDCCommentDetails(w io.Writer, c bbdc.PullRequestComment, target bool) error {
	indent := strings.Repeat("  ", min(c.Depth, maxCommentDepth))
	kind := "Comment"
	if strings.EqualFold(c.Severity, "BLOCKER") {
		kind = "Task"
	}
	var b strings.Builder
	b.WriteString(commentHeader(indent, kind, c.ID, dcCommentAuthor(c), target))
	if c.Anchor != nil {
		b.WriteString(indent + "File: " + c.Anchor.Path)
		if c.Anchor.Line > 0 {
			b.WriteString(":" + strconv.Itoa(c.Anchor.Line))
		}
		if c.Anchor.Orphaned {
			b.WriteString(" (orphaned)")
		}
		b.WriteString("\n")
	}
	if kind == "Task" {
		fmt.Fprintf(&b, "%sComplete: %s\n", indent, yesNo(strings.EqualFold(c.State, "RESOLVED")))
	}
	fmt.Fprintf(&b, "%sResolved: %s\n", indent, yesNo(c.ThreadResolved))
	fmt.Fprintf(&b, "\n%s%s\n\n", indent, c.Text)
	_, err := io.WriteString(w, b.String())
	return err
}

// printCloudCommentDetails writes one Cloud comment in the --details layout,
// indented by depth.
func printCloudCommentDetails(w io.Writer, c bbcloud.PullRequestComment, depth int, target bool) error {
	indent := strings.Repeat("  ", min(depth, maxCommentDepth))
	var b strings.Builder
	b.WriteString(commentHeader(indent, "Comment", c.ID, cloudCommentAuthor(c), target))
	if c.Deleted {
		b.WriteString(indent + "Deleted: yes\n\n")
		_, err := io.WriteString(w, b.String())
		return err
	}
	if c.Inline != nil {
		b.WriteString(indent + "File: " + c.Inline.Path)
		if c.Inline.To != nil {
			b.WriteString(":" + strconv.Itoa(*c.Inline.To))
		} else if c.Inline.From != nil {
			b.WriteString(":" + strconv.Itoa(*c.Inline.From))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "%sResolved: %s\n", indent, yesNo(c.Resolution != nil))
	fmt.Fprintf(&b, "\n%s%s\n\n", indent, c.Content.Raw)
	_, err := io.WriteString(w, b.String())
	return err
}

// printDiffContext writes diff lines as "marker source destination sign text";
// the anchored line is marked with ">".
func printDiffContext(w io.Writer, lines []bbdc.PullRequestDiffLine) error {
	var b strings.Builder
	for _, l := range lines {
		marker := " "
		if l.Anchored {
			marker = ">"
		}
		src, dst, sign := lineNumber(l.Source), lineNumber(l.Destination), " "
		switch l.Type {
		case "ADDED":
			src, sign = "", "+"
		case "REMOVED":
			dst, sign = "", "-"
		}
		fmt.Fprintf(&b, "%s %5s %5s %s%s\n", marker, src, dst, sign, l.Line)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func lineNumber(n int) string {
	if n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
