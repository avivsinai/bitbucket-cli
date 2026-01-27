package bbcloud

import (
	"context"
	"fmt"
	"net/url"
)

// PipelineVariable represents a Bitbucket Cloud pipeline variable.
type PipelineVariable struct {
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
	Value   string `json:"value,omitempty"`
	Secured bool   `json:"secured"`
}

// VariableListOptions configures variable list requests.
type VariableListOptions struct {
	Limit int
}

type variableListPage struct {
	Values []PipelineVariable `json:"values"`
	Next   string             `json:"next"`
}

// ListRepositoryVariables lists pipeline variables for a repository.
func (c *Client) ListRepositoryVariables(ctx context.Context, workspace, repoSlug string, opts VariableListOptions) ([]PipelineVariable, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 100
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines_config/variables?pagelen=%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		pageLen,
	)

	var variables []PipelineVariable
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page variableListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		variables = append(variables, page.Values...)

		if opts.Limit > 0 && len(variables) >= opts.Limit {
			variables = variables[:opts.Limit]
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

	return variables, nil
}

// CreateRepositoryVariableInput configures repository variable creation.
type CreateRepositoryVariableInput struct {
	Key     string
	Value   string
	Secured bool
}

// CreateRepositoryVariable creates a pipeline variable for a repository.
func (c *Client) CreateRepositoryVariable(ctx context.Context, workspace, repoSlug string, input CreateRepositoryVariableInput) (*PipelineVariable, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if input.Key == "" {
		return nil, fmt.Errorf("variable key is required")
	}

	body := map[string]any{
		"key":     input.Key,
		"value":   input.Value,
		"secured": input.Secured,
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines_config/variables",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var variable PipelineVariable
	if err := c.http.Do(req, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// UpdateRepositoryVariableInput configures repository variable updates.
type UpdateRepositoryVariableInput struct {
	Key     string
	Value   string
	Secured bool
}

// UpdateRepositoryVariable updates a pipeline variable for a repository.
// The variableUUID identifies the variable to update.
func (c *Client) UpdateRepositoryVariable(ctx context.Context, workspace, repoSlug, variableUUID string, input UpdateRepositoryVariableInput) (*PipelineVariable, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if variableUUID == "" {
		return nil, fmt.Errorf("variable UUID is required")
	}
	if input.Key == "" {
		return nil, fmt.Errorf("variable key is required")
	}

	body := map[string]any{
		"key":     input.Key,
		"value":   input.Value,
		"secured": input.Secured,
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines_config/variables/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(variableUUID),
	)

	req, err := c.http.NewRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	var variable PipelineVariable
	if err := c.http.Do(req, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// DeleteRepositoryVariable deletes a pipeline variable from a repository.
func (c *Client) DeleteRepositoryVariable(ctx context.Context, workspace, repoSlug, variableUUID string) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if variableUUID == "" {
		return fmt.Errorf("variable UUID is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines_config/variables/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(variableUUID),
	)

	req, err := c.http.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// --- Workspace-level variable methods ---

// ListWorkspaceVariables lists pipeline variables for a workspace.
func (c *Client) ListWorkspaceVariables(ctx context.Context, workspace string, opts VariableListOptions) ([]PipelineVariable, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 100
	}

	path := fmt.Sprintf("/workspaces/%s/pipelines-config/variables?pagelen=%d",
		url.PathEscape(workspace),
		pageLen,
	)

	var variables []PipelineVariable
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page variableListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		variables = append(variables, page.Values...)

		if opts.Limit > 0 && len(variables) >= opts.Limit {
			variables = variables[:opts.Limit]
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

	return variables, nil
}

// CreateWorkspaceVariableInput configures workspace variable creation.
type CreateWorkspaceVariableInput struct {
	Key     string
	Value   string
	Secured bool
}

// CreateWorkspaceVariable creates a pipeline variable for a workspace.
func (c *Client) CreateWorkspaceVariable(ctx context.Context, workspace string, input CreateWorkspaceVariableInput) (*PipelineVariable, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}
	if input.Key == "" {
		return nil, fmt.Errorf("variable key is required")
	}

	body := map[string]any{
		"key":     input.Key,
		"value":   input.Value,
		"secured": input.Secured,
	}

	path := fmt.Sprintf("/workspaces/%s/pipelines-config/variables",
		url.PathEscape(workspace),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var variable PipelineVariable
	if err := c.http.Do(req, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// UpdateWorkspaceVariableInput configures workspace variable updates.
type UpdateWorkspaceVariableInput struct {
	Key     string
	Value   string
	Secured bool
}

// UpdateWorkspaceVariable updates a pipeline variable for a workspace.
func (c *Client) UpdateWorkspaceVariable(ctx context.Context, workspace, variableUUID string, input UpdateWorkspaceVariableInput) (*PipelineVariable, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}
	if variableUUID == "" {
		return nil, fmt.Errorf("variable UUID is required")
	}
	if input.Key == "" {
		return nil, fmt.Errorf("variable key is required")
	}

	body := map[string]any{
		"key":     input.Key,
		"value":   input.Value,
		"secured": input.Secured,
	}

	path := fmt.Sprintf("/workspaces/%s/pipelines-config/variables/%s",
		url.PathEscape(workspace),
		url.PathEscape(variableUUID),
	)

	req, err := c.http.NewRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	var variable PipelineVariable
	if err := c.http.Do(req, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// DeleteWorkspaceVariable deletes a pipeline variable from a workspace.
func (c *Client) DeleteWorkspaceVariable(ctx context.Context, workspace, variableUUID string) error {
	if workspace == "" {
		return fmt.Errorf("workspace is required")
	}
	if variableUUID == "" {
		return fmt.Errorf("variable UUID is required")
	}

	path := fmt.Sprintf("/workspaces/%s/pipelines-config/variables/%s",
		url.PathEscape(workspace),
		url.PathEscape(variableUUID),
	)

	req, err := c.http.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// --- Deployment environment variable methods ---

// DeploymentEnvironment represents a deployment environment in Bitbucket Cloud.
type DeploymentEnvironment struct {
	UUID            string `json:"uuid"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	EnvironmentType struct {
		Name string `json:"name"`
	} `json:"environment_type"`
}

type deploymentEnvironmentListPage struct {
	Values []DeploymentEnvironment `json:"values"`
	Next   string                  `json:"next"`
}

// ListDeploymentEnvironments lists deployment environments for a repository.
func (c *Client) ListDeploymentEnvironments(ctx context.Context, workspace, repoSlug string) ([]DeploymentEnvironment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/environments?pagelen=100",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	var environments []DeploymentEnvironment
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page deploymentEnvironmentListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		environments = append(environments, page.Values...)

		if page.Next == "" {
			break
		}

		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return environments, nil
}

// ListDeploymentVariables lists pipeline variables for a deployment environment.
func (c *Client) ListDeploymentVariables(ctx context.Context, workspace, repoSlug, environmentUUID string, opts VariableListOptions) ([]PipelineVariable, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if environmentUUID == "" {
		return nil, fmt.Errorf("environment UUID is required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 100
	}

	path := fmt.Sprintf("/repositories/%s/%s/deployments_config/environments/%s/variables?pagelen=%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(environmentUUID),
		pageLen,
	)

	var variables []PipelineVariable
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page variableListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		variables = append(variables, page.Values...)

		if opts.Limit > 0 && len(variables) >= opts.Limit {
			variables = variables[:opts.Limit]
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

	return variables, nil
}

// CreateDeploymentVariableInput configures deployment variable creation.
type CreateDeploymentVariableInput struct {
	Key     string
	Value   string
	Secured bool
}

// CreateDeploymentVariable creates a pipeline variable for a deployment environment.
func (c *Client) CreateDeploymentVariable(ctx context.Context, workspace, repoSlug, environmentUUID string, input CreateDeploymentVariableInput) (*PipelineVariable, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if environmentUUID == "" {
		return nil, fmt.Errorf("environment UUID is required")
	}
	if input.Key == "" {
		return nil, fmt.Errorf("variable key is required")
	}

	body := map[string]any{
		"key":     input.Key,
		"value":   input.Value,
		"secured": input.Secured,
	}

	path := fmt.Sprintf("/repositories/%s/%s/deployments_config/environments/%s/variables",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(environmentUUID),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var variable PipelineVariable
	if err := c.http.Do(req, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// UpdateDeploymentVariableInput configures deployment variable updates.
type UpdateDeploymentVariableInput struct {
	Key     string
	Value   string
	Secured bool
}

// UpdateDeploymentVariable updates a pipeline variable for a deployment environment.
func (c *Client) UpdateDeploymentVariable(ctx context.Context, workspace, repoSlug, environmentUUID, variableUUID string, input UpdateDeploymentVariableInput) (*PipelineVariable, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if environmentUUID == "" {
		return nil, fmt.Errorf("environment UUID is required")
	}
	if variableUUID == "" {
		return nil, fmt.Errorf("variable UUID is required")
	}
	if input.Key == "" {
		return nil, fmt.Errorf("variable key is required")
	}

	body := map[string]any{
		"key":     input.Key,
		"value":   input.Value,
		"secured": input.Secured,
	}

	path := fmt.Sprintf("/repositories/%s/%s/deployments_config/environments/%s/variables/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(environmentUUID),
		url.PathEscape(variableUUID),
	)

	req, err := c.http.NewRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	var variable PipelineVariable
	if err := c.http.Do(req, &variable); err != nil {
		return nil, err
	}
	return &variable, nil
}

// DeleteDeploymentVariable deletes a pipeline variable from a deployment environment.
func (c *Client) DeleteDeploymentVariable(ctx context.Context, workspace, repoSlug, environmentUUID, variableUUID string) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if environmentUUID == "" {
		return fmt.Errorf("environment UUID is required")
	}
	if variableUUID == "" {
		return fmt.Errorf("variable UUID is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/deployments_config/environments/%s/variables/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(environmentUUID),
		url.PathEscape(variableUUID),
	)

	req, err := c.http.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}
