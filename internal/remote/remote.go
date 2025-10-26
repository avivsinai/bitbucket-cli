package remote

import "errors"

// ErrNotImplemented signals that remote detection is not yet implemented.
var ErrNotImplemented = errors.New("remote detection not implemented")

// Locator represents a repository identifier derived from a git remote.
type Locator struct {
	Host       string
	Kind       string // dc | cloud
	Workspace  string
	ProjectKey string
	RepoSlug   string
}

// Detect attempts to infer the locator from git remotes.
func Detect(repoPath string) (Locator, error) {
	return Locator{}, ErrNotImplemented
}
