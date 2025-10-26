package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// DiffStat aggregates additions/deletions for a pull request diff.
type DiffStat struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     int `json:"files"`
}

// PullRequestDiffStat retrieves diff statistics for the given pull request.
func (c *Client) PullRequestDiffStat(ctx context.Context, projectKey, repoSlug string, prID int) (*DiffStat, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/changes?withCounts=true&limit=1000",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	)

	stat := &DiffStat{}

	start := 0
	for {
		req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("%s&start=%d", u, start), nil)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Values []struct {
				Path struct {
					ToString string `json:"toString"`
				} `json:"path"`
				Stats struct {
					Additions int `json:"additions"`
					Deletions int `json:"deletions"`
				} `json:"stats"`
			} `json:"values"`
			IsLastPage    bool `json:"isLastPage"`
			NextPageStart int  `json:"nextPageStart"`
		}

		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		for _, change := range resp.Values {
			stat.Files++
			stat.Additions += change.Stats.Additions
			stat.Deletions += change.Stats.Deletions
		}

		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}
		start = resp.NextPageStart
	}

	return stat, nil
}
