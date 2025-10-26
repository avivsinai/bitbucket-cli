package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Permission represents a Bitbucket permission assignment.
type Permission struct {
	User       User   `json:"user"`
	Permission string `json:"permission"`
}

// ListRepoPermissions returns repository user permissions.
func (c *Client) ListRepoPermissions(ctx context.Context, projectKey, repoSlug string, limit int) ([]Permission, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	return c.listPermissions(ctx, fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/permissions/users",
		url.PathEscape(projectKey), url.PathEscape(repoSlug)), limit)
}

// ListProjectPermissions returns project user permissions.
func (c *Client) ListProjectPermissions(ctx context.Context, projectKey string, limit int) ([]Permission, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}
	return c.listPermissions(ctx, fmt.Sprintf("/rest/api/1.0/projects/%s/permissions/users", url.PathEscape(projectKey)), limit)
}

func (c *Client) listPermissions(ctx context.Context, path string, limit int) ([]Permission, error) {
	pageLimit := valueOrPositive(limit, 100)
	start := 0
	var out []Permission

	for {
		u := fmt.Sprintf("%s?limit=%d&start=%d", path, pageLimit, start)
		req, err := c.http.NewRequest(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}
		var resp paged[Permission]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Values...)
		if resp.IsLastPage || len(resp.Values) == 0 || (limit > 0 && len(out) >= limit) {
			if limit > 0 && len(out) > limit {
				out = out[:limit]
			}
			break
		}
		start = resp.NextPageStart
	}
	return out, nil
}

// GrantRepoPermission assigns a permission to a user for a repository.
func (c *Client) GrantRepoPermission(ctx context.Context, projectKey, repoSlug, username, permission string) error {
	if projectKey == "" || repoSlug == "" || username == "" || permission == "" {
		return fmt.Errorf("project, repo, username, and permission are required")
	}

	req, err := c.http.NewRequest(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/permissions/users?name=%s&permission=%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.QueryEscape(username),
		url.QueryEscape(strings.ToUpper(permission)),
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// GrantProjectPermission assigns a permission to a user for a project.
func (c *Client) GrantProjectPermission(ctx context.Context, projectKey, username, permission string) error {
	if projectKey == "" || username == "" || permission == "" {
		return fmt.Errorf("project key, username, and permission are required")
	}

	req, err := c.http.NewRequest(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/permissions/users?name=%s&permission=%s",
		url.PathEscape(projectKey),
		url.QueryEscape(username),
		url.QueryEscape(strings.ToUpper(permission)),
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RevokeRepoPermission removes a repository permission for a user.
func (c *Client) RevokeRepoPermission(ctx context.Context, projectKey, repoSlug, username string) error {
	if projectKey == "" || repoSlug == "" || username == "" {
		return fmt.Errorf("project, repo, and username are required")
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/permissions/users?name=%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.QueryEscape(username),
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RevokeProjectPermission removes a project permission for a user.
func (c *Client) RevokeProjectPermission(ctx context.Context, projectKey, username string) error {
	if projectKey == "" || username == "" {
		return fmt.Errorf("project key and username are required")
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/permissions/users?name=%s",
		url.PathEscape(projectKey),
		url.QueryEscape(username),
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}
