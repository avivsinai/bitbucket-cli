package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Branch represents a repository branch.
type Branch struct {
	ID           string `json:"id"`
	DisplayID    string `json:"displayId"`
	Type         string `json:"type"`
	LatestCommit string `json:"latestCommit"`
	IsDefault    bool   `json:"isDefault"`
}

// BranchListOptions filters branch listing.
type BranchListOptions struct {
	Filter string
	Limit  int
}

// ListBranches retrieves branches for a repository.
func (c *Client) ListBranches(ctx context.Context, projectKey, repoSlug string, opts BranchListOptions) ([]Branch, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	query := fmt.Sprintf("limit=%d", valueOrPositive(opts.Limit, 25))
	if opts.Filter != "" {
		query += "&filterText=" + url.QueryEscape(opts.Filter)
	}

	start := 0
	var branches []Branch

	for {
		u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/branches?%s&start=%d",
			url.PathEscape(projectKey),
			url.PathEscape(repoSlug),
			query,
			start,
		)
		req, err := c.http.NewRequest(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}

		var resp paged[Branch]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		branches = append(branches, resp.Values...)

		if resp.IsLastPage || len(resp.Values) == 0 || (opts.Limit > 0 && len(branches) >= opts.Limit) {
			if opts.Limit > 0 && len(branches) > opts.Limit {
				branches = branches[:opts.Limit]
			}
			break
		}

		start = resp.NextPageStart
	}

	return branches, nil
}

// CreateBranchInput describes branch creation payload.
type CreateBranchInput struct {
	Name       string
	StartPoint string
	Message    string
}

// CreateBranch creates a new branch within the repository.
func (c *Client) CreateBranch(ctx context.Context, projectKey, repoSlug string, in CreateBranchInput) (*Branch, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("branch name is required")
	}
	if in.StartPoint == "" {
		return nil, fmt.Errorf("start point (commit or branch) is required")
	}

	body := map[string]any{
		"name":       ensureRef(in.Name),
		"startPoint": ensureRef(in.StartPoint),
	}
	if in.Message != "" {
		body["message"] = in.Message
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/branch-utils/1.0/projects/%s/repos/%s/branches",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return nil, err
	}

	var branch Branch
	if err := c.http.Do(req, &branch); err != nil {
		return nil, err
	}
	return &branch, nil
}

// DeleteBranch removes a branch from the repository.
func (c *Client) DeleteBranch(ctx context.Context, projectKey, repoSlug, branch string, dryRun bool) error {
	if projectKey == "" || repoSlug == "" || branch == "" {
		return fmt.Errorf("project key, repository slug, and branch are required")
	}

	body := map[string]any{
		"name":   ensureRef(branch),
		"dryRun": dryRun,
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/branch-utils/1.0/projects/%s/repos/%s/branches",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// SetDefaultBranch updates the default branch for a repository.
func (c *Client) SetDefaultBranch(ctx context.Context, projectKey, repoSlug, branch string) error {
	if projectKey == "" || repoSlug == "" || branch == "" {
		return fmt.Errorf("project key, repository slug, and branch are required")
	}

	body := map[string]any{
		"id": ensureRef(branch),
	}

	req, err := c.http.NewRequest(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/settings/default-branch",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

func ensureRef(ref string) string {
	if strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/tags/") {
		return ref
	}
	return "refs/heads/" + ref
}

func valueOrPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
