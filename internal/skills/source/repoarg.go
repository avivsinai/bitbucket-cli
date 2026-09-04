package source

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/avivsinai/bitbucket-cli/internal/remote"
)

// RepoRef identifies the repository a user asked for, before it is bound to a
// configured Bitbucket host.
type RepoRef struct {
	Host  string // hostname when the argument was a URL; empty for OWNER/REPO shorthand
	Kind  string // "cloud" or "dc" when derivable from a URL; empty otherwise
	Owner string // Cloud workspace or Data Center project key
	Slug  string
}

var shorthandPattern = regexp.MustCompile(`^([^/\s]+)/([^/\s]+)$`)

// ParseRepoArg accepts "WORKSPACE/REPO" (Cloud), "PROJECT/REPO" (Data Center),
// or any Bitbucket clone/web URL and returns the repository identifiers.
func ParseRepoArg(arg string) (RepoRef, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return RepoRef{}, fmt.Errorf("repository is required")
	}

	if strings.Contains(arg, "://") || strings.Contains(arg, "@") {
		loc, err := remote.ParseLocator(arg)
		if err != nil {
			return RepoRef{}, fmt.Errorf("invalid repository %q: %w", arg, err)
		}
		ref := RepoRef{Host: loc.Host, Kind: loc.Kind, Slug: loc.RepoSlug}
		if loc.Kind == "cloud" {
			ref.Owner = loc.Workspace
		} else {
			ref.Owner = loc.ProjectKey
		}
		if ref.Owner == "" || ref.Slug == "" {
			return RepoRef{}, fmt.Errorf("invalid repository %q: could not determine owner and repository", arg)
		}
		return ref, nil
	}

	m := shorthandPattern.FindStringSubmatch(arg)
	if m == nil {
		return RepoRef{}, fmt.Errorf("invalid repository %q: expected WORKSPACE/REPO (Cloud), PROJECT/REPO (Data Center), or a Bitbucket URL", arg)
	}
	return RepoRef{Owner: m[1], Slug: strings.TrimSuffix(m[2], ".git")}, nil
}
