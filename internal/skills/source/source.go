// Package source abstracts read access to a Bitbucket repository so that skill
// discovery and installation do not depend on the Cloud or Data Center API shapes.
package source

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound is returned by Repository lookups when a ref, path, or file does not exist.
var ErrNotFound = errors.New("not found")

// Ref is a resolved git reference.
type Ref struct {
	Ref string // "refs/heads/<branch>", "refs/tags/<tag>", or a bare commit SHA
	SHA string // commit SHA the ref points at
}

// File describes one file in a repository tree.
type File struct {
	Path string // slash-separated path relative to the repository root
	Size int64
}

// Repository is the platform-neutral read surface used by skill discovery and
// installation. Cloud and Data Center adapters implement it over their REST clients.
type Repository interface {
	// FullName is "workspace/slug" (Cloud) or "PROJECT/slug" (Data Center).
	FullName() string
	// WebURL is the canonical browser URL recorded in installed skill metadata.
	WebURL() string
	// CloneURL is the HTTPS clone URL recorded in the skill lock file.
	CloneURL() string

	// Branch returns the head commit of a branch, or ErrNotFound.
	Branch(ctx context.Context, name string) (string, error)
	// Tag returns the commit a tag points at (annotated tags dereferenced), or ErrNotFound.
	Tag(ctx context.Context, name string) (string, error)
	// Commit returns the full SHA for a commit reference, or ErrNotFound.
	Commit(ctx context.Context, ref string) (string, error)
	// LatestTag returns the most recently created tag and its commit, or
	// ErrNotFound when the repository has no tags.
	LatestTag(ctx context.Context) (name, sha string, err error)
	// DefaultBranch returns the repository's default branch name.
	DefaultBranch(ctx context.Context) (string, error)

	// ListFiles lists every file under dir (recursively) at the given commit.
	// An empty dir lists the whole repository. A missing dir yields ErrNotFound.
	ListFiles(ctx context.Context, sha, dir string) ([]File, error)
	// ReadFile returns the raw bytes of a file at the given commit.
	ReadFile(ctx context.Context, sha, path string) ([]byte, error)
	// LatestCommit returns the most recent commit reachable from sha that touched dir.
	LatestCommit(ctx context.Context, sha, dir string) (string, error)
}

// IsFullyQualifiedRef returns true if ref uses the "refs/heads/" or "refs/tags/" prefix.
func IsFullyQualifiedRef(ref string) bool {
	return strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/tags/")
}

// ShortRef strips the "refs/heads/" or "refs/tags/" prefix from a fully
// qualified ref. A ref that is not fully qualified is returned as-is.
func ShortRef(ref string) string {
	if after, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
		return after
	}
	if after, ok := strings.CutPrefix(ref, "refs/tags/"); ok {
		return after
	}
	return ref
}

// ResolveRef determines the commit to use for a repository.
// Priority: explicit version > newest tag > default branch. Bitbucket has no
// release objects, so "newest tag" stands in for gh's "latest release".
func ResolveRef(ctx context.Context, repo Repository, version string) (Ref, error) {
	if version != "" {
		return resolveExplicitRef(ctx, repo, version)
	}

	name, sha, err := repo.LatestTag(ctx)
	if err == nil {
		return Ref{Ref: "refs/tags/" + name, SHA: sha}, nil
	}
	// Only fall back to the default branch when the repository genuinely has
	// no tags. Any other error is surfaced so it cannot silently select an
	// unexpected ref.
	if !errors.Is(err, ErrNotFound) {
		return Ref{}, fmt.Errorf("could not fetch latest tag: %w", err)
	}

	branch, err := repo.DefaultBranch(ctx)
	if err != nil {
		return Ref{}, fmt.Errorf("could not determine default branch: %w", err)
	}
	if branch == "" {
		return Ref{}, fmt.Errorf("could not determine default branch for %s", repo.FullName())
	}
	return resolveBranchRef(ctx, repo, branch)
}

// resolveExplicitRef resolves a user-supplied version string. It supports:
//   - fully qualified refs: "refs/tags/v1.0" or "refs/heads/main"
//   - short names: tried as branch first, then tag, then commit SHA
//
// When a short name matches both a branch and a tag, the branch wins.
func resolveExplicitRef(ctx context.Context, repo Repository, ref string) (Ref, error) {
	if after, ok := strings.CutPrefix(ref, "refs/tags/"); ok {
		return resolveTagRef(ctx, repo, after)
	}
	if after, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
		return resolveBranchRef(ctx, repo, after)
	}

	if resolved, err := resolveBranchRef(ctx, repo, ref); err == nil {
		return resolved, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Ref{}, err
	}
	if resolved, err := resolveTagRef(ctx, repo, ref); err == nil {
		return resolved, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Ref{}, err
	}

	sha, err := repo.Commit(ctx, ref)
	if err == nil {
		return Ref{Ref: sha, SHA: sha}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Ref{}, fmt.Errorf("could not resolve commit %q in %s: %w", ref, repo.FullName(), err)
	}

	return Ref{}, fmt.Errorf("ref %q not found as branch, tag, or commit in %s", ref, repo.FullName())
}

func resolveTagRef(ctx context.Context, repo Repository, tag string) (Ref, error) {
	sha, err := repo.Tag(ctx, tag)
	if err != nil {
		return Ref{}, fmt.Errorf("tag %q not found in %s: %w", tag, repo.FullName(), err)
	}
	return Ref{Ref: "refs/tags/" + tag, SHA: sha}, nil
}

func resolveBranchRef(ctx context.Context, repo Repository, branch string) (Ref, error) {
	sha, err := repo.Branch(ctx, branch)
	if err != nil {
		return Ref{}, fmt.Errorf("branch %q not found in %s: %w", branch, repo.FullName(), err)
	}
	return Ref{Ref: "refs/heads/" + branch, SHA: sha}, nil
}
