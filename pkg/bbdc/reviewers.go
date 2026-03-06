package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// ReviewerGroup represents a Bitbucket default reviewer group association.
type ReviewerGroup struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// ListReviewerGroups returns reviewer groups associated with a repository's default reviewers.
func (c *Client) ListReviewerGroups(ctx context.Context, projectKey, repoSlug string) ([]ReviewerGroup, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-reviewers/groups",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), nil)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Values []ReviewerGroup `json:"values"`
	}
	if err := c.http.Do(req, &payload); err != nil {
		return nil, err
	}

	return payload.Values, nil
}

// AddReviewerGroup adds a reviewer group to the repository default reviewers.
func (c *Client) AddReviewerGroup(ctx context.Context, projectKey, repoSlug, group string) error {
	if projectKey == "" || repoSlug == "" || group == "" {
		return fmt.Errorf("project key, repository slug, and group name are required")
	}

	endpoint := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-reviewers/groups?name=%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.QueryEscape(group),
	)

	req, err := c.http.NewRequest(ctx, "PUT", endpoint, nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// ReviewerCondition represents a default reviewer condition returned by the
// default-reviewers plugin. Each condition may match on source/target branch
// patterns and includes a list of reviewers.
type ReviewerCondition struct {
	ID              int    `json:"id"`
	SourceRefMatcher refMatcher `json:"sourceRefMatcher"`
	TargetRefMatcher refMatcher `json:"targetRefMatcher"`
	Reviewers        []User    `json:"reviewers"`
	RequiredApprovals int      `json:"requiredApprovals"`
}

type refMatcher struct {
	ID   string `json:"id"`
	Type struct {
		ID string `json:"id"`
	} `json:"type"`
	Active bool `json:"active"`
}

// GetDefaultReviewers returns the users configured as default reviewers for
// pull requests from sourceRef to targetRef in the given repository.
// It uses the dedicated reviewers endpoint which resolves conditions
// server-side based on source and target refs.
func (c *Client) GetDefaultReviewers(ctx context.Context, projectKey, repoSlug, sourceRef, targetRef string) ([]User, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	endpoint := fmt.Sprintf("/rest/default-reviewers/1.0/projects/%s/repos/%s/reviewers",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	)

	params := url.Values{}
	if sourceRef != "" {
		params.Set("sourceRefId", sourceRef)
	}
	if targetRef != "" {
		params.Set("targetRefId", targetRef)
	}
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := c.http.NewRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := c.http.Do(req, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// RemoveReviewerGroup removes a reviewer group association from repository defaults.
func (c *Client) RemoveReviewerGroup(ctx context.Context, projectKey, repoSlug, group string) error {
	if projectKey == "" || repoSlug == "" || group == "" {
		return fmt.Errorf("project key, repository slug, and group name are required")
	}

	endpoint := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-reviewers/groups?name=%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.QueryEscape(group),
	)

	req, err := c.http.NewRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}
