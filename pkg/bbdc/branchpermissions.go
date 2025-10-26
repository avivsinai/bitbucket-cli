package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// BranchRestriction represents a branch permission rule.
type BranchRestriction struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Matcher struct {
		ID   string `json:"id"`
		Type struct {
			ID string `json:"id"`
		} `json:"type"`
		DisplayID string `json:"displayId"`
	} `json:"matcher"`
	Users  []User   `json:"users"`
	Groups []string `json:"groups"`
}

// BranchRestrictionInput controls creation of a branch restriction.
type BranchRestrictionInput struct {
	Type        string
	MatcherID   string
	MatcherType string
	Users       []string
	Groups      []string
}

// ListBranchRestrictions lists restriction rules for the repository.
func (c *Client) ListBranchRestrictions(ctx context.Context, projectKey, repoSlug string) ([]BranchRestriction, error) {
	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/branch-permissions/2.0/projects/%s/repos/%s/restrictions",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []BranchRestriction `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}

	return resp.Values, nil
}

// CreateBranchRestriction creates a new restriction.
func (c *Client) CreateBranchRestriction(ctx context.Context, projectKey, repoSlug string, in BranchRestrictionInput) (*BranchRestriction, error) {
	if in.Type == "" {
		return nil, fmt.Errorf("restriction type is required")
	}
	if in.MatcherID == "" {
		return nil, fmt.Errorf("matcher id is required")
	}
	if in.MatcherType == "" {
		in.MatcherType = "BRANCH"
	}

	body := map[string]any{
		"type": map[string]any{"id": strings.ToUpper(in.Type)},
		"matcher": map[string]any{
			"id":        in.MatcherID,
			"displayId": in.MatcherID,
			"type": map[string]any{
				"id": strings.ToUpper(in.MatcherType),
			},
		},
	}

	if len(in.Users) > 0 {
		body["users"] = in.Users
	}
	if len(in.Groups) > 0 {
		body["groups"] = in.Groups
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/branch-permissions/2.0/projects/%s/repos/%s/restrictions",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return nil, err
	}

	var restriction BranchRestriction
	if err := c.http.Do(req, &restriction); err != nil {
		return nil, err
	}
	return &restriction, nil
}

// DeleteBranchRestriction deletes a restriction by id.
func (c *Client) DeleteBranchRestriction(ctx context.Context, projectKey, repoSlug string, restrictionID int) error {
	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/branch-permissions/2.0/projects/%s/repos/%s/restrictions/%d",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		restrictionID,
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}
