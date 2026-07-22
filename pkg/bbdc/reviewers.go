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
	return c.GetDefaultReviewersForRepositories(
		ctx,
		projectKey, repoSlug,
		projectKey, repoSlug,
		sourceRef, targetRef,
	)
}

// GetDefaultReviewersForRepositories returns the users required as reviewers
// for a pull request whose source and target refs may live in different
// repositories. The target repository owns the default-reviewer rules.
func (c *Client) GetDefaultReviewersForRepositories(
	ctx context.Context,
	targetProjectKey, targetRepoSlug string,
	sourceProjectKey, sourceRepoSlug string,
	sourceRef, targetRef string,
) ([]User, error) {
	if targetProjectKey == "" || targetRepoSlug == "" || sourceProjectKey == "" || sourceRepoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	sourceRef = strings.TrimSpace(sourceRef)
	targetRef = strings.TrimSpace(targetRef)
	if sourceRef == "" || targetRef == "" {
		return nil, fmt.Errorf("source and target refs are required")
	}

	targetRepo, err := c.GetRepository(ctx, targetProjectKey, targetRepoSlug)
	if err != nil {
		return nil, fmt.Errorf("fetch target repository: %w", err)
	}
	sourceRepo := targetRepo
	if !strings.EqualFold(sourceProjectKey, targetProjectKey) || !strings.EqualFold(sourceRepoSlug, targetRepoSlug) {
		sourceRepo, err = c.GetRepository(ctx, sourceProjectKey, sourceRepoSlug)
		if err != nil {
			return nil, fmt.Errorf("fetch source repository: %w", err)
		}
	}

	endpoint := fmt.Sprintf("/rest/default-reviewers/1.0/projects/%s/repos/%s/reviewers",
		url.PathEscape(targetProjectKey),
		url.PathEscape(targetRepoSlug),
	)

	params := url.Values{}
	params.Set("sourceRepoId", fmt.Sprintf("%d", sourceRepo.ID))
	params.Set("targetRepoId", fmt.Sprintf("%d", targetRepo.ID))
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
