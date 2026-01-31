package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

// Options configure the Bitbucket Cloud client.
type Options struct {
	BaseURL     string
	Username    string
	Token       string
	Workspace   string
	EnableCache bool
	Retry       httpx.RetryPolicy
}

// Client wraps Bitbucket Cloud REST endpoints.
type Client struct {
	http *httpx.Client
}

// HTTP exposes the underlying HTTP client for advanced scenarios.
func (c *Client) HTTP() *httpx.Client {
	return c.http
}

// New constructs a Bitbucket Cloud client.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.bitbucket.org/2.0"
	}

	httpClient, err := httpx.New(httpx.Options{
		BaseURL:     opts.BaseURL,
		Username:    opts.Username,
		Password:    opts.Token,
		UserAgent:   "bkt-cli",
		EnableCache: opts.EnableCache,
		Retry:       opts.Retry,
	})
	if err != nil {
		return nil, err
	}

	return &Client{http: httpClient}, nil
}

// User represents a Bitbucket Cloud user profile.
type User struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Display  string `json:"display_name"`
}

// CurrentUser retrieves the authenticated user.
func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	req, err := c.http.NewRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, err
	}
	var user User
	if err := c.http.Do(req, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// Repository identifies a Bitbucket Cloud repository.
type Repository struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	SCM       string `json:"scm"`
	IsPrivate bool   `json:"is_private"`
	Links     struct {
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Workspace struct {
		Slug string `json:"slug"`
	} `json:"workspace"`
	Project struct {
		Key string `json:"key"`
	} `json:"project"`
}

// Pipeline represents a pipeline execution.
type Pipeline struct {
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	State       struct {
		Result struct {
			Name string `json:"name"`
		} `json:"result"`
		Stage struct {
			Name string `json:"name"`
		} `json:"stage"`
		Name string `json:"name"`
	} `json:"state"`
	Target struct {
		Type string `json:"type"`
		Ref  struct {
			Name string `json:"name"`
		} `json:"ref"`
	} `json:"target"`
	CreatedOn   string `json:"created_on"`
	CompletedOn string `json:"completed_on"`
}

// normalizeUUID ensures a UUID has curly braces, as required by Bitbucket Cloud API.
func normalizeUUID(uuid string) string {
	uuid = strings.Trim(uuid, "{}")
	return "{" + uuid + "}"
}

// PipelinePage encapsulates paginated pipeline results.
type PipelinePage struct {
	Values []Pipeline `json:"values"`
	Next   string     `json:"next"`
}

// ListPipelines lists recent pipelines.
func (c *Client) ListPipelines(ctx context.Context, workspace, repoSlug string, limit int) ([]Pipeline, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := limit
	if pageLen <= 0 || pageLen > 50 {
		pageLen = 20
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines/?pagelen=%d&sort=-created_on",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		pageLen,
	)

	var pipelines []Pipeline

	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page PipelinePage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		pipelines = append(pipelines, page.Values...)

		if limit > 0 && len(pipelines) >= limit {
			pipelines = pipelines[:limit]
			break
		}

		if page.Next == "" {
			break
		}

		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		if nextURL.IsAbs() {
			if uri := nextURL.RequestURI(); uri != "" {
				path = uri
			} else {
				path = nextURL.String()
			}
		} else {
			path = nextURL.String()
		}
	}

	return pipelines, nil
}

// RepositoryListPage encapsulates paginated repository responses.
type repositoryListPage struct {
	Values []Repository `json:"values"`
	Next   string       `json:"next"`
}

// ListRepositories enumerates repositories for the workspace.
func (c *Client) ListRepositories(ctx context.Context, workspace string, limit int) ([]Repository, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}

	pageLen := limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 20
	}

	path := fmt.Sprintf("/repositories/%s?pagelen=%d",
		url.PathEscape(workspace),
		pageLen,
	)

	var repos []Repository

	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page repositoryListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		repos = append(repos, page.Values...)

		if limit > 0 && len(repos) >= limit {
			repos = repos[:limit]
			break
		}

		if page.Next == "" {
			break
		}

		// Bitbucket returns absolute URLs for next; reuse as-is.
		pathURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = pathURL.RequestURI()
	}

	return repos, nil
}

// GetRepository retrieves repository details.
func (c *Client) GetRepository(ctx context.Context, workspace, repoSlug string) (*Repository, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.http.Do(req, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// CreateRepositoryInput describes repository creation parameters.
type CreateRepositoryInput struct {
	Slug        string
	Name        string
	Description string
	IsPrivate   bool
	ProjectKey  string
}

// CreateRepository creates a repository within the workspace.
func (c *Client) CreateRepository(ctx context.Context, workspace string, input CreateRepositoryInput) (*Repository, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}
	if input.Slug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}

	body := map[string]any{
		"scm":        "git",
		"is_private": input.IsPrivate,
	}

	if input.Name != "" {
		body["name"] = input.Name
	}
	if input.Description != "" {
		body["description"] = input.Description
	}
	if input.ProjectKey != "" {
		body["project"] = map[string]any{
			"key": input.ProjectKey,
		}
	}

	path := fmt.Sprintf("/repositories/%s/%s",
		url.PathEscape(workspace),
		url.PathEscape(input.Slug),
	)
	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.http.Do(req, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// TriggerPipelineInput configures a pipeline run.
type TriggerPipelineInput struct {
	Ref       string
	Variables map[string]string
}

// TriggerPipeline triggers a new pipeline for the repo.
func (c *Client) TriggerPipeline(ctx context.Context, workspace, repoSlug string, in TriggerPipelineInput) (*Pipeline, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if in.Ref == "" {
		return nil, fmt.Errorf("ref is required")
	}

	body := map[string]any{
		"target": map[string]any{
			"ref_type": "branch",
			"type":     "pipeline_ref_target",
			"ref_name": in.Ref,
		},
	}
	if len(in.Variables) > 0 {
		vars := make([]map[string]any, 0, len(in.Variables))
		for k, v := range in.Variables {
			vars = append(vars, map[string]any{
				"key":     k,
				"value":   v,
				"secured": false,
			})
		}
		body["variables"] = vars
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines/",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var pipeline Pipeline
	if err := c.http.Do(req, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// GetPipeline fetches pipeline details.
func (c *Client) GetPipeline(ctx context.Context, workspace, repoSlug, uuid string) (*Pipeline, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(normalizeUUID(uuid)),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var pipeline Pipeline
	if err := c.http.Do(req, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// GetPipelineByBuildNumber fetches a pipeline by its build number.
func (c *Client) GetPipelineByBuildNumber(ctx context.Context, workspace, repoSlug string, buildNumber int) (*Pipeline, error) {
	// Bitbucket Cloud API supports querying by build number via the same endpoint
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		buildNumber,
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var pipeline Pipeline
	if err := c.http.Do(req, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// PipelineStep represents an individual pipeline step execution.
type PipelineStep struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	State struct {
		Name string `json:"name"`
	} `json:"state"`
	Result struct {
		Name string `json:"name"`
	} `json:"result"`
}

// ListPipelineSteps enumerates step executions for the pipeline.
func (c *Client) ListPipelineSteps(ctx context.Context, workspace, repoSlug, pipelineUUID string) ([]PipelineStep, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(normalizeUUID(pipelineUUID)),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []PipelineStep `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

// PipelineLog represents a step log chunk.
type PipelineLog struct {
	StepUUID string `json:"step_uuid"`
	Type     string `json:"type"`
	Log      string `json:"log"`
}

// CommitStatus describes build status for a commit.
// Type alias to shared types.CommitStatus for backward compatibility.
type CommitStatus = types.CommitStatus

// GetPipelineLogs fetches logs for a pipeline step.
func (c *Client) GetPipelineLogs(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string) ([]byte, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/%s/log",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(normalizeUUID(pipelineUUID)),
		url.PathEscape(normalizeUUID(stepUUID)),
	)

	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	// Override Accept header - logs endpoint returns octet-stream, not JSON
	req.Header.Set("Accept", "application/octet-stream")

	var buf strings.Builder
	if err := c.http.Do(req, &buf); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// CommitStatuses returns build statuses for a commit.
func (c *Client) CommitStatuses(ctx context.Context, workspace, repoSlug, commit string) ([]CommitStatus, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if commit == "" {
		return nil, fmt.Errorf("commit SHA is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/commit/%s/statuses",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(commit),
	)

	var statuses []CommitStatus
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Values []CommitStatus `json:"values"`
			Next   string         `json:"next"`
		}
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		statuses = append(statuses, resp.Values...)

		if resp.Next == "" {
			break
		}
		nextURL, err := url.Parse(resp.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return statuses, nil
}
