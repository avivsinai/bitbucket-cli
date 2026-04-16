package bbcloud

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// RepositoryRef identifies a repository inside a pull request's source or
// destination. The Bitbucket Cloud API returns full_name and clone links here,
// which we need to resolve fork remotes during checkout.
type RepositoryRef struct {
	Slug     string `json:"slug"`
	FullName string `json:"full_name"`
	Links    struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"` // "https" or "ssh"
		} `json:"clone"`
	} `json:"links"`
}

// PullRequest models a Bitbucket Cloud pull request.
type PullRequest struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	State     string `json:"state"`
	Draft     bool   `json:"draft"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Author    struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
		UUID        string `json:"uuid"`
		AccountID   string `json:"account_id"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
		Repository RepositoryRef `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository RepositoryRef `json:"repository"`
	} `json:"destination"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Reviewers []User `json:"reviewers"`
	Summary   struct {
		Raw string `json:"raw"`
	} `json:"summary"`
}

// PullRequestListOptions configure PR listings.
type PullRequestListOptions struct {
	State string
	Limit int
	Mine  string
}

type pullRequestListPage struct {
	Values []PullRequest `json:"values"`
	Next   string        `json:"next"`
}

// ListPullRequests lists pull requests for a repository.
func (c *Client) ListPullRequests(ctx context.Context, workspace, repoSlug string, opts PullRequestListOptions) ([]PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 20
	}

	var params []string
	params = append(params, fmt.Sprintf("pagelen=%d", pageLen))
	if state := strings.TrimSpace(opts.State); state != "" && !strings.EqualFold(state, "all") {
		params = append(params, "state="+url.QueryEscape(strings.ToUpper(state)))
	}
	if opts.Mine != "" {
		params = append(params, "q="+url.QueryEscape(fmt.Sprintf("author.username=\"%s\"", opts.Mine)))
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Join(params, "&"),
	)

	var prs []PullRequest
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page pullRequestListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		prs = append(prs, page.Values...)

		if opts.Limit > 0 && len(prs) >= opts.Limit {
			prs = prs[:opts.Limit]
			break
		}

		if page.Next == "" {
			break
		}
		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return prs, nil
}

// GetPullRequest fetches a pull request by ID.
func (c *Client) GetPullRequest(ctx context.Context, workspace, repoSlug string, id int) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// DeclinePullRequest declines (rejects) a pull request.
// An optional message may be provided; leave empty to omit it.
func (c *Client) DeclinePullRequest(ctx context.Context, workspace, repoSlug string, id int, message string) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}

	var body any
	if message != "" {
		body = map[string]any{"message": message}
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/decline",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)
	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// ReopenPullRequest reopens a previously declined pull request by updating its state to OPEN.
func (c *Client) ReopenPullRequest(ctx context.Context, workspace, repoSlug string, id int) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}

	body := map[string]any{
		"state": "OPEN",
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)
	req, err := c.http.NewRequest(ctx, "PUT", path, body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// CreatePullRequestInput configures PR creation.
type CreatePullRequestInput struct {
	Title       string
	Description string
	Source      string
	Destination string
	CloseSource bool
	Reviewers   []string
	Draft       bool
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(ctx context.Context, workspace, repoSlug string, input CreatePullRequestInput) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(input.Source) == "" || strings.TrimSpace(input.Destination) == "" {
		return nil, fmt.Errorf("source and destination branches are required")
	}

	body := map[string]any{
		"title":               input.Title,
		"close_source_branch": input.CloseSource,
		"draft":               input.Draft,
		"source": map[string]any{
			"branch": map[string]string{"name": input.Source},
		},
		"destination": map[string]any{
			"branch": map[string]string{"name": input.Destination},
		},
	}
	if input.Description != "" {
		body["description"] = input.Description
	}
	if len(input.Reviewers) > 0 {
		var reviewers []map[string]string
		for _, reviewer := range input.Reviewers {
			reviewers = append(reviewers, reviewerIdentity(reviewer))
		}
		body["reviewers"] = reviewers
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// EffectiveDefaultReviewer represents a reviewer returned by the
// effective-default-reviewers endpoint, which wraps each user in a nested object.
type EffectiveDefaultReviewer struct {
	User User `json:"user"`
}

// GetEffectiveDefaultReviewers returns the effective default reviewers for a repository.
func (c *Client) GetEffectiveDefaultReviewers(ctx context.Context, workspace, repoSlug string) ([]User, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/effective-default-reviewers?pagelen=100",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	var users []User
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page struct {
			Values []EffectiveDefaultReviewer `json:"values"`
			Next   string                     `json:"next"`
		}
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		for _, v := range page.Values {
			users = append(users, v.User)
		}

		if page.Next == "" {
			break
		}
		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return users, nil
}

// reviewerIdentity returns the correct API identity map for a reviewer string.
func reviewerIdentity(reviewer string) map[string]string {
	if LooksLikeUUID(reviewer) {
		return map[string]string{"uuid": NormalizeUUID(reviewer)}
	}
	if LooksLikeAccountID(reviewer) {
		return map[string]string{"account_id": reviewer}
	}
	return map[string]string{"username": reviewer}
}

// UpdatePullRequestInput configures PR updates. Use pointers to distinguish
// between "not set" and "set to empty string" for clearing fields.
type UpdatePullRequestInput struct {
	Title       *string
	Description *string
	Draft       *bool
	// Reviewers sets the PR reviewer list. nil = don't change; non-nil = replace
	// with this list (empty slice clears all reviewers).
	Reviewers []string
}

// UpdatePullRequest updates an existing pull request's title and/or description.
func (c *Client) UpdatePullRequest(ctx context.Context, workspace, repoSlug string, id int, input UpdatePullRequestInput) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	body := make(map[string]any)
	if input.Title != nil {
		body["title"] = *input.Title
	}
	if input.Description != nil {
		body["description"] = *input.Description
	}
	if input.Draft != nil {
		body["draft"] = *input.Draft
	}
	if input.Reviewers != nil {
		reviewers := make([]map[string]string, 0, len(input.Reviewers))
		for _, reviewer := range input.Reviewers {
			reviewers = append(reviewers, reviewerIdentity(reviewer))
		}
		body["reviewers"] = reviewers
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("at least one field must be provided")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)

	req, err := c.http.NewRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// CommentOptions configures a pull request comment.
type CommentOptions struct {
	Text     string
	ParentID int
	File     string
	FromLine int
	ToLine   int
	Pending  bool
}

// CommentPullRequest adds a comment to the pull request.
// When ParentID > 0, the comment is a threaded reply.
// When File is set with FromLine or ToLine, the comment targets a specific diff line.
func (c *Client) CommentPullRequest(ctx context.Context, workspace, repoSlug string, prID int, opts CommentOptions) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(opts.Text) == "" {
		return fmt.Errorf("comment text is required")
	}

	body := map[string]any{
		"content": map[string]string{
			"raw": opts.Text,
		},
	}
	if opts.Pending {
		body["pending"] = true
	}
	if opts.ParentID > 0 {
		body["parent"] = map[string]int{"id": opts.ParentID}
	}
	if opts.File != "" {
		inline := map[string]any{"path": opts.File}
		if opts.ToLine > 0 {
			inline["to"] = opts.ToLine
		}
		if opts.FromLine > 0 {
			inline["from"] = opts.FromLine
		}
		body["inline"] = inline
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
	)
	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// PullRequestDiff streams the unified diff for the given pull request into w.
func (c *Client) PullRequestDiff(ctx context.Context, workspace, repoSlug string, id int, w io.Writer) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if w == nil {
		return fmt.Errorf("writer is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/diff",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/plain")

	return c.http.Do(req, w)
}

// validMergeStrategies lists the strategies accepted by Bitbucket Cloud.
var validMergeStrategies = map[string]bool{
	"merge_commit": true,
	"squash":       true,
	"fast_forward": true,
}

// mergeTaskStatus represents the async merge task status response.
type mergeTaskStatus struct {
	TaskID string `json:"task_id"`
	Status string `json:"task_status"`
}

// MergePullRequest merges the given pull request.
// The Bitbucket Cloud API may return 202 for long-running merges with a task_id
// that must be polled until completion.
func (c *Client) MergePullRequest(ctx context.Context, workspace, repoSlug string, id int, message, strategy string, closeSource bool) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if strategy != "" && !validMergeStrategies[strategy] {
		return fmt.Errorf("invalid merge strategy %q: must be one of merge_commit, squash, fast_forward", strategy)
	}

	body := map[string]any{
		"close_source_branch": closeSource,
	}
	if message != "" {
		body["message"] = message
	}
	if strategy != "" {
		body["merge_strategy"] = strategy
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)
	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return err
	}

	var result mergeTaskStatus
	if err := c.http.Do(req, &result); err != nil {
		return err
	}

	if result.TaskID != "" {
		return c.pollMergeTask(ctx, workspace, repoSlug, id, result.TaskID)
	}

	return nil
}

// maxMergePollAttempts is the upper bound on polling iterations for async merges.
// At 2 seconds per iteration this gives ~5 minutes before giving up.
const maxMergePollAttempts = 150

// pollMergeTask polls the merge task status until it completes, the context
// expires, or maxMergePollAttempts is reached.
func (c *Client) pollMergeTask(ctx context.Context, workspace, repoSlug string, prID int, taskID string) error {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge/task-status/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
		url.PathEscape(taskID),
	)

	for attempt := 0; attempt < maxMergePollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("merge task timed out: %w", ctx.Err())
		case <-time.After(c.mergePollInterval):
		}

		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return err
		}

		var status mergeTaskStatus
		if err := c.http.Do(req, &status); err != nil {
			return fmt.Errorf("polling merge task: %w", err)
		}

		if status.Status == "SUCCESS" {
			return nil
		}
		if status.Status != "PENDING" {
			return fmt.Errorf("merge task failed with status: %s", status.Status)
		}
	}

	return fmt.Errorf("merge task %s did not complete after %d poll attempts", taskID, maxMergePollAttempts)
}

// PullRequestComment models a comment on a Bitbucket Cloud pull request.
type PullRequestComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	User       *Account `json:"user"`
	CreatedOn  string   `json:"created_on"`
	UpdatedOn  string   `json:"updated_on"`
	Resolution *struct {
		User      *Account `json:"user"`
		CreatedOn string   `json:"created_on"`
	} `json:"resolution"`
	Parent *struct {
		ID int `json:"id"`
	} `json:"parent"`
	Inline *struct {
		Path string `json:"path"`
		From *int   `json:"from"`
		To   *int   `json:"to"`
	} `json:"inline,omitempty"`
}

type pullRequestCommentListPage struct {
	Values []PullRequestComment `json:"values"`
	Next   string               `json:"next"`
}

// ListPullRequestComments lists comments on a pull request.
func (c *Client) ListPullRequestComments(ctx context.Context, workspace, repoSlug string, prID int, limit int) ([]PullRequestComment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 100
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments?pagelen=%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
		pageLen,
	)

	var comments []PullRequestComment
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page pullRequestCommentListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		comments = append(comments, page.Values...)

		if limit > 0 && len(comments) >= limit {
			comments = comments[:limit]
			break
		}

		if page.Next == "" {
			break
		}

		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return comments, nil
}

// ApprovePullRequest approves the given pull request.
func (c *Client) ApprovePullRequest(ctx context.Context, workspace, repoSlug string, id int) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)
	req, err := c.http.NewRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}
