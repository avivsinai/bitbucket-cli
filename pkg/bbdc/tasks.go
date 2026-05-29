package bbdc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	pullRequestTaskPageSize = 25

	TaskStateOpen     = "OPEN"
	TaskStateResolved = "RESOLVED"
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

type blockerComment struct {
	ID        int    `json:"id"`
	Version   int    `json:"version"`
	State     string `json:"state"`
	Text      string `json:"text"`
	Author    User   `json:"author"`
	CreatedAt int64  `json:"createdDate"`
	UpdatedAt int64  `json:"updatedDate"`
}

func (comment blockerComment) task() PullRequestTask {
	return PullRequestTask{
		ID:        comment.ID,
		State:     comment.State,
		Text:      comment.Text,
		Author:    comment.Author,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
}

// ListPullRequestTasks lists legacy tasks for the pull request.
func (c *Client) ListPullRequestTasks(ctx context.Context, projectKey, repoSlug string, prID int) ([]PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(projectKey, repoSlug, prID); err != nil {
		return nil, err
	}

	var (
		start int
		tasks []PullRequestTask
	)

	for {
		req, err := c.http.NewRequest(ctx, http.MethodGet, pagedTaskPath(legacyPullRequestTasksPath(projectKey, repoSlug, prID), start), nil)
		if err != nil {
			return nil, err
		}

		var resp paged[PullRequestTask]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		tasks = append(tasks, resp.Values...)
		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}
		start = resp.NextPageStart
	}

	return tasks, nil
}

// ListBlockerComments lists blocker comments for the pull request.
func (c *Client) ListBlockerComments(ctx context.Context, projectKey, repoSlug string, prID int) ([]PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(projectKey, repoSlug, prID); err != nil {
		return nil, err
	}

	var (
		start int
		tasks []PullRequestTask
	)

	for {
		req, err := c.http.NewRequest(ctx, http.MethodGet, pagedTaskPath(blockerCommentsPath(projectKey, repoSlug, prID), start), nil)
		if err != nil {
			return nil, err
		}

		var resp paged[blockerComment]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		for _, comment := range resp.Values {
			tasks = append(tasks, comment.task())
		}
		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}
		start = resp.NextPageStart
	}

	return tasks, nil
}

// CreateBlockerComment creates a blocker comment for the pull request.
func (c *Client) CreateBlockerComment(ctx context.Context, projectKey, repoSlug string, prID int, text string) (*PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(projectKey, repoSlug, prID); err != nil {
		return nil, err
	}
	if err := validateTaskText(text); err != nil {
		return nil, err
	}

	body := map[string]any{
		"text": text,
	}

	req, err := c.http.NewRequest(ctx, http.MethodPost, blockerCommentsPath(projectKey, repoSlug, prID), body)
	if err != nil {
		return nil, err
	}

	var comment blockerComment
	if err := c.http.Do(req, &comment); err != nil {
		return nil, err
	}

	task := comment.task()
	return &task, nil
}

// SetBlockerCommentState updates a blocker comment state and returns the updated task.
func (c *Client) SetBlockerCommentState(ctx context.Context, projectKey, repoSlug string, prID, commentID int, resolved bool) (*PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(projectKey, repoSlug, prID); err != nil {
		return nil, err
	}
	if commentID <= 0 {
		return nil, fmt.Errorf("comment id must be positive")
	}

	current, err := c.getBlockerComment(ctx, projectKey, repoSlug, prID, commentID)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"version": current.Version,
		"state":   taskState(resolved),
	}

	req, err := c.http.NewRequest(ctx, http.MethodPut, blockerCommentPath(projectKey, repoSlug, prID, commentID), body)
	if err != nil {
		return nil, err
	}

	var updated blockerComment
	if err := c.http.Do(req, &updated); err != nil {
		return nil, err
	}

	task := updated.task()
	return &task, nil
}

func (c *Client) getBlockerComment(ctx context.Context, projectKey, repoSlug string, prID, commentID int) (*blockerComment, error) {
	req, err := c.http.NewRequest(ctx, http.MethodGet, blockerCommentPath(projectKey, repoSlug, prID, commentID), nil)
	if err != nil {
		return nil, err
	}

	var comment blockerComment
	if err := c.http.Do(req, &comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

// CreateLegacyTask creates a legacy task anchored to an existing comment.
func (c *Client) CreateLegacyTask(ctx context.Context, projectKey, repoSlug string, prID, commentID int, text string) (*PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(projectKey, repoSlug, prID); err != nil {
		return nil, err
	}
	if commentID <= 0 {
		return nil, fmt.Errorf("comment id must be positive")
	}
	if err := validateTaskText(text); err != nil {
		return nil, err
	}

	body := map[string]any{
		"anchor": map[string]any{
			"id":   commentID,
			"type": "COMMENT",
		},
		"text": text,
	}

	req, err := c.http.NewRequest(ctx, http.MethodPost, "/rest/api/1.0/tasks", body)
	if err != nil {
		return nil, err
	}

	var task PullRequestTask
	if err := c.http.Do(req, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// SetLegacyTaskState updates a legacy task state and returns the updated task.
func (c *Client) SetLegacyTaskState(ctx context.Context, taskID int, resolved bool) (*PullRequestTask, error) {
	if taskID <= 0 {
		return nil, fmt.Errorf("task id must be positive")
	}

	body := map[string]any{
		"state": taskState(resolved),
	}

	req, err := c.http.NewRequest(ctx, http.MethodPut, fmt.Sprintf("/rest/api/1.0/tasks/%d", taskID), body)
	if err != nil {
		return nil, err
	}

	var task PullRequestTask
	if err := c.http.Do(req, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// DeleteLegacyTask deletes a legacy task.
func (c *Client) DeleteLegacyTask(ctx context.Context, taskID int) error {
	if taskID <= 0 {
		return fmt.Errorf("task id must be positive")
	}

	req, err := c.http.NewRequest(ctx, http.MethodDelete, fmt.Sprintf("/rest/api/1.0/tasks/%d", taskID), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

func validatePullRequestTaskTarget(projectKey, repoSlug string, prID int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}
	if prID <= 0 {
		return fmt.Errorf("pull request id must be positive")
	}
	return nil
}

func validateTaskText(text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("task text is required")
	}
	return nil
}

func taskState(resolved bool) string {
	if resolved {
		return TaskStateResolved
	}
	return TaskStateOpen
}

func legacyPullRequestTasksPath(projectKey, repoSlug string, prID int) string {
	return pullRequestPath(projectKey, repoSlug, prID, "/tasks")
}

func blockerCommentsPath(projectKey, repoSlug string, prID int) string {
	return pullRequestPath(projectKey, repoSlug, prID, "/blocker-comments")
}

func blockerCommentPath(projectKey, repoSlug string, prID, commentID int) string {
	return fmt.Sprintf("%s/%d", blockerCommentsPath(projectKey, repoSlug, prID), commentID)
}

func pullRequestPath(projectKey, repoSlug string, prID int, suffix string) string {
	return fmt.Sprintf(
		"/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		suffix,
	)
}

func pagedTaskPath(path string, start int) string {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(pullRequestTaskPageSize))
	query.Set("start", strconv.Itoa(start))
	return path + "?" + query.Encode()
}
