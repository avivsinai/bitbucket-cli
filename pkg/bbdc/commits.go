package bbdc

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// CommitDiff streams the raw unified diff between two refs into w.
// from and to can be commit SHAs, branch names, or tags.
func (c *Client) CommitDiff(ctx context.Context, projectKey, repoSlug, from, to string, w io.Writer) error {
	if projectKey == "" {
		return fmt.Errorf("project key is required")
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug is required")
	}
	if from == "" {
		return fmt.Errorf("from ref is required")
	}
	if to == "" {
		return fmt.Errorf("to ref is required")
	}
	if w == nil {
		return fmt.Errorf("writer is required")
	}

	params := url.Values{}
	params.Set("from", from)
	params.Set("to", to)
	path := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/compare/diff?%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		params.Encode(),
	)

	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/plain")

	return c.http.Do(req, w)
}
