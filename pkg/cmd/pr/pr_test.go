package pr_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/root"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestPRDeclineDataCenter(t *testing.T) {
	var declineCalled bool
	var declineMethod, declinePath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pull-requests/42"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id":      42,
				"title":   "Test PR",
				"state":   "OPEN",
				"version": 7,
				"fromRef": map[string]any{
					"id":        "refs/heads/feature",
					"displayId": "feature",
				},
				"toRef": map[string]any{
					"id":        "refs/heads/main",
					"displayId": "main",
				},
			})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pull-requests/42/decline"):
			declineCalled = true
			declineMethod = r.Method
			declinePath = r.URL.Path
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["version"].(float64); !ok || int(v) != 7 {
				t.Errorf("expected version=7, got %v", body["version"])
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "decline", "42")
	if err != nil {
		t.Fatalf("pr decline error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !declineCalled {
		t.Fatal("decline endpoint was not called")
	}
	if declineMethod != "POST" {
		t.Errorf("expected POST, got %s", declineMethod)
	}
	if !strings.Contains(declinePath, "/pull-requests/42/decline") {
		t.Errorf("unexpected path: %s", declinePath)
	}
	if !strings.Contains(stdout, "Declined pull request #42") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

func TestPRDeclineWithDeleteSource(t *testing.T) {
	var deleteBranchCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pull-requests/10"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id":      10,
				"title":   "Test PR",
				"state":   "OPEN",
				"version": 2,
				"fromRef": map[string]any{
					"id":        "refs/heads/feature-branch",
					"displayId": "feature-branch",
				},
				"toRef": map[string]any{
					"id":        "refs/heads/main",
					"displayId": "main",
				},
			})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pull-requests/10/decline"):
			w.WriteHeader(http.StatusOK)
		case r.Method == "DELETE" && strings.HasSuffix(r.URL.Path, "/branches"):
			deleteBranchCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "decline", "10", "--delete-source")
	if err != nil {
		t.Fatalf("pr decline --delete-source error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !deleteBranchCalled {
		t.Fatal("delete branch endpoint was not called")
	}
	if !strings.Contains(stdout, "Declined pull request #10") {
		t.Errorf("expected decline message in output: %s", stdout)
	}
	if !strings.Contains(stdout, "Deleted source branch feature-branch") {
		t.Errorf("expected branch deletion message in output: %s", stdout)
	}
}

func TestPRReopenDataCenter(t *testing.T) {
	var reopenCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pull-requests/42"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id":      42,
				"title":   "Test PR",
				"state":   "DECLINED",
				"version": 8,
			})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pull-requests/42/reopen"):
			reopenCalled = true
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if v, ok := body["version"].(float64); !ok || int(v) != 8 {
				t.Errorf("expected version=8, got %v", body["version"])
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "reopen", "42")
	if err != nil {
		t.Fatalf("pr reopen error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !reopenCalled {
		t.Fatal("reopen endpoint was not called")
	}
	if !strings.Contains(stdout, "Reopened pull request #42") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

func TestPRDeclineRequiresArg(t *testing.T) {
	_, _, err := runCLI(t, dcConfig("http://localhost"), "pr", "decline")
	if err == nil {
		t.Fatal("expected error when no PR ID provided")
	}
}

func TestPRReopenRequiresArg(t *testing.T) {
	_, _, err := runCLI(t, dcConfig("http://localhost"), "pr", "reopen")
	if err == nil {
		t.Fatal("expected error when no PR ID provided")
	}
}

func TestPRDeclineInvalidID(t *testing.T) {
	_, _, err := runCLI(t, dcConfig("http://localhost"), "pr", "decline", "abc")
	if err == nil {
		t.Fatal("expected error for invalid PR ID")
	}
	if !strings.Contains(err.Error(), "invalid pull request id") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPRReopenInvalidID(t *testing.T) {
	_, _, err := runCLI(t, dcConfig("http://localhost"), "pr", "reopen", "abc")
	if err == nil {
		t.Fatal("expected error for invalid PR ID")
	}
	if !strings.Contains(err.Error(), "invalid pull request id") {
		t.Errorf("unexpected error: %v", err)
	}
}

func dcConfig(baseURL string) *config.Config {
	return &config.Config{
		ActiveContext: "test",
		Contexts: map[string]*config.Context{
			"test": {
				Host:        "mock",
				ProjectKey:  "PROJ",
				DefaultRepo: "my-repo",
			},
		},
		Hosts: map[string]*config.Host{
			"mock": {
				Kind:     "dc",
				BaseURL:  baseURL,
				Username: "admin",
				Token:    "token",
			},
		},
	}
}

func runCLI(t *testing.T, cfg *config.Config, args ...string) (string, string, error) {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    stdout,
		ErrOut: stderr,
	}

	factory := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      ios,
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	rootCmd, err := root.NewCmdRoot(factory)
	if err != nil {
		t.Fatalf("NewCmdRoot: %v", err)
	}
	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	t.Setenv("BKT_NO_UPDATE_CHECK", "1")
	t.Setenv("NO_COLOR", "1")

	// Suppress usage output on errors.
	rootCmd.SilenceUsage = true

	err = rootCmd.ExecuteContext(context.Background())
	_ = fmt.Sprint() // satisfy import
	return stdout.String(), stderr.String(), err
}
