package pr

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/remote"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

const (
	// Standard timeouts for API calls.
	timeoutRead      = 15 * time.Second
	timeoutWrite     = 10 * time.Second
	prListTimeLayout = "2006-01-02 15:04"
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
		Long: `Create, list, review, merge, and manage pull requests on Bitbucket Data Center
and Bitbucket Cloud. Most subcommands work on both platforms; platform-specific
limitations are noted in each subcommand's help.`,
		Example: `  # List open pull requests in the current repository
  bkt pr list

  # View details of a specific pull request
  bkt pr view 42

  # Create a pull request from the current branch
  bkt pr create --title "Add user authentication"

  # Approve and merge a pull request
  bkt pr approve 42
  bkt pr merge 42`,
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
	cmd.AddCommand(newPublishCmd(f))

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
		Long: `List pull requests for a repository, filtered by state. On Data Center, the
project and repo are resolved from the active context (or --project/--repo).
On Cloud, the workspace and repo are used instead.

When --mine is set without a specific repository, the command lists pull
requests authored by the authenticated user across all repositories. On Data
Center this uses the dashboard API; on Cloud it queries the workspace.`,
		Example: `  # List open pull requests
  bkt pr list

  # List merged pull requests
  bkt pr list --state MERGED

  # List your own pull requests across all repositories
  bkt pr list --mine

  # List pull requests with a limit
  bkt pr list --limit 50 --state OPEN`,
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

		if opts.Mine {
			if host.Username == "" {
				return fmt.Errorf("--mine requires a username; bearer-only logins must re-authenticate with --username or use the dashboard endpoint (omit --project and --repo)")
			}
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
				created := formatPRListUnixMilli(pr.CreatedDate)
				if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\t%s\n", pr.ID, pr.State, pr.Title, created); err != nil {
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
				created := formatPRListRFC3339(pr.CreatedOn)
				if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\t%s\n", pr.ID, pr.State, pr.Title, created); err != nil {
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
			created := formatPRListUnixMilli(pr.CreatedDate)
			// Use ToRef.Repository (destination) to show where the PR merges into,
			// which is more useful for fork-based PRs than the source repo
			repoInfo := ""
			if pr.ToRef.Repository.Slug != "" {
				repoInfo = pr.ToRef.Repository.Slug
				if pr.ToRef.Repository.Project != nil && pr.ToRef.Repository.Project.Key != "" {
					repoInfo = pr.ToRef.Repository.Project.Key + "/" + repoInfo
				}
			}
			if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\t%s\n", pr.ID, pr.State, pr.Title, created); err != nil {
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
			created := formatPRListRFC3339(pr.CreatedOn)
			// Use Destination.Repository.Slug (where PR merges into) as primary source,
			// fall back to URL parsing for backwards compatibility
			repoInfo := pr.Destination.Repository.Slug
			if repoInfo == "" {
				repoInfo = extractRepoFromCloudPRLink(pr.Links.HTML.Href)
			}
			if _, err := fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\t%s\n", pr.ID, pr.State, pr.Title, created); err != nil {
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

func formatPRListUnixMilli(unixMilli int64) string {
	if unixMilli == 0 {
		return ""
	}

	return time.UnixMilli(unixMilli).Local().Format(prListTimeLayout)
}

func formatPRListRFC3339(value string) string {
	if value == "" {
		return ""
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return ""
	}

	return parsed.Local().Format(prListTimeLayout)
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
		Long: `Display the title, state, author, source and target branches, description, and
reviewers for a pull request. Use --web to open the pull request in your
default browser instead of printing to the terminal.

Works on both Data Center and Cloud.`,
		Example: `  # View pull request details
  bkt pr view 42

  # Open the pull request in a browser
  bkt pr view 42 --web

  # View a pull request in a different repository
  bkt pr view 10 --repo my-other-repo`,
		Args: cobra.ExactArgs(1),
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

type createResult struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
}

type createOptions struct {
	Project              string
	Workspace            string
	Repo                 string
	Title                string
	Source               string
	Target               string
	Description          string
	Body                 string
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

// mergeCloudReviewers combines explicit reviewer strings with default reviewer
// users for Bitbucket Cloud, deduplicating across username and UUID formats.
// A default reviewer is skipped if any explicit entry matches by username or
// by normalized UUID. Defaults without a username are identified by UUID.
func mergeCloudReviewers(explicit []string, defaults []bbcloud.User) []string {
	seen := make(map[string]bool, len(explicit)+len(defaults))
	var merged []string

	for _, r := range explicit {
		key := normalizeReviewerKey(r)
		if key != "" && !seen[key] {
			seen[key] = true
			merged = append(merged, r)
		}
	}

	for _, u := range defaults {
		uuidKey := ""
		if u.UUID != "" {
			uuidKey = bbcloud.NormalizeUUID(u.UUID)
		}
		if (u.Username != "" && seen[u.Username]) || (uuidKey != "" && seen[uuidKey]) {
			continue
		}
		id := u.Username
		if id == "" {
			id = uuidKey
		}
		if id == "" {
			continue
		}
		seen[id] = true
		if uuidKey != "" {
			seen[uuidKey] = true
		}
		if u.Username != "" {
			seen[u.Username] = true
		}
		merged = append(merged, id)
	}

	return merged
}

// normalizeReviewerKey returns a canonical key for overlap/dedup checks.
// Cloud UUIDs (bare or braced) are normalized to braced form; everything
// else is returned as-is.
func normalizeReviewerKey(s string) string {
	if bbcloud.LooksLikeUUID(s) {
		return bbcloud.NormalizeUUID(s)
	}
	return s
}

// reviewerOverlap returns the first entry that appears in both add and remove,
// or "" if there is no overlap. Cloud UUIDs are normalized before comparison
// so that bare and braced forms of the same UUID are treated as equal.
func reviewerOverlap(add, remove []string) string {
	addSet := make(map[string]bool, len(add))
	for _, r := range add {
		addSet[normalizeReviewerKey(r)] = true
	}
	for _, r := range remove {
		if addSet[normalizeReviewerKey(r)] {
			return r
		}
	}
	return ""
}

// editDCReviewers computes the new reviewer list for a DC pull request by
// adding and removing reviewers from the current list. Warnings for
// already-present additions or missing removals are written to w.
func editDCReviewers(w io.Writer, current []bbdc.PullRequestReviewer, add, remove []string) []bbdc.PullRequestReviewer {
	removeSet := make(map[string]bool, len(remove))
	for _, name := range remove {
		removeSet[name] = true
	}

	currentNames := make(map[string]bool, len(current))
	for _, r := range current {
		currentNames[r.User.Name] = true
	}

	for _, name := range remove {
		if !currentNames[name] {
			fmt.Fprintf(w, "warning: reviewer %q is not on this pull request\n", name)
		}
	}

	var result []bbdc.PullRequestReviewer
	for _, r := range current {
		if !removeSet[r.User.Name] {
			result = append(result, r)
		}
	}

	added := make(map[string]bool)
	for _, name := range add {
		if currentNames[name] || added[name] {
			fmt.Fprintf(w, "warning: reviewer %q is already on this pull request\n", name)
		} else {
			added[name] = true
			result = append(result, bbdc.PullRequestReviewer{User: bbdc.User{Name: name}})
		}
	}

	return result
}

// matchesCloudUser reports whether input (a username or UUID string) identifies
// the given Cloud user. UUIDs are compared after normalizing braces.
func matchesCloudUser(u bbcloud.User, input string) bool {
	if bbcloud.LooksLikeUUID(input) {
		return u.UUID == bbcloud.NormalizeUUID(input)
	}
	return u.Username == input
}

// editCloudReviewers computes the new reviewer list for a Cloud pull request.
// It returns a string slice suitable for UpdatePullRequestInput.Reviewers.
// Each existing reviewer is preserved by UUID; new reviewers use the caller's
// input format (username or UUID). Warnings are written to w.
//
// Returns an error if the same identity appears in both add and remove via
// different identifier formats (e.g. username in add, UUID in remove).
func editCloudReviewers(w io.Writer, current []bbcloud.User, add, remove []string) ([]string, error) {
	// Cross-identity overlap check: resolve add and remove against current
	// reviewers to detect the same person referenced by different formats.
	for _, a := range add {
		for _, r := range remove {
			// Exact string match is already caught by reviewerOverlap.
			// Here we check whether different strings resolve to the same user.
			for _, u := range current {
				if matchesCloudUser(u, a) && matchesCloudUser(u, r) {
					display := u.Username
					if display == "" {
						display = u.UUID
					}
					return nil, fmt.Errorf("reviewer %q (--reviewer %s, --remove-reviewer %s) cannot be in both flags", display, a, r)
				}
			}
		}
	}

	// Warn about removals that don't exist
	for _, r := range remove {
		found := false
		for _, u := range current {
			if matchesCloudUser(u, r) {
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(w, "warning: reviewer %q is not on this pull request\n", r)
		}
	}

	// Keep existing reviewers not being removed (use UUID for stability)
	result := make([]string, 0, len(current)+len(add))
	for _, u := range current {
		removed := false
		for _, r := range remove {
			if matchesCloudUser(u, r) {
				removed = true
				break
			}
		}
		if !removed {
			if u.UUID != "" {
				result = append(result, u.UUID)
			} else if u.Username != "" {
				result = append(result, u.Username)
			}
		}
	}

	// Add new reviewers
	added := make(map[string]bool)
	for _, r := range add {
		key := normalizeReviewerKey(r)
		alreadyOnPR := false
		for _, u := range current {
			if matchesCloudUser(u, r) {
				alreadyOnPR = true
				break
			}
		}
		if alreadyOnPR || added[key] {
			fmt.Fprintf(w, "warning: reviewer %q is already on this pull request\n", r)
		} else {
			added[key] = true
			result = append(result, r)
		}
	}

	return result, nil
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new pull request",
		Long: `Create a new pull request. The source branch defaults to the current git branch,
and the target branch defaults to the remote's default branch (e.g. main).
The title defaults to the first unique commit subject on the source branch.

Reviewers can be added with repeatable --reviewer flags.
--with-default-reviewers merges the repository's configured default reviewers
into the reviewer list. On Cloud, the current user is automatically excluded.

Draft pull requests are supported on Cloud (always) and on Data Center 8.18+
via the --draft flag.`,
		Example: `  # Create a pull request with auto-detected title
  bkt pr create

  # Create with an explicit title and description
  bkt pr create --title "Add OAuth2 support" --description "Implements RFC 6749"

  # Create with reviewers and close source branch on merge
  bkt pr create -t "Fix login bug" --reviewer alice --reviewer bob --close-source

  # Create a draft pull request
  bkt pr create --title "WIP: new feature" --draft`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --body and --description are mutually exclusive aliases
			if cmd.Flags().Changed("body") && cmd.Flags().Changed("description") {
				return fmt.Errorf("specify only one of --body or --description")
			}

			// --body is an alias for --description (for gh ergonomics)
			if cmd.Flags().Changed("body") {
				opts.Description = opts.Body
			}

			return runCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Title, "title", "", "Pull request title (defaults to the first unique commit subject)")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Pull request description")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Pull request description (alias for --description)")
	cmd.Flags().StringVar(&opts.Source, "source", "", "Source branch (defaults to the current branch)")
	cmd.Flags().StringVar(&opts.Target, "target", "", "Target branch (defaults to the remote's default branch)")
	cmd.Flags().StringSliceVar(&opts.Reviewers, "reviewer", nil, "Reviewer username or {UUID} (repeatable)")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", false, "Close source branch on merge")
	cmd.Flags().BoolVar(&opts.WithDefaultReviewers, "with-default-reviewers", false, "Add repository default reviewers")
	cmd.Flags().BoolVarP(&opts.Draft, "draft", "d", false, "Create pull request as a draft (DC 8.18+, Cloud always supported)")

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

	if err := applyCreateDefaults(cmd.Context(), opts, host); err != nil {
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
			defaultUsers, err := getDCDefaultReviewers(ctx, client, projectKey, repoSlug, opts.Source, opts.Target)
			if err != nil {
				return err
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

		result := createResult{ID: pr.ID, Title: pr.Title, URL: firstPRLinkDC(pr, "self")}
		return cmdutil.WriteOutput(cmd, ios.Out, result, func() error {
			kind := "pull request"
			if opts.Draft {
				kind = "draft pull request"
			}
			msg := fmt.Sprintf("✓ Created %s #%d: %s\n", kind, result.ID, result.Title)
			if result.URL != "" {
				msg += result.URL + "\n"
			}
			_, err := fmt.Fprint(ios.Out, msg)
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

		reviewers := opts.Reviewers
		if opts.WithDefaultReviewers {
			me, err := client.CurrentUser(ctx)
			if err != nil {
				return fmt.Errorf("resolving current user: %w", err)
			}
			defaultUsers, err := getCloudDefaultReviewers(ctx, client, workspace, repoSlug, *me)
			if err != nil {
				return err
			}
			reviewers = mergeCloudReviewers(reviewers, defaultUsers)
		}

		pr, err := client.CreatePullRequest(ctx, workspace, repoSlug, bbcloud.CreatePullRequestInput{
			Title:       opts.Title,
			Description: opts.Description,
			Source:      opts.Source,
			Destination: opts.Target,
			CloseSource: opts.CloseSource,
			Reviewers:   reviewers,
			Draft:       opts.Draft,
		})
		if err != nil {
			return err
		}

		result := createResult{ID: pr.ID, Title: pr.Title, URL: firstPRLinkCloud(pr)}
		return cmdutil.WriteOutput(cmd, ios.Out, result, func() error {
			kind := "pull request"
			if opts.Draft {
				kind = "draft pull request"
			}
			msg := fmt.Sprintf("✓ Created %s #%d: %s\n", kind, result.ID, result.Title)
			if result.URL != "" {
				msg += result.URL + "\n"
			}
			_, err := fmt.Fprint(ios.Out, msg)
			return err
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func applyCreateDefaults(ctx context.Context, opts *createOptions, host *config.Host) error {
	if strings.TrimSpace(opts.Source) == "" {
		source, err := currentGitBranch(ctx)
		if err != nil {
			return fmt.Errorf("could not determine default --source from git; pass --source explicitly: %w", err)
		}
		opts.Source = source
	}

	remoteName := ""
	if strings.TrimSpace(opts.Target) == "" {
		target, rn, err := defaultGitTargetBranch(ctx, host)
		if err != nil {
			return fmt.Errorf("could not determine default --target from git; pass --target explicitly: %w", err)
		}
		opts.Target = target
		remoteName = rn
	}

	if strings.TrimSpace(opts.Title) == "" {
		// When --target was explicit, remoteName is empty; resolve it now
		// so the merge-base can use the remote-tracking ref.
		if remoteName == "" {
			if rn, err := preferredGitRemote(ctx, host); err == nil {
				remoteName = rn
			}
		}
		title, err := defaultGitPRTitle(ctx, opts.Target, remoteName)
		if err != nil {
			return fmt.Errorf("could not determine default --title from git history; pass --title explicitly: %w", err)
		}
		opts.Title = title
	}

	return nil
}

func currentGitBranch(ctx context.Context) (string, error) {
	out, err := runGitOutput(ctx, "branch", "--show-current")
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(out)
	if branch == "" {
		return "", fmt.Errorf("git HEAD is detached")
	}

	return branch, nil
}

func defaultGitTargetBranch(ctx context.Context, host *config.Host) (branch string, remoteName string, err error) {
	remoteName, err = preferredGitRemote(ctx, host)
	if err != nil {
		return "", "", err
	}

	out, err := runGitOutput(ctx, "symbolic-ref", "--quiet", "--short", fmt.Sprintf("refs/remotes/%s/HEAD", remoteName))
	if err != nil {
		return "", "", err
	}

	ref := strings.TrimSpace(out)
	prefix := remoteName + "/"
	if !strings.HasPrefix(ref, prefix) {
		return "", "", fmt.Errorf("unexpected remote HEAD ref %q", ref)
	}

	branch = strings.TrimPrefix(ref, prefix)
	if branch == "" {
		return "", "", fmt.Errorf("remote HEAD does not point to a branch")
	}

	return branch, remoteName, nil
}

func defaultGitPRTitle(ctx context.Context, targetBranch, remoteName string) (string, error) {
	baseRef, err := resolveGitBaseRef(ctx, targetBranch, remoteName)
	if err != nil {
		return "", err
	}

	mergeBase, err := runGitOutput(ctx, "merge-base", "HEAD", baseRef)
	if err != nil {
		return "", err
	}

	out, err := runGitOutput(ctx, "log", "--reverse", "--format=%s", fmt.Sprintf("%s..HEAD", strings.TrimSpace(mergeBase)))
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(out, "\n") {
		title := strings.TrimSpace(line)
		if title != "" {
			return title, nil
		}
	}

	return "", fmt.Errorf("no commits found relative to %s", targetBranch)
}

func preferredGitRemote(ctx context.Context, host *config.Host) (string, error) {
	remotes, err := remote.ListRemotes(".")
	if err != nil {
		return "", err
	}
	if len(remotes) == 0 {
		return "", fmt.Errorf("no git remotes found")
	}

	// Build ordered name list: origin, upstream, then rest.
	names := orderedRemoteNames(remotes)

	// First pass: find a remote matching the active Bitbucket host.
	if host != nil {
		for _, name := range names {
			for _, rawURL := range remotes[name] {
				loc, locErr := remote.ParseLocator(rawURL)
				if locErr != nil {
					continue
				}
				if cmdutil.LocatorMatchesHost(host, loc) {
					return name, nil
				}
			}
		}
	}

	// Second pass: fall back in priority order.
	return names[0], nil
}

func orderedRemoteNames(remotes map[string][]string) []string {
	var names []string
	for _, preferred := range []string{"origin", "upstream"} {
		if _, ok := remotes[preferred]; ok {
			names = append(names, preferred)
		}
	}
	var rest []string
	for name := range remotes {
		if name != "origin" && name != "upstream" {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	return append(names, rest...)
}

func resolveGitBaseRef(ctx context.Context, targetBranch, remoteName string) (string, error) {
	targetBranch = strings.TrimSpace(targetBranch)
	if targetBranch == "" {
		return "", fmt.Errorf("target branch is empty")
	}

	var candidates []string
	if remoteName != "" {
		candidates = append(candidates, fmt.Sprintf("refs/remotes/%s/%s", remoteName, targetBranch))
	}
	candidates = append(candidates, fmt.Sprintf("refs/heads/%s", targetBranch), targetBranch)

	for _, candidate := range candidates {
		if _, err := runGitOutput(ctx, "rev-parse", "--verify", "--quiet", candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not resolve base branch %q", targetBranch)
}

type editOptions struct {
	Project              string
	Workspace            string
	Repo                 string
	ID                   int
	Title                string
	Description          string
	Body                 string
	Reviewers            []string
	RemoveReviewers      []string
	WithDefaultReviewers bool
}

func newEditCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{}
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a pull request",
		Long:  "Edit a pull request's title, description, and/or reviewers.",
		Example: `  # Update pull request title
  bkt pr edit 123 --title "New feature: user authentication"

  # Update pull request description
  bkt pr edit 123 --body "This PR adds OAuth2 support"

  # Update both title and description
  bkt pr edit 123 -t "Fix login bug" -b "Resolves issue with session timeout"

  # Add reviewers
  bkt pr edit 123 --reviewer alice --reviewer bob

  # Remove a reviewer
  bkt pr edit 123 --remove-reviewer alice

  # Add repository default reviewers
  bkt pr edit 123 --with-default-reviewers

  # Add and remove reviewers in one call
  bkt pr edit 123 --reviewer charlie --remove-reviewer alice`,
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

			// Same user cannot appear in both --reviewer and --remove-reviewer
			if overlap := reviewerOverlap(opts.Reviewers, opts.RemoveReviewers); overlap != "" {
				return fmt.Errorf("reviewer %q cannot be in both --reviewer and --remove-reviewer", overlap)
			}

			// Require at least one field to update
			hasFieldChange := cmd.Flags().Changed("title") || cmd.Flags().Changed("description") || cmd.Flags().Changed("body")
			hasReviewerChange := cmd.Flags().Changed("reviewer") || cmd.Flags().Changed("remove-reviewer") || opts.WithDefaultReviewers
			if !hasFieldChange && !hasReviewerChange {
				return fmt.Errorf("at least one of --title, --body, --description, --reviewer, --remove-reviewer, or --with-default-reviewers is required")
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
	cmd.Flags().StringSliceVar(&opts.Reviewers, "reviewer", nil, "Reviewer username or {UUID} to add (repeatable)")
	cmd.Flags().StringSliceVar(&opts.RemoveReviewers, "remove-reviewer", nil, "Reviewer username or {UUID} to remove (repeatable)")
	cmd.Flags().BoolVar(&opts.WithDefaultReviewers, "with-default-reviewers", false, "Add repository default reviewers")

	return cmd
}

func mergeDCPRReviewers(current []bbdc.PullRequestReviewer, defaults []bbdc.User) []bbdc.PullRequestReviewer {
	seen := make(map[string]bool, len(current)+len(defaults))
	result := make([]bbdc.PullRequestReviewer, 0, len(current)+len(defaults))

	for _, reviewer := range current {
		name := reviewer.User.Name
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, reviewer)
	}

	for _, user := range defaults {
		if user.Name == "" || seen[user.Name] {
			continue
		}
		seen[user.Name] = true
		result = append(result, bbdc.PullRequestReviewer{User: user})
	}

	return result
}

func mergeCloudPRReviewers(current, defaults []bbcloud.User) []bbcloud.User {
	seen := make(map[string]bool, len(current)+len(defaults))
	result := make([]bbcloud.User, 0, len(current)+len(defaults))

	addUser := func(user bbcloud.User) {
		keys := cloudUserKeys(user)
		if len(keys) == 0 {
			return
		}
		for _, key := range keys {
			if seen[key] {
				return
			}
		}
		for _, key := range keys {
			seen[key] = true
		}
		result = append(result, user)
	}

	for _, user := range current {
		addUser(user)
	}
	for _, user := range defaults {
		addUser(user)
	}

	return result
}

func cloudReviewerIDs(reviewers []bbcloud.User) []string {
	seen := make(map[string]bool, len(reviewers))
	ids := make([]string, 0, len(reviewers))
	for _, reviewer := range reviewers {
		keys := cloudUserKeys(reviewer)
		if len(keys) == 0 {
			continue
		}
		duplicate := false
		for _, key := range keys {
			if seen[key] {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		for _, key := range keys {
			seen[key] = true
		}
		if reviewer.UUID != "" {
			ids = append(ids, reviewer.UUID)
			continue
		}
		if reviewer.Username != "" {
			ids = append(ids, reviewer.Username)
		}
	}
	return ids
}

func getDCDefaultReviewers(ctx context.Context, client *bbdc.Client, projectKey, repoSlug, sourceRef, targetRef string) ([]bbdc.User, error) {
	defaultUsers, err := client.GetDefaultReviewers(ctx, projectKey, repoSlug, sourceRef, targetRef)
	if err != nil {
		return nil, fmt.Errorf("fetching default reviewers: %w", err)
	}
	return defaultUsers, nil
}

func cloudUserKeys(user bbcloud.User) []string {
	keys := make([]string, 0, 3)
	if user.UUID != "" {
		keys = append(keys, "uuid:"+bbcloud.NormalizeUUID(user.UUID))
	}
	if user.AccountID != "" {
		keys = append(keys, "account_id:"+user.AccountID)
	}
	if user.Username != "" {
		keys = append(keys, "username:"+user.Username)
	}
	return keys
}

func sameCloudUser(a, b bbcloud.User) bool {
	keys := cloudUserKeys(a)
	if len(keys) == 0 {
		return false
	}
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		seen[key] = true
	}
	for _, key := range cloudUserKeys(b) {
		if seen[key] {
			return true
		}
	}
	return false
}

func filterCloudUsers(users []bbcloud.User, excluded bbcloud.User) []bbcloud.User {
	filtered := make([]bbcloud.User, 0, len(users))
	for _, user := range users {
		if sameCloudUser(user, excluded) {
			continue
		}
		filtered = append(filtered, user)
	}
	return filtered
}

func getCloudDefaultReviewers(ctx context.Context, client *bbcloud.Client, workspace, repoSlug string, excluded bbcloud.User) ([]bbcloud.User, error) {
	defaultUsers, err := client.GetEffectiveDefaultReviewers(ctx, workspace, repoSlug)
	if err != nil {
		return nil, fmt.Errorf("fetching default reviewers: %w", err)
	}
	return filterCloudUsers(defaultUsers, excluded), nil
}

func runEdit(cmd *cobra.Command, f *cmdutil.Factory, opts *editOptions) error {
	ios, _ := f.Streams()

	override := cmdutil.FlagValue(cmd, "context")
	var err error
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

		newReviewers := pr.Reviewers
		if opts.WithDefaultReviewers {
			defaultUsers, err := getDCDefaultReviewers(ctx, client, projectKey, repoSlug, pr.FromRef.ID, pr.ToRef.ID)
			if err != nil {
				return err
			}
			newReviewers = mergeDCPRReviewers(newReviewers, defaultUsers)
		}
		if cmd.Flags().Changed("reviewer") || cmd.Flags().Changed("remove-reviewer") {
			newReviewers = editDCReviewers(ios.ErrOut, newReviewers, opts.Reviewers, opts.RemoveReviewers)
		}

		updatedPR, err := client.UpdatePullRequest(ctx, projectKey, repoSlug, opts.ID, pr.Version, bbdc.UpdatePROptions{
			Title:       newTitle,
			Description: newDesc,
			Reviewers:   newReviewers,
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
		if cmd.Flags().Changed("reviewer") || cmd.Flags().Changed("remove-reviewer") || opts.WithDefaultReviewers {
			pr, err := client.GetPullRequest(ctx, workspace, repoSlug, opts.ID)
			if err != nil {
				return err
			}

			currentReviewers := pr.Reviewers
			if opts.WithDefaultReviewers {
				defaultUsers, err := getCloudDefaultReviewers(ctx, client, workspace, repoSlug, bbcloud.User{
					UUID:      pr.Author.UUID,
					Username:  pr.Author.Username,
					AccountID: pr.Author.AccountID,
				})
				if err != nil {
					return err
				}
				currentReviewers = mergeCloudPRReviewers(currentReviewers, defaultUsers)
			}

			if cmd.Flags().Changed("reviewer") || cmd.Flags().Changed("remove-reviewer") {
				reviewers, err := editCloudReviewers(ios.ErrOut, currentReviewers, opts.Reviewers, opts.RemoveReviewers)
				if err != nil {
					return err
				}
				input.Reviewers = reviewers
			} else {
				input.Reviewers = cloudReviewerIDs(currentReviewers)
			}
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
		Long: `Fetch and check out the source branch of a pull request into a local branch.
By default the local branch is named pr/<id>.

On Data Center, the PR head is fetched via the refs/pull-requests/<id>/from
ref. On Cloud, the source branch name is resolved from the API and fetched
directly. For fork-based Cloud pull requests, a remote is automatically
added (or reused) for the fork repository.`,
		Example: `  # Check out pull request #42
  bkt pr checkout 42

  # Check out into a custom branch name
  bkt pr checkout 42 --branch feature-review

  # Check out from a specific remote
  bkt pr checkout 42 --remote upstream`,
		Args: cobra.ExactArgs(1),
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
		Long: `Display the full unified diff for a pull request, streamed through the
configured pager when available. Use --stat for a compact summary of changed
files, additions, and deletions instead of the full patch.

Works on both Data Center and Cloud. On Cloud, --stat also lists per-file
change counts.`,
		Example: `  # Show the full diff
  bkt pr diff 42

  # Show diff statistics only
  bkt pr diff 42 --stat`,
		Args: cobra.ExactArgs(1),
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
		Long: `Approve a pull request as the authenticated user. This adds your approval to
the pull request, which may satisfy merge checks that require reviewer
approvals.

Works on both Data Center and Cloud.`,
		Example: `  # Approve a pull request
  bkt pr approve 42

  # Approve a pull request in a specific repository
  bkt pr approve 42 --repo my-service`,
		Args: cobra.ExactArgs(1),
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

type publishOptions struct {
	Workspace string
	Project   string
	Repo      string
	ID        int
	Undo      bool
}

func newPublishCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &publishOptions{}
	cmd := &cobra.Command{
		Use:     "publish <id>",
		Aliases: []string{"ready"},
		Short:   "Mark a draft pull request as ready for review",
		Long: `Toggle a pull request between draft and published states. By default, this
command publishes a draft pull request so it is visible for review. Use --undo
to convert a published pull request back to draft.

Works on both Data Center (8.18+) and Cloud. If the pull request is already
in the desired state, the command prints a warning and exits without error.`,
		Example: `  # Publish a draft pull request
  bkt pr publish 42

  # Convert a pull request back to draft
  bkt pr publish 42 --undo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runPublish(cmd, f, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Undo, "undo", false, "Convert a pull request back to draft")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runPublish(cmd *cobra.Command, f *cmdutil.Factory, opts *publishOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	wantDraft := opts.Undo

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

		pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		if pr.Draft == wantDraft {
			if wantDraft {
				if _, err := fmt.Fprintf(ios.ErrOut, "! Pull request #%d is already a draft\n", opts.ID); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(ios.ErrOut, "! Pull request #%d is already published\n", opts.ID); err != nil {
					return err
				}
			}
			return nil
		}

		_, err = client.UpdatePullRequest(ctx, projectKey, repoSlug, opts.ID, pr.Version, bbdc.UpdatePROptions{
			Title:       pr.Title,
			Description: pr.Description,
			Draft:       &wantDraft,
			Reviewers:   pr.Reviewers,
			FromRef:     &pr.FromRef,
			ToRef:       &pr.ToRef,
		})
		if err != nil {
			return err
		}

		if wantDraft {
			if _, err := fmt.Fprintf(ios.Out, "✓ Unpublished pull request #%d\n", opts.ID); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(ios.Out, "✓ Published pull request #%d\n", opts.ID); err != nil {
				return err
			}
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

		pr, err := client.GetPullRequest(ctx, workspace, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		if pr.Draft == wantDraft {
			if wantDraft {
				if _, err := fmt.Fprintf(ios.ErrOut, "! Pull request #%d is already a draft\n", opts.ID); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(ios.ErrOut, "! Pull request #%d is already published\n", opts.ID); err != nil {
					return err
				}
			}
			return nil
		}

		_, err = client.UpdatePullRequest(ctx, workspace, repoSlug, opts.ID, bbcloud.UpdatePullRequestInput{
			Draft: &wantDraft,
		})
		if err != nil {
			return err
		}

		if wantDraft {
			if _, err := fmt.Fprintf(ios.Out, "✓ Unpublished pull request #%d\n", opts.ID); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(ios.Out, "✓ Published pull request #%d\n", opts.ID); err != nil {
				return err
			}
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
		Long: `Merge a pull request. The source branch is closed by default (use
--close-source=false to keep it). An optional merge strategy can be specified
(e.g. fast-forward, squash) and a custom merge commit message can be provided.

Works on both Data Center and Cloud. On Data Center, the current PR version
is used for optimistic locking.`,
		Example: `  # Merge a pull request
  bkt pr merge 42

  # Merge with a custom commit message
  bkt pr merge 42 --message "Release v1.2.0"

  # Merge using fast-forward strategy and keep source branch
  bkt pr merge 42 --strategy fast-forward --close-source=false`,
		Args: cobra.ExactArgs(1),
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
		Long: `Decline (close without merging) a pull request. On Data Center, the optional
--delete-source flag also deletes the source branch after declining. This flag
is not supported on Cloud.

Works on both Data Center and Cloud.`,
		Example: `  # Decline a pull request
  bkt pr decline 42

  # Decline and delete the source branch (Data Center only)
  bkt pr decline 42 --delete-source`,
		Args: cobra.ExactArgs(1),
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
		Long: `Reopen a previously declined pull request, returning it to the OPEN state.

Works on both Data Center and Cloud. On Data Center, the current PR version
is used for optimistic locking.`,
		Example: `  # Reopen a declined pull request
  bkt pr reopen 42`,
		Args: cobra.ExactArgs(1),
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
		Long: `Add a comment to a pull request. Comments can be general (activity-level),
threaded replies (via --parent), or inline on a specific file and line in the
diff (via --file with --from-line or --to-line). Use --pending to create a
draft review comment that is not visible until submitted.

Works on both Data Center and Cloud.`,
		Example: `  # Add a general comment
  bkt pr comment 42 --text "Looks good to me"

  # Reply to an existing comment thread
  bkt pr comment 42 --text "Fixed in the latest push" --parent 1001

  # Add an inline comment on a specific line in the new file
  bkt pr comment 42 --text "Nit: rename this variable" --file src/main.go --to-line 55

  # Add a pending (draft) inline comment
  bkt pr comment 42 --text "Consider error handling here" --file api.go --to-line 30 --pending`,
		Args: cobra.ExactArgs(1),
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
		Long: `Display CI/build statuses for the head commit of a pull request. Use --wait to
poll until all builds complete, with exponential backoff and jitter. The
--fail-fast flag exits on the first failure. Use --web to open the first
build's URL in your browser.

On Data Center, statuses are fetched from the commit build-status API using
the source branch's latest commit. On Cloud, the source commit hash from the
pull request is used.

Exit codes in --wait mode: 0 = all passed, 1 = a build failed,
8 = timed out with builds still pending.`,
		Example: `  # Show current build status
  bkt pr checks 42

  # Wait for all builds to finish
  bkt pr checks 42 --wait

  # Wait with fail-fast and a custom timeout
  bkt pr checks 42 --wait --fail-fast --timeout 10m

  # Open the first build URL in a browser
  bkt pr checks 42 --web`,
		Args: cobra.ExactArgs(1),
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
