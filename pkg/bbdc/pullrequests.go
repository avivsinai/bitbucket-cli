package bbdc

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// PullRequestReviewer represents a reviewer assignment.
type PullRequestReviewer struct {
	User User `json:"user"`
}

// PullRequestParticipant wraps a reviewer/participant entry.
type PullRequestParticipant struct {
	User     User   `json:"user"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	Approved bool   `json:"approved"`
}

// PullRequestComment represents a PR comment.
type PullRequestComment struct {
	ID     int    `json:"id"`
	Text   string `json:"text"`
	Author struct {
		User User `json:"user"`
	} `json:"author"`
}

// CreatePROptions configures pull request creation.
type CreatePROptions struct {
	Title        string
	Description  string
	SourceBranch string
	TargetBranch string
	Reviewers    []string
	CloseSource  bool
}

// CreatePullRequest creates a pull request between branches.
func (c *Client) CreatePullRequest(ctx context.Context, projectKey, repoSlug string, opts CreatePROptions) (*PullRequest, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if opts.SourceBranch == "" || opts.TargetBranch == "" {
		return nil, fmt.Errorf("source and target branches are required")
	}
	if opts.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	body := map[string]any{
		"title":       opts.Title,
		"description": opts.Description,
		"fromRef": map[string]any{
			"id": ensureRef(opts.SourceBranch),
			"repository": map[string]any{
				"slug":    repoSlug,
				"project": map[string]any{"key": strings.ToUpper(projectKey)},
			},
		},
		"toRef": map[string]any{
			"id": ensureRef(opts.TargetBranch),
			"repository": map[string]any{
				"slug":    repoSlug,
				"project": map[string]any{"key": strings.ToUpper(projectKey)},
			},
		},
		"closeSourceBranch": opts.CloseSource,
	}

	if len(opts.Reviewers) > 0 {
		reviewers := make([]map[string]any, 0, len(opts.Reviewers))
		for _, reviewer := range opts.Reviewers {
			reviewers = append(reviewers, map[string]any{
				"user": map[string]string{"name": reviewer},
			})
		}
		body["reviewers"] = reviewers
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// MergePROptions controls pull request merges.
type MergePROptions struct {
	Message           string
	Strategy          string
	CloseSourceBranch bool
}

// MergePullRequest merges the pull request.
func (c *Client) MergePullRequest(ctx context.Context, projectKey, repoSlug string, prID int, version int, opts MergePROptions) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	body := map[string]any{
		"version":           version,
		"message":           opts.Message,
		"closeSourceBranch": opts.CloseSourceBranch,
	}
	if opts.Strategy != "" {
		body["mergeStrategyId"] = opts.Strategy
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/merge",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// ApprovePullRequest records an approval for the current token.
func (c *Client) ApprovePullRequest(ctx context.Context, projectKey, repoSlug string, prID int) error {
	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/approve",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// CommentPullRequest adds a comment to the pull request.
func (c *Client) CommentPullRequest(ctx context.Context, projectKey, repoSlug string, prID int, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("comment text is required")
	}

	body := map[string]any{"text": text}
	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// DeclinePullRequest declines (rejects) a pull request.
func (c *Client) DeclinePullRequest(ctx context.Context, projectKey, repoSlug string, prID int, version int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	body := map[string]any{
		"version": version,
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/decline",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// ReopenPullRequest reopens a previously declined pull request.
func (c *Client) ReopenPullRequest(ctx context.Context, projectKey, repoSlug string, prID int, version int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	body := map[string]any{
		"version": version,
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/reopen",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// PullRequestDiff streams the diff for the given pull request into w.
func (c *Client) PullRequestDiff(ctx context.Context, projectKey, repoSlug string, id int, w io.Writer) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}
	if w == nil {
		return fmt.Errorf("writer is required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/diff",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		id,
	), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/plain")

	return c.http.Do(req, w)
}
