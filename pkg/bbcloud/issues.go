package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Issue represents a Bitbucket Cloud issue.
type Issue struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Kind     string `json:"kind"`
	Priority string `json:"priority"`
	Content  struct {
		Raw    string `json:"raw"`
		Markup string `json:"markup"`
		HTML   string `json:"html"`
	} `json:"content"`
	Reporter *Account `json:"reporter"`
	Assignee *Account `json:"assignee"`
	Links    struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
	Component  *IssueComponent `json:"component"`
	Milestone  *IssueMilestone `json:"milestone"`
	Version    *IssueVersion   `json:"version"`
	Votes      int             `json:"votes"`
	Watches    int             `json:"watches"`
	CreatedOn  string          `json:"created_on"`
	UpdatedOn  string          `json:"updated_on"`
	EditedOn   string          `json:"edited_on"`
	Repository *Repository     `json:"repository"`
}

// Account represents a Bitbucket Cloud account (user).
type Account struct {
	UUID        string `json:"uuid"`
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
}

// IssueComponent represents an issue component.
type IssueComponent struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// IssueMilestone represents an issue milestone.
type IssueMilestone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// IssueVersion represents an issue version.
type IssueVersion struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// IssueComment represents a comment on an issue.
type IssueComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw    string `json:"raw"`
		Markup string `json:"markup"`
		HTML   string `json:"html"`
	} `json:"content"`
	User      *Account `json:"user"`
	CreatedOn string   `json:"created_on"`
	UpdatedOn string   `json:"updated_on"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

// IssueListOptions configures issue list requests.
type IssueListOptions struct {
	State     string
	Kind      string
	Priority  string
	Assignee  string
	Reporter  string
	Milestone string
	Query     string
	Limit     int
}

type issueListPage struct {
	Values []Issue `json:"values"`
	Next   string  `json:"next"`
}

// ListIssues lists issues for a repository.
func (c *Client) ListIssues(ctx context.Context, workspace, repoSlug string, opts IssueListOptions) ([]Issue, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 30
	}

	var params []string
	params = append(params, fmt.Sprintf("pagelen=%d", pageLen))

	// Build BBQL query
	var queryParts []string
	if state := strings.TrimSpace(opts.State); state != "" && !strings.EqualFold(state, "all") {
		queryParts = append(queryParts, fmt.Sprintf("state = \"%s\"", state))
	}
	if kind := strings.TrimSpace(opts.Kind); kind != "" {
		queryParts = append(queryParts, fmt.Sprintf("kind = \"%s\"", kind))
	}
	if priority := strings.TrimSpace(opts.Priority); priority != "" {
		queryParts = append(queryParts, fmt.Sprintf("priority = \"%s\"", priority))
	}
	if assignee := strings.TrimSpace(opts.Assignee); assignee != "" {
		queryParts = append(queryParts, fmt.Sprintf("assignee.username = \"%s\"", assignee))
	}
	if reporter := strings.TrimSpace(opts.Reporter); reporter != "" {
		queryParts = append(queryParts, fmt.Sprintf("reporter.username = \"%s\"", reporter))
	}
	if milestone := strings.TrimSpace(opts.Milestone); milestone != "" {
		queryParts = append(queryParts, fmt.Sprintf("milestone.name = \"%s\"", milestone))
	}
	if opts.Query != "" {
		queryParts = append(queryParts, opts.Query)
	}

	if len(queryParts) > 0 {
		params = append(params, "q="+url.QueryEscape(strings.Join(queryParts, " AND ")))
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues?%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Join(params, "&"),
	)

	var issues []Issue
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page issueListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		issues = append(issues, page.Values...)

		if opts.Limit > 0 && len(issues) >= opts.Limit {
			issues = issues[:opts.Limit]
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

	return issues, nil
}

// GetIssue fetches a single issue by ID.
func (c *Client) GetIssue(ctx context.Context, workspace, repoSlug string, issueID int) (*Issue, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
	)

	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := c.http.Do(req, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// CreateIssueInput configures issue creation.
type CreateIssueInput struct {
	Title     string
	Content   string
	Kind      string
	Priority  string
	Assignee  string
	Milestone string
	Component string
	Version   string
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(ctx context.Context, workspace, repoSlug string, input CreateIssueInput) (*Issue, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}

	body := map[string]any{
		"title": input.Title,
	}

	if input.Content != "" {
		body["content"] = map[string]any{
			"raw": input.Content,
		}
	}
	if input.Kind != "" {
		body["kind"] = input.Kind
	}
	if input.Priority != "" {
		body["priority"] = input.Priority
	}
	if input.Assignee != "" {
		body["assignee"] = map[string]any{
			"username": input.Assignee,
		}
	}
	if input.Milestone != "" {
		body["milestone"] = map[string]any{
			"name": input.Milestone,
		}
	}
	if input.Component != "" {
		body["component"] = map[string]any{
			"name": input.Component,
		}
	}
	if input.Version != "" {
		body["version"] = map[string]any{
			"name": input.Version,
		}
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := c.http.Do(req, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// UpdateIssueInput configures issue updates.
type UpdateIssueInput struct {
	Title     *string
	Content   *string
	State     *string
	Kind      *string
	Priority  *string
	Assignee  *string
	Milestone *string
	Component *string
	Version   *string
}

// UpdateIssue updates an existing issue.
func (c *Client) UpdateIssue(ctx context.Context, workspace, repoSlug string, issueID int, input UpdateIssueInput) (*Issue, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	body := make(map[string]any)

	if input.Title != nil {
		body["title"] = *input.Title
	}
	if input.Content != nil {
		body["content"] = map[string]any{
			"raw": *input.Content,
		}
	}
	if input.State != nil {
		body["state"] = *input.State
	}
	if input.Kind != nil {
		body["kind"] = *input.Kind
	}
	if input.Priority != nil {
		body["priority"] = *input.Priority
	}
	if input.Assignee != nil {
		if *input.Assignee == "" {
			body["assignee"] = nil
		} else {
			body["assignee"] = map[string]any{
				"username": *input.Assignee,
			}
		}
	}
	if input.Milestone != nil {
		if *input.Milestone == "" {
			body["milestone"] = nil
		} else {
			body["milestone"] = map[string]any{
				"name": *input.Milestone,
			}
		}
	}
	if input.Component != nil {
		if *input.Component == "" {
			body["component"] = nil
		} else {
			body["component"] = map[string]any{
				"name": *input.Component,
			}
		}
	}
	if input.Version != nil {
		if *input.Version == "" {
			body["version"] = nil
		} else {
			body["version"] = map[string]any{
				"name": *input.Version,
			}
		}
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
	)

	req, err := c.http.NewRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := c.http.Do(req, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// DeleteIssue deletes an issue.
func (c *Client) DeleteIssue(ctx context.Context, workspace, repoSlug string, issueID int) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
	)

	req, err := c.http.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

type issueCommentListPage struct {
	Values []IssueComment `json:"values"`
	Next   string         `json:"next"`
}

// ListIssueComments lists comments on an issue.
func (c *Client) ListIssueComments(ctx context.Context, workspace, repoSlug string, issueID int, limit int) ([]IssueComment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 30
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d/comments?pagelen=%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
		pageLen,
	)

	var comments []IssueComment
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page issueCommentListPage
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

// CreateIssueComment creates a comment on an issue.
func (c *Client) CreateIssueComment(ctx context.Context, workspace, repoSlug string, issueID int, body string) (*IssueComment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("comment body is required")
	}

	payload := map[string]any{
		"content": map[string]any{
			"raw": body,
		},
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues/%d/comments",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		issueID,
	)

	req, err := c.http.NewRequest(ctx, "POST", path, payload)
	if err != nil {
		return nil, err
	}

	var comment IssueComment
	if err := c.http.Do(req, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}
