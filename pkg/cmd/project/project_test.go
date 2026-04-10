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
