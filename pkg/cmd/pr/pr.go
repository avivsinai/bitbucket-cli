package pr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdPR returns the pull request command tree.
func NewCmdPR(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage pull requests",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newCheckoutCmd(f))
	cmd.AddCommand(newDiffCmd(f))
	cmd.AddCommand(newApproveCmd(f))
	cmd.AddCommand(newMergeCmd(f))
	cmd.AddCommand(newDeclineCmd(f))
	cmd.AddCommand(newReopenCmd(f))
	cmd.AddCommand(newCommentCmd(f))
	cmd.AddCommand(newReviewerGroupCmd(f))
	cmd.AddCommand(newAutoMergeCmd(f))
	cmd.AddCommand(newTaskCmd(f))
	cmd.AddCommand(newReactionCmd(f))
	cmd.AddCommand(newSuggestionCmd(f))

	return cmd
}

type listOptions struct {
	Project   string
	Workspace string
	Repo      string
	State     string
	Limit     int
	Mine      bool
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{State: "OPEN", Limit: 20}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.State, "state", opts.State, "Filter by state (OPEN, MERGED, DECLINED)")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum pull requests to list (0 for all)")
	cmd.Flags().BoolVar(&opts.Mine, "mine", false, "Show pull requests authored by the authenticated user")

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
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		prs, err := client.ListPullRequests(ctx, projectKey, repoSlug, opts.State, opts.Limit)
		if err != nil {
			return err
		}

		if opts.Mine && host.Username != "" {
			filtered := prs[:0]
			current := strings.ToLower(host.Username)
			for _, pr := range prs {
				author := strings.ToLower(firstNonEmpty(pr.Author.User.Name, pr.Author.User.Slug))
				if author == current {
					filtered = append(filtered, pr)
				}
			}
			prs = filtered
		}

		payload := map[string]any{
			"project":       projectKey,
			"repo":          repoSlug,
			"pull_requests": prs,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(prs) == 0 {
				_, err := fmt.Fprintf(ios.Out, "No pull requests (%s).\n", strings.ToUpper(opts.State))
				return err
			}

			for _, pr := range prs {
				author := firstNonEmpty(pr.Author.User.FullName, pr.Author.User.Name)
				if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\n", pr.ID, pr.State, pr.Title); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(ios.Out, "    %s -> %s\tby %s\n", pr.FromRef.DisplayID, pr.ToRef.DisplayID, author); err != nil {
					return err
				}
			}
			return nil
		})

	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		mine := ""
		if opts.Mine && host.Username != "" {
			mine = host.Username
		}

		prs, err := client.ListPullRequests(ctx, workspace, repoSlug, bbcloud.PullRequestListOptions{
			State: opts.State,
			Limit: opts.Limit,
			Mine:  mine,
		})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace":     workspace,
			"repo":          repoSlug,
			"pull_requests": prs,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(prs) == 0 {
				_, err := fmt.Fprintf(ios.Out, "No pull requests (%s).\n", strings.ToUpper(opts.State))
				return err
			}

			for _, pr := range prs {
				author := firstNonEmpty(pr.Author.DisplayName, pr.Author.Username)
				if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\n", pr.ID, pr.State, pr.Title); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(ios.Out, "    %s -> %s\tby %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name, author); err != nil {
					return err
				}
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type viewOptions struct {
	Project   string
	Workspace string
	Repo      string
	ID        int
	Web       bool
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "Show details for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runView(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.Web, "web", false, "Open the pull request in your browser")

	return cmd
}

func runView(cmd *cobra.Command, f *cmdutil.Factory, opts *viewOptions) error {
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
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":      projectKey,
			"repo":         repoSlug,
			"pull_request": pr,
		}

		if opts.Web {
			if link := firstPRLinkDC(pr, "self"); link != "" {
				if err := f.BrowserOpener().Open(link); err != nil {
					return fmt.Errorf("open browser: %w", err)
				}
			} else {
				return fmt.Errorf("pull request does not expose a web URL")
			}
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if _, err := fmt.Fprintf(ios.Out, "Pull Request #%d: %s\n", pr.ID, pr.Title); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "State: %s\n", pr.State); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "Author: %s\n", firstNonEmpty(pr.Author.User.FullName, pr.Author.User.Name)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "From: %s\nTo:   %s\n", pr.FromRef.DisplayID, pr.ToRef.DisplayID); err != nil {
				return err
			}
			if strings.TrimSpace(pr.Description) != "" {
				if _, err := fmt.Fprintf(ios.Out, "\n%s\n", pr.Description); err != nil {
					return err
				}
			}

			if len(pr.Reviewers) > 0 {
				if _, err := fmt.Fprintln(ios.Out, "\nReviewers:"); err != nil {
					return err
				}
				for _, reviewer := range pr.Reviewers {
					if _, err := fmt.Fprintf(ios.Out, "  %s\n", firstNonEmpty(reviewer.User.FullName, reviewer.User.Name)); err != nil {
						return err
					}
				}
			}
			return nil
		})

	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.GetPullRequest(ctx, workspace, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace":    workspace,
			"repo":         repoSlug,
			"pull_request": pr,
		}

		if opts.Web {
			if link := firstPRLinkCloud(pr); link != "" {
				if err := f.BrowserOpener().Open(link); err != nil {
					return fmt.Errorf("open browser: %w", err)
				}
			} else {
				return fmt.Errorf("pull request does not expose a web URL")
			}
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if _, err := fmt.Fprintf(ios.Out, "Pull Request #%d: %s\n", pr.ID, pr.Title); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "State: %s\n", pr.State); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "Author: %s\n", firstNonEmpty(pr.Author.DisplayName, pr.Author.Username)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "From: %s\nTo:   %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name); err != nil {
				return err
			}
			if strings.TrimSpace(pr.Summary.Raw) != "" {
				if _, err := fmt.Fprintf(ios.Out, "\n%s\n", pr.Summary.Raw); err != nil {
					return err
				}
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func firstPRLinkDC(pr *bbdc.PullRequest, kind string) string {
	if pr == nil {
		return ""
	}
	switch kind {
	case "self":
		for _, link := range pr.Links.Self {
			if strings.TrimSpace(link.Href) != "" {
				return link.Href
			}
		}
	}
	return ""
}

func firstPRLinkCloud(pr *bbcloud.PullRequest) string {
	if pr == nil {
		return ""
	}
	if pr.Links.HTML.Href != "" {
		return pr.Links.HTML.Href
	}
	return ""
}

type createOptions struct {
	Project     string
	Workspace   string
	Repo        string
	Title       string
	Source      string
	Target      string
	Description string
	Reviewers   []string
	CloseSource bool
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Title, "title", "", "Pull request title (required)")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Pull request description")
	cmd.Flags().StringVar(&opts.Source, "source", "", "Source branch (required)")
	cmd.Flags().StringVar(&opts.Target, "target", "", "Target branch (required)")
	cmd.Flags().StringSliceVar(&opts.Reviewers, "reviewer", nil, "Reviewers to request (repeatable)")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", false, "Close source branch on merge")

	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("target")

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
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.CreatePullRequest(ctx, projectKey, repoSlug, bbdc.CreatePROptions{
			Title:        opts.Title,
			Description:  opts.Description,
			SourceBranch: opts.Source,
			TargetBranch: opts.Target,
			Reviewers:    opts.Reviewers,
			CloseSource:  opts.CloseSource,
		})
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Created pull request #%d\n", pr.ID); err != nil {
			return err
		}
		return nil

	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.CreatePullRequest(ctx, workspace, repoSlug, bbcloud.CreatePullRequestInput{
			Title:       opts.Title,
			Description: opts.Description,
			Source:      opts.Source,
			Destination: opts.Target,
			CloseSource: opts.CloseSource,
			Reviewers:   opts.Reviewers,
		})
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Created pull request #%d\n", pr.ID); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type checkoutOptions struct {
	Project string
	Repo    string
	ID      int
	Branch  string
	Remote  string
}

func newCheckoutCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &checkoutOptions{Remote: "origin"}
	cmd := &cobra.Command{
		Use:   "checkout <id>",
		Short: "Check out the pull request branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runCheckout(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "Local branch name (defaults to pr/<id>)")
	cmd.Flags().StringVar(&opts.Remote, "remote", opts.Remote, "Git remote name to fetch from")

	return cmd
}

func runCheckout(cmd *cobra.Command, f *cmdutil.Factory, opts *checkoutOptions) error {
	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("pr checkout currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	branchName := opts.Branch
	if branchName == "" {
		branchName = fmt.Sprintf("pr/%d", opts.ID)
	}

	ref := fmt.Sprintf("refs/pull-requests/%d/from", opts.ID)
	fetchArgs := []string{"fetch", opts.Remote, fmt.Sprintf("%s:%s", ref, branchName)}
	if err := runGit(cmd.Context(), fetchArgs...); err != nil {
		return err
	}

	if err := runGit(cmd.Context(), "checkout", branchName); err != nil {
		return err
	}
	return nil
}

type diffOptions struct {
	Project string
	Repo    string
	ID      int
	Stat    bool
}

func newDiffCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &diffOptions{}
	cmd := &cobra.Command{
		Use:   "diff <id>",
		Short: "Show the diff for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runDiff(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.Stat, "stat", false, "Show diff statistics instead of full patch")

	return cmd
}

func runDiff(cmd *cobra.Command, f *cmdutil.Factory, opts *diffOptions) error {
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
		return fmt.Errorf("pr diff currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	if opts.Stat {
		stat, err := client.PullRequestDiffStat(ctx, projectKey, repoSlug, opts.ID)
		if err != nil {
			return err
		}
		payload := map[string]any{
			"project":      projectKey,
			"repo":         repoSlug,
			"pull_request": opts.ID,
			"stats":        stat,
		}
		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			_, err := fmt.Fprintf(ios.Out, "Files: %d\nAdditions: %d\nDeletions: %d\n", stat.Files, stat.Additions, stat.Deletions)
			return err
		})
	}

	pager := f.PagerManager()
	if pager.Enabled() {
		w, err := pager.Start()
		if err == nil {
			defer func() { _ = pager.Stop() }()
			return client.PullRequestDiff(ctx, projectKey, repoSlug, opts.ID, w)
		}
	}

	return client.PullRequestDiff(ctx, projectKey, repoSlug, opts.ID, ios.Out)
}

func newApproveCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runApprove(cmd, f, id)
		},
	}
	return cmd
}

func runApprove(cmd *cobra.Command, f *cmdutil.Factory, id int) error {
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
		return fmt.Errorf("pr approve currently supports Data Center contexts only")
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

	if err := client.ApprovePullRequest(ctx, projectKey, repoSlug, id); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Approved pull request #%d\n", id); err != nil {
		return err
	}
	return nil
}

type mergeOptions struct {
	Message     string
	Strategy    string
	CloseSource bool
	Project     string
	Repo        string
}

func newMergeCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &mergeOptions{}
	cmd := &cobra.Command{
		Use:   "merge <id>",
		Short: "Merge a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runMerge(cmd, f, id, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Message, "message", "", "Merge commit message override")
	cmd.Flags().StringVar(&opts.Strategy, "strategy", "", "Merge strategy ID (e.g., fast-forward)")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", true, "Close source branch on merge")

	return cmd
}

func runMerge(cmd *cobra.Command, f *cmdutil.Factory, id int, opts *mergeOptions) error {
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
		return fmt.Errorf("pr merge currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, id)
	if err != nil {
		return err
	}

	if err := client.MergePullRequest(ctx, projectKey, repoSlug, id, pr.Version, bbdc.MergePROptions{
		Message:           opts.Message,
		Strategy:          opts.Strategy,
		CloseSourceBranch: opts.CloseSource,
	}); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Merged pull request #%d\n", id); err != nil {
		return err
	}
	return nil
}

type declineOptions struct {
	Project      string
	Workspace    string
	Repo         string
	DeleteSource bool
}

func newDeclineCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &declineOptions{}
	cmd := &cobra.Command{
		Use:   "decline <id>",
		Short: "Decline a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runDecline(cmd, f, id, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.DeleteSource, "delete-source", false, "Delete the source branch after declining")

	return cmd
}

func runDecline(cmd *cobra.Command, f *cmdutil.Factory, id int, opts *declineOptions) error {
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
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, id)
		if err != nil {
			return err
		}

		if err := client.DeclinePullRequest(ctx, projectKey, repoSlug, id, pr.Version); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "Declined pull request #%d\n", id); err != nil {
			return err
		}

		if opts.DeleteSource {
			sourceBranch := pr.FromRef.DisplayID
			if sourceBranch == "" {
				sourceBranch = pr.FromRef.ID
			}
			if sourceBranch != "" {
				if err := client.DeleteBranch(ctx, projectKey, repoSlug, sourceBranch, false); err != nil {
					return fmt.Errorf("declined PR but failed to delete source branch %q: %w", sourceBranch, err)
				}
				if _, err := fmt.Fprintf(ios.Out, "Deleted source branch %s\n", sourceBranch); err != nil {
					return err
				}
			}
		}

		return nil

	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		if err := client.DeclinePullRequest(ctx, workspace, repoSlug, id); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "Declined pull request #%d\n", id); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func newReopenCmd(f *cmdutil.Factory) *cobra.Command {
	var project, workspace, repo string
	cmd := &cobra.Command{
		Use:   "reopen <id>",
		Short: "Reopen a declined pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runReopen(cmd, f, id, project, workspace, repo)
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository slug override")

	return cmd
}

func runReopen(cmd *cobra.Command, f *cmdutil.Factory, id int, project, workspace, repo string) error {
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
		projectKey := firstNonEmpty(project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, id)
		if err != nil {
			return err
		}

		if err := client.ReopenPullRequest(ctx, projectKey, repoSlug, id, pr.Version); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "Reopened pull request #%d\n", id); err != nil {
			return err
		}
		return nil

	case "cloud":
		ws := firstNonEmpty(workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(repo, ctxCfg.DefaultRepo)
		if ws == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		if err := client.ReopenPullRequest(ctx, ws, repoSlug, id); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "Reopened pull request #%d\n", id); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type commentOptions struct {
	Project string
	Repo    string
	Text    string
}

func newCommentCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentOptions{}
	cmd := &cobra.Command{
		Use:   "comment <id> --text <message>",
		Short: "Comment on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runComment(cmd, f, id, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Text, "text", "", "Comment text")
	_ = cmd.MarkFlagRequired("text")

	return cmd
}

func runComment(cmd *cobra.Command, f *cmdutil.Factory, id int, opts *commentOptions) error {
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
		return fmt.Errorf("pr comment currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := client.CommentPullRequest(ctx, projectKey, repoSlug, id, opts.Text); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Commented on pull request #%d\n", id); err != nil {
		return err
	}
	return nil
}

func runGit(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
