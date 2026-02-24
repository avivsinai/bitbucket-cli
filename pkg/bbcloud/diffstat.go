package bbcloud

import (
	"context"
	"fmt"
	"net/url"
)

// DiffStatEntry represents the diff statistics for a single file.
type DiffStatEntry struct {
	Status       string `json:"status"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	OldPath      string `json:"old_path,omitempty"`
	NewPath      string `json:"new_path,omitempty"`
}

// DiffStatResult aggregates per-file diff statistics for a pull request.
type DiffStatResult struct {
	Entries      []DiffStatEntry `json:"entries"`
	TotalAdded   int             `json:"total_added"`
	TotalRemoved int             `json:"total_removed"`
}

// diffStatPage models a single page of the diffstat API response.
type diffStatPage struct {
	Values []struct {
		Status       string `json:"status"`
		LinesAdded   int    `json:"lines_added"`
		LinesRemoved int    `json:"lines_removed"`
		Old          *struct {
			Path string `json:"path"`
		} `json:"old"`
		New *struct {
			Path string `json:"path"`
		} `json:"new"`
	} `json:"values"`
	Next string `json:"next"`
}

// PullRequestDiffStat retrieves per-file diff statistics for the given pull request.
func (c *Client) PullRequestDiffStat(ctx context.Context, workspace, repoSlug string, prID int) (*DiffStatResult, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/diffstat",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
	)

	const maxDiffStatPages = 50 // safety limit: 50 pages × ~500 entries = ~25,000 files

	result := &DiffStatResult{}

	for pageNum := 0; path != "" && pageNum < maxDiffStatPages; pageNum++ {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page diffStatPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		for _, v := range page.Values {
			entry := DiffStatEntry{
				Status:       v.Status,
				LinesAdded:   v.LinesAdded,
				LinesRemoved: v.LinesRemoved,
			}
			if v.Old != nil {
				entry.OldPath = v.Old.Path
			}
			if v.New != nil {
				entry.NewPath = v.New.Path
			}
			result.Entries = append(result.Entries, entry)
			result.TotalAdded += v.LinesAdded
			result.TotalRemoved += v.LinesRemoved
		}

		if page.Next == "" {
			break
		}
		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return result, nil
}
