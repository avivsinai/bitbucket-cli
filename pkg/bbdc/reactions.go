package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// Reaction represents an emoji reaction on a pull request comment.
type Reaction struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

// ListCommentReactions lists reactions for a given comment.
func (c *Client) ListCommentReactions(ctx context.Context, projectKey, repoSlug string, prID, commentID int) ([]Reaction, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/reactions",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
	), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []Reaction `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}

	return resp.Values, nil
}

// AddCommentReaction adds a reaction to a comment.
func (c *Client) AddCommentReaction(ctx context.Context, projectKey, repoSlug string, prID, commentID int, emoji string) error {
	if projectKey == "" || repoSlug == "" || emoji == "" {
		return fmt.Errorf("project key, repository slug, and emoji are required")
	}

	body := map[string]any{"emoji": emoji}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/reactions",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// RemoveCommentReaction removes a reaction from a comment.
func (c *Client) RemoveCommentReaction(ctx context.Context, projectKey, repoSlug string, prID, commentID int, emoji string) error {
	if projectKey == "" || repoSlug == "" || emoji == "" {
		return fmt.Errorf("project key, repository slug, and emoji are required")
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/reactions/%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
		url.PathEscape(emoji),
	), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}
