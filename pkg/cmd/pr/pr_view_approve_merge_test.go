package pr_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/browser"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/root"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

type recordingBrowser struct {
	url string
	err error
}

func (b *recordingBrowser) Open(url string) error {
	b.url = url
	return b.err
}

func TestPRViewDataCenter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pull-requests/42") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          42,
				"title":       "Speed up tests",
				"state":       "OPEN",
				"description": "Make slow tests fast",
				"author": map[string]any{
					"user": map[string]any{
						"name":        "alice",
						"displayName": "Alice",
					},
				},
				"fromRef": map[string]any{
					"displayId": "feature/tests",
				},
				"toRef": map[string]any{
					"displayId": "master",
				},
				"reviewers": []map[string]any{
					{
						"user": map[string]any{
							"name":        "bob",
							"displayName": "Bob",
						},
					},
				},
				"links": map[string]any{
					"self": []map[string]any{
						{"href": "https://bitbucket.example.com/projects/PROJ/repos/my-repo/pull-requests/42"},
					},
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "view", "42")
	if err != nil {
		t.Fatalf("pr view error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	for _, want := range []string{
		"Pull Request #42: Speed up tests",
		"State: OPEN",
		"Author: Alice",
		"From: feature/tests",
		"To:   master",
		"Make slow tests fast",
		"Reviewers:",
		"Bob",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestPRViewCloudWeb(t *testing.T) {
	b := &recordingBrowser{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method == "GET" && r.URL.Path == "/repositories/myworkspace/my-repo/pullrequests/7" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    7,
				"title": "Cloud PR",
				"state": "OPEN",
				"author": map[string]any{
					"display_name": "Cloud User",
					"username":     "cloud-user",
				},
				"source": map[string]any{
					"branch": map[string]any{"name": "feature/cloud"},
				},
				"destination": map[string]any{
					"branch": map[string]any{"name": "master"},
				},
				"summary": map[string]any{
					"raw": "Cloud summary",
				},
				"links": map[string]any{
					"html": map[string]any{
						"href": "https://bitbucket.org/myworkspace/my-repo/pull-requests/7",
					},
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLIWithBrowser(t, cloudConfig(srv.URL), b, "pr", "view", "7", "--web")
	if err != nil {
		t.Fatalf("pr view --web error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if b.url != "https://bitbucket.org/myworkspace/my-repo/pull-requests/7" {
		t.Fatalf("browser opened %q", b.url)
	}
	for _, want := range []string{
		"Pull Request #7: Cloud PR",
		"Author: Cloud User",
		"From: feature/cloud",
		"To:   master",
		"Cloud summary",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestPRApproveDataCenter(t *testing.T) {
	var approveCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pull-requests/42/approve") {
			approveCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "approve", "42")
	if err != nil {
		t.Fatalf("pr approve error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !approveCalled {
		t.Fatal("approve endpoint not called")
	}
	if !strings.Contains(stdout, "Approved pull request #42") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestPRApproveCloud(t *testing.T) {
	var approveCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method == "POST" && r.URL.Path == "/repositories/myworkspace/my-repo/pullrequests/42/approve" {
			approveCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "approve", "42")
	if err != nil {
		t.Fatalf("pr approve error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if !approveCalled {
		t.Fatal("approve endpoint not called")
	}
	if !strings.Contains(stdout, "Approved pull request #42") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestPRMergeDataCenter(t *testing.T) {
	var mergeBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pull-requests/42"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      42,
				"title":   "Merge me",
				"state":   "OPEN",
				"version": 3,
			})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pull-requests/42/merge"):
			_ = json.NewDecoder(r.Body).Decode(&mergeBody)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "merge", "42", "--message", "Ship it", "--strategy", "squash", "--close-source=false")
	if err != nil {
		t.Fatalf("pr merge error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if got := int(mergeBody["version"].(float64)); got != 3 {
		t.Fatalf("version = %d", got)
	}
	if got := mergeBody["message"]; got != "Ship it" {
		t.Fatalf("message = %v", got)
	}
	if got := mergeBody["mergeStrategyId"]; got != "squash" {
		t.Fatalf("mergeStrategyId = %v", got)
	}
	if got := mergeBody["closeSourceBranch"]; got != false {
		t.Fatalf("closeSourceBranch = %v", got)
	}
	if !strings.Contains(stdout, "Merged pull request #42") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestPRMergeCloud(t *testing.T) {
	var mergeBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token"))
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method == "POST" && r.URL.Path == "/repositories/myworkspace/my-repo/pullrequests/42/merge" {
			_ = json.NewDecoder(r.Body).Decode(&mergeBody)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "merge", "42", "--message", "Ship it", "--strategy", "squash", "--close-source=false")
	if err != nil {
		t.Fatalf("pr merge error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	if got := mergeBody["message"]; got != "Ship it" {
		t.Fatalf("message = %v", got)
	}
	if got := mergeBody["merge_strategy"]; got != "squash" {
		t.Fatalf("merge_strategy = %v", got)
	}
	if got := mergeBody["close_source_branch"]; got != false {
		t.Fatalf("close_source_branch = %v", got)
	}
	if !strings.Contains(stdout, "Merged pull request #42") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func runCLIWithBrowser(t *testing.T, cfg *config.Config, b browser.Browser, args ...string) (string, string, error) {
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
		Browser:        b,
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
	rootCmd.SilenceUsage = true

	t.Setenv("BKT_NO_UPDATE_CHECK", "1")
	t.Setenv("NO_COLOR", "1")

	err = rootCmd.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}
