package project

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func newTestFactory(cfg *config.Config) (*cmdutil.Factory, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    stdout,
		ErrOut: stderr,
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      ios,
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}
	return f, stdout, stderr
}

func runProjectCmd(t *testing.T, f *cmdutil.Factory, args ...string) error {
	t.Helper()

	cmd := NewCmdProject(f)
	cmd.PersistentFlags().String("context", "", "Named context to use")
	cmd.PersistentFlags().String("output", "text", "Output format")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	cmd.SetArgs(args)
	cmd.SetOut(f.IOStreams.Out)
	cmd.SetErr(f.IOStreams.ErrOut)

	return cmd.ExecuteContext(context.Background())
}

func dcConfig(baseURL string) *config.Config {
	return &config.Config{
		ActiveContext: "test",
		Contexts: map[string]*config.Context{
			"test": {Host: "mock", ProjectKey: "PROJ"},
		},
		Hosts: map[string]*config.Host{
			"mock": {Kind: "dc", BaseURL: baseURL, Username: "admin", Token: "token"},
		},
	}
}

func cloudConfig(baseURL string) *config.Config {
	return &config.Config{
		ActiveContext: "test",
		Contexts: map[string]*config.Context{
			"test": {Host: "mock", Workspace: "ws"},
		},
		Hosts: map[string]*config.Host{
			"mock": {Kind: "cloud", BaseURL: baseURL, Username: "admin", Token: "token"},
		},
	}
}

func TestProjectList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/1.0/projects") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"size":       2,
			"limit":      25,
			"isLastPage": true,
			"start":      0,
			"values": []map[string]any{
				{
					"id":          1,
					"key":         "team",
					"name":        "Team Project",
					"description": "  Team space  ",
					"public":      true,
					"type":        "NORMAL",
				},
				{
					"id":     2,
					"key":    "ops",
					"name":   "Operations",
					"public": false,
					"type":   "NORMAL",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runProjectCmd(t, f, "list"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	out := stdout.String()
	// Key is uppercased in the formatter.
	if !strings.Contains(out, "TEAM\tTeam Project") {
		t.Errorf("expected TEAM row, got: %s", out)
	}
	if !strings.Contains(out, "OPS\tOperations") {
		t.Errorf("expected OPS row, got: %s", out)
	}
	// Web URL uses the uppercased key.
	if !strings.Contains(out, srv.URL+"/projects/TEAM") {
		t.Errorf("expected TEAM project link, got: %s", out)
	}
	// Description should be trimmed.
	if !strings.Contains(out, "desc: Team space") {
		t.Errorf("expected trimmed description, got: %s", out)
	}
	// Visibility only printed for public projects.
	if !strings.Contains(out, "visibility: public") {
		t.Errorf("expected 'visibility: public' for TEAM, got: %s", out)
	}
	// Private projects omit the visibility line.
	if strings.Count(out, "visibility: public") != 1 {
		t.Errorf("expected exactly one 'visibility: public' line, got: %s", out)
	}
}

func TestProjectListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"size":       0,
			"limit":      25,
			"isLastPage": true,
			"start":      0,
			"values":     []any{},
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, _ := newTestFactory(dcConfig(srv.URL))
	if err := runProjectCmd(t, f, "list"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No projects visible") {
		t.Errorf("expected empty message, got: %s", stdout.String())
	}
}

func TestProjectListRejectsCloud(t *testing.T) {
	f, _, _ := newTestFactory(cloudConfig("http://localhost"))
	err := runProjectCmd(t, f, "list")
	if err == nil {
		t.Fatal("expected error on Cloud host")
	}
	if !strings.Contains(err.Error(), "Data Center") {
		t.Errorf("unexpected error: %v", err)
	}
}

func reviewerGroupsHandler(t *testing.T, wantProject string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/rest/api/1.0/projects/" + wantProject + "/settings/reviewer-groups"
		if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", r.URL.Path, wantPath)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{
					"id":          7,
					"name":        "backend-team",
					"description": "  Backend reviewers  ",
					"scope":       map[string]any{"type": "PROJECT", "resourceId": 3},
					"users": []map[string]any{
						{"name": "alice", "slug": "alice", "id": 10, "displayName": "Alice"},
						{"name": "bob", "slug": "bob", "id": 20, "displayName": "Bob"},
					},
				},
				{"id": 8, "name": "frontend-team"},
			},
		})
	})
}

func TestProjectReviewerGroupsList(t *testing.T) {
	srv := httptest.NewServer(reviewerGroupsHandler(t, "PROJ"))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runProjectCmd(t, f, "reviewer-groups", "list"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "backend-team\t(id: 7, members: 2)") {
		t.Errorf("expected backend-team row, got: %s", out)
	}
	// Description should be trimmed.
	if !strings.Contains(out, "desc: Backend reviewers") {
		t.Errorf("expected trimmed description, got: %s", out)
	}
	if !strings.Contains(out, "member: Alice (alice)") {
		t.Errorf("expected Alice member line, got: %s", out)
	}
	if !strings.Contains(out, "frontend-team\t(id: 8, members: 0)") {
		t.Errorf("expected frontend-team row, got: %s", out)
	}
}

func TestProjectReviewerGroupsListProjectOverride(t *testing.T) {
	srv := httptest.NewServer(reviewerGroupsHandler(t, "OTHER"))
	t.Cleanup(srv.Close)

	f, _, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runProjectCmd(t, f, "reviewer-groups", "list", "--project", "OTHER"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}
}

func TestProjectReviewerGroupsListLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values":     []any{},
		})
	}))
	t.Cleanup(srv.Close)

	f, _, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runProjectCmd(t, f, "reviewer-groups", "list", "--limit", "5"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}
}

func TestProjectReviewerGroupsListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values":     []any{},
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, _ := newTestFactory(dcConfig(srv.URL))
	if err := runProjectCmd(t, f, "reviewer-groups", "list"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No reviewer groups defined for project PROJ.") {
		t.Errorf("expected empty message, got: %s", stdout.String())
	}
}

func TestProjectReviewerGroupsListRequiresProject(t *testing.T) {
	cfg := dcConfig("http://localhost")
	cfg.Contexts["test"].ProjectKey = ""

	f, _, _ := newTestFactory(cfg)
	err := runProjectCmd(t, f, "reviewer-groups", "list")
	if err == nil {
		t.Fatal("expected error without project")
	}
	if !strings.Contains(err.Error(), "--project") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProjectReviewerGroupsListRejectsCloud(t *testing.T) {
	f, _, _ := newTestFactory(cloudConfig("http://localhost"))
	err := runProjectCmd(t, f, "reviewer-groups", "list")
	if err == nil {
		t.Fatal("expected error on Cloud host")
	}
	if !strings.Contains(err.Error(), "Data Center") {
		t.Errorf("unexpected error: %v", err)
	}
}
