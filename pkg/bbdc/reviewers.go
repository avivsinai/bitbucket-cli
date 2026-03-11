package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// ReviewerGroup represents a Bitbucket default reviewer group association.
type ReviewerGroup struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type defaultReviewerCondition struct {
	ID                int                       `json:"id"`
	SourceRefMatcher  defaultReviewerRefMatcher `json:"sourceRefMatcher"`
	TargetRefMatcher  defaultReviewerRefMatcher `json:"targetRefMatcher"`
	Reviewers         []defaultReviewerGroup    `json:"reviewers"`
	ReviewerGroups    []defaultReviewerGroup    `json:"reviewerGroups"`
	RequiredApprovals int                       `json:"requiredApprovals"`
}

type defaultReviewerRefMatcher struct {
	ID        string `json:"id"`
	DisplayID string `json:"displayId"`
	Type      struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"type"`
}

type defaultReviewerGroup struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Users       []User `json:"users"`
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

// GetDefaultReviewers returns the users required as reviewers for a pull request
// from sourceRef to targetRef in the given repository.
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
		params.Set("sourceRefId", normalizeRefID(sourceRef))
	}
	if targetRef != "" {
		params.Set("targetRefId", normalizeRefID(targetRef))
	}
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := c.http.NewRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create default reviewers request: %w", err)
	}

	var conditions []defaultReviewerCondition
	if err := c.http.Do(req, &conditions); err != nil {
		return nil, fmt.Errorf("fetch default reviewers: %w", err)
	}

	reviewers := make([]User, 0)
	seen := make(map[string]struct{})
	for _, condition := range conditions {
		groups := append([]defaultReviewerGroup{}, condition.Reviewers...)
		groups = append(groups, condition.ReviewerGroups...)
		for _, group := range groups {
			for _, user := range group.Users {
				if _, ok := seen[user.Name]; ok {
					continue
				}
				seen[user.Name] = struct{}{}
				reviewers = append(reviewers, user)
			}
		}
	}

	return reviewers, nil
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

func normalizeRefID(ref string) string {
	if strings.HasPrefix(ref, "refs/") {
		return ref
	}
	return "refs/heads/" + ref
}
