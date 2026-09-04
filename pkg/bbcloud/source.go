package bbcloud

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

// srcMaxDepth bounds recursive source listings. Skill directories are shallow;
// this keeps a single request from walking an entire monorepo.
const srcMaxDepth = 10

// SourceEntry is one file in a repository listing.
type SourceEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // commit_file or commit_directory
	Size int64  `json:"size"`
}

type sourcePage struct {
	Values []SourceEntry `json:"values"`
	Next   string        `json:"next"`
}

// ListSource lists the files under path at the given commit, recursively.
// An empty path lists the whole repository. Directories are omitted.
//
// The request must not set format=meta: on a directory that returns metadata
// about the directory itself rather than a listing of its contents.
func (c *Client) ListSource(ctx context.Context, workspace, repoSlug, commit, path string) ([]SourceEntry, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if commit == "" {
		return nil, fmt.Errorf("commit is required")
	}

	next := fmt.Sprintf("/repositories/%s/%s/src/%s/%s?pagelen=100&max_depth=%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(commit),
		escapePath(path),
		srcMaxDepth,
	)

	var files []SourceEntry
	for next != "" {
		req, err := c.http.NewRequest(ctx, "GET", next, nil)
		if err != nil {
			return nil, err
		}
		var page sourcePage
		if err := c.http.Do(req, &page); err != nil {
			return nil, translateNotFound(err)
		}
		for _, entry := range page.Values {
			if entry.Type == "commit_file" {
				files = append(files, entry)
			}
		}
		next = page.Next
	}
	return files, nil
}

// ReadSource returns the raw bytes of a file at the given commit.
func (c *Client) ReadSource(ctx context.Context, workspace, repoSlug, commit, path string) ([]byte, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("file path is required")
	}

	apiPath := fmt.Sprintf("/repositories/%s/%s/src/%s/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(commit),
		escapePath(path),
	)

	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return nil, err
	}
	// Source files are arbitrary bytes, not JSON.
	req.Header.Set("Accept", "*/*")

	var buf strings.Builder
	if err := c.http.Do(req, &buf); err != nil {
		return nil, translateNotFound(err)
	}
	return []byte(buf.String()), nil
}

// Tag is a Bitbucket Cloud tag reference.
type Tag struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

type tagPage struct {
	Values []Tag  `json:"values"`
	Next   string `json:"next"`
}

// ListTags returns up to limit tags, newest first by target commit date.
func (c *Client) ListTags(ctx context.Context, workspace, repoSlug string, limit int) ([]Tag, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	pageLen := limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 100
	}

	apiPath := fmt.Sprintf("/repositories/%s/%s/refs/tags?sort=-target.date&pagelen=%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		pageLen,
	)
	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return nil, err
	}
	var page tagPage
	if err := c.http.Do(req, &page); err != nil {
		return nil, translateNotFound(err)
	}
	if limit > 0 && len(page.Values) > limit {
		return page.Values[:limit], nil
	}
	return page.Values, nil
}

// GetTag returns the commit a tag points at.
func (c *Client) GetTag(ctx context.Context, workspace, repoSlug, name string) (string, error) {
	if workspace == "" || repoSlug == "" {
		return "", fmt.Errorf("workspace and repository slug are required")
	}
	apiPath := fmt.Sprintf("/repositories/%s/%s/refs/tags/%s",
		url.PathEscape(workspace),
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
	if tag.Target.Hash == "" {
		return "", fmt.Errorf("tag %q has no target commit", name)
	}
	return tag.Target.Hash, nil
}

// GetBranch returns the head commit of a branch.
func (c *Client) GetBranch(ctx context.Context, workspace, repoSlug, name string) (string, error) {
	if workspace == "" || repoSlug == "" {
		return "", fmt.Errorf("workspace and repository slug are required")
	}
	apiPath := fmt.Sprintf("/repositories/%s/%s/refs/branches/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		escapePath(name),
	)
	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return "", err
	}
	var branch Branch
	if err := c.http.Do(req, &branch); err != nil {
		return "", translateNotFound(err)
	}
	if branch.Target.Hash == "" {
		return "", fmt.Errorf("branch %q has no target commit", name)
	}
	return branch.Target.Hash, nil
}

// Commit is a Bitbucket Cloud commit.
type Commit struct {
	Hash string `json:"hash"`
}

type commitPage struct {
	Values []Commit `json:"values"`
}

// GetCommit resolves a commit reference to its full hash.
func (c *Client) GetCommit(ctx context.Context, workspace, repoSlug, ref string) (string, error) {
	if workspace == "" || repoSlug == "" {
		return "", fmt.Errorf("workspace and repository slug are required")
	}
	apiPath := fmt.Sprintf("/repositories/%s/%s/commit/%s",
		url.PathEscape(workspace),
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
	if commit.Hash == "" {
		return "", ErrNotFound
	}
	return commit.Hash, nil
}

// LatestCommitForPath returns the most recent commit reachable from ref that
// touched path. It identifies the version of a directory: Bitbucket exposes no
// per-directory tree hash.
func (c *Client) LatestCommitForPath(ctx context.Context, workspace, repoSlug, ref, path string) (string, error) {
	if workspace == "" || repoSlug == "" {
		return "", fmt.Errorf("workspace and repository slug are required")
	}
	apiPath := fmt.Sprintf("/repositories/%s/%s/commits/%s?pagelen=1",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(ref),
	)
	if trimmed := strings.Trim(path, "/"); trimmed != "" {
		apiPath += "&path=" + url.QueryEscape(trimmed)
	}

	req, err := c.http.NewRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return "", err
	}
	var page commitPage
	if err := c.http.Do(req, &page); err != nil {
		return "", translateNotFound(err)
	}
	if len(page.Values) == 0 || page.Values[0].Hash == "" {
		return "", fmt.Errorf("no commits found for %q at %s", path, ref)
	}
	return page.Values[0].Hash, nil
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
