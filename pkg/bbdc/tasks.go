package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// PullRequestTask models a task attached to a pull request comment or diff.
type PullRequestTask struct {
	ID        int    `json:"id"`
	State     string `json:"state"`
	Text      string `json:"text"`
	Author    User   `json:"author"`
	CreatedAt int64  `json:"createdDate"`
	UpdatedAt int64  `json:"updatedDate"`
}

// ListPullRequestTasks lists tasks for the pull request.
func (c *Client) ListPullRequestTasks(ctx context.Context, projectKey, repoSlug string, prID int) ([]PullRequestTask, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/tasks",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []PullRequestTask `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}

	return resp.Values, nil
}

// CreatePullRequestTask creates a new task attached to the pull request.
func (c *Client) CreatePullRequestTask(ctx context.Context, projectKey, repoSlug string, prID int, text string) (*PullRequestTask, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("task text is required")
	}

	body := map[string]any{
		"text": text,
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/tasks",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return nil, err
	}

	var task PullRequestTask
	if err := c.http.Do(req, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// CompletePullRequestTask marks a task as resolved.
func (c *Client) CompletePullRequestTask(ctx context.Context, projectKey, repoSlug string, prID, taskID int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/tasks/%d/resolve",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		taskID,
	), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// ReopenPullRequestTask reopens a resolved task.
func (c *Client) ReopenPullRequestTask(ctx context.Context, projectKey, repoSlug string, prID, taskID int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/tasks/%d/reopen",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		taskID,
	), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}
