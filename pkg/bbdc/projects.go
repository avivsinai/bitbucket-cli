package bbdc

import (
	"context"
	"fmt"
)

// ListProjects enumerates projects visible to the authenticated user.
func (c *Client) ListProjects(ctx context.Context, limit int) ([]Project, error) {
	const defaultPageSize = 25

	var (
		start = 0
		found []Project
	)

	for {
		pageSize := defaultPageSize
		if limit > 0 {
			remaining := limit - len(found)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		path := fmt.Sprintf("/rest/api/1.0/projects?limit=%d&start=%d", pageSize, start)
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var resp paged[Project]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		found = append(found, resp.Values...)

		if limit > 0 && len(found) >= limit {
			found = found[:limit]
			break
		}

		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}

		start = resp.NextPageStart
	}

	return found, nil
}
