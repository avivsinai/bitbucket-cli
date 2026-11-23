package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Cloud pull request task states.
const (
	TaskStateUnresolved = "UNRESOLVED"
	TaskStateResolved   = "RESOLVED"
)

// PullRequestTask models a task on a Bitbucket Cloud pull request. Unlike Data
// Center, Cloud tasks are first-class resources that may be linked to comments.
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
	return c.createPullRequestTask(ctx, workspace, repoSlug, prID, text, 0)
}

// CreateInlinePullRequestTask creates an inline comment and attaches a task to it.
// The comment ID is returned even if the task request fails.
func (c *Client) CreateInlinePullRequestTask(ctx context.Context, workspace, repoSlug string, prID int, opts CommentOptions) (*PullRequestTask, int, error) {
	if err := validatePullRequestTaskTarget(workspace, repoSlug, prID); err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(opts.File) == "" || opts.FromLine < 0 || opts.ToLine < 0 || (opts.FromLine > 0) == (opts.ToLine > 0) {
		return nil, 0, fmt.Errorf("inline task requires a file and exactly one positive diff line")
	}
	if opts.ParentID != 0 || opts.Pending {
		return nil, 0, fmt.Errorf("inline tasks do not support a parent or pending state")
	}

	ctx = httpx.WithoutRetries(httpx.WithoutRedirects(ctx))
	comment, err := c.createPullRequestComment(ctx, workspace, repoSlug, prID, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("could not confirm inline comment creation. No task request was sent: %w", err)
	}
	if comment.ID <= 0 {
		return nil, 0, fmt.Errorf("inline comment created on pull request #%d without a comment ID in the response. No task request was sent", prID)
	}

	task, err := c.createPullRequestTask(ctx, workspace, repoSlug, prID, opts.Text, comment.ID)
	if err == nil && task.ID <= 0 {
		err = fmt.Errorf("Bitbucket did not return a task ID")
	}
	if err != nil {
		return nil, comment.ID, fmt.Errorf("created inline comment %d on pull request #%d. Could not confirm task creation: %w", comment.ID, prID, err)
	}
	return task, comment.ID, nil
}

func (c *Client) createPullRequestTask(ctx context.Context, workspace, repoSlug string, prID int, text string, commentID int) (*PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(workspace, repoSlug, prID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("task text is required")
	}

	body := map[string]any{
		"content": map[string]string{"raw": text},
	}
	if commentID > 0 {
		body["comment"] = map[string]int{"id": commentID}
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
