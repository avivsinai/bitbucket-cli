package pr

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

// Sentinel errors for checks command
var (
	ErrNoSourceCommit = errors.New("pull request has no source commit")
	ErrBuildsFailed   = errors.New("one or more builds failed")
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
	cmd.AddCommand(newCommentCmd(f))
	cmd.AddCommand(newReviewerGroupCmd(f))
	cmd.AddCommand(newAutoMergeCmd(f))
	cmd.AddCommand(newTaskCmd(f))
	cmd.AddCommand(newReactionCmd(f))
	cmd.AddCommand(newSuggestionCmd(f))
	cmd.AddCommand(newChecksCmd(f))

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

type checksOptions struct {
	Project     string
	Workspace   string
	Repo        string
	ID          int
	Web         bool
	Wait        bool
	FailFast    bool
	Interval    time.Duration
	MaxInterval time.Duration
	Timeout     time.Duration
}

func newChecksCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &checksOptions{}
	cmd := &cobra.Command{
		Use:     "checks <id>",
		Aliases: []string{"builds"},
		Short:   "Show build/CI status for a pull request",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id

			// Validate flag combinations: polling flags require --wait
			if !opts.Wait {
				if cmd.Flags().Changed("interval") {
					return fmt.Errorf("--interval requires --wait")
				}
				if cmd.Flags().Changed("max-interval") {
					return fmt.Errorf("--max-interval requires --wait")
				}
				if cmd.Flags().Changed("timeout") {
					return fmt.Errorf("--timeout requires --wait")
				}
				if opts.FailFast {
					return fmt.Errorf("--fail-fast requires --wait")
				}
			}

			return runChecks(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.Web, "web", false, "Open the build URL in your browser (first build)")
	cmd.Flags().BoolVar(&opts.Wait, "wait", false, "Wait for all builds to complete")
	cmd.Flags().BoolVar(&opts.FailFast, "fail-fast", false, "Exit immediately when a check fails (requires --wait)")
	cmd.Flags().DurationVar(&opts.Interval, "interval", 10*time.Second, "Initial polling interval when using --wait")
	cmd.Flags().DurationVar(&opts.MaxInterval, "max-interval", 2*time.Minute, "Maximum polling interval (backoff cap)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 30*time.Minute, "Maximum time to wait for builds (0 for no timeout)")

	return cmd
}

func runChecks(cmd *cobra.Command, f *cmdutil.Factory, opts *checksOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	colorEnabled := ios.ColorEnabled()

	// Set up context with signal handling for graceful cancellation
	ctx := cmd.Context()
	if opts.Wait {
		var stop context.CancelFunc
		ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Apply timeout if specified
		if opts.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
		}
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

		fetchCtx, fetchCancel := context.WithTimeout(ctx, 15*time.Second)
		defer fetchCancel()

		pr, err := client.GetPullRequest(fetchCtx, projectKey, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		commitSHA := pr.FromRef.LatestCommit
		if commitSHA == "" {
			return ErrNoSourceCommit
		}

		return executeStatusCheck(&checksResult{
			ctx:          ctx,
			ios:          ios,
			cmd:          cmd,
			opts:         opts,
			colorEnabled: colorEnabled,
			commitSHA:    commitSHA,
			browserOpen:  f.BrowserOpener().Open,
			payload: map[string]any{
				"project":      projectKey,
				"repo":         repoSlug,
				"pull_request": opts.ID,
				"commit":       commitSHA,
			},
			fetchFunc: func() ([]types.CommitStatus, error) {
				statusCtx, statusCancel := context.WithTimeout(ctx, 15*time.Second)
				defer statusCancel()
				return client.CommitStatuses(statusCtx, commitSHA)
			},
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

		fetchCtx, fetchCancel := context.WithTimeout(ctx, 15*time.Second)
		defer fetchCancel()

		pr, err := client.GetPullRequest(fetchCtx, workspace, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		commitSHA := pr.Source.Commit.Hash
		if commitSHA == "" {
			return ErrNoSourceCommit
		}

		return executeStatusCheck(&checksResult{
			ctx:          ctx,
			ios:          ios,
			cmd:          cmd,
			opts:         opts,
			colorEnabled: colorEnabled,
			commitSHA:    commitSHA,
			browserOpen:  f.BrowserOpener().Open,
			payload: map[string]any{
				"workspace":    workspace,
				"repo":         repoSlug,
				"pull_request": opts.ID,
				"commit":       commitSHA,
			},
			fetchFunc: func() ([]types.CommitStatus, error) {
				statusCtx, statusCancel := context.WithTimeout(ctx, 15*time.Second)
				defer statusCancel()
				return client.CommitStatuses(statusCtx, workspace, repoSlug, commitSHA)
			},
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

// checksResult holds the parameters for executing status checks after the fetch function is set up
type checksResult struct {
	ctx          context.Context
	ios          *iostreams.IOStreams
	cmd          *cobra.Command
	opts         *checksOptions
	fetchFunc    func() ([]types.CommitStatus, error)
	colorEnabled bool
	commitSHA    string
	payload      map[string]any
	browserOpen  func(string) error
}

// executeStatusCheck handles the common logic for both DC and Cloud:
// polling/fetching, error handling, output, and exit code.
func executeStatusCheck(r *checksResult) error {
	var statuses []types.CommitStatus
	var err error
	var timedOutWithPending bool

	if r.opts.Wait {
		// Use alternate screen buffer for cleaner watch output
		r.ios.StartAlternateScreenBuffer()
		statuses, err = pollUntilComplete(r.ctx, r.ios, r.opts, r.fetchFunc, r.colorEnabled, r.commitSHA)
		r.ios.StopAlternateScreenBuffer()

		// Handle cancellation gracefully
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(r.ios.ErrOut, "\nOperation cancelled")
			return nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Fprintln(r.ios.ErrOut, "\nTimeout waiting for builds to complete")
			// Check if any builds are still pending
			timedOutWithPending = !allBuildsComplete(statuses)
		}
	} else {
		statuses, err = r.fetchFunc()
	}
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	r.payload["statuses"] = statuses

	if r.opts.Web && len(statuses) > 0 {
		if link := statuses[0].URL; link != "" {
			if err := r.browserOpen(link); err != nil {
				return fmt.Errorf("open browser: %w", err)
			}
		}
	}

	writeErr := cmdutil.WriteOutput(r.cmd, r.ios.Out, r.payload, func() error {
		return printStatuses(r.ios, r.opts.ID, r.commitSHA, statuses, r.colorEnabled)
	})
	if writeErr != nil {
		return writeErr
	}

	// Return appropriate exit code based on final state
	if r.opts.Wait {
		// Timeout with pending checks: exit code 8
		if timedOutWithPending {
			return cmdutil.ErrPending
		}
		// Any build failed: exit code 1 (silent - details already visible)
		if anyBuildFailed(statuses) {
			return cmdutil.ErrSilent
		}
	}
	return nil
}

// pollUntilComplete polls for build statuses until all are complete or context is cancelled.
// Uses exponential backoff with jitter to avoid overwhelming the API.
func pollUntilComplete(
	ctx context.Context,
	ios *iostreams.IOStreams,
	opts *checksOptions,
	fetch func() ([]types.CommitStatus, error),
	colorEnabled bool,
	commitSHA string,
) ([]types.CommitStatus, error) {
	iteration := 0
	consecutiveErrors := 0
	const maxConsecutiveErrors = 3

	for {
		statuses, err := fetch()
		if err != nil {
			consecutiveErrors++
			// After multiple consecutive errors, back off more aggressively
			if consecutiveErrors >= maxConsecutiveErrors {
				return nil, fmt.Errorf("fetch failed after %d attempts: %w", consecutiveErrors, err)
			}
			// Log error and continue with extended backoff
			fmt.Fprintf(ios.ErrOut, "  ⚠ Error fetching status (attempt %d/%d): %v\n", consecutiveErrors, maxConsecutiveErrors, err)
			// Use iteration + consecutiveErrors to back off faster on errors
			errorBackoff := calculatePollInterval(opts.Interval, opts.MaxInterval, iteration+consecutiveErrors)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(errorBackoff):
				continue
			}
		}
		consecutiveErrors = 0 // Reset on success

		// Print current status (clear screen on updates for cleaner output)
		if iteration > 0 {
			ios.ClearScreen()
		}
		if err := printStatuses(ios, opts.ID, commitSHA, statuses, colorEnabled); err != nil {
			return nil, err
		}

		// On first iteration, if no builds exist, exit immediately (don't poll forever)
		if iteration == 0 && len(statuses) == 0 {
			return statuses, nil
		}

		if allBuildsComplete(statuses) {
			return statuses, nil
		}

		// Exit early on first failure if --fail-fast is set
		if opts.FailFast && anyBuildFailed(statuses) {
			return statuses, nil
		}

		// Calculate next polling interval with exponential backoff and jitter
		nextInterval := calculatePollInterval(opts.Interval, opts.MaxInterval, iteration)

		// Show waiting message with current interval
		var waitMsg string
		if len(statuses) == 0 {
			// No builds found yet - explain we're waiting for them to appear
			waitMsg = fmt.Sprintf("\n  Waiting for builds to appear... (next poll in %s, Ctrl-C to cancel)", nextInterval.Round(time.Second))
		} else {
			inProgress := 0
			for _, s := range statuses {
				if !isTerminalState(s.State) {
					inProgress++
				}
			}
			waitMsg = fmt.Sprintf("\n  Waiting for %d build(s)... (next poll in %s, Ctrl-C to cancel)", inProgress, nextInterval.Round(time.Second))
		}
		fmt.Fprintln(ios.Out, waitMsg)

		iteration++

		select {
		case <-ctx.Done():
			return statuses, ctx.Err()
		case <-time.After(nextInterval):
			continue
		}
	}
}

// printStatuses prints build statuses with optional color coding
func printStatuses(ios *iostreams.IOStreams, prID int, commitSHA string, statuses []types.CommitStatus, colorEnabled bool) error {
	if _, err := fmt.Fprintf(ios.Out, "Build Status for PR #%d (commit %s):\n", prID, commitSHA[:min(12, len(commitSHA))]); err != nil {
		return err
	}

	if len(statuses) == 0 {
		_, err := fmt.Fprintln(ios.Out, "  No builds found.")
		return err
	}

	for _, s := range statuses {
		name := firstNonEmpty(s.Name, s.Key)
		icon := stateIcon(s.State)
		colorPrefix, colorSuffix := stateColor(s.State, colorEnabled)
		if _, err := fmt.Fprintf(ios.Out, "  %s%s %s: %s%s\n", colorPrefix, icon, name, s.State, colorSuffix); err != nil {
			return err
		}
		if s.URL != "" {
			if _, err := fmt.Fprintf(ios.Out, "      %s\n", s.URL); err != nil {
				return err
			}
		}
	}
	return nil
}

func stateIcon(state string) string {
	switch strings.ToUpper(state) {
	case "SUCCESSFUL", "SUCCESS":
		return "✓"
	case "FAILED", "FAILURE":
		return "✗"
	case "INPROGRESS", "IN_PROGRESS", "PENDING":
		return "○"
	case "STOPPED":
		return "■"
	case "CANCELLED":
		return "⊘"
	default:
		return "?"
	}
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

func stateColor(state string, colorEnabled bool) (prefix, suffix string) {
	if !colorEnabled {
		return "", ""
	}
	switch strings.ToUpper(state) {
	case "SUCCESSFUL", "SUCCESS":
		return colorGreen, colorReset
	case "FAILED", "FAILURE":
		return colorRed, colorReset
	case "INPROGRESS", "IN_PROGRESS", "PENDING", "CANCELLED", "STOPPED":
		return colorYellow, colorReset
	default:
		return "", ""
	}
}

// isTerminalState returns true if the build state is final (not in progress)
func isTerminalState(state string) bool {
	switch strings.ToUpper(state) {
	case "SUCCESSFUL", "SUCCESS", "FAILED", "FAILURE", "STOPPED", "CANCELLED":
		return true
	default:
		return false
	}
}

// allBuildsComplete returns true if all statuses are in a terminal state
func allBuildsComplete(statuses []types.CommitStatus) bool {
	if len(statuses) == 0 {
		return false // No builds means we should keep waiting
	}
	for _, s := range statuses {
		if !isTerminalState(s.State) {
			return false
		}
	}
	return true
}

// anyBuildFailed returns true if any build has failed
func anyBuildFailed(statuses []types.CommitStatus) bool {
	for _, s := range statuses {
		switch strings.ToUpper(s.State) {
		case "FAILED", "FAILURE":
			return true
		}
	}
	return false
}

// backoffMultiplier is the factor by which the polling interval increases each iteration
const backoffMultiplier = 1.5

// jitterFraction is the maximum random adjustment (±15%) applied to intervals
const jitterFraction = 0.15

// calculatePollInterval computes the next polling interval using exponential backoff with jitter.
// The formula is: min(baseInterval * multiplier^iteration, maxInterval) ± jitter
func calculatePollInterval(baseInterval, maxInterval time.Duration, iteration int) time.Duration {
	if iteration <= 0 {
		return addJitter(baseInterval)
	}

	// Calculate exponential backoff: base * 1.5^iteration
	interval := float64(baseInterval)
	for i := 0; i < iteration; i++ {
		interval *= backoffMultiplier
		if interval >= float64(maxInterval) {
			interval = float64(maxInterval)
			break
		}
	}

	// Cap at max interval
	if interval > float64(maxInterval) {
		interval = float64(maxInterval)
	}

	return addJitter(time.Duration(interval))
}

// addJitter applies ±15% random jitter to a duration to prevent thundering herd.
// Uses crypto/rand for better randomness distribution.
func addJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}

	// Calculate jitter range: ±15% of the duration
	jitterRange := int64(float64(d) * jitterFraction * 2) // Total range is 2x the fraction
	if jitterRange <= 0 {
		return d
	}

	// Generate random value in range [0, jitterRange)
	n, err := rand.Int(rand.Reader, big.NewInt(jitterRange))
	if err != nil {
		// Fallback to no jitter on error
		return d
	}

	// Apply jitter: subtract half the range, then add random value
	// This gives us a value in [-jitterFraction, +jitterFraction]
	jitter := n.Int64() - (jitterRange / 2)
	result := time.Duration(int64(d) + jitter)

	// Ensure we don't go below 1 second minimum
	if result < time.Second {
		result = time.Second
	}

	return result
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
