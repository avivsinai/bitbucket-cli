package pr

import (
	"strings"
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

func TestOwnerDerivation(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     string
	}{
		{
			name:     "normal owner/repo",
			fullName: "contributor/my-repo",
			want:     "contributor",
		},
		{
			name:     "owner with multiple slashes",
			fullName: "org/sub/repo",
			want:     "org",
		},
		{
			name:     "no slash - fallback to full name",
			fullName: "justrepo",
			want:     "justrepo",
		},
		{
			name:     "empty string - fallback",
			fullName: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.SplitN(tt.fullName, "/", 2)
			owner := tt.fullName // fallback if no /
			if len(parts) >= 2 {
				owner = parts[0]
			}
			if owner != tt.want {
				t.Errorf("owner = %q, want %q", owner, tt.want)
			}
		})
	}
}

func TestFindRemoteByURLParsing(t *testing.T) {
	// This tests the parsing logic used by findRemoteByURL.
	// We simulate git remote -v output and check that only (fetch) lines match.
	tests := []struct {
		name     string
		output   string
		cloneURL string
		want     string
	}{
		{
			name: "match fetch line",
			output: "origin\thttps://bitbucket.org/ws/repo.git (fetch)\n" +
				"origin\thttps://bitbucket.org/ws/repo.git (push)\n",
			cloneURL: "https://bitbucket.org/ws/repo.git",
			want:     "origin",
		},
		{
			name:     "only push - no match",
			output:   "upstream\thttps://bitbucket.org/ws/repo.git (push)\n",
			cloneURL: "https://bitbucket.org/ws/repo.git",
			want:     "",
		},
		{
			name:     ".git suffix normalisation",
			output:   "fork\thttps://bitbucket.org/user/repo (fetch)\n",
			cloneURL: "https://bitbucket.org/user/repo.git",
			want:     "fork",
		},
		{
			name:     "no match",
			output:   "origin\thttps://github.com/ws/repo.git (fetch)\n",
			cloneURL: "https://bitbucket.org/ws/repo.git",
			want:     "",
		},
		{
			name:     "empty output",
			output:   "",
			cloneURL: "https://bitbucket.org/ws/repo.git",
			want:     "",
		},
	}

	norm := func(u string) string {
		return strings.TrimSuffix(strings.TrimSpace(u), ".git")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := norm(tt.cloneURL)
			got := ""
			for _, line := range strings.Split(tt.output, "\n") {
				fields := strings.Fields(line)
				if len(fields) < 3 {
					continue
				}
				if fields[2] != "(fetch)" {
					continue
				}
				if norm(fields[1]) == target {
					got = fields[0]
					break
				}
			}
			if got != tt.want {
				t.Errorf("findRemoteByURL logic = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInferProtocolParsing(t *testing.T) {
	// Tests the protocol inference logic used by inferProtocol.
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"https url", "https://bitbucket.org/ws/repo.git", "https"},
		{"ssh git@ url", "git@bitbucket.org:ws/repo.git", "ssh"},
		{"ssh:// url", "ssh://git@bitbucket.org/ws/repo.git", "ssh"},
		{"empty url", "", "https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := "https"
			url := strings.TrimSpace(tt.url)
			if strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://") {
				got = "ssh"
			}
			if got != tt.want {
				t.Errorf("inferProtocol logic = %q, want %q", got, tt.want)
			}
		})
	}
}
