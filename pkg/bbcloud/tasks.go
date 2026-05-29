package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Cloud pull request task states.
const (
	TaskStateUnresolved = "UNRESOLVED"
	TaskStateResolved   = "RESOLVED"
)

// PullRequestTask models a task on a Bitbucket Cloud pull request. Unlike Data
// Center, Cloud tasks are a first-class, standalone PR-level resource separate
// from comments.
type PullRequestTask struct {
	ID      int    `json:"id"`
	State   string `json:"state"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	Creator    *Account `json:"creator"`
	CreatedOn  string   `json:"created_on"`
	UpdatedOn  string   `json:"updated_on"`
	ResolvedOn string   `json:"resolved_on,omitempty"`
}

type pullRequestTaskListPage struct {
	Values []PullRequestTask `json:"values"`
	Next   string            `json:"next"`
}

func pullRequestTasksPath(workspace, repoSlug string, prID int) string {
	return fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/tasks",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
	)
}

func validatePullRequestTaskTarget(workspace, repoSlug string, prID int) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if prID <= 0 {
		return fmt.Errorf("pull request id must be positive")
	}
	return nil
}

// ListPullRequestTasks lists tasks on a pull request, following pagination. A
// limit of 0 returns all tasks.
func (c *Client) ListPullRequestTasks(ctx context.Context, workspace, repoSlug string, prID, limit int) ([]PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(workspace, repoSlug, prID); err != nil {
		return nil, err
	}

	// Bitbucket Cloud's pagelen ranges from 10 to 100. Request within that band
	// and truncate the accumulated result to the caller's limit below.
	pageLen := limit
	switch {
	case pageLen <= 0 || pageLen > 100:
		pageLen = 100
	case pageLen < 10:
		pageLen = 10
	}

	path := fmt.Sprintf("%s?pagelen=%d", pullRequestTasksPath(workspace, repoSlug, prID), pageLen)

	var tasks []PullRequestTask
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page pullRequestTaskListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		tasks = append(tasks, page.Values...)

		if limit > 0 && len(tasks) >= limit {
			tasks = tasks[:limit]
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

	return tasks, nil
}

// CreatePullRequestTask creates a standalone PR-level task with the given text.
func (c *Client) CreatePullRequestTask(ctx context.Context, workspace, repoSlug string, prID int, text string) (*PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(workspace, repoSlug, prID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("task text is required")
	}

	body := map[string]any{
		"content": map[string]string{"raw": text},
	}

	req, err := c.http.NewRequest(ctx, "POST", pullRequestTasksPath(workspace, repoSlug, prID), body)
	if err != nil {
		return nil, err
	}

	var task PullRequestTask
	if err := c.http.Do(req, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// SetPullRequestTaskState resolves (resolved=true) or reopens (resolved=false)
// a task by updating its state.
func (c *Client) SetPullRequestTaskState(ctx context.Context, workspace, repoSlug string, prID, taskID int, resolved bool) (*PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(workspace, repoSlug, prID); err != nil {
		return nil, err
	}
	if taskID <= 0 {
		return nil, fmt.Errorf("task id must be positive")
	}

	state := TaskStateUnresolved
	if resolved {
		state = TaskStateResolved
	}

	path := fmt.Sprintf("%s/%d", pullRequestTasksPath(workspace, repoSlug, prID), taskID)
	req, err := c.http.NewRequest(ctx, "PUT", path, map[string]any{"state": state})
	if err != nil {
		return nil, err
	}

	var task PullRequestTask
	if err := c.http.Do(req, &task); err != nil {
		return nil, err
	}

	return &task, nil
}
