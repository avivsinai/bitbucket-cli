package bbcloud

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// PullRequest models a Bitbucket Cloud pull request.
type PullRequest struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
		Repository struct {
			Slug string `json:"slug"`
		} `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			Slug string `json:"slug"`
		} `json:"repository"`
	} `json:"destination"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Summary struct {
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
func (c *Client) DeclinePullRequest(ctx context.Context, workspace, repoSlug string, id int) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/decline",
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
			reviewers = append(reviewers, map[string]string{"username": reviewer})
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

// UpdatePullRequestInput configures PR updates. Use pointers to distinguish
// between "not set" and "set to empty string" for clearing fields.
type UpdatePullRequestInput struct {
	Title       *string
	Description *string
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

	if len(body) == 0 {
		return nil, fmt.Errorf("at least one field (title or description) must be provided")
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

// CommentPullRequest adds a comment to the pull request.
func (c *Client) CommentPullRequest(ctx context.Context, workspace, repoSlug string, prID int, text string) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("comment text is required")
	}

	body := map[string]any{
		"content": map[string]string{
			"raw": text,
		},
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
