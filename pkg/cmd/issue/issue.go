package issue

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdIssue wires issue subcommands.
func NewCmdIssue(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Work with Bitbucket Cloud issues",
		Long: `Create and manage issues in Bitbucket Cloud repositories.

Note: The issue tracker is only available for Bitbucket Cloud. Bitbucket Data Center
uses Jira for issue tracking.`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newEditCmd(f))
	cmd.AddCommand(newCloseCmd(f))
	cmd.AddCommand(newReopenCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newCommentCmd(f))
	cmd.AddCommand(newStatusCmd(f))

	return cmd
}

// --- List Command ---

type listOptions struct {
	Workspace string
	Repo      string
	State     string
	Kind      string
	Priority  string
	Assignee  string
	Milestone string
	Limit     int
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{
		Limit: 30,
		State: "open",
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List issues in a repository",
		Example: `  # List all open issues
  bkt issue list

  # List bugs with major priority
  bkt issue list --kind bug --priority major

  # List issues assigned to a user
  bkt issue list --assignee {uuid}`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")
	cmd.Flags().StringVarP(&opts.State, "state", "s", opts.State, "Filter by state (new, open, resolved, on hold, invalid, duplicate, wontfix, closed, all); defaults to open")
	cmd.Flags().StringVarP(&opts.Kind, "kind", "k", "", "Filter by kind (bug, enhancement, proposal, task)")
	cmd.Flags().StringVarP(&opts.Priority, "priority", "p", "", "Filter by priority (trivial, minor, major, critical, blocker)")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Filter by assignee (UUID, e.g., {abc-123})")
	cmd.Flags().StringVar(&opts.Milestone, "milestone", "", "Filter by milestone")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", opts.Limit, "Maximum issues to display")

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

	if host.Kind != "cloud" {
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	issues, err := client.ListIssues(ctx, workspace, repoSlug, bbcloud.IssueListOptions{
		State:     opts.State,
		Kind:      opts.Kind,
		Priority:  opts.Priority,
		Assignee:  opts.Assignee,
		Milestone: opts.Milestone,
		Limit:     opts.Limit,
	})
	if err != nil {
		return err
	}

	type issueSummary struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		State     string `json:"state"`
		Kind      string `json:"kind"`
		Priority  string `json:"priority"`
		Assignee  string `json:"assignee,omitempty"`
		CreatedOn string `json:"created_on"`
		URL       string `json:"url,omitempty"`
	}

	var summaries []issueSummary
	for _, issue := range issues {
		s := issueSummary{
			ID:        issue.ID,
			Title:     issue.Title,
			State:     issue.State,
			Kind:      issue.Kind,
			Priority:  issue.Priority,
			CreatedOn: issue.CreatedOn,
			URL:       issue.Links.HTML.Href,
		}
		if issue.Assignee != nil {
			s.Assignee = issue.Assignee.DisplayName
		}
		summaries = append(summaries, s)
	}

	payload := struct {
		Workspace  string         `json:"workspace"`
		Repository string         `json:"repository"`
		Issues     []issueSummary `json:"issues"`
	}{
		Workspace:  workspace,
		Repository: repoSlug,
		Issues:     summaries,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(summaries) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No issues found in %s/%s.\n", workspace, repoSlug)
			return err
		}

		for _, s := range summaries {
			assignee := ""
			if s.Assignee != "" {
				assignee = fmt.Sprintf(" (@%s)", s.Assignee)
			}
			if _, err := fmt.Fprintf(ios.Out, "#%d\t%s\t[%s/%s]%s\n",
				s.ID, s.Title, s.State, s.Priority, assignee); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- View Command ---

type viewOptions struct {
	Workspace string
	Repo      string
	Web       bool
	Comments  bool
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <issue-id>",
		Short: "Display details for an issue",
		Example: `  # View issue #42
  bkt issue view 42

  # View issue with comments
  bkt issue view 42 --comments

  # Output as JSON
  bkt issue view 42 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runView(cmd, f, opts, issueID)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open in browser")
	cmd.Flags().BoolVarP(&opts.Comments, "comments", "c", false, "Show comments")

	return cmd
}

func runView(cmd *cobra.Command, f *cmdutil.Factory, opts *viewOptions, issueID int) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	issue, err := client.GetIssue(ctx, workspace, repoSlug, issueID)
	if err != nil {
		return err
	}

	if opts.Web && issue.Links.HTML.Href != "" {
		browser := f.BrowserOpener()
		return browser.Open(issue.Links.HTML.Href)
	}

	type commentSummary struct {
		ID        int    `json:"id"`
		Author    string `json:"author"`
		Body      string `json:"body"`
		CreatedOn string `json:"created_on"`
	}

	type issueDetails struct {
		ID        int              `json:"id"`
		Title     string           `json:"title"`
		State     string           `json:"state"`
		Kind      string           `json:"kind"`
		Priority  string           `json:"priority"`
		Reporter  string           `json:"reporter,omitempty"`
		Assignee  string           `json:"assignee,omitempty"`
		Milestone string           `json:"milestone,omitempty"`
		Component string           `json:"component,omitempty"`
		Body      string           `json:"body,omitempty"`
		URL       string           `json:"url,omitempty"`
		CreatedOn string           `json:"created_on"`
		UpdatedOn string           `json:"updated_on,omitempty"`
		Comments  []commentSummary `json:"comments,omitempty"`
	}

	details := issueDetails{
		ID:        issue.ID,
		Title:     issue.Title,
		State:     issue.State,
		Kind:      issue.Kind,
		Priority:  issue.Priority,
		Body:      issue.Content.Raw,
		URL:       issue.Links.HTML.Href,
		CreatedOn: issue.CreatedOn,
		UpdatedOn: issue.UpdatedOn,
	}
	if issue.Reporter != nil {
		details.Reporter = issue.Reporter.DisplayName
	}
	if issue.Assignee != nil {
		details.Assignee = issue.Assignee.DisplayName
	}
	if issue.Milestone != nil {
		details.Milestone = issue.Milestone.Name
	}
	if issue.Component != nil {
		details.Component = issue.Component.Name
	}

	if opts.Comments {
		comments, err := client.ListIssueComments(ctx, workspace, repoSlug, issueID, 50)
		if err != nil {
			return err
		}
		for _, c := range comments {
			cs := commentSummary{
				ID:        c.ID,
				Body:      c.Content.Raw,
				CreatedOn: c.CreatedOn,
			}
			if c.User != nil {
				cs.Author = c.User.DisplayName
			}
			details.Comments = append(details.Comments, cs)
		}
	}

	return cmdutil.WriteOutput(cmd, ios.Out, details, func() error {
		if _, err := fmt.Fprintf(ios.Out, "#%d: %s\n", details.ID, details.Title); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(ios.Out, "State: %s | Kind: %s | Priority: %s\n",
			details.State, details.Kind, details.Priority); err != nil {
			return err
		}
		if details.Reporter != "" {
			if _, err := fmt.Fprintf(ios.Out, "Reporter: %s\n", details.Reporter); err != nil {
				return err
			}
		}
		if details.Assignee != "" {
			if _, err := fmt.Fprintf(ios.Out, "Assignee: %s\n", details.Assignee); err != nil {
				return err
			}
		}
		if details.Milestone != "" {
			if _, err := fmt.Fprintf(ios.Out, "Milestone: %s\n", details.Milestone); err != nil {
				return err
			}
		}
		if details.Component != "" {
			if _, err := fmt.Fprintf(ios.Out, "Component: %s\n", details.Component); err != nil {
				return err
			}
		}
		if details.URL != "" {
			if _, err := fmt.Fprintf(ios.Out, "URL: %s\n", details.URL); err != nil {
				return err
			}
		}
		if details.Body != "" {
			if _, err := fmt.Fprintf(ios.Out, "\n%s\n", details.Body); err != nil {
				return err
			}
		}

		if len(details.Comments) > 0 {
			if _, err := fmt.Fprintf(ios.Out, "\n--- Comments (%d) ---\n", len(details.Comments)); err != nil {
				return err
			}
			for _, c := range details.Comments {
				if _, err := fmt.Fprintf(ios.Out, "\n@%s (%s):\n%s\n",
					c.Author, c.CreatedOn, c.Body); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// --- Create Command ---

type createOptions struct {
	Workspace string
	Repo      string
	Title     string
	Body      string
	Kind      string
	Priority  string
	Assignee  string
	Milestone string
	Component string
	Version   string
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{
		Kind: "bug",
	}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Example: `  # Create a bug
  bkt issue create -t "Login button broken" -b "The login button does not respond"

  # Create an enhancement
  bkt issue create -t "Add dark mode" -k enhancement -p minor

  # Create with assignee (use UUID from user profile)
  bkt issue create -t "Fix memory leak" -a "{abc-123-def}"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Issue title (required)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Issue body/description")
	cmd.Flags().StringVarP(&opts.Kind, "kind", "k", opts.Kind, "Issue kind (bug, enhancement, proposal, task)")
	cmd.Flags().StringVarP(&opts.Priority, "priority", "p", "", "Priority (trivial, minor, major, critical, blocker)")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Assignee UUID (e.g., {abc-123})")
	cmd.Flags().StringVar(&opts.Milestone, "milestone", "", "Milestone name")
	cmd.Flags().StringVar(&opts.Component, "component", "", "Component name")
	cmd.Flags().StringVar(&opts.Version, "version", "", "Version name")

	_ = cmd.MarkFlagRequired("title")

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

	if host.Kind != "cloud" {
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	if strings.TrimSpace(opts.Title) == "" {
		return fmt.Errorf("title is required; use --title or -t")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	issue, err := client.CreateIssue(ctx, workspace, repoSlug, bbcloud.CreateIssueInput{
		Title:     opts.Title,
		Content:   opts.Body,
		Kind:      opts.Kind,
		Priority:  opts.Priority,
		Assignee:  opts.Assignee,
		Milestone: opts.Milestone,
		Component: opts.Component,
		Version:   opts.Version,
	})
	if err != nil {
		return err
	}

	type issueCreated struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		State string `json:"state"`
		Kind  string `json:"kind"`
		URL   string `json:"url"`
	}

	result := issueCreated{
		ID:    issue.ID,
		Title: issue.Title,
		State: issue.State,
		Kind:  issue.Kind,
		URL:   issue.Links.HTML.Href,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, result, func() error {
		_, err := fmt.Fprintf(ios.Out, "Created issue #%d: %s\n%s\n",
			result.ID, result.Title, result.URL)
		return err
	})
}

// --- Edit Command ---

type editOptions struct {
	Workspace string
	Repo      string
	Title     string
	Body      string
	State     string
	Kind      string
	Priority  string
	Assignee  string
	Milestone string
	Component string
	Version   string
}

func newEditCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{}
	cmd := &cobra.Command{
		Use:   "edit <issue-id>",
		Short: "Edit an existing issue",
		Example: `  # Update title
  bkt issue edit 42 --title "New title"

  # Change state and priority
  bkt issue edit 42 --state resolved --priority critical

  # Assign to user
  bkt issue edit 42 --assignee {uuid}`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runEdit(cmd, f, opts, issueID)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Update title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Update body/description")
	cmd.Flags().StringVarP(&opts.State, "state", "s", "", "Update state (new, open, resolved, on hold, invalid, duplicate, wontfix, closed)")
	cmd.Flags().StringVarP(&opts.Kind, "kind", "k", "", "Update kind (bug, enhancement, proposal, task)")
	cmd.Flags().StringVarP(&opts.Priority, "priority", "p", "", "Update priority (trivial, minor, major, critical, blocker)")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Update assignee UUID (use empty string to unassign)")
	cmd.Flags().StringVar(&opts.Milestone, "milestone", "", "Update milestone (use empty string to clear)")
	cmd.Flags().StringVar(&opts.Component, "component", "", "Update component (use empty string to clear)")
	cmd.Flags().StringVar(&opts.Version, "version", "", "Update version (use empty string to clear)")

	return cmd
}

func runEdit(cmd *cobra.Command, f *cmdutil.Factory, opts *editOptions, issueID int) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	// Build update input only with flags that were set
	input := bbcloud.UpdateIssueInput{}
	hasUpdates := false

	if cmd.Flags().Changed("title") {
		input.Title = &opts.Title
		hasUpdates = true
	}
	if cmd.Flags().Changed("body") {
		input.Content = &opts.Body
		hasUpdates = true
	}
	if cmd.Flags().Changed("state") {
		input.State = &opts.State
		hasUpdates = true
	}
	if cmd.Flags().Changed("kind") {
		input.Kind = &opts.Kind
		hasUpdates = true
	}
	if cmd.Flags().Changed("priority") {
		input.Priority = &opts.Priority
		hasUpdates = true
	}
	if cmd.Flags().Changed("assignee") {
		input.Assignee = &opts.Assignee
		hasUpdates = true
	}
	if cmd.Flags().Changed("milestone") {
		input.Milestone = &opts.Milestone
		hasUpdates = true
	}
	if cmd.Flags().Changed("component") {
		input.Component = &opts.Component
		hasUpdates = true
	}
	if cmd.Flags().Changed("version") {
		input.Version = &opts.Version
		hasUpdates = true
	}

	if !hasUpdates {
		return fmt.Errorf("no updates specified; use flags like --title, --state, --assignee, etc.")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	issue, err := client.UpdateIssue(ctx, workspace, repoSlug, issueID, input)
	if err != nil {
		return err
	}

	type issueUpdated struct {
		ID       int    `json:"id"`
		Title    string `json:"title"`
		State    string `json:"state"`
		Kind     string `json:"kind"`
		Priority string `json:"priority"`
		URL      string `json:"url"`
	}

	result := issueUpdated{
		ID:       issue.ID,
		Title:    issue.Title,
		State:    issue.State,
		Kind:     issue.Kind,
		Priority: issue.Priority,
		URL:      issue.Links.HTML.Href,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, result, func() error {
		_, err := fmt.Fprintf(ios.Out, "Updated issue #%d: %s [%s]\n%s\n",
			result.ID, result.Title, result.State, result.URL)
		return err
	})
}

// --- Close Command ---

func newCloseCmd(f *cmdutil.Factory) *cobra.Command {
	var workspace, repo string
	cmd := &cobra.Command{
		Use:   "close <issue-id>",
		Short: "Close an issue",
		Example: `  # Close issue #42
  bkt issue close 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runStateChange(cmd, f, workspace, repo, issueID, "closed")
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository slug")

	return cmd
}

// --- Reopen Command ---

func newReopenCmd(f *cmdutil.Factory) *cobra.Command {
	var workspace, repo string
	cmd := &cobra.Command{
		Use:   "reopen <issue-id>",
		Short: "Reopen a closed issue",
		Example: `  # Reopen issue #42
  bkt issue reopen 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runStateChange(cmd, f, workspace, repo, issueID, "open")
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository slug")

	return cmd
}

func runStateChange(cmd *cobra.Command, f *cmdutil.Factory, workspace, repo string, issueID int, newState string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	ws := strings.TrimSpace(workspace)
	if ws == "" {
		ws = ctxCfg.Workspace
	}
	if ws == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	issue, err := client.UpdateIssue(ctx, ws, repoSlug, issueID, bbcloud.UpdateIssueInput{
		State: &newState,
	})
	if err != nil {
		return err
	}

	action := "Closed"
	if newState == "open" {
		action = "Reopened"
	}

	type result struct {
		ID     int    `json:"id"`
		Title  string `json:"title"`
		State  string `json:"state"`
		URL    string `json:"url"`
		Action string `json:"action"`
	}

	r := result{
		ID:     issue.ID,
		Title:  issue.Title,
		State:  issue.State,
		URL:    issue.Links.HTML.Href,
		Action: action,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, r, func() error {
		_, err := fmt.Fprintf(ios.Out, "%s issue #%d: %s\n", action, r.ID, r.Title)
		return err
	})
}

// --- Delete Command ---

type deleteOptions struct {
	Workspace string
	Repo      string
	Confirm   bool
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete <issue-id>",
		Short: "Delete an issue",
		Example: `  # Delete issue #42 (will prompt for confirmation)
  bkt issue delete 42

  # Delete without confirmation
  bkt issue delete 42 --confirm`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runDelete(cmd, f, opts, issueID)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")
	cmd.Flags().BoolVar(&opts.Confirm, "confirm", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(cmd *cobra.Command, f *cmdutil.Factory, opts *deleteOptions, issueID int) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	// Fetch issue first to show title and confirm
	issue, err := client.GetIssue(ctx, workspace, repoSlug, issueID)
	if err != nil {
		return err
	}

	if !opts.Confirm {
		prompter := f.Prompt()
		confirmed, err := prompter.Confirm(fmt.Sprintf("Delete issue #%d: %s?", issue.ID, issue.Title), false)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(ios.Out, "Aborted.")
			return nil
		}
	}

	if err := client.DeleteIssue(ctx, workspace, repoSlug, issueID); err != nil {
		return err
	}

	type result struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Deleted bool   `json:"deleted"`
	}

	r := result{
		ID:      issue.ID,
		Title:   issue.Title,
		Deleted: true,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, r, func() error {
		_, err := fmt.Fprintf(ios.Out, "Deleted issue #%d: %s\n", r.ID, r.Title)
		return err
	})
}

// --- Comment Command ---

type commentOptions struct {
	Workspace string
	Repo      string
	Body      string
	List      bool
}

func newCommentCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentOptions{}
	cmd := &cobra.Command{
		Use:   "comment <issue-id>",
		Short: "Add or list comments on an issue",
		Example: `  # Add a comment
  bkt issue comment 42 -b "This is fixed in the latest release"

  # List comments
  bkt issue comment 42 --list`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runComment(cmd, f, opts, issueID)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Comment body (ignored if --list is specified)")
	cmd.Flags().BoolVar(&opts.List, "list", false, "List existing comments (takes precedence over --body)")

	return cmd
}

func runComment(cmd *cobra.Command, f *cmdutil.Factory, opts *commentOptions, issueID int) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	if opts.List {
		comments, err := client.ListIssueComments(ctx, workspace, repoSlug, issueID, 50)
		if err != nil {
			return err
		}

		type commentSummary struct {
			ID        int    `json:"id"`
			Author    string `json:"author"`
			Body      string `json:"body"`
			CreatedOn string `json:"created_on"`
		}

		var summaries []commentSummary
		for _, c := range comments {
			cs := commentSummary{
				ID:        c.ID,
				Body:      c.Content.Raw,
				CreatedOn: c.CreatedOn,
			}
			if c.User != nil {
				cs.Author = c.User.DisplayName
			}
			summaries = append(summaries, cs)
		}

		payload := struct {
			IssueID  int              `json:"issue_id"`
			Comments []commentSummary `json:"comments"`
		}{
			IssueID:  issueID,
			Comments: summaries,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(summaries) == 0 {
				_, err := fmt.Fprintf(ios.Out, "No comments on issue #%d.\n", issueID)
				return err
			}

			for _, c := range summaries {
				if _, err := fmt.Fprintf(ios.Out, "@%s (%s):\n%s\n\n",
					c.Author, c.CreatedOn, c.Body); err != nil {
					return err
				}
			}
			return nil
		})
	}

	// Add comment
	if strings.TrimSpace(opts.Body) == "" {
		return fmt.Errorf("comment body is required; use --body or -b")
	}

	comment, err := client.CreateIssueComment(ctx, workspace, repoSlug, issueID, opts.Body)
	if err != nil {
		return err
	}

	type result struct {
		ID        int    `json:"id"`
		IssueID   int    `json:"issue_id"`
		Body      string `json:"body"`
		CreatedOn string `json:"created_on"`
	}

	r := result{
		ID:        comment.ID,
		IssueID:   issueID,
		Body:      comment.Content.Raw,
		CreatedOn: comment.CreatedOn,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, r, func() error {
		_, err := fmt.Fprintf(ios.Out, "Added comment to issue #%d\n", issueID)
		return err
	})
}

// --- Status Command ---

type statusOptions struct {
	Workspace string
	Repo      string
}

func newStatusCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &statusOptions{}
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show issues relevant to you",
		Long: `Show issues assigned to you, created by you, and recently updated.

This command requires authentication to identify the current user.`,
		Example: `  # Show your issues
  bkt issue status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")

	return cmd
}

func runStatus(cmd *cobra.Command, f *cmdutil.Factory, opts *statusOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// Get current user
	user, err := client.CurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	type issueSummary struct {
		ID       int    `json:"id"`
		Title    string `json:"title"`
		State    string `json:"state"`
		Kind     string `json:"kind"`
		Priority string `json:"priority"`
	}

	// Fetch issues assigned to user
	assignedIssues, err := client.ListIssues(ctx, workspace, repoSlug, bbcloud.IssueListOptions{
		Assignee: user.UUID,
		Limit:    10,
	})
	if err != nil {
		return err
	}

	// Fetch issues created by user
	createdIssues, err := client.ListIssues(ctx, workspace, repoSlug, bbcloud.IssueListOptions{
		Reporter: user.UUID,
		Limit:    10,
	})
	if err != nil {
		return err
	}

	// Fetch recently updated issues (excluding those already shown)
	recentIssues, err := client.ListIssues(ctx, workspace, repoSlug, bbcloud.IssueListOptions{
		Sort:  "-updated_on", // Most recently updated first
		Limit: 10,
	})
	if err != nil {
		return err
	}

	// Filter out issues already shown in assigned/created
	seen := make(map[int]bool)
	for _, issue := range assignedIssues {
		seen[issue.ID] = true
	}
	for _, issue := range createdIssues {
		seen[issue.ID] = true
	}
	var filteredRecent []bbcloud.Issue
	for _, issue := range recentIssues {
		if !seen[issue.ID] {
			filteredRecent = append(filteredRecent, issue)
		}
	}

	toSummary := func(issues []bbcloud.Issue) []issueSummary {
		var summaries []issueSummary
		for _, issue := range issues {
			summaries = append(summaries, issueSummary{
				ID:       issue.ID,
				Title:    issue.Title,
				State:    issue.State,
				Kind:     issue.Kind,
				Priority: issue.Priority,
			})
		}
		return summaries
	}

	payload := struct {
		User            string         `json:"user"`
		Assigned        []issueSummary `json:"assigned"`
		Created         []issueSummary `json:"created"`
		RecentlyUpdated []issueSummary `json:"recently_updated"`
	}{
		User:            user.Username,
		Assigned:        toSummary(assignedIssues),
		Created:         toSummary(createdIssues),
		RecentlyUpdated: toSummary(filteredRecent),
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if _, err := fmt.Fprintf(ios.Out, "Issues for @%s in %s/%s\n\n", user.Username, workspace, repoSlug); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(ios.Out, "Assigned to you:"); err != nil {
			return err
		}
		if len(payload.Assigned) == 0 {
			if _, err := fmt.Fprintln(ios.Out, "  No issues assigned to you."); err != nil {
				return err
			}
		} else {
			for _, s := range payload.Assigned {
				if _, err := fmt.Fprintf(ios.Out, "  #%d\t%s\t[%s]\n", s.ID, s.Title, s.State); err != nil {
					return err
				}
			}
		}

		if _, err := fmt.Fprintln(ios.Out, "\nCreated by you:"); err != nil {
			return err
		}
		if len(payload.Created) == 0 {
			if _, err := fmt.Fprintln(ios.Out, "  No issues created by you."); err != nil {
				return err
			}
		} else {
			for _, s := range payload.Created {
				if _, err := fmt.Fprintf(ios.Out, "  #%d\t%s\t[%s]\n", s.ID, s.Title, s.State); err != nil {
					return err
				}
			}
		}

		if _, err := fmt.Fprintln(ios.Out, "\nRecently updated:"); err != nil {
			return err
		}
		if len(payload.RecentlyUpdated) == 0 {
			if _, err := fmt.Fprintln(ios.Out, "  No other recently updated issues."); err != nil {
				return err
			}
		} else {
			for _, s := range payload.RecentlyUpdated {
				if _, err := fmt.Fprintf(ios.Out, "  #%d\t%s\t[%s]\n", s.ID, s.Title, s.State); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
