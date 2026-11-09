package bbcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"gopkg.in/yaml.v3"
)

// PullRequestMergeability summarizes a read-only Cloud merge simulation.
// Checks retain every field returned by Bitbucket, including restriction details.
// Even CLEAN is not a guarantee: some conditions require an actual merge attempt.
type PullRequestMergeability struct {
	Mergeable        string            `json:"mergeable"`
	MergeStateStatus string            `json:"mergeStateStatus"`
	Checks           []json.RawMessage `json:"checks"`
}

// MarshalYAML preserves the JSON field names and renders raw checks as objects,
// rather than exposing their underlying byte slices as YAML integer sequences.
func (m PullRequestMergeability) MarshalYAML() (any, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	return node.Content[0], nil
}

type mergeabilityCheck struct {
	Type     string `json:"type"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
	Required *bool  `json:"required"`
	Blocking *bool  `json:"blocking"`
}

// GetPullRequestMergeability returns UNKNOWN along with an error if the check
// cannot be fetched or decoded. It never attempts a merge or changes the PR.
func (c *Client) GetPullRequestMergeability(ctx context.Context, workspace, repoSlug string, id int) (*PullRequestMergeability, error) {
	result := &PullRequestMergeability{
		Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN", Checks: []json.RawMessage{},
	}
	if workspace == "" || repoSlug == "" || id <= 0 {
		return result, fmt.Errorf("workspace, repository slug, and positive pull request id are required")
	}
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/mergeability/checks",
		url.PathEscape(workspace), url.PathEscape(repoSlug), id)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return result, err
	}
	var response struct {
		Values *[]json.RawMessage `json:"values"`
	}
	if err := c.http.Do(req, &response); err != nil {
		return result, fmt.Errorf("fetch mergeability checks: %w", err)
	}
	if response.Values == nil {
		return result, fmt.Errorf("mergeability response is missing checks")
	}
	checks := make([]mergeabilityCheck, len(*response.Values))
	for i, raw := range *response.Values {
		if err := json.Unmarshal(raw, &checks[i]); err != nil {
			return result, fmt.Errorf("decode mergeability check: %w", err)
		}
	}
	result.Checks = *response.Values
	known := len(checks) > 0
	blocked := false
	gitCount := 0
	statePassed, permissionPassed := false, false
	for _, check := range checks {
		if check.Blocking == nil || check.Required == nil || check.Type == "" {
			known = false
		}
		if check.Blocking != nil && *check.Blocking {
			blocked = true
		}
		switch check.Status {
		case "PASSED":
		case "FAILED", "SKIPPED":
			if check.Required == nil || *check.Required {
				known = false
			}
		default:
			known = false
		}
		switch check.Type {
		case "pullrequest_state_check":
			statePassed = check.Status == "PASSED"
		case "current_user_permission_check":
			permissionPassed = check.Status == "PASSED"
		case "git_mergeability_check":
			gitCount++
			if check.Status == "PASSED" && check.Reason == "clean" {
				result.Mergeable = "MERGEABLE"
			} else if check.Status == "FAILED" && check.Reason == "conflicts" {
				result.Mergeable = "CONFLICTING"
			}
		}
	}
	if gitCount != 1 {
		result.Mergeable = "UNKNOWN"
	}
	switch {
	case result.Mergeable == "CONFLICTING":
		result.MergeStateStatus = "DIRTY"
	case blocked:
		result.MergeStateStatus = "BLOCKED"
	case known && statePassed && permissionPassed && result.Mergeable == "MERGEABLE":
		result.MergeStateStatus = "CLEAN"
	}
	return result, nil
}
