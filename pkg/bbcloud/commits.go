package bbcloud

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// CommitDiff streams the raw unified diff between two refs into w.
// spec must be "ref1..ref2" where refs can be commit SHAs, branch names, or tags.
func (c *Client) CommitDiff(ctx context.Context, workspace, repoSlug, spec string, w io.Writer) error {
	if workspace == "" {
		return fmt.Errorf("workspace is required")
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug is required")
	}
	if spec == "" {
		return fmt.Errorf("spec is required")
	}
	if w == nil {
		return fmt.Errorf("writer is required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/diff/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(spec),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/plain")

	return c.http.Do(req, w)
}
