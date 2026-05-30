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
	sourceRef = strings.TrimSpace(sourceRef)
	targetRef = strings.TrimSpace(targetRef)
	if sourceRef == "" || targetRef == "" {
		return nil, fmt.Errorf("source and target refs are required")
	}

	repo, err := c.GetRepository(ctx, projectKey, repoSlug)
	if err != nil {
		return nil, fmt.Errorf("fetch repository: %w", err)
	}

	endpoint := fmt.Sprintf("/rest/default-reviewers/1.0/projects/%s/repos/%s/reviewers",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	)

	params := url.Values{}
	params.Set("sourceRepoId", fmt.Sprintf("%d", repo.ID))
	params.Set("targetRepoId", fmt.Sprintf("%d", repo.ID))
	params.Set("sourceRefId", defaultReviewerRefID(sourceRef))
	params.Set("targetRefId", defaultReviewerRefID(targetRef))
	endpoint += "?" + params.Encode()

	req, err := c.http.NewRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create default reviewers request: %w", err)
	}

	var reviewers []User
	if err := c.http.Do(req, &reviewers); err != nil {
		return nil, fmt.Errorf("fetch default reviewers: %w", err)
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

func defaultReviewerRefID(ref string) string {
	ref = strings.TrimPrefix(ref, "refs/heads/")
	ref = strings.TrimPrefix(ref, "refs/tags/")
	return ref
}
