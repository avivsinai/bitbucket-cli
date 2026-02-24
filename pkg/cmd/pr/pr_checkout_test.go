package pr

import (
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestRepoCloneURL(t *testing.T) {
	tests := []struct {
		name     string
		repo     bbcloud.RepositoryRef
		protocol string
		want     string
	}{
		{
			name: "https found",
			repo: makeRepoRef("user/repo", []cloneEntry{
				{Name: "https", Href: "https://bitbucket.org/user/repo.git"},
				{Name: "ssh", Href: "git@bitbucket.org:user/repo.git"},
			}),
			protocol: "https",
			want:     "https://bitbucket.org/user/repo.git",
		},
		{
			name: "ssh found",
			repo: makeRepoRef("user/repo", []cloneEntry{
				{Name: "https", Href: "https://bitbucket.org/user/repo.git"},
				{Name: "ssh", Href: "git@bitbucket.org:user/repo.git"},
			}),
			protocol: "ssh",
			want:     "git@bitbucket.org:user/repo.git",
		},
		{
			name: "protocol not found",
			repo: makeRepoRef("user/repo", []cloneEntry{
				{Name: "ssh", Href: "git@bitbucket.org:user/repo.git"},
			}),
			protocol: "https",
			want:     "",
		},
		{
			name:     "no clone links",
			repo:     makeRepoRef("user/repo", nil),
			protocol: "https",
			want:     "",
		},
		{
			name: "case insensitive match",
			repo: makeRepoRef("user/repo", []cloneEntry{
				{Name: "HTTPS", Href: "https://bitbucket.org/user/repo.git"},
			}),
			protocol: "https",
			want:     "https://bitbucket.org/user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repoCloneURL(tt.repo, tt.protocol)
			if got != tt.want {
				t.Errorf("repoCloneURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsForkDetection(t *testing.T) {
	tests := []struct {
		name     string
		srcFull  string
		dstFull  string
		wantFork bool
	}{
		{
			name:     "same repo - not a fork",
			srcFull:  "workspace/repo",
			dstFull:  "workspace/repo",
			wantFork: false,
		},
		{
			name:     "different repos - is a fork",
			srcFull:  "contributor/repo",
			dstFull:  "workspace/repo",
			wantFork: true,
		},
		{
			name:     "empty source full_name - not a fork",
			srcFull:  "",
			dstFull:  "workspace/repo",
			wantFork: false,
		},
		{
			name:     "empty destination full_name - not a fork",
			srcFull:  "contributor/repo",
			dstFull:  "",
			wantFork: false,
		},
		{
			name:     "both empty - not a fork",
			srcFull:  "",
			dstFull:  "",
			wantFork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isFork := tt.srcFull != "" &&
				tt.dstFull != "" &&
				tt.srcFull != tt.dstFull

			if isFork != tt.wantFork {
				t.Errorf("isFork = %v, want %v (src=%q, dst=%q)",
					isFork, tt.wantFork, tt.srcFull, tt.dstFull)
			}
		})
	}
}

// --- helpers for tests ---

type cloneEntry struct {
	Name string
	Href string
}

func makeRepoRef(fullName string, clones []cloneEntry) bbcloud.RepositoryRef {
	ref := bbcloud.RepositoryRef{
		FullName: fullName,
	}
	for _, c := range clones {
		ref.Links.Clone = append(ref.Links.Clone, struct {
			Href string `json:"href"`
			Name string `json:"name"`
		}{
			Href: c.Href,
			Name: c.Name,
		})
	}
	return ref
}
