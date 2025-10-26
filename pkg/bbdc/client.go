package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Options configure the Bitbucket Data Center client.
type Options struct {
	BaseURL  string
	Username string
	Token    string
}

// Client wraps Bitbucket Data Center REST endpoints.
type Client struct {
	http *httpx.Client
}

// New constructs a Bitbucket Data Center client.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	httpClient, err := httpx.New(httpx.Options{
		BaseURL:   opts.BaseURL,
		Username:  opts.Username,
		Password:  opts.Token,
		UserAgent: "bkt-cli",
	})
	if err != nil {
		return nil, err
	}

	return &Client{http: httpClient}, nil
}

// User represents a Bitbucket user.
type User struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ID       int    `json:"id"`
	Email    string `json:"emailAddress"`
	Active   bool   `json:"active"`
	FullName string `json:"displayName"`
	Type     string `json:"type"`
}

// Repository represents a Bitbucket repository.
type Repository struct {
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	ID            int      `json:"id"`
	Project       *Project `json:"project"`
	DefaultBranch string   `json:"defaultBranch,omitempty"`
	Links         struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
		Web []struct {
			Href string `json:"href"`
		} `json:"web"`
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
	} `json:"links"`
}

// Project represents a Bitbucket project.
type Project struct {
	Key         string `json:"key"`
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Public      bool   `json:"public"`
}

// PullRequest models a Bitbucket pull request.
type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Version     int    `json:"version"`
	Author      struct {
		User User `json:"user"`
	} `json:"author"`
	FromRef      Ref                      `json:"fromRef"`
	ToRef        Ref                      `json:"toRef"`
	Reviewers    []PullRequestReviewer    `json:"reviewers"`
	Participants []PullRequestParticipant `json:"participants"`
	Links        struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

// Ref describes a SCM ref.
type Ref struct {
	ID           string     `json:"id"`
	DisplayID    string     `json:"displayId"`
	LatestCommit string     `json:"latestCommit"`
	Repository   Repository `json:"repository"`
}

// CommitStatus describes build status for a commit.
type CommitStatus struct {
	State       string `json:"state"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type paged[T any] struct {
	Size          int  `json:"size"`
	Limit         int  `json:"limit"`
	IsLastPage    bool `json:"isLastPage"`
	Start         int  `json:"start"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []T  `json:"values"`
}

// CurrentUser fetches the user identified by slug.
func (c *Client) CurrentUser(ctx context.Context, userSlug string) (*User, error) {
	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/users/%s", url.PathEscape(userSlug)), nil)
	if err != nil {
		return nil, err
	}
	var user User
	if err := c.http.Do(req, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListRepositories enumerates repositories for a project, handling pagination.
func (c *Client) ListRepositories(ctx context.Context, projectKey string, limit int) ([]Repository, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}

	const defaultPageSize = 25

	var (
		start = 0
		found []Repository
	)

	for {
		pageSize := defaultPageSize
		if limit > 0 {
			remaining := limit - len(found)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos?limit=%d&start=%d", url.PathEscape(projectKey), pageSize, start)
		req, err := c.http.NewRequest(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}

		var resp paged[Repository]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		found = append(found, resp.Values...)

		if limit > 0 && len(found) >= limit {
			found = found[:limit]
			break
		}

		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}
		start = resp.NextPageStart
	}

	return found, nil
}

// GetRepository fetches details for a repository.
func (c *Client) GetRepository(ctx context.Context, projectKey, repoSlug string) (*Repository, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s", url.PathEscape(projectKey), url.PathEscape(repoSlug)), nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.http.Do(req, &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// GetPullRequest fetches a pull request by id.
func (c *Client) GetPullRequest(ctx context.Context, projectKey, repoSlug string, id int) (*PullRequest, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", url.PathEscape(projectKey), url.PathEscape(repoSlug), id), nil)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// ListPullRequests lists pull requests for a repository.
func (c *Client) ListPullRequests(ctx context.Context, projectKey, repoSlug, state string, limit int) ([]PullRequest, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	const defaultPageSize = 25

	var (
		start = 0
		all   []PullRequest
	)

	for {
		pageSize := defaultPageSize
		if limit > 0 {
			remaining := limit - len(all)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		params := []string{fmt.Sprintf("limit=%d", pageSize)}
		if state != "" {
			params = append(params, "state="+url.QueryEscape(strings.ToUpper(state)))
		}

		u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests?%s&start=%d",
			url.PathEscape(projectKey),
			url.PathEscape(repoSlug),
			strings.Join(params, "&"),
			start,
		)
		req, err := c.http.NewRequest(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}

		var resp paged[PullRequest]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		all = append(all, resp.Values...)

		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}
		start = resp.NextPageStart
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}

// CommitStatuses returns build statuses for a commit.
func (c *Client) CommitStatuses(ctx context.Context, sha string) ([]CommitStatus, error) {
	if sha == "" {
		return nil, fmt.Errorf("commit SHA is required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/build-status/1.0/commits/%s", sha), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []CommitStatus `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}
