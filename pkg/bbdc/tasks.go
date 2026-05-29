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
	pullRequestTaskAPIHint  = "DC tasks use the blocker-comments API introduced in Bitbucket Data Center 7.2+"

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

// ListPullRequestTasks lists pull request tasks.
//
// Bitbucket Data Center exposes pull request tasks as blocker comments.
func (c *Client) ListPullRequestTasks(ctx context.Context, projectKey, repoSlug string, prID int) ([]PullRequestTask, error) {
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
			return nil, wrapPullRequestTaskAPIError("list pull request tasks", err)
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

// CreatePullRequestTask creates a pull request task.
//
// Bitbucket Data Center exposes pull request tasks as blocker comments.
func (c *Client) CreatePullRequestTask(ctx context.Context, projectKey, repoSlug string, prID int, text string) (*PullRequestTask, error) {
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
		return nil, wrapPullRequestTaskAPIError("create pull request task", err)
	}

	task := comment.task()
	return &task, nil
}

// SetPullRequestTaskState updates a pull request task state and returns the updated task.
func (c *Client) SetPullRequestTaskState(ctx context.Context, projectKey, repoSlug string, prID, taskID int, resolved bool) (*PullRequestTask, error) {
	if err := validatePullRequestTaskTarget(projectKey, repoSlug, prID); err != nil {
		return nil, err
	}
	if taskID <= 0 {
		return nil, fmt.Errorf("task id must be positive")
	}

	current, err := c.getBlockerComment(ctx, projectKey, repoSlug, prID, taskID)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"version": current.Version,
		"state":   taskState(resolved),
	}

	req, err := c.http.NewRequest(ctx, http.MethodPut, blockerCommentPath(projectKey, repoSlug, prID, taskID), body)
	if err != nil {
		return nil, err
	}

	var updated blockerComment
	if err := c.http.Do(req, &updated); err != nil {
		return nil, wrapPullRequestTaskAPIError("set pull request task state", err)
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
		return nil, wrapPullRequestTaskAPIError("get pull request task", err)
	}

	return &comment, nil
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

func wrapPullRequestTaskAPIError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w. Hint: %s", op, err, pullRequestTaskAPIHint)
}
