package commit_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/commit"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestCommitDiffDC(t *testing.T) {
	diffBody := "diff --git a/foo.go b/foo.go\nindex abc123..def456 100644\n--- a/foo.go\n+++ b/foo.go\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/1.0/projects/PROJ/repos/my-repo/compare/diff") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Verify query params
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if from != "abc123" {
			t.Errorf("expected from=abc123, got from=%s", from)
		}
		if to != "def456" {
			t.Errorf("expected to=def456, got to=%s", to)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(diffBody))
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "my-repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: server.URL, Username: "u", Token: "t"},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    stdout,
			ErrOut: stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := commit.NewCmdCommit(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"diff", "abc123", "def456"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "diff --git a/foo.go b/foo.go") {
		t.Errorf("expected diff output, got: %s", output)
	}
}

func TestCommitDiffCloud(t *testing.T) {
	diffBody := "diff --git a/bar.js b/bar.js\nindex 111222..333444 100644\n--- a/bar.js\n+++ b/bar.js\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/repositories/myws/my-repo/diff/abc123..def456"
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected path: %s, expected: %s", r.URL.Path, expectedPath)
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(diffBody))
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "cloud", Workspace: "myws", DefaultRepo: "my-repo"},
		},
		Hosts: map[string]*config.Host{
			"cloud": {Kind: "cloud", BaseURL: server.URL, Username: "u", Token: "t"},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    stdout,
			ErrOut: stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := commit.NewCmdCommit(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"diff", "abc123", "def456"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "diff --git a/bar.js b/bar.js") {
		t.Errorf("expected diff output, got: %s", output)
	}
}

func TestCommitDiffEmptyDC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/1.0/projects/PROJ/repos/my-repo/compare/diff") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		// Empty body — simulates reversed refs with no diff
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "my-repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: server.URL, Username: "u", Token: "t"},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    stdout,
			ErrOut: stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := commit.NewCmdCommit(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"diff", "def456", "abc123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if stdout.String() != "" {
		t.Errorf("expected empty stdout, got: %s", stdout.String())
	}

	if !strings.Contains(stderr.String(), "(empty diff)") {
		t.Errorf("expected stderr to contain \"(empty diff)\", got: %s", stderr.String())
	}
}

func TestCommitDiffEmptyCloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/repositories/myws/my-repo/diff/def456..abc123"
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected path: %s, expected: %s", r.URL.Path, expectedPath)
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		// Empty body — simulates reversed refs with no diff
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "cloud", Workspace: "myws", DefaultRepo: "my-repo"},
		},
		Hosts: map[string]*config.Host{
			"cloud": {Kind: "cloud", BaseURL: server.URL, Username: "u", Token: "t"},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    stdout,
			ErrOut: stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := commit.NewCmdCommit(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"diff", "def456", "abc123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if stdout.String() != "" {
		t.Errorf("expected empty stdout, got: %s", stdout.String())
	}

	if !strings.Contains(stderr.String(), "(empty diff)") {
		t.Errorf("expected stderr to contain \"(empty diff)\", got: %s", stderr.String())
	}
}

func TestCommitDiffTwoArgsMissing(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "my-repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: "https://bitbucket.example.com", Username: "u", Token: "t"},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    stdout,
			ErrOut: stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := commit.NewCmdCommit(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"diff", "abc123"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when only one positional arg supplied")
	}
}

func TestCommitDiffUnsupportedHostKind(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "alien"},
		},
		Hosts: map[string]*config.Host{
			"alien": {Kind: "gitea", BaseURL: "https://example.com", Username: "u", Token: "t"},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    stdout,
			ErrOut: stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := commit.NewCmdCommit(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"diff", "abc123", "def456"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported host kind")
	}
	if !strings.Contains(err.Error(), "unsupported host kind") {
		t.Errorf("expected 'unsupported host kind' error, got: %v", err)
	}
}

func TestCommitDiffMissingContext(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: "https://bitbucket.example.com", Username: "u", Token: "t"},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    stdout,
			ErrOut: stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := commit.NewCmdCommit(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"diff", "abc123", "def456"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing context")
	}

	if !strings.Contains(err.Error(), "project and repo") {
		t.Errorf("expected error containing 'project and repo', got: %v", err)
	}
}
