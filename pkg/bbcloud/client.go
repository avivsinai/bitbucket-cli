package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
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
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Pipeline represents a pipeline execution.
type Pipeline struct {
	UUID  string `json:"uuid"`
	State struct {
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

	path := fmt.Sprintf("/repositories/%s/%s/pipelines/?pagelen=%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		pageLen,
	)

	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var page PipelinePage
	if err := c.http.Do(req, &page); err != nil {
		return nil, err
	}

	return page.Values, nil
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
		strings.Trim(uuid, "{}"),
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
		strings.Trim(pipelineUUID, "{}"),
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

// GetPipelineLogs fetches logs for a pipeline step.
func (c *Client) GetPipelineLogs(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string) ([]byte, error) {
	pipelineUUID = strings.Trim(pipelineUUID, "{}")
	stepUUID = strings.Trim(stepUUID, "{}")
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/%s/log",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		pipelineUUID,
		stepUUID,
	)

	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var buf strings.Builder
	if err := c.http.Do(req, &buf); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}
