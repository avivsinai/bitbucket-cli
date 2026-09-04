package source

import (
	"strings"
	"testing"
)

func TestParseRepoArg(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    RepoRef
		wantErr string
	}{
		{name: "cloud shorthand", arg: "myteam/agent-skills", want: RepoRef{Owner: "myteam", Slug: "agent-skills"}},
		{name: "dc shorthand keeps key as typed", arg: "PROJ/agent-skills.git", want: RepoRef{Owner: "PROJ", Slug: "agent-skills"}},
		{name: "cloud https url", arg: "https://bitbucket.org/myteam/agent-skills.git", want: RepoRef{Host: "bitbucket.org", Kind: "cloud", Owner: "myteam", Slug: "agent-skills"}},
		{name: "cloud web url with trailing path", arg: "https://bitbucket.org/myteam/agent-skills/src/main/", want: RepoRef{Host: "bitbucket.org", Kind: "cloud", Owner: "myteam", Slug: "agent-skills"}},
		{name: "cloud ssh scp url", arg: "git@bitbucket.org:myteam/agent-skills.git", want: RepoRef{Host: "bitbucket.org", Kind: "cloud", Owner: "myteam", Slug: "agent-skills"}},
		{name: "dc web url", arg: "https://bitbucket.example.com/projects/proj/repos/agent-skills/browse", want: RepoRef{Host: "bitbucket.example.com", Kind: "dc", Owner: "PROJ", Slug: "agent-skills"}},
		{name: "dc scm clone url", arg: "https://bitbucket.example.com/scm/proj/agent-skills.git", want: RepoRef{Host: "bitbucket.example.com", Kind: "dc", Owner: "PROJ", Slug: "agent-skills"}},
		{name: "dc ssh url with port", arg: "ssh://git@bitbucket.example.com:7999/proj/agent-skills.git", want: RepoRef{Host: "bitbucket.example.com", Kind: "dc", Owner: "PROJ", Slug: "agent-skills"}},
		{name: "empty", arg: "  ", wantErr: "repository is required"},
		{name: "missing slash", arg: "agent-skills", wantErr: "expected WORKSPACE/REPO (Cloud), PROJECT/REPO (Data Center), or a Bitbucket URL"},
		{name: "too many segments", arg: "a/b/c", wantErr: "invalid repository"},
		{name: "unparseable url", arg: "https://bitbucket.org/onlyowner", wantErr: "invalid repository"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRepoArg(tt.arg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRepoArg(%q): %v", tt.arg, err)
			}
			if got != tt.want {
				t.Fatalf("ParseRepoArg(%q) = %+v, want %+v", tt.arg, got, tt.want)
			}
		})
	}
}
