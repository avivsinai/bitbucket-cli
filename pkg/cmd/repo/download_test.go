package repo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
)

func cloudRepoConfig(baseURL string) *config.Config {
	return &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", Workspace: "ws", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "cloud", BaseURL: baseURL, Token: "test-token"},
		},
	}
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestDownloadListCloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/repositories/other/svc/downloads" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"name": "before.png", "size": 10},
				{"name": "after.png", "size": 20},
			},
		})
	}))
	t.Cleanup(server.Close)

	stdout := &strings.Builder{}
	cmd := newDownloadListCmd(repoTestFactory(cloudRepoConfig(server.URL), stdout))
	cmd.SetContext(context.Background())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--workspace", "other", "--repo", "svc"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := stdout.String(), "before.png\nafter.png\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestDownloadListCloudEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"values":[]}`)
	}))
	t.Cleanup(server.Close)

	stdout := &strings.Builder{}
	cmd := newDownloadListCmd(repoTestFactory(cloudRepoConfig(server.URL), stdout))
	cmd.SetContext(context.Background())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := stdout.String(), "No downloads in ws/repo.\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestDownloadUploadCloud(t *testing.T) {
	dir := t.TempDir()
	first := writeTempFile(t, dir, "before.png", "one")
	second := writeTempFile(t, dir, "after image.png", "two")

	var uploaded []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/repositories/ws/repo/downloads" {
			http.NotFound(w, r)
			return
		}
		file, header, err := r.FormFile("files")
		if err != nil {
			t.Errorf("read form file: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		body, _ := io.ReadAll(file)
		uploaded = append(uploaded, header.Filename+"="+string(body))
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	stdout := &strings.Builder{}
	cmd := newDownloadUploadCmd(repoTestFactory(cloudRepoConfig(server.URL), stdout))
	cmd.SetContext(context.Background())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{first, second})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := strings.Join(uploaded, ","), "before.png=one,after image.png=two"; got != want {
		t.Fatalf("uploaded = %q, want %q", got, want)
	}
	want := "Uploaded: before.png\nhttps://bitbucket.org/ws/repo/downloads/before.png\n" +
		"Uploaded: after image.png\nhttps://bitbucket.org/ws/repo/downloads/after%20image.png\n"
	if got := stdout.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestDownloadUploadValidation(t *testing.T) {
	dir := t.TempDir()
	valid := writeTempFile(t, dir, "ok.png", "x")
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}
	dupe := writeTempFile(t, subdir, "ok.png", "y")

	tests := []struct {
		name          string
		hostKind      string
		cfg           func(*config.Config)
		args          []string
		errorContains string
	}{
		{
			name:          "data center rejected",
			hostKind:      "dc",
			args:          []string{valid},
			errorContains: "only available for Bitbucket Cloud",
		},
		{
			name:          "missing workspace",
			cfg:           func(c *config.Config) { c.Contexts["default"].Workspace = "" },
			args:          []string{valid},
			errorContains: "workspace required",
		},
		{
			name:          "missing repo",
			cfg:           func(c *config.Config) { c.Contexts["default"].DefaultRepo = "" },
			args:          []string{valid},
			errorContains: "repository slug required",
		},
		{
			name:          "missing file",
			args:          []string{valid, filepath.Join(dir, "nope.png")},
			errorContains: "file not found",
		},
		{
			name:          "directory",
			args:          []string{valid, subdir},
			errorContains: "cannot upload directory",
		},
		{
			name:          "duplicate base name",
			args:          []string{valid, dupe},
			errorContains: `duplicate file name "ok.png"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				w.WriteHeader(http.StatusCreated)
			}))
			t.Cleanup(server.Close)

			cfg := cloudRepoConfig(server.URL)
			if tt.hostKind != "" {
				cfg.Hosts["main"].Kind = tt.hostKind
			}
			if tt.cfg != nil {
				tt.cfg(cfg)
			}

			cmd := newDownloadUploadCmd(repoTestFactory(cfg, &strings.Builder{}))
			cmd.SetContext(context.Background())
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("error = %q, want substring %q", err, tt.errorContains)
			}
			if hits != 0 {
				t.Fatalf("expected validation to avoid HTTP requests, got %d", hits)
			}
		})
	}
}
