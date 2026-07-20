package bbcloud

import (
	"context"
	"encoding/json"
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
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	State       string `json:"state"`
	Draft       bool   `json:"draft"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Author      struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
		UUID        string `json:"uuid"`
		AccountID   string `json:"account_id"`
	} `json:"author"`
	AuthorNickname string `json:"-"`
	Source         struct {
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
	Reviewers    []User                   `json:"reviewers"`
	Participants []PullRequestParticipant `json:"participants,omitempty"`
	Summary      struct {
		Raw string `json:"raw"`
	} `json:"summary"`
}

// UnmarshalJSON preserves the current anonymous Author field shape for
// source compatibility while retaining Cloud's nickname field separately.
func (p *PullRequest) UnmarshalJSON(data []byte) error {
	type pullRequestAlias PullRequest
	var decoded pullRequestAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var extra struct {
		Author struct {
			Nickname string `json:"nickname"`
		} `json:"author"`
	}
	if err := json.Unmarshal(data, &extra); err != nil {
		return err
	}
	*p = PullRequest(decoded)
	p.AuthorNickname = extra.Author.Nickname
	return nil
}

// PullRequestParticipant retains the approval state Bitbucket Cloud returns
// separately from the pull request's reviewer identities.
type PullRequestParticipant struct {
	User     User   `json:"user"`
	Role     string `json:"role,omitempty"`
	State    string `json:"state,omitempty"`
	Approved *bool  `json:"approved,omitempty"`
}

// PullRequestListOptions configure PR listings. Mine and Reviewer carry a
// user identity (UUID, account id, or nickname) and are encoded upstream as
// BBQL author/reviewer filters before any limiting happens.
type PullRequestListOptions struct {
	State    string
	Limit    int
	Mine     string
	Reviewer string
}

type pullRequestListPage struct {
	Values []PullRequest `json:"values"`
	Next   string        `json:"next"`
}

// PullRequestsPage is one bounded page of pull requests. Next is an opaque
// reference to the following page; empty means the last page.
type PullRequestsPage struct {
	Values []PullRequest
	Next   string
}

// ListRepoPullRequestsPage fetches a single page of repository pull
// requests. Pass next="" for the first page (built from opts) or a Next
// value from a previous page.
func (c *Client) ListRepoPullRequestsPage(ctx context.Context, workspace, repoSlug string, opts PullRequestListOptions, next string) (*PullRequestsPage, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	if next != "" {
		endpoint := fmt.Sprintf("/repositories/%s/%s/pullrequests",
			url.PathEscape(workspace), url.PathEscape(repoSlug))
		normalized, err := normalizeNextRef(next, endpoint)
		if err != nil {
			return nil, err
		}
		return c.fetchPullRequestsPage(ctx, normalized)
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 20
	}

	var params []string
	params = append(params, fmt.Sprintf("pagelen=%d", pageLen))
	params = append(params, pullRequestStateParams(opts.State)...)
	if q := pullRequestQFilter(opts); q != "" {
		params = append(params, "q="+url.QueryEscape(q))
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Join(params, "&"),
	)

	return c.fetchPullRequestsPage(ctx, path)
}

func pullRequestStateParams(raw string) []string {
	state := strings.ToUpper(strings.TrimSpace(raw))
	if state == "" {
		return nil
	}
	if state == "ALL" {
		return []string{"state=OPEN", "state=MERGED", "state=DECLINED"}
	}
	return []string{"state=" + url.QueryEscape(state)}
}

// pullRequestQFilter builds the upstream BBQL q parameter from the identity
// filters; multiple filters combine with AND.
func pullRequestQFilter(opts PullRequestListOptions) string {
	var terms []string
	if mine := strings.TrimSpace(opts.Mine); mine != "" {
		terms = append(terms, bbqlEquals(authorFilterField(mine), mine))
	}
	if reviewer := strings.TrimSpace(opts.Reviewer); reviewer != "" {
		terms = append(terms, bbqlEquals(reviewerFilterField(reviewer), reviewer))
	}
	return strings.Join(terms, " AND ")
}

// normalizeNextRef hardens caller-supplied opaque next references: the
// reference is reduced to its request URI (so it can never point the
// authenticated client at another host) and its path must END at the
// endpoint the page sequence started from. HasSuffix (not Contains) enforces
// terminal endpoint identity — a trailing "/1" or a glued "/prefixrepos..."
// is rejected — while the endpoint's leading slash still admits a legitimate
// base-path prefix such as /2.0.
func normalizeNextRef(next, endpoint string) (string, error) {
	u, err := url.Parse(next)
	if err != nil {
		return "", fmt.Errorf("invalid next page reference: %w", err)
	}
	path := u.EscapedPath()
	if path != endpoint && !strings.HasSuffix(path, endpoint) {
		return "", fmt.Errorf("next page reference does not target %s", endpoint)
	}
	return u.RequestURI(), nil
}

func (c *Client) fetchPullRequestsPage(ctx context.Context, path string) (*PullRequestsPage, error) {
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var page pullRequestListPage
	if err := c.http.Do(req, &page); err != nil {
		return nil, err
	}

	next := ""
	if page.Next != "" {
		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		next = nextURL.RequestURI()
	}

	return &PullRequestsPage{Values: page.Values, Next: next}, nil
}

// ListPullRequests lists pull requests for a repository, flattening pages up
// to opts.Limit.
func (c *Client) ListPullRequests(ctx context.Context, workspace, repoSlug string, opts PullRequestListOptions) ([]PullRequest, error) {
	var prs []PullRequest
	next := ""
	for {
		page, err := c.ListRepoPullRequestsPage(ctx, workspace, repoSlug, opts, next)
		if err != nil {
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
		next = page.Next
	}

	return prs, nil
}

func authorFilterField(identity string) string {
	switch {
	case LooksLikeUUID(identity):
		return "author.uuid"
	case LooksLikeAccountID(identity):
		return "author.account_id"
	default:
		return "author.nickname"
	}
}

// reviewerFilterField picks the BBQL reviewers field matching the identity
// shape, mirroring authorFilterField.
func reviewerFilterField(identity string) string {
	switch {
	case LooksLikeUUID(identity):
		return "reviewers.uuid"
	case LooksLikeAccountID(identity):
		return "reviewers.account_id"
	default:
		return "reviewers.nickname"
	}
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

	// Capture the created comment so a requested threaded reply can be
	// verified. Bitbucket Cloud echoes the created representation, including
	// parent.id, on a successful POST.
	var created PullRequestComment
	if err := c.http.Do(req, &created); err != nil {
		return err
	}

	// Only verify threading when a reply was requested and Bitbucket actually
	// returned a parseable body (created.ID != 0). Some responses (and test
	// stubs) return a 2xx with an empty body, which is still a success.
	if opts.ParentID > 0 && created.ID != 0 {
		if created.Parent == nil || created.Parent.ID != opts.ParentID {
			return fmt.Errorf(
				"comment created (id %d) but Bitbucket did not thread it under parent %d; the parent may not be a valid reply target (e.g. resolved, deleted, or a comment that does not accept replies) — delete the stray comment or retry against a top-level comment",
				created.ID, opts.ParentID,
			)
		}
	}

	return nil
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

// validMergeStrategyValues lists the strategies accepted by Bitbucket Cloud.
var validMergeStrategyValues = []string{
	"merge_commit",
	"squash",
	"fast_forward",
	"squash_fast_forward",
	"rebase_fast_forward",
	"rebase_merge",
}

var validMergeStrategies = mergeStrategySet(validMergeStrategyValues)

func mergeStrategySet(strategies []string) map[string]bool {
	set := make(map[string]bool, len(strategies))
	for _, strategy := range strategies {
		set[strategy] = true
	}
	return set
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
		return fmt.Errorf("invalid merge strategy %q: must be one of %s", strategy, strings.Join(validMergeStrategyValues, ", "))
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
			return fmt.Errorf("merge task %s for pull request #%d may still be running; polling timed out: %w", taskID, prID, ctx.Err())
		case <-time.After(c.mergePollInterval):
		}

		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return err
		}

		var status mergeTaskStatus
		if err := c.http.Do(req, &status); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("merge task %s for pull request #%d may still be running; polling timed out: %w", taskID, prID, ctx.Err())
			}
			return fmt.Errorf("polling merge task %s for pull request #%d: %w", taskID, prID, err)
		}

		if status.Status == "SUCCESS" {
			return nil
		}
		if status.Status != "PENDING" {
			return fmt.Errorf("merge task %s for pull request #%d failed with status: %s", taskID, prID, status.Status)
		}
	}

	return fmt.Errorf("merge task %s for pull request #%d may still be running; did not complete after %d poll attempts", taskID, prID, maxMergePollAttempts)
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
	Deleted    bool     `json:"deleted"`
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

// PullRequestCommentResolution preserves Bitbucket Cloud's resolution response
// for a resolve operation.
type PullRequestCommentResolution map[string]any

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

// GetPullRequestComment retrieves a single pull request comment.
func (c *Client) GetPullRequestComment(ctx context.Context, workspace, repoSlug string, prID, commentID int) (*PullRequestComment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if prID <= 0 {
		return nil, fmt.Errorf("pull request id must be positive")
	}
	if commentID <= 0 {
		return nil, fmt.Errorf("comment id must be positive")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
		commentID,
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var comment PullRequestComment
	if err := c.http.Do(req, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// DeletePullRequestComment deletes a pull request comment.
func (c *Client) DeletePullRequestComment(ctx context.Context, workspace, repoSlug string, prID, commentID int) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if prID <= 0 {
		return fmt.Errorf("pull request id must be positive")
	}
	if commentID <= 0 {
		return fmt.Errorf("comment id must be positive")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
		commentID,
	)
	req, err := c.http.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// SetPullRequestCommentThreadResolved resolves or reopens a top-level pull
// request comment thread. Resolve returns Bitbucket Cloud's resolution object
// when provided; reopen returns nil on success.
func (c *Client) SetPullRequestCommentThreadResolved(ctx context.Context, workspace, repoSlug string, prID, commentID int, resolved bool) (*PullRequestCommentResolution, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if prID <= 0 {
		return nil, fmt.Errorf("pull request id must be positive")
	}
	if commentID <= 0 {
		return nil, fmt.Errorf("comment id must be positive")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments/%d/resolve",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
		commentID,
	)

	method := "DELETE"
	if resolved {
		method = "POST"
	}

	req, err := c.http.NewRequest(ctx, method, path, nil)
	if err != nil {
		return nil, err
	}

	if !resolved {
		return nil, c.http.Do(req, nil)
	}

	var resolution PullRequestCommentResolution
	if err := c.http.Do(req, &resolution); err != nil {
		return nil, err
	}
	return &resolution, nil
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
