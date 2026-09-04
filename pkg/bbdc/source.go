package bbdc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// ErrNotFound reports that a ref, path, or file does not exist.
var ErrNotFound = errors.New("not found")

// filesPageLimit is the page size for recursive file listings.
const filesPageLimit = 1000

// ListFiles returns the paths of every file under path at the given ref,
// recursively. Paths are relative to path, matching the Data Center API.
// An empty path lists the whole repository.
func (c *Client) ListFiles(ctx context.Context, projectKey, repoSlug, ref, path string) ([]string, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	base := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/files",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	)
	if escaped := escapePath(path); escaped != "" {
		base += "/" + escaped
	}

	var files []string
	start := 0
	for {
		apiPath := fmt.Sprintf("%s?limit=%d&start=%d", base, filesPageLimit, start)
		if ref != "" {
			apiPath += "&at=" + url.QueryEscape(ref)
		}

		req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
		if err != nil {
			return nil, err
		}
		var page paged[string]
		if err := c.http.Do(req, &page); err != nil {
			return nil, translateNotFound(err)
		}
		files = append(files, page.Values...)
		if page.IsLastPage || page.NextPageStart == 0 {
			break
		}
		start = page.NextPageStart
	}
	return files, nil
}

// ReadFile returns the raw bytes of a file at the given ref.
func (c *Client) ReadFile(ctx context.Context, projectKey, repoSlug, ref, path string) ([]byte, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	escaped := escapePath(path)
	if escaped == "" {
		return nil, fmt.Errorf("file path is required")
	}

	apiPath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/raw/%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		escaped,
	)
	if ref != "" {
		apiPath += "?at=" + url.QueryEscape(ref)
	}

	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return nil, err
	}
	// Raw files are arbitrary bytes, not JSON.
	req.Header.Set("Accept", "*/*")

	var buf strings.Builder
	if err := c.http.Do(req, &buf); err != nil {
		return nil, translateNotFound(err)
	}
	return []byte(buf.String()), nil
}

// Tag is a Bitbucket Data Center tag reference.
type Tag struct {
	ID           string `json:"id"`
	DisplayID    string `json:"displayId"`
	LatestCommit string `json:"latestCommit"`
	Hash         string `json:"hash"`
}

// Commit returns the commit the tag points at. For annotated tags the API
// reports the tag object in Hash and the commit in LatestCommit.
func (t Tag) Commit() string {
	if t.LatestCommit != "" {
		return t.LatestCommit
	}
	return t.Hash
}

// ListTags returns up to limit tags, most recently modified first.
func (c *Client) ListTags(ctx context.Context, projectKey, repoSlug string, limit int) ([]Tag, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	apiPath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/tags?orderBy=MODIFICATION&limit=%d",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		limit,
	)
	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return nil, err
	}
	var page paged[Tag]
	if err := c.http.Do(req, &page); err != nil {
		return nil, translateNotFound(err)
	}
	return page.Values, nil
}

// GetTag returns the commit a tag points at.
func (c *Client) GetTag(ctx context.Context, projectKey, repoSlug, name string) (string, error) {
	if projectKey == "" || repoSlug == "" {
		return "", fmt.Errorf("project key and repository slug are required")
	}
	apiPath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/tags/%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		escapePath(name),
	)
	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return "", err
	}
	var tag Tag
	if err := c.http.Do(req, &tag); err != nil {
		return "", translateNotFound(err)
	}
	if commit := tag.Commit(); commit != "" {
		return commit, nil
	}
	return "", fmt.Errorf("tag %q has no commit", name)
}

// GetBranch returns the head commit of a branch. Data Center has no lookup by
// exact name, so the filtered branch list is matched on displayId.
func (c *Client) GetBranch(ctx context.Context, projectKey, repoSlug, name string) (string, error) {
	branches, err := c.ListBranches(ctx, projectKey, repoSlug, BranchListOptions{Filter: name, Limit: 100})
	if err != nil {
		return "", translateNotFound(err)
	}
	for _, branch := range branches {
		if branch.DisplayID == name || branch.ID == "refs/heads/"+name {
			if branch.LatestCommit == "" {
				return "", fmt.Errorf("branch %q has no commit", name)
			}
			return branch.LatestCommit, nil
		}
	}
	return "", ErrNotFound
}

// GetDefaultBranch returns the repository's default branch name.
func (c *Client) GetDefaultBranch(ctx context.Context, projectKey, repoSlug string) (string, error) {
	if projectKey == "" || repoSlug == "" {
		return "", fmt.Errorf("project key and repository slug are required")
	}
	apiPath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-branch",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	)
	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return "", err
	}
	var branch Branch
	if err := c.http.Do(req, &branch); err != nil {
		return "", translateNotFound(err)
	}
	if branch.DisplayID == "" {
		return "", fmt.Errorf("repository %s/%s has no default branch", projectKey, repoSlug)
	}
	return branch.DisplayID, nil
}

// Commit is a Bitbucket Data Center commit.
type Commit struct {
	ID string `json:"id"`
}

// GetCommit resolves a commit reference to its full ID.
func (c *Client) GetCommit(ctx context.Context, projectKey, repoSlug, ref string) (string, error) {
	if projectKey == "" || repoSlug == "" {
		return "", fmt.Errorf("project key and repository slug are required")
	}
	apiPath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/commits/%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.PathEscape(ref),
	)
	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return "", err
	}
	var commit Commit
	if err := c.http.Do(req, &commit); err != nil {
		return "", translateNotFound(err)
	}
	if commit.ID == "" {
		return "", ErrNotFound
	}
	return commit.ID, nil
}

// LatestCommitForPath returns the most recent commit reachable from ref that
// touched path. It identifies the version of a directory: Bitbucket exposes no
// per-directory tree hash.
func (c *Client) LatestCommitForPath(ctx context.Context, projectKey, repoSlug, ref, path string) (string, error) {
	if projectKey == "" || repoSlug == "" {
		return "", fmt.Errorf("project key and repository slug are required")
	}

	apiPath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/commits?limit=1",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	)
	if ref != "" {
		apiPath += "&until=" + url.QueryEscape(ref)
	}
	if trimmed := strings.Trim(path, "/"); trimmed != "" {
		apiPath += "&path=" + url.QueryEscape(trimmed)
	}

	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return "", err
	}
	var page paged[Commit]
	if err := c.http.Do(req, &page); err != nil {
		return "", translateNotFound(err)
	}
	if len(page.Values) == 0 || page.Values[0].ID == "" {
		return "", fmt.Errorf("no commits found for %q at %s", path, ref)
	}
	return page.Values[0].ID, nil
}

// escapePath escapes each segment of a slash-separated path, keeping the
// separators intact.
func escapePath(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

// translateNotFound converts a 404 response into ErrNotFound so callers can
// distinguish "does not exist" from a transport or permission failure.
func translateNotFound(err error) error {
	var httpErr *httpx.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
		return fmt.Errorf("%w: %s", ErrNotFound, httpErr.Error())
	}
	return err
}
