package source_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/internal/skills/sourcetest"
)

func TestShortRef(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{name: "branch ref", ref: "refs/heads/main", want: "main"},
		{name: "tag ref", ref: "refs/tags/v1.0.0", want: "v1.0.0"},
		{name: "bare sha unchanged", ref: "abc123", want: "abc123"},
		{name: "empty", ref: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := source.ShortRef(tt.ref); got != tt.want {
				t.Fatalf("ShortRef(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}

func TestResolveRef(t *testing.T) {
	newRepo := func() *sourcetest.Repo {
		r := sourcetest.New("myteam/skills", nil)
		r.Branches = map[string]string{"main": "sha-main", "v1": "sha-branch-v1"}
		r.Tags = map[string]string{"v1": "sha-tag-v1", "v2.0.0": "sha-tag-v2"}
		r.TagOrder = []string{"v2.0.0", "v1"}
		return r
	}

	tests := []struct {
		name    string
		setup   func(*sourcetest.Repo)
		version string
		want    source.Ref
		wantErr string
	}{
		{
			name: "no version uses newest tag",
			want: source.Ref{Ref: "refs/tags/v2.0.0", SHA: "sha-tag-v2"},
		},
		{
			name:  "no version and no tags falls back to default branch",
			setup: func(r *sourcetest.Repo) { r.Tags = nil; r.TagOrder = nil },
			want:  source.Ref{Ref: "refs/heads/main", SHA: "sha-main"},
		},
		{
			name:    "no version and tag lookup fails surfaces error",
			setup:   func(r *sourcetest.Repo) { r.Err = errors.New("boom") },
			wantErr: "could not fetch latest tag: boom",
		},
		{
			name:    "short name prefers tag over branch",
			version: "v1",
			want:    source.Ref{Ref: "refs/tags/v1", SHA: "sha-tag-v1"},
		},
		{
			name:    "short name resolves tag when no branch",
			version: "v2.0.0",
			want:    source.Ref{Ref: "refs/tags/v2.0.0", SHA: "sha-tag-v2"},
		},
		{
			name:    "fully qualified tag bypasses branch",
			version: "refs/tags/v1",
			want:    source.Ref{Ref: "refs/tags/v1", SHA: "sha-tag-v1"},
		},
		{
			name:    "fully qualified branch",
			version: "refs/heads/main",
			want:    source.Ref{Ref: "refs/heads/main", SHA: "sha-main"},
		},
		{
			name:    "bare commit sha",
			version: "sha-main",
			want:    source.Ref{Ref: "sha-main", SHA: "sha-main"},
		},
		{
			name:    "unknown ref reports all three lookups",
			version: "nope",
			wantErr: `ref "nope" not found as branch, tag, or commit in myteam/skills`,
		},
		{
			name:    "fully qualified tag missing",
			version: "refs/tags/missing",
			wantErr: `tag "missing" not found in myteam/skills`,
		},
		{
			name:    "non-404 branch error is surfaced",
			setup:   func(r *sourcetest.Repo) { r.Err = errors.New("503 service unavailable") },
			version: "main",
			wantErr: "503 service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo()
			if tt.setup != nil {
				tt.setup(repo)
			}
			got, err := source.ResolveRef(context.Background(), repo, tt.version)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (ref=%+v)", tt.wantErr, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveRef returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolveRef = %+v, want %+v", got, tt.want)
			}
		})
	}
}
