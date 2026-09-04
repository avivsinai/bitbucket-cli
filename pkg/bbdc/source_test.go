package bbdc_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

// newSourceServer serves canned JSON and records the raw URLs requested.
func newSourceServer(t *testing.T, handler http.HandlerFunc) (*bbdc.Client, *[]string) {
	t.Helper()
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.String())
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client, &requested
}

func writeJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestListFilesPaginatesAndReturnsRelativePaths(t *testing.T) {
	client, requested := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("start") == "2" {
			writeJSON(t, w, map[string]any{
				"values":     []string{"reference/notes.md"},
				"isLastPage": true,
			})
			return
		}
		writeJSON(t, w, map[string]any{
			"values":        []string{"SKILL.md", "scripts/run.sh"},
			"isLastPage":    false,
			"nextPageStart": 2,
		})
	})

	files, err := client.ListFiles(context.Background(), "PROJ", "repo", "refs/heads/main", "skills/alpha")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	want := []string{"SKILL.md", "scripts/run.sh", "reference/notes.md"}
	if strings.Join(files, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v (paths stay relative to the requested directory)", files, want)
	}

	first := (*requested)[0]
	for _, part := range []string{"/rest/api/1.0/projects/PROJ/repos/repo/files/skills/alpha", "limit=1000", "start=0", "at=refs%2Fheads%2Fmain"} {
		if !strings.Contains(first, part) {
			t.Errorf("first request %q missing %q", first, part)
		}
	}
	if len(*requested) != 2 {
		t.Fatalf("expected 2 requests (pagination), got %d", len(*requested))
	}
}

func TestListFilesEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		project    string
		repo       string
		path       string
		body       any
		wantURL    string
		wantErr    string
		wantNotFnd bool
	}{
		{
			name:    "empty path lists the repository root",
			status:  http.StatusOK,
			project: "PROJ",
			repo:    "repo",
			path:    "",
			body:    map[string]any{"values": []string{"README.md"}, "isLastPage": true},
			wantURL: "/rest/api/1.0/projects/PROJ/repos/repo/files?",
		},
		{
			name:       "404 becomes ErrNotFound",
			status:     http.StatusNotFound,
			project:    "PROJ",
			repo:       "repo",
			path:       "skills/missing",
			wantNotFnd: true,
		},
		{
			name:    "project key required",
			project: "",
			repo:    "repo",
			wantErr: "project key and repository slug are required",
		},
		{
			name:    "last page without nextPageStart terminates",
			status:  http.StatusOK,
			project: "PROJ",
			repo:    "repo",
			body:    map[string]any{"values": []string{"a.md"}, "isLastPage": false, "nextPageStart": 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, requested := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
				if tt.status != http.StatusOK {
					w.WriteHeader(tt.status)
					return
				}
				writeJSON(t, w, tt.body)
			})

			_, err := client.ListFiles(context.Background(), tt.project, tt.repo, "main", tt.path)
			switch {
			case tt.wantNotFnd:
				if !errors.Is(err, bbdc.ErrNotFound) {
					t.Fatalf("error = %v, want ErrNotFound", err)
				}
			case tt.wantErr != "":
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
			default:
				if err != nil {
					t.Fatalf("ListFiles: %v", err)
				}
			}
			if tt.wantURL != "" && !strings.Contains((*requested)[0], tt.wantURL) {
				t.Fatalf("requested %q, want it to contain %q", (*requested)[0], tt.wantURL)
			}
		})
	}
}

func TestReadFileReturnsRawBytes(t *testing.T) {
	client, requested := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# raw\n"))
	})

	content, err := client.ReadFile(context.Background(), "PROJ", "repo", "refs/tags/v1", "skills/alpha/SKILL.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "# raw\n" {
		t.Fatalf("content = %q", content)
	}
	got := (*requested)[0]
	for _, part := range []string{"/rest/api/1.0/projects/PROJ/repos/repo/raw/skills/alpha/SKILL.md", "at=refs%2Ftags%2Fv1"} {
		if !strings.Contains(got, part) {
			t.Errorf("requested %q missing %q", got, part)
		}
	}

	if _, err := client.ReadFile(context.Background(), "PROJ", "repo", "main", ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestTagsBranchesAndCommits(t *testing.T) {
	client, requested := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/tags/v1.0.0"):
			writeJSON(t, w, map[string]any{"displayId": "v1.0.0", "hash": "tagobject", "latestCommit": "tagcommit"})
		case strings.Contains(r.URL.Path, "/tags"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{
				{"id": "refs/tags/v2.0.0", "displayId": "v2.0.0", "latestCommit": "newcommit"},
			}, "isLastPage": true})
		case strings.Contains(r.URL.Path, "/default-branch"):
			writeJSON(t, w, map[string]any{"id": "refs/heads/develop", "displayId": "develop"})
		case strings.Contains(r.URL.Path, "/branches"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{
				{"id": "refs/heads/main-old", "displayId": "main-old", "latestCommit": "wrong"},
				{"id": "refs/heads/main", "displayId": "main", "latestCommit": "branchcommit"},
			}, "isLastPage": true})
		case strings.Contains(r.URL.Path, "/commits/"):
			writeJSON(t, w, map[string]any{"id": "fullcommitid"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	ctx := context.Background()

	tags, err := client.ListTags(ctx, "PROJ", "repo", 1)
	if err != nil || len(tags) != 1 || tags[0].Commit() != "newcommit" {
		t.Fatalf("ListTags = %+v, %v", tags, err)
	}
	if !strings.Contains((*requested)[0], "orderBy=MODIFICATION") {
		t.Fatalf("tag listing must order by modification, got %q", (*requested)[0])
	}

	// Annotated tags report the tag object in hash and the commit in latestCommit.
	if sha, err := client.GetTag(ctx, "PROJ", "repo", "v1.0.0"); err != nil || sha != "tagcommit" {
		t.Fatalf("GetTag = %q, %v; want the dereferenced commit", sha, err)
	}
	// Branch lookup filters by name and must match exactly, not by prefix.
	if sha, err := client.GetBranch(ctx, "PROJ", "repo", "main"); err != nil || sha != "branchcommit" {
		t.Fatalf("GetBranch = %q, %v", sha, err)
	}
	if branch, err := client.GetDefaultBranch(ctx, "PROJ", "repo"); err != nil || branch != "develop" {
		t.Fatalf("GetDefaultBranch = %q, %v", branch, err)
	}
	if sha, err := client.GetCommit(ctx, "PROJ", "repo", "abc"); err != nil || sha != "fullcommitid" {
		t.Fatalf("GetCommit = %q, %v", sha, err)
	}
}

func TestGetBranchNotFound(t *testing.T) {
	client, _ := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"values": []map[string]any{}, "isLastPage": true})
	})
	if _, err := client.GetBranch(context.Background(), "PROJ", "repo", "missing"); !errors.Is(err, bbdc.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestTagCommitPrefersLatestCommit(t *testing.T) {
	tests := []struct {
		name string
		tag  bbdc.Tag
		want string
	}{
		{name: "annotated tag", tag: bbdc.Tag{Hash: "tagobject", LatestCommit: "commit"}, want: "commit"},
		{name: "lightweight tag", tag: bbdc.Tag{Hash: "commit"}, want: "commit"},
		{name: "empty", tag: bbdc.Tag{}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tag.Commit(); got != tt.want {
				t.Fatalf("Commit() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLatestCommitForPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		values    []map[string]any
		wantParts []string
		wantSHA   string
		wantErr   string
	}{
		{
			name:      "path and ref become query parameters",
			path:      "skills/alpha",
			values:    []map[string]any{{"id": "commit1"}},
			wantParts: []string{"limit=1", "until=refs%2Fheads%2Fmain", "path=skills%2Falpha"},
			wantSHA:   "commit1",
		},
		{
			name:      "empty path omits the filter",
			path:      "",
			values:    []map[string]any{{"id": "commit2"}},
			wantParts: []string{"limit=1"},
			wantSHA:   "commit2",
		},
		{
			name:    "no commits is an error",
			path:    "skills/alpha",
			values:  []map[string]any{},
			wantErr: `no commits found for "skills/alpha"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, requested := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, map[string]any{"values": tt.values, "isLastPage": true})
			})

			sha, err := client.LatestCommitForPath(context.Background(), "PROJ", "repo", "refs/heads/main", tt.path)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LatestCommitForPath: %v", err)
			}
			if sha != tt.wantSHA {
				t.Fatalf("sha = %q, want %q", sha, tt.wantSHA)
			}
			for _, part := range tt.wantParts {
				if !strings.Contains((*requested)[0], part) {
					t.Errorf("requested %q missing %q", (*requested)[0], part)
				}
			}
			if tt.path == "" && strings.Contains((*requested)[0], "path=") {
				t.Errorf("empty path must not add a path filter: %q", (*requested)[0])
			}
		})
	}
}
