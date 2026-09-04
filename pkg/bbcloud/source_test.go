package bbcloud_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

// newSourceServer serves canned JSON per request path and records the raw URLs
// the client requested.
func newSourceServer(t *testing.T, handler http.HandlerFunc) (*bbcloud.Client, *[]string) {
	t.Helper()
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.String())
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
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

func TestListSourceRecursesAndFiltersDirectories(t *testing.T) {
	// The "next" link is absolute, so it is built from the incoming request host.
	var requested []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.String())
		if r.URL.Query().Get("page") == "2" {
			writeJSON(t, w, map[string]any{"values": []map[string]any{
				{"path": "skills/alpha/scripts/run.sh", "type": "commit_file", "size": 12},
			}})
			return
		}
		writeJSON(t, w, map[string]any{
			"values": []map[string]any{
				{"path": "skills/alpha", "type": "commit_directory"},
				{"path": "skills/alpha/SKILL.md", "type": "commit_file", "size": 34},
			},
			"next": "http://" + r.Host + "/repositories/ws/repo/src/sha1/skills?format=meta&page=2",
		})
	}))
	t.Cleanup(srv.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: srv.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	files, err := client.ListSource(context.Background(), "ws", "repo", "sha1", "skills")
	if err != nil {
		t.Fatalf("ListSource: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %+v, want 2 entries (directories filtered out)", files)
	}
	if files[0].Path != "skills/alpha/SKILL.md" || files[0].Size != 34 {
		t.Errorf("first file = %+v", files[0])
	}
	if files[1].Path != "skills/alpha/scripts/run.sh" {
		t.Errorf("second page not followed, got %+v", files[1])
	}

	first := requested[0]
	for _, want := range []string{"/repositories/ws/repo/src/sha1/skills", "pagelen=100", "max_depth=10"} {
		if !strings.Contains(first, want) {
			t.Errorf("first request %q missing %q", first, want)
		}
	}
	// format=meta would return metadata about the directory itself instead of
	// a listing of its contents, silently yielding zero files.
	if strings.Contains(first, "format=meta") {
		t.Errorf("directory listing must not request format=meta: %q", first)
	}
}

func TestListSourceEscapesSegmentsAndReportsNotFound(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		workspace  string
		repo       string
		commit     string
		path       string
		wantInURL  string
		wantErr    string
		wantNotFnd bool
	}{
		{
			name:      "escapes each path segment",
			status:    http.StatusOK,
			workspace: "my team",
			repo:      "repo",
			commit:    "refs/heads/main",
			path:      "skills/my skill",
			wantInURL: "/repositories/my%20team/repo/src/refs%2Fheads%2Fmain/skills/my%20skill?",
		},
		{
			name:       "404 becomes ErrNotFound",
			status:     http.StatusNotFound,
			workspace:  "ws",
			repo:       "repo",
			commit:     "sha",
			path:       "skills/missing",
			wantNotFnd: true,
		},
		{
			name:      "500 is surfaced as-is",
			status:    http.StatusInternalServerError,
			workspace: "ws",
			repo:      "repo",
			commit:    "sha",
			path:      "skills/x",
			wantErr:   "500",
		},
		{
			name:      "workspace required",
			workspace: "",
			repo:      "repo",
			commit:    "sha",
			wantErr:   "workspace and repository slug are required",
		},
		{
			name:      "commit required",
			workspace: "ws",
			repo:      "repo",
			commit:    "",
			wantErr:   "commit is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, requested := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
				if tt.status != http.StatusOK {
					w.WriteHeader(tt.status)
					return
				}
				writeJSON(t, w, map[string]any{"values": []map[string]any{}})
			})

			_, err := client.ListSource(context.Background(), tt.workspace, tt.repo, tt.commit, tt.path)

			switch {
			case tt.wantNotFnd:
				if !errors.Is(err, bbcloud.ErrNotFound) {
					t.Fatalf("error = %v, want ErrNotFound", err)
				}
			case tt.wantErr != "":
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
			default:
				if err != nil {
					t.Fatalf("ListSource: %v", err)
				}
			}

			if tt.wantInURL != "" {
				if len(*requested) == 0 || !strings.Contains((*requested)[0], tt.wantInURL) {
					t.Fatalf("requested %v, want a URL containing %q", *requested, tt.wantInURL)
				}
			}
		})
	}
}

func TestReadSourceReturnsRawBytes(t *testing.T) {
	client, requested := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("---\nname: alpha\n---\n# Body\n"))
	})

	content, err := client.ReadSource(context.Background(), "ws", "repo", "sha1", "skills/alpha/SKILL.md")
	if err != nil {
		t.Fatalf("ReadSource: %v", err)
	}
	if string(content) != "---\nname: alpha\n---\n# Body\n" {
		t.Fatalf("content = %q", content)
	}
	if !strings.Contains((*requested)[0], "/repositories/ws/repo/src/sha1/skills/alpha/SKILL.md") {
		t.Fatalf("requested %q", (*requested)[0])
	}
	if strings.Contains((*requested)[0], "format=meta") {
		t.Fatal("raw reads must not request the meta format")
	}

	if _, err := client.ReadSource(context.Background(), "ws", "repo", "sha1", ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestTagsAndRefLookups(t *testing.T) {
	client, requested := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/refs/tags/v1.0.0"):
			writeJSON(t, w, map[string]any{"name": "v1.0.0", "target": map[string]any{"hash": "tagsha"}})
		case strings.HasSuffix(r.URL.Path, "/refs/tags"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{
				{"name": "v2.0.0", "target": map[string]any{"hash": "newsha"}},
				{"name": "v1.0.0", "target": map[string]any{"hash": "oldsha"}},
			}})
		case strings.Contains(r.URL.Path, "/refs/branches/main"):
			writeJSON(t, w, map[string]any{"name": "main", "target": map[string]any{"hash": "branchsha"}})
		case strings.Contains(r.URL.Path, "/commit/"):
			writeJSON(t, w, map[string]any{"hash": "commitsha"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	ctx := context.Background()

	tags, err := client.ListTags(ctx, "ws", "repo", 1)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v2.0.0" || tags[0].Target.Hash != "newsha" {
		t.Fatalf("tags = %+v, want the newest tag only", tags)
	}
	if !strings.Contains((*requested)[0], "sort=-target.date") {
		t.Fatalf("tag listing must sort newest first, got %q", (*requested)[0])
	}

	if sha, err := client.GetTag(ctx, "ws", "repo", "v1.0.0"); err != nil || sha != "tagsha" {
		t.Fatalf("GetTag = %q, %v", sha, err)
	}
	if sha, err := client.GetBranch(ctx, "ws", "repo", "main"); err != nil || sha != "branchsha" {
		t.Fatalf("GetBranch = %q, %v", sha, err)
	}
	if sha, err := client.GetCommit(ctx, "ws", "repo", "abc123"); err != nil || sha != "commitsha" {
		t.Fatalf("GetCommit = %q, %v", sha, err)
	}

	if _, err := client.GetBranch(ctx, "ws", "repo", "missing"); !errors.Is(err, bbcloud.ErrNotFound) {
		t.Fatalf("missing branch error = %v, want ErrNotFound", err)
	}
	if _, err := client.GetTag(ctx, "ws", "repo", "missing"); !errors.Is(err, bbcloud.ErrNotFound) {
		t.Fatalf("missing tag error = %v, want ErrNotFound", err)
	}
}

func TestListTagsEmptyRepository(t *testing.T) {
	client, _ := newSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"values": []map[string]any{}})
	})
	tags, err := client.ListTags(context.Background(), "ws", "repo", 1)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("tags = %+v, want none", tags)
	}
}

func TestLatestCommitForPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		values    []map[string]any
		wantQuery string
		wantSHA   string
		wantErr   string
	}{
		{
			name:      "directory path is sent as a query parameter",
			path:      "skills/alpha",
			values:    []map[string]any{{"hash": "commit1"}},
			wantQuery: "path=skills%2Falpha",
			wantSHA:   "commit1",
		},
		{
			name:      "empty path omits the filter",
			path:      "",
			values:    []map[string]any{{"hash": "commit2"}},
			wantQuery: "pagelen=1",
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
				writeJSON(t, w, map[string]any{"values": tt.values})
			})

			sha, err := client.LatestCommitForPath(context.Background(), "ws", "repo", "refs/heads/main", tt.path)
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
			if !strings.Contains((*requested)[0], tt.wantQuery) {
				t.Fatalf("requested %q, want it to contain %q", (*requested)[0], tt.wantQuery)
			}
			if tt.path == "" && strings.Contains((*requested)[0], "path=") {
				t.Fatalf("empty path must not add a path filter: %q", (*requested)[0])
			}
		})
	}
}
