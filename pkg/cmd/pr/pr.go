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

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

const (
	// Standard timeouts for API calls.
	timeoutRead  = 15 * time.Second
	timeoutWrite = 10 * time.Second
)

// Sentinel errors for checks command
var (
	ErrNoSourceCommit = errors.New("pull request has no source commit")
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
	cmd.AddCommand(newEditCmd(f))
	cmd.AddCommand(newCheckoutCmd(f))
	cmd.AddCommand(newDiffCmd(f))
	cmd.AddCommand(newApproveCmd(f))
	cmd.AddCommand(newMergeCmd(f))
	cmd.AddCommand(newDeclineCmd(f))
	cmd.AddCommand(newReopenCmd(f))
	cmd.AddCommand(newCommentCmd(f))
	cmd.AddCommand(newCommentsCmd(f))
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
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)

		// If no repo specified, use the dashboard endpoint (requires --mine)
		if repoSlug == "" {
			if !opts.Mine {
				return fmt.Errorf("--mine is required when not specifying a repository")
			}
			return runListDashboardDC(cmd, f, ios, host, opts)
		}

		if projectKey == "" {
			return fmt.Errorf("context must supply project; use --project if needed")
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
				author := strings.ToLower(cmdutil.FirstNonEmpty(pr.Author.User.Name, pr.Author.User.Slug))
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
				author := cmdutil.FirstNonEmpty(pr.Author.User.FullName, pr.Author.User.Name)
				created := time.UnixMilli(pr.CreatedDate).Format("2006-01-02 15:04")
				if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\t%s\n", pr.ID, pr.State, created, pr.Title); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(ios.Out, "    %s -> %s\tby %s\n", pr.FromRef.DisplayID, pr.ToRef.DisplayID, author); err != nil {
					return err
				}
			}
			return nil
		})

	case "cloud":
		workspace := cmdutil.FirstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)

		// If no repo specified, use the workspace endpoint (requires --mine)
		if repoSlug == "" {
			if !opts.Mine {
				return fmt.Errorf("--mine is required when not specifying a repository")
			}
			if workspace == "" {
				return fmt.Errorf("context must supply workspace; use --workspace if needed")
			}
			return runListWorkspaceCloud(cmd, f, ios, host, workspace, opts)
		}

		if workspace == "" {
			return fmt.Errorf("context must supply workspace; use --workspace if needed")
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
				author := cmdutil.FirstNonEmpty(pr.Author.DisplayName, pr.Author.Username)
				created, _ := time.Parse(time.RFC3339, pr.CreatedOn)
				createdStr := created.Format("2006-01-02 15:04")
				if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\t%s\n", pr.ID, pr.State, createdStr, pr.Title); err != nil {
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

// runListDashboardDC lists pull requests for the authenticated user across all repositories (Data Center).
func runListDashboardDC(cmd *cobra.Command, f *cmdutil.Factory, ios *iostreams.IOStreams, host *config.Host, opts *listOptions) error {
	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	prs, err := client.ListDashboardPullRequests(ctx, bbdc.DashboardPullRequestsOptions{
		State: opts.State,
		Role:  "AUTHOR",
		Limit: opts.Limit,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{
		"pull_requests": prs,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(prs) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No pull requests (%s).\n", strings.ToUpper(opts.State))
			return err
		}

		for _, pr := range prs {
			author := cmdutil.FirstNonEmpty(pr.Author.User.FullName, pr.Author.User.Name)
			created := time.UnixMilli(pr.CreatedDate).Format("2006-01-02 15:04")
			// Use ToRef.Repository (destination) to show where the PR merges into,
			// which is more useful for fork-based PRs than the source repo
			repoInfo := ""
			if pr.ToRef.Repository.Slug != "" {
				repoInfo = pr.ToRef.Repository.Slug
				if pr.ToRef.Repository.Project != nil && pr.ToRef.Repository.Project.Key != "" {
					repoInfo = pr.ToRef.Repository.Project.Key + "/" + repoInfo
				}
			}
			if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\t%s\n", pr.ID, pr.State, created, pr.Title); err != nil {
				return err
			}
			if repoInfo != "" {
				if _, err := fmt.Fprintf(ios.Out, "    %s\t%s -> %s\tby %s\n", repoInfo, pr.FromRef.DisplayID, pr.ToRef.DisplayID, author); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(ios.Out, "    %s -> %s\tby %s\n", pr.FromRef.DisplayID, pr.ToRef.DisplayID, author); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// runListWorkspaceCloud lists pull requests for the authenticated user across all repositories (Cloud).
func runListWorkspaceCloud(cmd *cobra.Command, f *cmdutil.Factory, ios *iostreams.IOStreams, host *config.Host, workspace string, opts *listOptions) error {
	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// Fetch the current user to get the actual Bitbucket username (not the email used for auth)
	currentUser, err := client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Determine username for API call. Username may be empty for newer Bitbucket
	// accounts, so fall back to AccountID, then configured host username.
	username := currentUser.Username
	if username == "" {
		username = currentUser.AccountID
	}
	if username == "" && host.Username != "" && !strings.Contains(host.Username, "@") {
		username = host.Username
	}
	if username == "" {
		return fmt.Errorf("could not determine username; Bitbucket Cloud account may lack username field")
	}

	prs, err := client.ListWorkspacePullRequests(ctx, workspace, username, bbcloud.WorkspacePullRequestsOptions{
		State: opts.State,
		Limit: opts.Limit,
	})
	if err != nil {
		return err
	}

	payload := map[string]any{
		"workspace":     workspace,
		"pull_requests": prs,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(prs) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No pull requests (%s).\n", strings.ToUpper(opts.State))
			return err
		}

		for _, pr := range prs {
			author := cmdutil.FirstNonEmpty(pr.Author.DisplayName, pr.Author.Username)
			created, _ := time.Parse(time.RFC3339, pr.CreatedOn)
			createdStr := created.Format("2006-01-02 15:04")
			// Use Destination.Repository.Slug (where PR merges into) as primary source,
			// fall back to URL parsing for backwards compatibility
			repoInfo := pr.Destination.Repository.Slug
			if repoInfo == "" {
				repoInfo = extractRepoFromCloudPRLink(pr.Links.HTML.Href)
			}
			if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\t%s\n", pr.ID, pr.State, createdStr, pr.Title); err != nil {
				return err
			}
			if repoInfo != "" {
				if _, err := fmt.Fprintf(ios.Out, "    %s\t%s -> %s\tby %s\n", repoInfo, pr.Source.Branch.Name, pr.Destination.Branch.Name, author); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(ios.Out, "    %s -> %s\tby %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name, author); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// extractRepoFromCloudPRLink extracts the repository slug from a Bitbucket Cloud PR URL.
// This is a fallback method; prefer using PullRequest.Destination.Repository.Slug directly.
// URL format: https://bitbucket.org/{workspace}/{repo}/pull-requests/{id}
func extractRepoFromCloudPRLink(href string) string {
	parts := strings.Split(href, "/")
	// Expected: ["https:", "", "bitbucket.org", "workspace", "repo", "pull-requests", "id"]
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
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
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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
			if _, err := fmt.Fprintf(ios.Out, "Author: %s\n", cmdutil.FirstNonEmpty(pr.Author.User.FullName, pr.Author.User.Name)); err != nil {
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
					if _, err := fmt.Fprintf(ios.Out, "  %s\n", cmdutil.FirstNonEmpty(reviewer.User.FullName, reviewer.User.Name)); err != nil {
						return err
					}
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
			if _, err := fmt.Fprintf(ios.Out, "Author: %s\n", cmdutil.FirstNonEmpty(pr.Author.DisplayName, pr.Author.Username)); err != nil {
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
	if kind == "self" {
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
	Project              string
	Workspace            string
	Repo                 string
	Title                string
	Source               string
	Target               string
	Description          string
	Reviewers            []string
	CloseSource          bool
	WithDefaultReviewers bool
	Draft                bool
}

// mergeReviewers combines explicit reviewer names with default reviewer users,
// deduplicating across both lists. The nameFunc extracts a username string
// from each default user value.
func mergeReviewers[T any](explicit []string, defaults []T, nameFunc func(T) string) []string {
	seen := make(map[string]bool, len(explicit)+len(defaults))
	var merged []string
	for _, r := range explicit {
		if r != "" && !seen[r] {
			seen[r] = true
			merged = append(merged, r)
		}
	}
	for _, u := range defaults {
		name := nameFunc(u)
		if name != "" && !seen[name] {
			seen[name] = true
			merged = append(merged, name)
		}
	}
	return merged
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
	cmd.Flags().StringSliceVar(&opts.Reviewers, "reviewer", nil, "Reviewer username or {UUID} (repeatable)")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", false, "Close source branch on merge")
	cmd.Flags().BoolVar(&opts.WithDefaultReviewers, "with-default-reviewers", false, "Add repository default reviewers (Data Center only)")
	cmd.Flags().BoolVarP(&opts.Draft, "draft", "d", false, "Create pull request as a draft (DC 8.18+, Cloud always supported)")

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
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		reviewers := opts.Reviewers
		if opts.WithDefaultReviewers {
			defaultUsers, err := client.GetDefaultReviewers(ctx, projectKey, repoSlug, opts.Source, opts.Target)
			if err != nil {
				return fmt.Errorf("fetching default reviewers: %w", err)
			}
			reviewers = mergeReviewers(reviewers, defaultUsers, func(u bbdc.User) string { return u.Name })
		}

		pr, err := client.CreatePullRequest(ctx, projectKey, repoSlug, bbdc.CreatePROptions{
			Title:        opts.Title,
			Description:  opts.Description,
			SourceBranch: opts.Source,
			TargetBranch: opts.Target,
			Reviewers:    reviewers,
			CloseSource:  opts.CloseSource,
			Draft:        opts.Draft,
		})
		if err != nil {
			return err
		}

		kind := "pull request"
		if opts.Draft {
			kind = "draft pull request"
		}
		if _, err := fmt.Fprintf(ios.Out, "✓ Created %s #%d\n", kind, pr.ID); err != nil {
			return err
		}
		return nil

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

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		if opts.WithDefaultReviewers {
			return fmt.Errorf("--with-default-reviewers is not yet supported for Bitbucket Cloud (see https://github.com/avivsinai/bitbucket-cli/issues/67)")
		}

		pr, err := client.CreatePullRequest(ctx, workspace, repoSlug, bbcloud.CreatePullRequestInput{
			Title:       opts.Title,
			Description: opts.Description,
			Source:      opts.Source,
			Destination: opts.Target,
			CloseSource: opts.CloseSource,
			Reviewers:   opts.Reviewers,
			Draft:       opts.Draft,
		})
		if err != nil {
			return err
		}

		kind := "pull request"
		if opts.Draft {
			kind = "draft pull request"
		}
		if _, err := fmt.Fprintf(ios.Out, "✓ Created %s #%d\n", kind, pr.ID); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type editOptions struct {
	Project     string
	Workspace   string
	Repo        string
	ID          int
	Title       string
	Description string
	Body        string
}

func newEditCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{}
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a pull request",
		Long:  "Edit a pull request's title and/or description.",
		Example: `  # Update pull request title
  bkt pr edit 123 --title "New feature: user authentication"

  # Update pull request description
  bkt pr edit 123 --body "This PR adds OAuth2 support"

  # Update both title and description
  bkt pr edit 123 -t "Fix login bug" -b "Resolves issue with session timeout"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id

			// --body and --description are mutually exclusive aliases
			if cmd.Flags().Changed("body") && cmd.Flags().Changed("description") {
				return fmt.Errorf("specify only one of --body or --description")
			}

			// --body is an alias for --description (for gh ergonomics)
			if cmd.Flags().Changed("body") {
				opts.Description = opts.Body
			}

			// Require at least one field to update
			if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("description") && !cmd.Flags().Changed("body") {
				return fmt.Errorf("at least one of --title, --body, or --description is required")
			}

			return runEdit(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override.")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud).")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override.")
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Set the new title.")
	cmd.Flags().StringVarP(&opts.Description, "description", "", "", "Set the new description.")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Set the new body (alias for --description).")

	return cmd
}

func runEdit(cmd *cobra.Command, f *cmdutil.Factory, opts *editOptions) error {
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
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		// Fetch current PR to get version (optimistic locking) and current values
		pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		// Compute new values: use flag value if changed, otherwise keep existing
		newTitle := pr.Title
		if cmd.Flags().Changed("title") {
			newTitle = opts.Title
		}
		newDesc := pr.Description
		if cmd.Flags().Changed("description") || cmd.Flags().Changed("body") {
			newDesc = opts.Description
		}

		updatedPR, err := client.UpdatePullRequest(ctx, projectKey, repoSlug, opts.ID, pr.Version, bbdc.UpdatePROptions{
			Title:       newTitle,
			Description: newDesc,
			Reviewers:   pr.Reviewers,
			FromRef:     &pr.FromRef,
			ToRef:       &pr.ToRef,
		})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":      projectKey,
			"repo":         repoSlug,
			"pull_request": updatedPR,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			_, err := fmt.Fprintf(ios.Out, "✓ Updated pull request #%d\n", updatedPR.ID)
			return err
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

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		// Build input with only changed fields
		input := bbcloud.UpdatePullRequestInput{}
		if cmd.Flags().Changed("title") {
			input.Title = &opts.Title
		}
		if cmd.Flags().Changed("description") || cmd.Flags().Changed("body") {
			input.Description = &opts.Description
		}

		updatedPR, err := client.UpdatePullRequest(ctx, workspace, repoSlug, opts.ID, input)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace":    workspace,
			"repo":         repoSlug,
			"pull_request": updatedPR,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			_, err := fmt.Fprintf(ios.Out, "✓ Updated pull request #%d\n", updatedPR.ID)
			return err
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type checkoutOptions struct {
	Workspace string
	Project   string
	Repo      string
	ID        int
	Branch    string
	Remote    string
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

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
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

	branchName := opts.Branch
	if branchName == "" {
		branchName = fmt.Sprintf("pr/%d", opts.ID)
	}

	switch host.Kind {
	case "dc":
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		ref := fmt.Sprintf("refs/pull-requests/%d/from", opts.ID)
		fetchArgs := []string{"fetch", opts.Remote, fmt.Sprintf("%s:%s", ref, branchName)}
		if err := runGit(cmd.Context(), fetchArgs...); err != nil {
			return err
		}

		return runGit(cmd.Context(), "checkout", branchName)

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

		pr, err := client.GetPullRequest(ctx, workspace, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		sourceBranch := pr.Source.Branch.Name
		if sourceBranch == "" {
			return fmt.Errorf("could not determine source branch for pull request #%d", opts.ID)
		}

		// Determine the correct remote to fetch from.
		remote := opts.Remote // default "origin"

		isFork := pr.Source.Repository.FullName != "" &&
			pr.Destination.Repository.FullName != "" &&
			pr.Source.Repository.FullName != pr.Destination.Repository.FullName

		var addedRemote bool
		if isFork {
			protocol := inferProtocol(cmd.Context(), opts.Remote)
			forkCloneURL := repoCloneURL(pr.Source.Repository, protocol)
			if forkCloneURL == "" {
				commitHash := pr.Source.Commit.Hash
				hint := ""
				if commitHash != "" {
					hint = fmt.Sprintf(
						"\nThe fork may have been deleted. You can manually fetch the PR's head commit:\n"+
							"  git fetch %s %s && git checkout -b %s FETCH_HEAD",
						opts.Remote, commitHash, branchName)
				}
				return fmt.Errorf(
					"cannot checkout fork-based PR #%d: source repository %q has no clone URL available%s",
					opts.ID, pr.Source.Repository.FullName, hint)
			}
			if err := cmdutil.ValidateGitPositionalArg(forkCloneURL, "fork clone URL"); err != nil {
				return err
			}

			// Reuse an existing remote if one already points to the fork.
			if existing := findRemoteByURL(cmd.Context(), forkCloneURL); existing != "" {
				remote = existing
			} else {
				// Derive remote name: fork/<owner>
				parts := strings.SplitN(pr.Source.Repository.FullName, "/", 2)
				owner := pr.Source.Repository.FullName // fallback if no /
				if len(parts) >= 2 {
					owner = parts[0]
				}
				remote = "fork/" + owner

				// If a remote with this name already exists (but different URL),
				// update its URL instead of failing.
				if existingURL, err := runGitOutput(cmd.Context(), "remote", "get-url", remote); err == nil && strings.TrimSpace(existingURL) != "" {
					if err := runGit(cmd.Context(), "remote", "set-url", remote, forkCloneURL); err != nil {
						return fmt.Errorf("failed to update remote %q URL for fork: %w", remote, err)
					}
				} else {
					if err := runGit(cmd.Context(), "remote", "add", remote, forkCloneURL); err != nil {
						return fmt.Errorf("failed to add remote %q for fork: %w", remote, err)
					}
					addedRemote = true
				}
			}
		}

		fetchArgs := []string{"fetch", remote, fmt.Sprintf("%s:%s", sourceBranch, branchName)}
		if err := runGit(cmd.Context(), fetchArgs...); err != nil {
			// Roll back a freshly added remote so re-runs don't fail
			// with "remote already exists" when the URL changes.
			if addedRemote {
				if rmErr := runGit(cmd.Context(), "remote", "remove", remote); rmErr != nil {
					return fmt.Errorf("fetch failed: %w (additionally, cleanup of remote %q failed: %v)", err, remote, rmErr)
				}
			}
			return err
		}

		return runGit(cmd.Context(), "checkout", branchName)

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type diffOptions struct {
	Workspace string
	Project   string
	Repo      string
	ID        int
	Stat      bool
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

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
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

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		if opts.Stat {
			result, err := client.PullRequestDiffStat(ctx, workspace, repoSlug, opts.ID)
			if err != nil {
				return err
			}
			payload := map[string]any{
				"workspace":    workspace,
				"repo":         repoSlug,
				"pull_request": opts.ID,
				"stats":        result,
			}
			return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
				// Compute max path length for alignment.
				maxLen := 0
				for _, e := range result.Entries {
					p := e.NewPath
					if p == "" {
						p = e.OldPath
					}
					if len(p) > maxLen {
						maxLen = len(p)
					}
				}
				if maxLen < 20 {
					maxLen = 20
				}

				for _, e := range result.Entries {
					prefix := "M"
					switch e.Status {
					case "added":
						prefix = "A"
					case "removed":
						prefix = "D"
					case "renamed":
						prefix = "R"
					default:
						if e.Status != "modified" && e.Status != "" {
							prefix = "?"
						}
					}
					filePath := e.NewPath
					if filePath == "" {
						filePath = e.OldPath
					}
					if _, err := fmt.Fprintf(ios.Out, "%s  %-*s +%d -%d\n", prefix, maxLen, filePath, e.LinesAdded, e.LinesRemoved); err != nil {
						return err
					}
				}
				_, err := fmt.Fprintf(ios.Out, "\n%d files changed, %d insertions(+), %d deletions(-)\n", len(result.Entries), result.TotalAdded, result.TotalRemoved)
				return err
			})
		}

		pager := f.PagerManager()
		if pager.Enabled() {
			w, err := pager.Start()
			if err == nil {
				defer func() { _ = pager.Stop() }()
				return client.PullRequestDiff(ctx, workspace, repoSlug, opts.ID, w)
			}
		}

		return client.PullRequestDiff(ctx, workspace, repoSlug, opts.ID, ios.Out)

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type approveOptions struct {
	Workspace string
	Project   string
	Repo      string
	ID        int
}

func newApproveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &approveOptions{}
	cmd := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runApprove(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runApprove(cmd *cobra.Command, f *cmdutil.Factory, opts *approveOptions) error {
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
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), timeoutWrite)
		defer cancel()

		if err := client.ApprovePullRequest(ctx, projectKey, repoSlug, opts.ID); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Approved pull request #%d\n", opts.ID); err != nil {
			return err
		}
		return nil

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

		ctx, cancel := context.WithTimeout(cmd.Context(), timeoutWrite)
		defer cancel()

		if err := client.ApprovePullRequest(ctx, workspace, repoSlug, opts.ID); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Approved pull request #%d\n", opts.ID); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type mergeOptions struct {
	Workspace   string
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

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
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

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		if err := client.MergePullRequest(ctx, workspace, repoSlug, id, opts.Message, opts.Strategy, opts.CloseSource); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Merged pull request #%d\n", id); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
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
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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
				// Use the source ref's own repository for deletion — it may
				// differ from the destination repo when the PR comes from a fork.
				srcProject := projectKey
				srcRepo := repoSlug
				if pr.FromRef.Repository.Project != nil && pr.FromRef.Repository.Project.Key != "" {
					srcProject = pr.FromRef.Repository.Project.Key
				}
				if pr.FromRef.Repository.Slug != "" {
					srcRepo = pr.FromRef.Repository.Slug
				}
				if err := client.DeleteBranch(ctx, srcProject, srcRepo, sourceBranch, false); err != nil {
					return fmt.Errorf("declined PR but failed to delete source branch %q: %w", sourceBranch, err)
				}
				if _, err := fmt.Fprintf(ios.Out, "Deleted source branch %s\n", sourceBranch); err != nil {
					return err
				}
			}
		}

		return nil

	case "cloud":
		if opts.DeleteSource {
			return fmt.Errorf("--delete-source is not supported for Bitbucket Cloud")
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

type reopenOptions struct {
	Project   string
	Workspace string
	Repo      string
}

func newReopenCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reopenOptions{}
	cmd := &cobra.Command{
		Use:   "reopen <id>",
		Short: "Reopen a declined pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runReopen(cmd, f, id, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runReopen(cmd *cobra.Command, f *cmdutil.Factory, id int, opts *reopenOptions) error {
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
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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
		ws := cmdutil.FirstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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
	Workspace string
	Project   string
	Repo      string
	Text      string
	ParentID  int
	File      string
	FromLine  int
	ToLine    int
	Pending   bool
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
			if cmd.Flags().Changed("parent") && opts.ParentID <= 0 {
				return fmt.Errorf("--parent must be a positive comment ID")
			}

			opts.File = strings.TrimSpace(opts.File)
			fileFlagChanged := cmd.Flags().Changed("file")
			hasFile := opts.File != ""
			hasFromLine := cmd.Flags().Changed("from-line")
			hasToLine := cmd.Flags().Changed("to-line")
			hasInline := hasFile || hasFromLine || hasToLine

			if fileFlagChanged && !hasFile {
				return fmt.Errorf("--file value must not be blank")
			}
			if (hasFromLine || hasToLine) && !hasFile {
				return fmt.Errorf("--file is required when --from-line or --to-line is specified")
			}
			if hasFile && !hasFromLine && !hasToLine {
				return fmt.Errorf("--file must be used with either --from-line or --to-line (file-level comments not yet supported)")
			}
			if hasFromLine && hasToLine {
				return fmt.Errorf("--from-line and --to-line are mutually exclusive")
			}
			if cmd.Flags().Changed("parent") && hasInline {
				return fmt.Errorf("--parent cannot be combined with inline comment flags (--file, --from-line, --to-line)")
			}
			if hasFromLine && opts.FromLine <= 0 {
				return fmt.Errorf("--from-line must be a positive integer")
			}
			if hasToLine && opts.ToLine <= 0 {
				return fmt.Errorf("--to-line must be a positive integer")
			}

			return runComment(cmd, f, id, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Text, "text", "", "Comment text")
	cmd.Flags().IntVar(&opts.ParentID, "parent", 0, "Parent comment ID for threaded replies")
	cmd.Flags().StringVar(&opts.File, "file", "", "File path in the diff (requires --from-line or --to-line)")
	cmd.Flags().IntVar(&opts.FromLine, "from-line", 0, "Line in the old file (removed/source side)")
	cmd.Flags().IntVar(&opts.ToLine, "to-line", 0, "Line in the new file (added/destination side)")
	cmd.Flags().BoolVar(&opts.Pending, "pending", false, "Create the comment as pending (draft review feedback)")
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

		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		if err := client.CommentPullRequest(ctx, projectKey, repoSlug, id, bbdc.CommentOptions{
			Text:     opts.Text,
			ParentID: opts.ParentID,
			File:     opts.File,
			FromLine: opts.FromLine,
			ToLine:   opts.ToLine,
			Pending:  opts.Pending,
		}); err != nil {
			return err
		}

		msg := "✓ Commented on pull request #%d\n"
		if opts.Pending {
			msg = "✓ Pending comment added to pull request #%d\n"
		}
		if _, err := fmt.Fprintf(ios.Out, msg, id); err != nil {
			return err
		}
		return nil

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

		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		if err := client.CommentPullRequest(ctx, workspace, repoSlug, id, bbcloud.CommentOptions{
			Text:     opts.Text,
			ParentID: opts.ParentID,
			File:     opts.File,
			FromLine: opts.FromLine,
			ToLine:   opts.ToLine,
			Pending:  opts.Pending,
		}); err != nil {
			return err
		}

		msg := "✓ Commented on pull request #%d\n"
		if opts.Pending {
			msg = "✓ Pending comment added to pull request #%d\n"
		}
		if _, err := fmt.Fprintf(ios.Out, msg, id); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
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

			// Validate interval values to prevent API hammering
			if opts.Wait {
				if opts.Interval <= 0 {
					return fmt.Errorf("--interval must be positive")
				}
				if opts.MaxInterval <= 0 {
					return fmt.Errorf("--max-interval must be positive")
				}
				if opts.MaxInterval < opts.Interval {
					return fmt.Errorf("--max-interval must be >= --interval")
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

	// Check if structured output is requested (--json/--yaml/--template/--jq)
	outputSettings, err := cmdutil.ResolveOutputSettings(cmd)
	if err != nil {
		return err
	}
	quietPoll := outputSettings.Format != "" || outputSettings.Template != "" || outputSettings.JQ != ""

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
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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
			quietPoll:    quietPoll,
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
		workspace := cmdutil.FirstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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
			quietPoll:    quietPoll,
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
	quietPoll    bool // suppress poll output for structured output (--json/--yaml)
}

// executeStatusCheck handles the common logic for both DC and Cloud:
// polling/fetching, error handling, output, and exit code.
func executeStatusCheck(r *checksResult) error {
	var statuses []types.CommitStatus
	var err error
	var timedOutWithPending bool

	if r.opts.Wait {
		// Use alternate screen buffer for cleaner watch output (skip for structured output)
		if !r.quietPoll {
			r.ios.StartAlternateScreenBuffer()
		}
		statuses, err = pollUntilComplete(r.ctx, r.ios, r.opts, r.fetchFunc, r.colorEnabled, r.commitSHA, r.quietPoll)
		if !r.quietPoll {
			r.ios.StopAlternateScreenBuffer()
		}

		// Handle cancellation gracefully
		if errors.Is(err, context.Canceled) {
			_, _ = fmt.Fprintln(r.ios.ErrOut, "\nOperation cancelled")
			return nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			_, _ = fmt.Fprintln(r.ios.ErrOut, "\nTimeout waiting for builds to complete")
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

	// Skip final print if we used wait mode without TTY (already printed during polling)
	// With TTY, alternate screen buffer means final print shows on main screen
	skipFinalPrint := r.opts.Wait && !r.ios.IsStdoutTTY()

	writeErr := cmdutil.WriteOutput(r.cmd, r.ios.Out, r.payload, func() error {
		if skipFinalPrint {
			return nil
		}
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
// When quietPoll is true, suppresses all output (for structured output like --json).
func pollUntilComplete(
	ctx context.Context,
	ios *iostreams.IOStreams,
	opts *checksOptions,
	fetch func() ([]types.CommitStatus, error),
	colorEnabled bool,
	commitSHA string,
	quietPoll bool,
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
			// Log error to stderr (doesn't corrupt structured output on stdout)
			_, _ = fmt.Fprintf(ios.ErrOut, "  ⚠ Error fetching status (attempt %d/%d): %v\n", consecutiveErrors, maxConsecutiveErrors, err)
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

		// Print current status (skip for structured output to avoid corrupting JSON/YAML)
		if !quietPoll {
			if iteration > 0 {
				ios.ClearScreen()
			}
			if err := printStatuses(ios, opts.ID, commitSHA, statuses, colorEnabled); err != nil {
				return nil, err
			}
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

		// Show waiting message (skip for structured output)
		if !quietPoll {
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
			_, _ = fmt.Fprintln(ios.Out, waitMsg)
		}

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
		name := cmdutil.FirstNonEmpty(s.Name, s.Key)
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

// runGitOutput runs a git command and returns its stdout as a string.
func runGitOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	return string(out), err
}

// repoCloneURL extracts the clone URL with the given protocol ("https" or "ssh")
// from a RepositoryRef. Returns "" if no matching link is found.
func repoCloneURL(repo bbcloud.RepositoryRef, protocol string) string {
	for _, link := range repo.Links.Clone {
		if strings.EqualFold(link.Name, protocol) {
			return link.Href
		}
	}
	return ""
}

// findRemoteByURL scans `git remote -v` output for a remote whose fetch URL
// matches the given cloneURL. Only fetch lines are considered.
// Returns the remote name, or "" if not found.
func findRemoteByURL(ctx context.Context, cloneURL string) string {
	out, err := runGitOutput(ctx, "remote", "-v")
	if err != nil {
		return ""
	}
	// Normalise for comparison: strip trailing ".git"
	norm := func(u string) string {
		return strings.TrimSuffix(strings.TrimSpace(u), ".git")
	}
	target := norm(cloneURL)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Only match fetch lines: name URL (fetch)
		if fields[2] != "(fetch)" {
			continue
		}
		if norm(fields[1]) == target {
			return fields[0]
		}
	}
	return ""
}

// inferProtocol examines the given remote's URL to determine whether
// SSH or HTTPS should be preferred for fork clone URLs.
// Falls back to "https" if the remote URL cannot be determined.
func inferProtocol(ctx context.Context, remoteName string) string {
	url, err := runGitOutput(ctx, "remote", "get-url", remoteName)
	if err != nil {
		return "https"
	}
	url = strings.TrimSpace(url)
	if strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://") {
		return "ssh"
	}
	return "https"
}
