package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
)

// CodeSearchResult is one file matched by Bitbucket Cloud code search.
type CodeSearchResult struct {
	ContentMatchCount int                  `json:"content_match_count"`
	File              CodeSearchResultFile `json:"file"`
}

type CodeSearchResultFile struct {
	Path  string `json:"path"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links,omitempty"`
	Commit struct {
		Repository struct {
			FullName string `json:"full_name"`
			Name     string `json:"name"`
			Slug     string `json:"slug"`
		} `json:"repository"`
	} `json:"commit,omitempty"`
}

type codeSearchPage struct {
	Next   string             `json:"next"`
	Values []CodeSearchResult `json:"values"`
}

// SearchWorkspaceSkills searches SKILL.md files across repositories in a workspace.
// The limit applies to skill files after defensive client-side filtering.
func (c *Client) SearchWorkspaceSkills(ctx context.Context, workspace, query string, limit int) ([]CodeSearchResult, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if limit <= 0 {
		return []CodeSearchResult{}, nil
	}

	endpoint := "/workspaces/" + url.PathEscape(workspace) + "/search/code"
	pageLen := min(limit, 100)
	params := url.Values{}
	params.Set("search_query", "("+query+") path:SKILL.md")
	params.Set("pagelen", strconv.Itoa(pageLen))
	// Repository identity is not part of the default result projection.
	params.Set("fields", "+values.file.commit.repository")
	next := endpoint + "?" + params.Encode()

	results := make([]CodeSearchResult, 0, min(limit, pageLen))
	for next != "" && len(results) < limit {
		req, err := c.http.NewRequest(ctx, "GET", next, nil)
		if err != nil {
			return nil, err
		}
		var page codeSearchPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}
		for _, result := range page.Values {
			if path.Base(result.File.Path) != "SKILL.md" {
				continue
			}
			results = append(results, result)
			if len(results) >= limit {
				break
			}
		}
		if page.Next == "" || len(results) >= limit {
			break
		}
		next, err = normalizeNextRef(page.Next, endpoint)
		if err != nil {
			return nil, fmt.Errorf("invalid code search next page: %w", err)
		}
	}
	return results, nil
}
