package pr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestCreateCommandDefinesSourceRepositoryFlags(t *testing.T) {
	cmd := newCreateCmd(nil)
	for _, name := range []string{"source-project", "source-repo"} {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("flag --%s is not defined", name)
		}
		if flag.DefValue != "" {
			t.Errorf("--%s default = %q, want empty", name, flag.DefValue)
		}
	}
}

func TestCreateDataCenterCrossRepositoryRequest(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/api/1.0/projects/DEST/repos/upstream/pull-requests" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 17, "title": "Fork PR"})
	}))
	t.Cleanup(server.Close)

	cmd := newCreateCmd(newForkCreateFactory(server.URL, "dc"))
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--title", "Fork PR",
		"--source", "feature",
		"--target", "main",
		"--source-project", "FORK",
		"--source-repo", "contributor-fork",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	assertCommandRequestRepository(t, gotBody, "fromRef", "FORK", "contributor-fork")
	assertCommandRequestRepository(t, gotBody, "toRef", "DEST", "upstream")
}

func TestCreateDataCenterCrossRepositoryDefaultReviewers(t *testing.T) {
	var gotCreateBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/1.0/projects/DEST/repos/upstream":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202, "slug": "upstream"})
		case "/rest/api/1.0/projects/FORK/repos/contributor-fork":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "slug": "contributor-fork"})
		case "/rest/default-reviewers/1.0/projects/DEST/repos/upstream/reviewers":
			query := r.URL.Query()
			if query.Get("sourceRepoId") != "101" || query.Get("targetRepoId") != "202" {
				t.Fatalf("default reviewer repo IDs = source %q, target %q", query.Get("sourceRepoId"), query.Get("targetRepoId"))
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "alice"}})
		case "/rest/api/1.0/projects/DEST/repos/upstream/pull-requests":
			if err := json.NewDecoder(r.Body).Decode(&gotCreateBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 17, "title": "Fork PR"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	cmd := newCreateCmd(newForkCreateFactory(server.URL, "dc"))
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--title", "Fork PR",
		"--source", "feature",
		"--target", "main",
		"--source-project", "FORK",
		"--source-repo", "contributor-fork",
		"--with-default-reviewers",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	assertCommandRequestRepository(t, gotCreateBody, "fromRef", "FORK", "contributor-fork")
	reviewers, ok := gotCreateBody["reviewers"].([]any)
	if !ok || len(reviewers) != 1 {
		t.Fatalf("reviewers = %#v, want one reviewer", gotCreateBody["reviewers"])
	}
	reviewer, ok := reviewers[0].(map[string]any)
	if !ok {
		t.Fatalf("reviewer = %#v, want object", reviewers[0])
	}
	user, ok := reviewer["user"].(map[string]any)
	if !ok || user["name"] != "alice" {
		t.Fatalf("reviewer user = %#v, want alice", reviewer["user"])
	}
}

func TestCreateCloudRejectsSourceRepositoryFlags(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"--source-project", "FORK"},
		{"--source-repo", "contributor-fork"},
	} {
		t.Run(args[0], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("unexpected Cloud request: %s %s", r.Method, r.URL.Path)
			}))
			t.Cleanup(server.Close)

			cmd := newCreateCmd(newForkCreateFactory(server.URL, "cloud"))
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetContext(context.Background())
			cmd.SetArgs(args)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "only supported on Bitbucket Data Center") {
				t.Fatalf("error = %v, want explicit Data Center-only rejection", err)
			}
		})
	}
}

func newForkCreateFactory(baseURL, kind string) *cmdutil.Factory {
	ctx := &config.Context{Host: "test", ProjectKey: "DEST", Workspace: "workspace", DefaultRepo: "upstream"}
	host := &config.Host{Kind: kind, BaseURL: baseURL, Username: "user", Token: "token"}
	return &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config: func() (*config.Config, error) {
			return &config.Config{
				ActiveContext: "default",
				Contexts:      map[string]*config.Context{"default": ctx},
				Hosts:         map[string]*config.Host{"test": host},
			}, nil
		},
	}
}

func assertCommandRequestRepository(t *testing.T, body map[string]any, ref, wantProject, wantRepo string) {
	t.Helper()
	refBody, ok := body[ref].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", ref, body[ref])
	}
	repository, ok := refBody["repository"].(map[string]any)
	if !ok {
		t.Fatalf("%s.repository = %#v, want object", ref, refBody["repository"])
	}
	if got := repository["slug"]; got != wantRepo {
		t.Errorf("%s.repository.slug = %v, want %q", ref, got, wantRepo)
	}
	project, ok := repository["project"].(map[string]any)
	if !ok {
		t.Fatalf("%s.repository.project = %#v, want object", ref, repository["project"])
	}
	if got := project["key"]; got != wantProject {
		t.Errorf("%s.repository.project.key = %v, want %q", ref, got, wantProject)
	}
}
