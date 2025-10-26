package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Branch represents a Bitbucket Cloud branch.
type Branch struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
		Type string `json:"type"`
	} `json:"target"`
	IsDefault bool `json:"default"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// BranchListOptions configure branch listings.
type BranchListOptions struct {
	Filter string
	Limit  int
}

// branchListPage wraps paginated branch responses.
type branchListPage struct {
	Values []Branch `json:"values"`
	Next   string   `json:"next"`
}

// ListBranches lists repository branches.
func (c *Client) ListBranches(ctx context.Context, workspace, repoSlug string, opts BranchListOptions) ([]Branch, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 30
	}

	var params []string
	params = append(params, fmt.Sprintf("pagelen=%d", pageLen))
	if strings.TrimSpace(opts.Filter) != "" {
		params = append(params, "q="+url.QueryEscape(fmt.Sprintf("name ~ \"%s\"", opts.Filter)))
	}

	path := fmt.Sprintf("/repositories/%s/%s/refs/branches?%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Join(params, "&"),
	)

	var branches []Branch
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page branchListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		branches = append(branches, page.Values...)

		if opts.Limit > 0 && len(branches) >= opts.Limit {
			branches = branches[:opts.Limit]
			break
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

	return branches, nil
}
