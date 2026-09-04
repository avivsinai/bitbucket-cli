// Package sourcetest provides an in-memory source.Repository for tests.
package sourcetest

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
)

// Repo is an in-memory source.Repository. Files are keyed by slash-separated
// path; every lookup that misses returns source.ErrNotFound.
type Repo struct {
	Name     string            // FullName, e.g. "myteam/agent-skills"
	Web      string            // WebURL
	Clone    string            // CloneURL
	Branches map[string]string // branch name -> commit SHA
	Tags     map[string]string // tag name -> commit SHA
	TagOrder []string          // tags newest first; first entry is LatestTag
	Default  string            // default branch name
	Files    map[string]string // path -> content
	Commits  map[string]string // dir -> latest commit touching it (default: "commit-" + dir)
	Err      error             // when set, every call returns this error
	readFile atomic.Int32      // ReadFile is called concurrently by FetchDescriptions

	CreatedTags        map[string]string // tag name -> commit, recorded by CreateTag
	CreatedTagMessages []string

	// Per-call failures, for exercising error paths that are not "not found".
	TagErr       error
	CommitErr    error
	CreateTagErr error
}

// ReadFileCalls reports how many times ReadFile has been called.
func (r *Repo) ReadFileCalls() int { return int(r.readFile.Load()) }

// ResetReadFileCalls zeroes the ReadFile counter.
func (r *Repo) ResetReadFileCalls() { r.readFile.Store(0) }

// New returns a Repo with sensible defaults for the given files.
func New(name string, files map[string]string) *Repo {
	return &Repo{
		Name:     name,
		Web:      "https://bitbucket.org/" + name,
		Clone:    "https://bitbucket.org/" + name + ".git",
		Branches: map[string]string{"main": "sha-main"},
		Default:  "main",
		Files:    files,
		Tags:     map[string]string{},
	}
}

func (r *Repo) FullName() string { return r.Name }
func (r *Repo) WebURL() string   { return r.Web }
func (r *Repo) CloneURL() string { return r.Clone }

func (r *Repo) Branch(_ context.Context, name string) (string, error) {
	if r.Err != nil {
		return "", r.Err
	}
	if sha, ok := r.Branches[name]; ok {
		return sha, nil
	}
	return "", source.ErrNotFound
}

func (r *Repo) Tag(_ context.Context, name string) (string, error) {
	if r.Err != nil {
		return "", r.Err
	}
	if r.TagErr != nil {
		return "", r.TagErr
	}
	if sha, ok := r.Tags[name]; ok {
		return sha, nil
	}
	return "", source.ErrNotFound
}

func (r *Repo) Commit(_ context.Context, ref string) (string, error) {
	if r.Err != nil {
		return "", r.Err
	}
	if r.CommitErr != nil {
		return "", r.CommitErr
	}
	for _, sha := range r.Branches {
		if sha == ref {
			return sha, nil
		}
	}
	for _, sha := range r.Tags {
		if sha == ref {
			return sha, nil
		}
	}
	return "", source.ErrNotFound
}

func (r *Repo) LatestTag(_ context.Context) (string, string, error) {
	if r.Err != nil {
		return "", "", r.Err
	}
	if len(r.TagOrder) == 0 {
		return "", "", source.ErrNotFound
	}
	name := r.TagOrder[0]
	return name, r.Tags[name], nil
}

func (r *Repo) DefaultBranch(_ context.Context) (string, error) {
	if r.Err != nil {
		return "", r.Err
	}
	return r.Default, nil
}

func (r *Repo) ListFiles(_ context.Context, _ string, dir string) ([]source.File, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	prefix := strings.Trim(dir, "/")
	if prefix != "" {
		prefix += "/"
	}
	var files []source.File
	for p, content := range r.Files {
		if strings.HasPrefix(p, prefix) {
			files = append(files, source.File{Path: p, Size: int64(len(content))})
		}
	}
	if prefix != "" && len(files) == 0 {
		return nil, source.ErrNotFound
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func (r *Repo) ReadFile(_ context.Context, _ string, path string) ([]byte, error) {
	r.readFile.Add(1)
	if r.Err != nil {
		return nil, r.Err
	}
	content, ok := r.Files[path]
	if !ok {
		return nil, fmt.Errorf("%s: %w", path, source.ErrNotFound)
	}
	return []byte(content), nil
}

func (r *Repo) LatestCommit(_ context.Context, _ string, dir string) (string, error) {
	if r.Err != nil {
		return "", r.Err
	}
	if sha, ok := r.Commits[dir]; ok {
		return sha, nil
	}
	return "commit-" + strings.ReplaceAll(strings.Trim(dir, "/"), "/", "-"), nil
}

// CreateTag records a created tag so tests can assert on it.
func (r *Repo) CreateTag(_ context.Context, name, commit, message string) error {
	if r.Err != nil {
		return r.Err
	}
	if r.CreateTagErr != nil {
		return r.CreateTagErr
	}
	if r.CreatedTags == nil {
		r.CreatedTags = map[string]string{}
	}
	r.CreatedTags[name] = commit
	r.CreatedTagMessages = append(r.CreatedTagMessages, message)
	if r.Tags == nil {
		r.Tags = map[string]string{}
	}
	r.Tags[name] = commit
	return nil
}
