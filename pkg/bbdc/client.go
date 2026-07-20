package bbdc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

// Options configure the Bitbucket Data Center client.
type Options struct {
	BaseURL     string
	Username    string
	Token       string
	AuthMethod  string // "basic" (default) or "bearer"
	EnableCache bool
	Retry       httpx.RetryPolicy
}

// Client wraps Bitbucket Data Center REST endpoints.
type Client struct {
	http *httpx.Client
}

const atlassianTokenNoCheck = "no-check"

// HTTP exposes the underlying HTTP client for advanced scenarios.
func (c *Client) HTTP() *httpx.Client {
	return c.http
}

// New constructs a Bitbucket Data Center client.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	httpClient, err := httpx.New(httpx.Options{
		BaseURL:     opts.BaseURL,
		Username:    opts.Username,
		Password:    opts.Token,
		AuthMethod:  opts.AuthMethod,
		UserAgent:   "bkt-cli",
		EnableCache: opts.EnableCache,
		Retry:       opts.Retry,
		RequestHook: addNoCheckHeader,
	})
	if err != nil {
		return nil, err
	}

	return &Client{http: httpClient}, nil
}

func addNoCheckHeader(req *http.Request) {
	if req == nil || !requiresNoCheck(req.Method) {
		return
	}
	req.Header.Set("X-Atlassian-Token", atlassianTokenNoCheck)
}

func requiresNoCheck(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
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
	Draft       bool   `json:"draft"`
	CreatedDate int64  `json:"createdDate"`
	UpdatedDate int64  `json:"updatedDate"`
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
// Type alias to shared types.CommitStatus for backward compatibility.
type CommitStatus = types.CommitStatus

type paged[T any] struct {
	Size          int  `json:"size"`
	Limit         int  `json:"limit"`
	IsLastPage    bool `json:"isLastPage"`
	Start         int  `json:"start"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []T  `json:"values"`
}

// RepositoriesPage is one bounded page of repositories.
type RepositoriesPage struct {
	Values    []Repository
	IsLast    bool
	NextStart int
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

		page, err := c.ListRepositoriesPage(ctx, projectKey, pageSize, start)
		if err != nil {
			return nil, err
		}

		found = append(found, page.Values...)

		if limit > 0 && len(found) >= limit {
			found = found[:limit]
			break
		}

		if page.IsLast || len(page.Values) == 0 {
			break
		}
		start = page.NextStart
	}

	return found, nil
}

// ListRepositoriesPage fetches one repository page and preserves upstream
// continuation metadata for bounded consumers.
func (c *Client) ListRepositoriesPage(ctx context.Context, projectKey string, limit, start int) (*RepositoriesPage, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos?limit=%d&start=%d", url.PathEscape(projectKey), limit, start)
	req, err := c.http.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	var resp paged[Repository]
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return &RepositoriesPage{
		Values:    resp.Values,
		IsLast:    resp.IsLastPage,
		NextStart: resp.NextPageStart,
	}, nil
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

// RepoPullRequestsOptions configures repository-scoped pull request pages.
// Role filtering happens upstream via the REST participant filter params
// (role.1/username.1); Role requires Username.
type RepoPullRequestsOptions struct {
	State    string
	Role     string // AUTHOR or REVIEWER
	Username string
	Limit    int // page size; <=0 or >100 uses the default
	Start    int // page offset as returned in NextStart
}

// PullRequestsPage is one bounded page of pull requests.
type PullRequestsPage struct {
	Values    []PullRequest
	IsLast    bool
	NextStart int
}

// ListRepoPullRequestsPage fetches a single page of repository pull
// requests with all filters encoded in the upstream query.
func (c *Client) ListRepoPullRequestsPage(ctx context.Context, projectKey, repoSlug string, opts RepoPullRequestsOptions) (*PullRequestsPage, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	params, err := repoPullRequestParams(opts)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests?%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		strings.Join(params, "&"),
	)
	req, err := c.http.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	var resp paged[PullRequest]
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}

	return &PullRequestsPage{
		Values:    resp.Values,
		IsLast:    resp.IsLastPage,
		NextStart: resp.NextPageStart,
	}, nil
}

func repoPullRequestParams(opts RepoPullRequestsOptions) ([]string, error) {
	pageSize := opts.Limit
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 25
	}

	params := []string{fmt.Sprintf("limit=%d", pageSize)}
	if opts.State != "" {
		params = append(params, "state="+url.QueryEscape(strings.ToUpper(opts.State)))
	}
	if opts.Role != "" {
		role := strings.ToUpper(strings.TrimSpace(opts.Role))
		if role != "AUTHOR" && role != "REVIEWER" {
			return nil, fmt.Errorf("unsupported participant role %q; use AUTHOR or REVIEWER", opts.Role)
		}
		if strings.TrimSpace(opts.Username) == "" {
			return nil, fmt.Errorf("participant role filtering requires a username")
		}
		params = append(params,
			"username.1="+url.QueryEscape(opts.Username),
			"role.1="+role,
		)
	}
	params = append(params, fmt.Sprintf("start=%d", opts.Start))
	return params, nil
}

// ListPullRequests lists pull requests for a repository, flattening pages up
// to limit.
func (c *Client) ListPullRequests(ctx context.Context, projectKey, repoSlug, state string, limit int) ([]PullRequest, error) {
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

		page, err := c.ListRepoPullRequestsPage(ctx, projectKey, repoSlug, RepoPullRequestsOptions{
			State: state,
			Limit: pageSize,
			Start: start,
		})
		if err != nil {
			return nil, err
		}

		all = append(all, page.Values...)

		if page.IsLast || len(page.Values) == 0 {
			break
		}
		start = page.NextStart
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

// CommitStatusesPage is one bounded page from the legacy Data Center build
// status endpoint. The endpoint exposes at most the 100 most recent statuses.
type CommitStatusesPage struct {
	Values    []CommitStatus
	IsLast    bool
	NextStart int
}

// CommitStatusesPage fetches one build-status page without flattening it.
func (c *Client) CommitStatusesPage(ctx context.Context, sha string, limit, start int) (*CommitStatusesPage, error) {
	if sha == "" {
		return nil, fmt.Errorf("commit SHA is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if start < 0 {
		return nil, fmt.Errorf("page start must not be negative")
	}

	path := fmt.Sprintf("/rest/build-status/1.0/commits/%s?limit=%d&start=%d",
		url.PathEscape(sha), limit, start)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp paged[CommitStatus]
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return &CommitStatusesPage{
		Values:    resp.Values,
		IsLast:    resp.IsLastPage,
		NextStart: resp.NextPageStart,
	}, nil
}

// DashboardPullRequestsOptions configures dashboard PR listings.
type DashboardPullRequestsOptions struct {
	State string
	Role  string // AUTHOR, REVIEWER, or PARTICIPANT
	Limit int
}

// ListDashboardPullRequestsPage fetches one page of the authenticated
// user's dashboard pull requests; the role filter (AUTHOR, REVIEWER,
// PARTICIPANT) is encoded upstream.
func (c *Client) ListDashboardPullRequestsPage(ctx context.Context, opts DashboardPullRequestsOptions, start int) (*PullRequestsPage, error) {
	pageSize := opts.Limit
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 25
	}

	params := []string{fmt.Sprintf("limit=%d", pageSize)}
	if opts.State != "" {
		params = append(params, "state="+url.QueryEscape(strings.ToUpper(opts.State)))
	}
	if opts.Role != "" {
		params = append(params, "role="+url.QueryEscape(strings.ToUpper(opts.Role)))
	}

	u := fmt.Sprintf("/rest/api/1.0/dashboard/pull-requests?%s&start=%d",
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

	return &PullRequestsPage{
		Values:    resp.Values,
		IsLast:    resp.IsLastPage,
		NextStart: resp.NextPageStart,
	}, nil
}

// ListDashboardPullRequests lists pull requests for the authenticated user across all repositories.
func (c *Client) ListDashboardPullRequests(ctx context.Context, opts DashboardPullRequestsOptions) ([]PullRequest, error) {
	const defaultPageSize = 25

	var (
		start = 0
		all   []PullRequest
	)

	for {
		pageSize := defaultPageSize
		if opts.Limit > 0 {
			remaining := opts.Limit - len(all)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		pageOpts := opts
		pageOpts.Limit = pageSize
		page, err := c.ListDashboardPullRequestsPage(ctx, pageOpts, start)
		if err != nil {
			return nil, err
		}

		all = append(all, page.Values...)

		if page.IsLast || len(page.Values) == 0 {
			break
		}
		start = page.NextStart
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}
