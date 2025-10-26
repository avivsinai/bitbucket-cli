package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// AutoMergeSettings controls automatic merge behaviour when all checks pass.
type AutoMergeSettings struct {
	Enabled       bool   `json:"enabled"`
	StrategyID    string `json:"strategyId,omitempty"`
	CommitMessage string `json:"commitMessage,omitempty"`
	CloseSource   bool   `json:"closeSourceBranch"`
}

// GetAutoMerge retrieves auto-merge settings for a pull request.
func (c *Client) GetAutoMerge(ctx context.Context, projectKey, repoSlug string, prID int) (*AutoMergeSettings, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/auto-merge",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), nil)
	if err != nil {
		return nil, err
	}

	var settings AutoMergeSettings
	if err := c.http.Do(req, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// EnableAutoMerge enables automatic merge for the pull request using the given strategy.
func (c *Client) EnableAutoMerge(ctx context.Context, projectKey, repoSlug string, prID int, settings AutoMergeSettings) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}
	settings.Enabled = true

	req, err := c.http.NewRequest(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/auto-merge",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), settings)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// DisableAutoMerge removes any automatic merge configuration for the pull request.
func (c *Client) DisableAutoMerge(ctx context.Context, projectKey, repoSlug string, prID int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/auto-merge",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}
