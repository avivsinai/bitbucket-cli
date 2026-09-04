package source_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

// newCloudRepo wires a Cloud adapter to a stub server and records request URLs.
func newCloudRepo(t *testing.T, handler http.HandlerFunc) (source.Repository, *[]string) {
	t.Helper()
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.String())
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatalf("bbcloud.New: %v", err)
	}
	return source.NewCloudRepository(client, "myteam", "agent-skills"), &requested
}

// newDCRepo wires a Data Center adapter to a stub server and records request URLs.
func newDCRepo(t *testing.T, handler http.HandlerFunc) (source.Repository, *[]string) {
	t.Helper()
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.String())
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatalf("bbdc.New: %v", err)
	}
	return source.NewDCRepository(client, "https://bitbucket.example.com", "PROJ", "agent-skills"), &requested
}

func writeJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func notFound(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }

func TestRepositoryIdentity(t *testing.T) {
	tests := []struct {
		name      string
		repo      source.Repository
		wantFull  string
		wantWeb   string
		wantClone string
	}{
		{
			name:      "cloud",
			repo:      source.NewCloudRepository(nil, "myteam", "agent-skills"),
			wantFull:  "myteam/agent-skills",
			wantWeb:   "https://bitbucket.org/myteam/agent-skills",
			wantClone: "https://bitbucket.org/myteam/agent-skills.git",
		},
		{
			// Data Center browse URLs keep the project key's case, but /scm/
			// clone URLs lower-case it.
			name:      "data center",
			repo:      source.NewDCRepository(nil, "https://bitbucket.example.com/", "PROJ", "agent-skills"),
			wantFull:  "PROJ/agent-skills",
			wantWeb:   "https://bitbucket.example.com/projects/PROJ/repos/agent-skills",
			wantClone: "https://bitbucket.example.com/scm/proj/agent-skills.git",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.repo.FullName(); got != tt.wantFull {
				t.Errorf("FullName = %q, want %q", got, tt.wantFull)
			}
			if got := tt.repo.WebURL(); got != tt.wantWeb {
				t.Errorf("WebURL = %q, want %q", got, tt.wantWeb)
			}
			if got := tt.repo.CloneURL(); got != tt.wantClone {
				t.Errorf("CloneURL = %q, want %q", got, tt.wantClone)
			}
		})
	}
}

func TestCloudAdapterReads(t *testing.T) {
	repo, requested := newCloudRepo(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/src/") && strings.HasSuffix(r.URL.Path, "SKILL.md"):
			_, _ = w.Write([]byte("# body\n"))
		case strings.Contains(r.URL.Path, "/src/"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{
				{"path": "skills/alpha", "type": "commit_directory"},
				{"path": "skills/alpha/SKILL.md", "type": "commit_file", "size": 7},
			}})
		case strings.HasSuffix(r.URL.Path, "/refs/tags"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{
				{"name": "v2.0.0", "target": map[string]any{"hash": "tag2"}},
			}})
		case strings.Contains(r.URL.Path, "/refs/tags/"):
			writeJSON(t, w, map[string]any{"name": "v1.0.0", "target": map[string]any{"hash": "tag1"}})
		case strings.Contains(r.URL.Path, "/refs/branches/"):
			writeJSON(t, w, map[string]any{"name": "main", "target": map[string]any{"hash": "branchsha"}})
		case strings.Contains(r.URL.Path, "/commits/"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{{"hash": "pathcommit"}}})
		case strings.Contains(r.URL.Path, "/commit/"):
			writeJSON(t, w, map[string]any{"hash": "commitsha"})
		default: // repository detail
			writeJSON(t, w, map[string]any{"mainbranch": map[string]any{"name": "develop"}})
		}
	})
	ctx := context.Background()

	if sha, err := repo.Branch(ctx, "main"); err != nil || sha != "branchsha" {
		t.Errorf("Branch = %q, %v", sha, err)
	}
	if sha, err := repo.Tag(ctx, "v1.0.0"); err != nil || sha != "tag1" {
		t.Errorf("Tag = %q, %v", sha, err)
	}
	if sha, err := repo.Commit(ctx, "abc"); err != nil || sha != "commitsha" {
		t.Errorf("Commit = %q, %v", sha, err)
	}
	name, sha, err := repo.LatestTag(ctx)
	if err != nil || name != "v2.0.0" || sha != "tag2" {
		t.Errorf("LatestTag = %q/%q, %v", name, sha, err)
	}
	if branch, err := repo.DefaultBranch(ctx); err != nil || branch != "develop" {
		t.Errorf("DefaultBranch = %q, %v", branch, err)
	}
	if commit, err := repo.LatestCommit(ctx, "sha", "skills/alpha"); err != nil || commit != "pathcommit" {
		t.Errorf("LatestCommit = %q, %v", commit, err)
	}

	// Directories are dropped; files keep their repository-root-relative path.
	files, err := repo.ListFiles(ctx, "sha", "skills")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 || files[0].Path != "skills/alpha/SKILL.md" || files[0].Size != 7 {
		t.Fatalf("files = %+v", files)
	}

	content, err := repo.ReadFile(ctx, "sha", "skills/alpha/SKILL.md")
	if err != nil || string(content) != "# body\n" {
		t.Fatalf("ReadFile = %q, %v", content, err)
	}

	if len(*requested) == 0 {
		t.Fatal("expected requests to be recorded")
	}
}

func TestCloudAdapterNotFound(t *testing.T) {
	repo, _ := newCloudRepo(t, notFound)
	ctx := context.Background()

	// Every lookup must report source.ErrNotFound, because ResolveRef's
	// branch -> tag -> commit fallback keys off it.
	tests := []struct {
		name string
		call func() error
	}{
		{"branch", func() error { _, err := repo.Branch(ctx, "missing"); return err }},
		{"tag", func() error { _, err := repo.Tag(ctx, "missing"); return err }},
		{"commit", func() error { _, err := repo.Commit(ctx, "missing"); return err }},
		{"list files", func() error { _, err := repo.ListFiles(ctx, "sha", "skills/missing"); return err }},
		{"read file", func() error { _, err := repo.ReadFile(ctx, "sha", "missing.md"); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, source.ErrNotFound) {
				t.Fatalf("error = %v, want source.ErrNotFound", err)
			}
		})
	}
}

func TestCloudAdapterEmptyListingForMissingDirectory(t *testing.T) {
	// Bitbucket answers an unknown directory with an empty listing rather than
	// a 404, so the adapter synthesises ErrNotFound for a named directory.
	repo, _ := newCloudRepo(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"values": []map[string]any{}})
	})
	if _, err := repo.ListFiles(context.Background(), "sha", "skills/missing"); !errors.Is(err, source.ErrNotFound) {
		t.Fatalf("error = %v, want source.ErrNotFound", err)
	}
	// An empty repository root is not an error.
	files, err := repo.ListFiles(context.Background(), "sha", "")
	if err != nil || len(files) != 0 {
		t.Fatalf("root listing = %+v, %v", files, err)
	}
}

func TestCloudAdapterLatestTagWithoutTags(t *testing.T) {
	repo, _ := newCloudRepo(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"values": []map[string]any{}})
	})
	if _, _, err := repo.LatestTag(context.Background()); !errors.Is(err, source.ErrNotFound) {
		t.Fatalf("error = %v, want source.ErrNotFound so ResolveRef falls back to the default branch", err)
	}
}

func TestDCAdapterReads(t *testing.T) {
	repo, requested := newDCRepo(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/raw/"):
			_, _ = w.Write([]byte("# body\n"))
		case strings.Contains(r.URL.Path, "/files"):
			// Data Center reports paths relative to the requested directory.
			writeJSON(t, w, map[string]any{"values": []string{"SKILL.md", "scripts/run.sh"}, "isLastPage": true})
		case strings.HasSuffix(r.URL.Path, "/tags"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{
				{"id": "refs/tags/v2.0.0", "displayId": "v2.0.0", "latestCommit": "tag2"},
			}, "isLastPage": true})
		case strings.Contains(r.URL.Path, "/tags/"):
			writeJSON(t, w, map[string]any{"displayId": "v1.0.0", "latestCommit": "tag1"})
		case strings.Contains(r.URL.Path, "/default-branch"):
			writeJSON(t, w, map[string]any{"id": "refs/heads/develop", "displayId": "develop"})
		case strings.Contains(r.URL.Path, "/branches"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{
				{"id": "refs/heads/main", "displayId": "main", "latestCommit": "branchsha"},
			}, "isLastPage": true})
		case strings.Contains(r.URL.Path, "/commits/"):
			writeJSON(t, w, map[string]any{"id": "commitsha"})
		case strings.Contains(r.URL.Path, "/commits"):
			writeJSON(t, w, map[string]any{"values": []map[string]any{{"id": "pathcommit"}}, "isLastPage": true})
		default:
			notFound(w, r)
		}
	})
	ctx := context.Background()

	if sha, err := repo.Branch(ctx, "main"); err != nil || sha != "branchsha" {
		t.Errorf("Branch = %q, %v", sha, err)
	}
	if sha, err := repo.Tag(ctx, "v1.0.0"); err != nil || sha != "tag1" {
		t.Errorf("Tag = %q, %v", sha, err)
	}
	if sha, err := repo.Commit(ctx, "abc"); err != nil || sha != "commitsha" {
		t.Errorf("Commit = %q, %v", sha, err)
	}
	name, sha, err := repo.LatestTag(ctx)
	if err != nil || name != "v2.0.0" || sha != "tag2" {
		t.Errorf("LatestTag = %q/%q, %v", name, sha, err)
	}
	if branch, err := repo.DefaultBranch(ctx); err != nil || branch != "develop" {
		t.Errorf("DefaultBranch = %q, %v", branch, err)
	}
	if commit, err := repo.LatestCommit(ctx, "sha", "skills/alpha"); err != nil || commit != "pathcommit" {
		t.Errorf("LatestCommit = %q, %v", commit, err)
	}

	content, err := repo.ReadFile(ctx, "sha", "skills/alpha/SKILL.md")
	if err != nil || string(content) != "# body\n" {
		t.Fatalf("ReadFile = %q, %v", content, err)
	}

	if len(*requested) == 0 {
		t.Fatal("expected requests to be recorded")
	}
}

func TestDCAdapterListFilesRestoresDirectoryPrefix(t *testing.T) {
	// The Data Center files endpoint returns paths relative to the requested
	// directory; discovery matches on repository-root-relative paths, so the
	// adapter has to put the prefix back.
	tests := []struct {
		name string
		dir  string
		want []string
	}{
		{name: "named directory", dir: "skills/alpha", want: []string{"skills/alpha/SKILL.md", "skills/alpha/scripts/run.sh"}},
		{name: "trailing slash trimmed", dir: "skills/alpha/", want: []string{"skills/alpha/SKILL.md", "skills/alpha/scripts/run.sh"}},
		{name: "repository root", dir: "", want: []string{"SKILL.md", "scripts/run.sh"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _ := newDCRepo(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, map[string]any{"values": []string{"SKILL.md", "scripts/run.sh"}, "isLastPage": true})
			})
			files, err := repo.ListFiles(context.Background(), "sha", tt.dir)
			if err != nil {
				t.Fatalf("ListFiles: %v", err)
			}
			var got []string
			for _, f := range files {
				got = append(got, f.Path)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("paths = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDCAdapterNotFound(t *testing.T) {
	repo, _ := newDCRepo(t, notFound)
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{"branch", func() error { _, err := repo.Branch(ctx, "missing"); return err }},
		{"tag", func() error { _, err := repo.Tag(ctx, "missing"); return err }},
		{"commit", func() error { _, err := repo.Commit(ctx, "missing"); return err }},
		{"list files", func() error { _, err := repo.ListFiles(ctx, "sha", "skills/missing"); return err }},
		{"read file", func() error { _, err := repo.ReadFile(ctx, "sha", "missing.md"); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, source.ErrNotFound) {
				t.Fatalf("error = %v, want source.ErrNotFound", err)
			}
		})
	}
}

func TestDCAdapterLatestTagWithoutTags(t *testing.T) {
	repo, _ := newDCRepo(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"values": []map[string]any{}, "isLastPage": true})
	})
	if _, _, err := repo.LatestTag(context.Background()); !errors.Is(err, source.ErrNotFound) {
		t.Fatalf("error = %v, want source.ErrNotFound so ResolveRef falls back to the default branch", err)
	}
}

// TestResolveRefOverAdapters exercises the real fallback chain end to end, since
// it depends on the adapters translating 404s into source.ErrNotFound.
func TestResolveRefOverAdapters(t *testing.T) {
	t.Run("cloud falls back to the default branch when untagged", func(t *testing.T) {
		repo, _ := newCloudRepo(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasSuffix(r.URL.Path, "/refs/tags"):
				writeJSON(t, w, map[string]any{"values": []map[string]any{}})
			case strings.Contains(r.URL.Path, "/refs/branches/"):
				writeJSON(t, w, map[string]any{"name": "main", "target": map[string]any{"hash": "mainsha"}})
			default:
				writeJSON(t, w, map[string]any{"mainbranch": map[string]any{"name": "main"}})
			}
		})
		ref, err := source.ResolveRef(context.Background(), repo, "")
		if err != nil {
			t.Fatalf("ResolveRef: %v", err)
		}
		if ref.Ref != "refs/heads/main" || ref.SHA != "mainsha" {
			t.Fatalf("ref = %+v", ref)
		}
	})

	t.Run("data center resolves a tag after the branch lookup misses", func(t *testing.T) {
		repo, _ := newDCRepo(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/branches"):
				writeJSON(t, w, map[string]any{"values": []map[string]any{}, "isLastPage": true})
			case strings.Contains(r.URL.Path, "/tags/"):
				writeJSON(t, w, map[string]any{"displayId": "v1.0.0", "latestCommit": "tagsha"})
			default:
				notFound(w, r)
			}
		})
		ref, err := source.ResolveRef(context.Background(), repo, "v1.0.0")
		if err != nil {
			t.Fatalf("ResolveRef: %v", err)
		}
		if ref.Ref != "refs/tags/v1.0.0" || ref.SHA != "tagsha" {
			t.Fatalf("ref = %+v", ref)
		}
	})
}

// Both adapters must satisfy TagCreator; publish depends on it.
var (
	_ source.TagCreator = source.NewCloudRepository(nil, "w", "r").(source.TagCreator)
	_ source.TagCreator = source.NewDCRepository(nil, "https://h", "P", "r").(source.TagCreator)
)

func TestAdapterCreateTag(t *testing.T) {
	tests := []struct {
		name     string
		newRepo  func(*testing.T, http.HandlerFunc) (source.Repository, *[]string)
		wantPath string
	}{
		{name: "cloud", newRepo: newCloudRepo, wantPath: "/repositories/myteam/agent-skills/refs/tags"},
		{name: "data center", newRepo: newDCRepo, wantPath: "/rest/api/1.0/projects/PROJ/repos/agent-skills/tags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			repo, _ := tt.newRepo(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				writeJSON(t, w, map[string]any{})
			})
			tagger, ok := repo.(source.TagCreator)
			if !ok {
				t.Fatal("adapter does not implement source.TagCreator")
			}
			if err := tagger.CreateTag(context.Background(), "v1.0.0", "abc123", "First release"); err != nil {
				t.Fatalf("CreateTag: %v", err)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestAdapterCreateTagNotFound(t *testing.T) {
	// A 404 must reach callers as source.ErrNotFound, not the client's sentinel.
	tests := []struct {
		name    string
		newRepo func(*testing.T, http.HandlerFunc) (source.Repository, *[]string)
	}{
		{name: "cloud", newRepo: newCloudRepo},
		{name: "data center", newRepo: newDCRepo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _ := tt.newRepo(t, notFound)
			err := repo.(source.TagCreator).CreateTag(context.Background(), "v1.0.0", "abc123", "")
			if !errors.Is(err, source.ErrNotFound) {
				t.Fatalf("error = %v, want source.ErrNotFound", err)
			}
		})
	}
}
