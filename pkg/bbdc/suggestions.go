package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// Suggestion represents a code suggestion attached to a pull request comment.
type Suggestion struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Applied   bool   `json:"applied"`
	CommentID int    `json:"commentId"`
}

// ApplySuggestion applies a code suggestion identified by comment+suggestion id.
func (c *Client) ApplySuggestion(ctx context.Context, projectKey, repoSlug string, prID, commentID, suggestionID int) error {
	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/suggestions/%d/apply",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
		suggestionID,
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// SuggestionPreview fetches the suggestion details for inspection prior to applying.
func (c *Client) SuggestionPreview(ctx context.Context, projectKey, repoSlug string, prID, commentID, suggestionID int) (*Suggestion, error) {
	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/suggestions/%d",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
		suggestionID,
	), nil)
	if err != nil {
		return nil, err
	}

	var suggestion Suggestion
	if err := c.http.Do(req, &suggestion); err != nil {
		return nil, err
	}
	return &suggestion, nil
}
