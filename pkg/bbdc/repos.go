package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// CreateRepositoryInput describes repository creation parameters.
type CreateRepositoryInput struct {
	Name          string
	SCMID         string
	Forkable      bool
	Public        bool
	Description   string
	DefaultBranch string
}

// CreateRepository creates a repository within the given project.
func (c *Client) CreateRepository(ctx context.Context, projectKey string, in CreateRepositoryInput) (*Repository, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("repository name is required")
	}

	body := map[string]any{
		"name":        in.Name,
		"scmId":       valueOrDefault(in.SCMID, "git"),
		"forkable":    in.Forkable,
		"public":      in.Public,
		"description": in.Description,
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos", url.PathEscape(projectKey)), body)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.http.Do(req, &repo); err != nil {
		return nil, err
	}

	if in.DefaultBranch != "" {
		if err := c.SetDefaultBranch(ctx, projectKey, repo.Slug, in.DefaultBranch); err != nil {
			return nil, err
		}
		repo.DefaultBranch = in.DefaultBranch
	}

	return &repo, nil
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
