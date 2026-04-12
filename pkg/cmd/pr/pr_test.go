package pr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

func TestListRequiresMineWithoutRepo(t *testing.T) {
	// Change to a temp directory without a git repo to prevent
	// applyRemoteDefaults from overwriting test context values.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	tests := []struct {
		name          string
		context       *config.Context
		host          *config.Host
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name: "dc without repo and without mine",
			context: &config.Context{
				Host:       "main",
				ProjectKey: "PROJ",
				// No DefaultRepo
			},
			host: &config.Host{
				Kind:     "dc",
				BaseURL:  "https://bitbucket.example.com",
				Username: "testuser",
				Token:    "test-token",
			},
			args:          []string{},
			expectError:   true,
			errorContains: "--mine is required when not specifying a repository",
		},
		{
			name: "cloud without repo and without mine",
			context: &config.Context{
				Host:      "cloud",
				Workspace: "workspace",
				// No DefaultRepo
			},
			host: &config.Host{
				Kind:     "cloud",
				BaseURL:  "https://api.bitbucket.org/2.0",
				Username: "testuser",
				Token:    "test-token",
			},
			args:          []string{},
			expectError:   true,
			errorContains: "--mine is required when not specifying a repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": tt.context,
				},
				Hosts: map[string]*config.Host{
					tt.context.Host: tt.host,
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

			cmd := newListCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func findOutputLine(t *testing.T, output, needle string) string {
	t.Helper()

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}

	t.Fatalf("expected output line containing %q, got:\n%s", needle, output)
	return ""
}

func TestListDashboardDC(t *testing.T) {
	formattedCreated := time.Date(2026, time.April, 10, 11, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60)).UnixMilli()
	prs := []bbdc.PullRequest{
		{
			ID:          1,
			Title:       "First PR",
			State:       "OPEN",
			CreatedDate: formattedCreated,
			FromRef: bbdc.Ref{
				DisplayID:  "feature-1",
				Repository: bbdc.Repository{Slug: "fork-repo1", Project: &bbdc.Project{Key: "~USER"}},
			},
			ToRef: bbdc.Ref{
				DisplayID:  "main",
				Repository: bbdc.Repository{Slug: "repo1", Project: &bbdc.Project{Key: "PROJ"}},
			},
		},
		{
			ID:    2,
			Title: "Second PR",
			State: "OPEN",
			FromRef: bbdc.Ref{
				DisplayID:  "feature-2",
				Repository: bbdc.Repository{Slug: "fork-repo2", Project: &bbdc.Project{Key: "~USER"}},
			},
			ToRef: bbdc.Ref{
				DisplayID:  "main",
				Repository: bbdc.Repository{Slug: "repo2", Project: &bbdc.Project{Key: "PROJ"}},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/dashboard/pull-requests") {
			resp := struct {
				Values     []bbdc.PullRequest `json:"values"`
				IsLastPage bool               `json:"isLastPage"`
			}{
				Values:     prs,
				IsLastPage: true,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:       "main",
				ProjectKey: "PROJ",
				// No DefaultRepo - this triggers dashboard mode
			},
		},
		Hosts: map[string]*config.Host{
			"main": {
				Kind:     "dc",
				BaseURL:  server.URL,
				Username: "testuser",
				Token:    "test-token",
			},
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

	cmd := newListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--mine"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "#1") {
		t.Errorf("expected output to contain PR #1, got:\n%s", output)
	}
	if !strings.Contains(output, "#2") {
		t.Errorf("expected output to contain PR #2, got:\n%s", output)
	}
	if !strings.Contains(output, "First PR") {
		t.Errorf("expected output to contain 'First PR', got:\n%s", output)
	}
	if !strings.Contains(output, "PROJ/repo1") {
		t.Errorf("expected output to contain repo info 'PROJ/repo1', got:\n%s", output)
	}

	firstLine := findOutputLine(t, output, "First PR")
	firstFields := strings.Split(firstLine, "\t")
	if len(firstFields) != 4 {
		t.Fatalf("expected 4 tab-separated fields for First PR, got %d: %q", len(firstFields), firstLine)
	}
	if got, want := firstFields[3], time.UnixMilli(formattedCreated).Local().Format(prListTimeLayout); got != want {
		t.Fatalf("expected formatted timestamp %q, got %q", want, got)
	}

	secondLine := findOutputLine(t, output, "Second PR")
	secondFields := strings.Split(secondLine, "\t")
	if len(secondFields) != 4 {
		t.Fatalf("expected 4 tab-separated fields for Second PR, got %d: %q", len(secondFields), secondLine)
	}
	if got := secondFields[3]; got != "" {
		t.Fatalf("expected empty timestamp for zero CreatedDate, got %q", got)
	}
	if strings.Contains(output, "1970-01-01 00:00") {
		t.Fatalf("expected zero CreatedDate to render empty string, got:\n%s", output)
	}
}

func TestListWorkspaceCloud(t *testing.T) {
	// Change to a temp directory without a git repo to prevent
	// applyRemoteDefaults from overwriting test context values.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	prs := []bbcloud.PullRequest{
		{
			ID:        1,
			Title:     "First PR",
			State:     "OPEN",
			CreatedOn: "2026-04-10T11:30:45.123456+02:00",
		},
		{
			ID:        2,
			Title:     "Second PR",
			State:     "OPEN",
			CreatedOn: "not-a-timestamp",
		},
	}
	// Set nested fields - use Destination.Repository.Slug as primary source
	prs[0].Source.Branch.Name = "feature-1"
	prs[0].Destination.Branch.Name = "main"
	prs[0].Destination.Repository.Slug = "repo1"
	prs[0].Links.HTML.Href = "https://bitbucket.org/workspace/repo1/pull-requests/1"
	prs[1].Source.Branch.Name = "feature-2"
	prs[1].Destination.Branch.Name = "main"
	prs[1].Destination.Repository.Slug = "repo2"
	prs[1].Links.HTML.Href = "https://bitbucket.org/workspace/repo2/pull-requests/2"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle /user endpoint to return current user
		if r.URL.Path == "/user" {
			user := bbcloud.User{
				UUID:     "{12345}",
				Username: "testuser",
				Display:  "Test User",
			}
			_ = json.NewEncoder(w).Encode(user)
			return
		}

		if strings.Contains(r.URL.Path, "/workspaces/") && strings.Contains(r.URL.Path, "/pullrequests/") {
			resp := struct {
				Values []bbcloud.PullRequest `json:"values"`
				Next   string                `json:"next"`
			}{
				Values: prs,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:      "cloud",
				Workspace: "workspace",
				// No DefaultRepo - this triggers workspace mode
			},
		},
		Hosts: map[string]*config.Host{
			"cloud": {
				Kind:     "cloud",
				BaseURL:  server.URL,
				Username: "testuser",
				Token:    "test-token",
			},
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

	cmd := newListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--mine"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "#1") {
		t.Errorf("expected output to contain PR #1, got:\n%s", output)
	}
	if !strings.Contains(output, "#2") {
		t.Errorf("expected output to contain PR #2, got:\n%s", output)
	}
	if !strings.Contains(output, "First PR") {
		t.Errorf("expected output to contain 'First PR', got:\n%s", output)
	}
	if !strings.Contains(output, "repo1") {
		t.Errorf("expected output to contain repo info 'repo1', got:\n%s", output)
	}

	firstLine := findOutputLine(t, output, "First PR")
	firstFields := strings.Split(firstLine, "\t")
	if len(firstFields) != 4 {
		t.Fatalf("expected 4 tab-separated fields for First PR, got %d: %q", len(firstFields), firstLine)
	}
	expectedFirst, err := time.Parse(time.RFC3339Nano, prs[0].CreatedOn)
	if err != nil {
		t.Fatalf("failed to parse test timestamp: %v", err)
	}
	if got, want := firstFields[3], expectedFirst.Local().Format(prListTimeLayout); got != want {
		t.Fatalf("expected formatted timestamp %q, got %q", want, got)
	}

	secondLine := findOutputLine(t, output, "Second PR")
	secondFields := strings.Split(secondLine, "\t")
	if len(secondFields) != 4 {
		t.Fatalf("expected 4 tab-separated fields for Second PR, got %d: %q", len(secondFields), secondLine)
	}
	if got := secondFields[3]; got != "" {
		t.Fatalf("expected empty timestamp for invalid CreatedOn, got %q", got)
	}
	if strings.Contains(output, "0001-01-01 00:00") {
		t.Fatalf("expected invalid CreatedOn to render empty string, got:\n%s", output)
	}
}

func TestListRepositoryDCIncludesCreationTimestamp(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	createdDate := time.Date(2026, time.April, 10, 9, 15, 0, 0, time.UTC).UnixMilli()
	prs := []bbdc.PullRequest{
		{
			ID:          7,
			Title:       "Repo PR",
			State:       "OPEN",
			CreatedDate: createdDate,
			FromRef: bbdc.Ref{
				DisplayID: "feature/repo",
			},
			ToRef: bbdc.Ref{
				DisplayID: "main",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/pull-requests") {
			resp := struct {
				Values     []bbdc.PullRequest `json:"values"`
				IsLastPage bool               `json:"isLastPage"`
			}{
				Values:     prs,
				IsLastPage: true,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:        "main",
				ProjectKey:  "PROJ",
				DefaultRepo: "repo1",
			},
		},
		Hosts: map[string]*config.Host{
			"main": {
				Kind:     "dc",
				BaseURL:  server.URL,
				Username: "testuser",
				Token:    "test-token",
			},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: stderr},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line := findOutputLine(t, stdout.String(), "Repo PR")
	fields := strings.Split(line, "\t")
	if len(fields) != 4 {
		t.Fatalf("expected 4 tab-separated fields, got %d: %q", len(fields), line)
	}
	if got, want := fields[3], time.UnixMilli(createdDate).Local().Format(prListTimeLayout); got != want {
		t.Fatalf("expected formatted timestamp %q, got %q", want, got)
	}
}

func TestListRepositoryCloudIncludesCreationTimestamp(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	prs := []bbcloud.PullRequest{
		{
			ID:        8,
			Title:     "Cloud Repo PR",
			State:     "OPEN",
			CreatedOn: "",
		},
	}
	prs[0].Source.Branch.Name = "feature/cloud"
	prs[0].Destination.Branch.Name = "main"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/repositories/") && strings.Contains(r.URL.Path, "/pullrequests") {
			resp := struct {
				Values []bbcloud.PullRequest `json:"values"`
				Next   string                `json:"next"`
			}{
				Values: prs,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:        "cloud",
				Workspace:   "workspace",
				DefaultRepo: "repo1",
			},
		},
		Hosts: map[string]*config.Host{
			"cloud": {
				Kind:     "cloud",
				BaseURL:  server.URL,
				Username: "testuser",
				Token:    "test-token",
			},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: stderr},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line := findOutputLine(t, stdout.String(), "Cloud Repo PR")
	fields := strings.Split(line, "\t")
	if len(fields) != 4 {
		t.Fatalf("expected 4 tab-separated fields, got %d: %q", len(fields), line)
	}
	if got := fields[3]; got != "" {
		t.Fatalf("expected empty timestamp for empty CreatedOn, got %q", got)
	}
}

// failingWriter returns an error on every Write. It exercises the error-return
// branches after the per-PR Fprintf calls in the list rendering paths.
type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestListWriteErrorPropagation(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	dcPRs := []bbdc.PullRequest{{
		ID:    1,
		Title: "DC PR",
		State: "OPEN",
		FromRef: bbdc.Ref{
			DisplayID:  "feature",
			Repository: bbdc.Repository{Slug: "repo1", Project: &bbdc.Project{Key: "PROJ"}},
		},
		ToRef: bbdc.Ref{
			DisplayID:  "main",
			Repository: bbdc.Repository{Slug: "repo1", Project: &bbdc.Project{Key: "PROJ"}},
		},
	}}
	dcHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := struct {
			Values     []bbdc.PullRequest `json:"values"`
			IsLastPage bool               `json:"isLastPage"`
		}{Values: dcPRs, IsLastPage: true}
		_ = json.NewEncoder(w).Encode(resp)
	})

	cloudPRs := []bbcloud.PullRequest{{
		ID:    1,
		Title: "Cloud PR",
		State: "OPEN",
	}}
	cloudPRs[0].Source.Branch.Name = "feature"
	cloudPRs[0].Destination.Branch.Name = "main"
	cloudPRs[0].Destination.Repository.Slug = "repo1"
	cloudHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/user" {
			_ = json.NewEncoder(w).Encode(bbcloud.User{UUID: "{1}", Username: "testuser", Display: "Test"})
			return
		}
		resp := struct {
			Values []bbcloud.PullRequest `json:"values"`
			Next   string                `json:"next"`
		}{Values: cloudPRs}
		_ = json.NewEncoder(w).Encode(resp)
	})

	cases := []struct {
		name        string
		handler     http.HandlerFunc
		kind        string
		workspace   string
		projectKey  string
		defaultRepo string
		args        []string
	}{
		{"repository DC", dcHandler, "dc", "", "PROJ", "repo1", nil},
		{"repository Cloud", cloudHandler, "cloud", "workspace", "", "repo1", nil},
		{"dashboard DC", dcHandler, "dc", "", "PROJ", "", []string{"--mine"}},
		{"workspace Cloud", cloudHandler, "cloud", "workspace", "", "", []string{"--mine"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "h",
						Workspace:   tc.workspace,
						ProjectKey:  tc.projectKey,
						DefaultRepo: tc.defaultRepo,
					},
				},
				Hosts: map[string]*config.Host{
					"h": {
						Kind:     tc.kind,
						BaseURL:  server.URL,
						Username: "testuser",
						Token:    "test-token",
					},
				},
			}

			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams: &iostreams.IOStreams{
					Out:    failingWriter{},
					ErrOut: &strings.Builder{},
				},
				Config: func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newListCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			if len(tc.args) > 0 {
				cmd.SetArgs(tc.args)
			}

			execErr := cmd.Execute()
			if execErr == nil {
				t.Fatal("expected error from failing writer, got nil")
			}
			if !strings.Contains(execErr.Error(), "write failed") {
				t.Fatalf("expected error to contain %q, got %v", "write failed", execErr)
			}
		})
	}
}

func TestStateIcon(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{
			name:     "successful uppercase",
			state:    "SUCCESSFUL",
			expected: "✓",
		},
		{
			name:     "success lowercase",
			state:    "success",
			expected: "✓",
		},
		{
			name:     "SUCCESS mixed case",
			state:    "Success",
			expected: "✓",
		},
		{
			name:     "failed uppercase",
			state:    "FAILED",
			expected: "✗",
		},
		{
			name:     "failure lowercase",
			state:    "failure",
			expected: "✗",
		},
		{
			name:     "FAILURE mixed case",
			state:    "Failure",
			expected: "✗",
		},
		{
			name:     "inprogress uppercase",
			state:    "INPROGRESS",
			expected: "○",
		},
		{
			name:     "in_progress with underscore",
			state:    "IN_PROGRESS",
			expected: "○",
		},
		{
			name:     "pending lowercase",
			state:    "pending",
			expected: "○",
		},
		{
			name:     "PENDING uppercase",
			state:    "PENDING",
			expected: "○",
		},
		{
			name:     "stopped uppercase",
			state:    "STOPPED",
			expected: "■",
		},
		{
			name:     "stopped lowercase",
			state:    "stopped",
			expected: "■",
		},
		{
			name:     "cancelled uppercase",
			state:    "CANCELLED",
			expected: "⊘",
		},
		{
			name:     "cancelled lowercase",
			state:    "cancelled",
			expected: "⊘",
		},
		{
			name:     "unknown state",
			state:    "UNKNOWN",
			expected: "?",
		},
		{
			name:     "empty state",
			state:    "",
			expected: "?",
		},
		{
			name:     "random state",
			state:    "CUSTOM_STATE",
			expected: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stateIcon(tt.state)
			if got != tt.expected {
				t.Errorf("stateIcon(%q) = %q, want %q", tt.state, got, tt.expected)
			}
		})
	}
}

func TestRunChecksDataCenter(t *testing.T) {
	tests := []struct {
		name           string
		prID           int
		prResponse     bbdc.PullRequest
		statusResponse []bbdc.CommitStatus
		expectError    bool
		errorContains  string
		outputContains []string
	}{
		{
			name: "successful with multiple statuses",
			prID: 123,
			prResponse: bbdc.PullRequest{
				ID:    123,
				Title: "Test PR",
				FromRef: bbdc.Ref{
					LatestCommit: "abc123def456",
				},
			},
			statusResponse: []bbdc.CommitStatus{
				{
					State: "SUCCESSFUL",
					Key:   "jenkins-build",
					Name:  "Jenkins Build",
					URL:   "https://jenkins.example.com/job/123",
				},
				{
					State: "INPROGRESS",
					Key:   "sonar-analysis",
					Name:  "SonarQube Analysis",
					URL:   "https://sonar.example.com/project",
				},
			},
			expectError: false,
			outputContains: []string{
				"Build Status for PR #123",
				"abc123def456",
				"✓ Jenkins Build: SUCCESSFUL",
				"○ SonarQube Analysis: INPROGRESS",
				"https://jenkins.example.com/job/123",
			},
		},
		{
			name: "no builds found",
			prID: 456,
			prResponse: bbdc.PullRequest{
				ID:    456,
				Title: "PR without builds",
				FromRef: bbdc.Ref{
					LatestCommit: "def456abc123",
				},
			},
			statusResponse: []bbdc.CommitStatus{},
			expectError:    false,
			outputContains: []string{
				"Build Status for PR #456",
				"No builds found",
			},
		},
		{
			name: "pr missing commit",
			prID: 789,
			prResponse: bbdc.PullRequest{
				ID:    789,
				Title: "PR without commit",
				FromRef: bbdc.Ref{
					LatestCommit: "",
				},
			},
			expectError:   true,
			errorContains: ErrNoSourceCommit.Error(),
		},
		{
			name: "status with fallback to key when name missing",
			prID: 234,
			prResponse: bbdc.PullRequest{
				ID:    234,
				Title: "Test PR",
				FromRef: bbdc.Ref{
					LatestCommit: "commit123",
				},
			},
			statusResponse: []bbdc.CommitStatus{
				{
					State: "FAILED",
					Key:   "test-key",
					Name:  "",
					URL:   "",
				},
			},
			expectError: false,
			outputContains: []string{
				"✗ test-key: FAILED",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var prCalled, statusCalled bool

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if strings.Contains(r.URL.Path, "/pull-requests/") {
					prCalled = true
					_ = json.NewEncoder(w).Encode(tt.prResponse)
					return
				}

				if strings.Contains(r.URL.Path, "/build-status/") {
					statusCalled = true
					resp := struct {
						Values []bbdc.CommitStatus `json:"values"`
					}{
						Values: tt.statusResponse,
					}
					_ = json.NewEncoder(w).Encode(resp)
					return
				}

				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "main",
						ProjectKey:  "PROJ",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"main": {
						Kind:     "dc",
						BaseURL:  server.URL,
						Username: "testuser",
						Token:    "test-token",
					},
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

			cmd := newChecksCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			ctx := context.Background()
			cmd.SetContext(ctx)

			opts := &checksOptions{
				ID: tt.prID,
			}

			err := runChecks(cmd, f, opts)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.prResponse.FromRef.LatestCommit != "" && !prCalled {
				t.Error("expected PR endpoint to be called")
			}

			if tt.prResponse.FromRef.LatestCommit != "" && !statusCalled {
				t.Error("expected status endpoint to be called")
			}

			output := stdout.String()
			for _, expected := range tt.outputContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestRunChecksCloud(t *testing.T) {
	tests := []struct {
		name           string
		prID           int
		prResponse     bbcloud.PullRequest
		statusResponse []bbcloud.CommitStatus
		expectError    bool
		errorContains  string
		outputContains []string
	}{
		{
			name: "successful with builds",
			prID: 123,
			prResponse: func() bbcloud.PullRequest {
				pr := bbcloud.PullRequest{
					ID:    123,
					Title: "Test PR",
				}
				pr.Source.Commit.Hash = "cloudcommit123"
				return pr
			}(),
			statusResponse: []bbcloud.CommitStatus{
				{
					State: "SUCCESSFUL",
					Key:   "bitbucket-pipelines",
					Name:  "Bitbucket Pipelines",
					URL:   "https://bitbucket.org/workspace/repo/addon/pipelines/home#!/results/1",
				},
			},
			expectError: false,
			outputContains: []string{
				"Build Status for PR #123",
				"cloudcommit1",
				"✓ Bitbucket Pipelines: SUCCESSFUL",
			},
		},
		{
			name: "pr without commit hash",
			prID: 456,
			prResponse: func() bbcloud.PullRequest {
				pr := bbcloud.PullRequest{
					ID:    456,
					Title: "PR without commit",
				}
				pr.Source.Commit.Hash = ""
				return pr
			}(),
			expectError:   true,
			errorContains: ErrNoSourceCommit.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var prCalled, statusCalled bool

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if strings.Contains(r.URL.Path, "/pullrequests/") {
					prCalled = true
					_ = json.NewEncoder(w).Encode(tt.prResponse)
					return
				}

				if strings.Contains(r.URL.Path, "/commit/") && strings.Contains(r.URL.Path, "/statuses") {
					statusCalled = true
					resp := struct {
						Values []bbcloud.CommitStatus `json:"values"`
						Next   string                 `json:"next"`
					}{
						Values: tt.statusResponse,
					}
					_ = json.NewEncoder(w).Encode(resp)
					return
				}

				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "cloud",
						Workspace:   "workspace",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"cloud": {
						Kind:     "cloud",
						BaseURL:  server.URL,
						Username: "testuser",
						Token:    "test-token",
					},
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

			cmd := newChecksCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			ctx := context.Background()
			cmd.SetContext(ctx)

			opts := &checksOptions{
				ID: tt.prID,
			}

			err := runChecks(cmd, f, opts)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.prResponse.Source.Commit.Hash != "" && !prCalled {
				t.Error("expected PR endpoint to be called")
			}

			if tt.prResponse.Source.Commit.Hash != "" && !statusCalled {
				t.Error("expected status endpoint to be called")
			}

			output := stdout.String()
			for _, expected := range tt.outputContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestChecksCommandArgumentParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid pr id",
			args:        []string{"123"},
			expectError: false,
		},
		{
			name:          "no arguments",
			args:          []string{},
			expectError:   true,
			errorContains: "accepts 1 arg(s), received 0",
		},
		{
			name:          "too many arguments",
			args:          []string{"123", "456"},
			expectError:   true,
			errorContains: "accepts 1 arg(s), received 2",
		},
		{
			name:          "invalid pr id",
			args:          []string{"not-a-number"},
			expectError:   true,
			errorContains: "invalid pull request id",
		},
		// Note: negative numbers like "-123" are parsed as flags by cobra,
		// so we don't test that case here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "main",
						ProjectKey:  "PROJ",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"main": {
						Kind:    "dc",
						BaseURL: "https://bitbucket.example.com",
						Token:   "test-token",
					},
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

			cmd := newChecksCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			// Note: Without a mock server, valid args will fail when trying to connect
			// We're only testing argument parsing here, not the full execution
		})
	}
}

func TestChecksCommandValidation(t *testing.T) {
	// Change to a temp directory without a git repo to prevent
	// applyRemoteDefaults from overwriting test context values.
	// In CI environments with a bitbucket.org remote, the git detection
	// would otherwise fill in workspace/repo and bypass validation.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	// Use a mock server for cloud tests to avoid hitting real API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 404 for any request - we're testing validation, not API calls
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	tests := []struct {
		name          string
		context       *config.Context
		host          *config.Host
		expectError   bool
		errorContains string
	}{
		{
			name: "data center missing project",
			context: &config.Context{
				Host:        "main",
				DefaultRepo: "repo",
			},
			host: &config.Host{
				Kind:     "dc",
				BaseURL:  "https://bitbucket.example.com",
				Username: "testuser",
				Token:    "test-token",
			},
			expectError:   true,
			errorContains: "context must supply project and repo",
		},
		{
			name: "data center missing repo",
			context: &config.Context{
				Host:       "main",
				ProjectKey: "PROJ",
			},
			host: &config.Host{
				Kind:     "dc",
				BaseURL:  "https://bitbucket.example.com",
				Username: "testuser",
				Token:    "test-token",
			},
			expectError:   true,
			errorContains: "context must supply project and repo",
		},
		{
			name: "cloud missing workspace",
			context: &config.Context{
				Host:        "cloud",
				DefaultRepo: "repo",
			},
			host: &config.Host{
				Kind:     "cloud",
				BaseURL:  mockServer.URL, // Use mock server instead of real API
				Username: "testuser",
				Token:    "test-token",
			},
			expectError:   true,
			errorContains: "context must supply workspace and repo",
		},
		{
			name: "cloud missing repo",
			context: &config.Context{
				Host:      "cloud",
				Workspace: "workspace",
			},
			host: &config.Host{
				Kind:     "cloud",
				BaseURL:  mockServer.URL, // Use mock server instead of real API
				Username: "testuser",
				Token:    "test-token",
			},
			expectError:   true,
			errorContains: "context must supply workspace and repo",
		},
		{
			name: "unsupported host kind",
			context: &config.Context{
				Host: "unknown",
			},
			host: &config.Host{
				Kind:     "unknown",
				BaseURL:  "https://example.com",
				Username: "testuser",
				Token:    "test-token",
			},
			expectError:   true,
			errorContains: "unsupported host kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": tt.context,
				},
				Hosts: map[string]*config.Host{
					tt.context.Host: tt.host,
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

			cmd := newChecksCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"123"})

			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestStateColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		state        string
		colorEnabled bool
		wantPrefix   string
		wantSuffix   string
	}{
		{
			name:         "successful with color",
			state:        "SUCCESSFUL",
			colorEnabled: true,
			wantPrefix:   colorGreen,
			wantSuffix:   colorReset,
		},
		{
			name:         "success lowercase with color",
			state:        "success",
			colorEnabled: true,
			wantPrefix:   colorGreen,
			wantSuffix:   colorReset,
		},
		{
			name:         "failed with color",
			state:        "FAILED",
			colorEnabled: true,
			wantPrefix:   colorRed,
			wantSuffix:   colorReset,
		},
		{
			name:         "failure with color",
			state:        "failure",
			colorEnabled: true,
			wantPrefix:   colorRed,
			wantSuffix:   colorReset,
		},
		{
			name:         "inprogress with color",
			state:        "INPROGRESS",
			colorEnabled: true,
			wantPrefix:   colorYellow,
			wantSuffix:   colorReset,
		},
		{
			name:         "pending with color",
			state:        "pending",
			colorEnabled: true,
			wantPrefix:   colorYellow,
			wantSuffix:   colorReset,
		},
		{
			name:         "cancelled with color",
			state:        "CANCELLED",
			colorEnabled: true,
			wantPrefix:   colorYellow,
			wantSuffix:   colorReset,
		},
		{
			name:         "stopped with color",
			state:        "STOPPED",
			colorEnabled: true,
			wantPrefix:   colorYellow,
			wantSuffix:   colorReset,
		},
		{
			name:         "unknown state with color",
			state:        "UNKNOWN",
			colorEnabled: true,
			wantPrefix:   "",
			wantSuffix:   "",
		},
		{
			name:         "successful without color",
			state:        "SUCCESSFUL",
			colorEnabled: false,
			wantPrefix:   "",
			wantSuffix:   "",
		},
		{
			name:         "failed without color",
			state:        "FAILED",
			colorEnabled: false,
			wantPrefix:   "",
			wantSuffix:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, suffix := stateColor(tt.state, tt.colorEnabled)
			if prefix != tt.wantPrefix {
				t.Errorf("stateColor(%q, %v) prefix = %q, want %q", tt.state, tt.colorEnabled, prefix, tt.wantPrefix)
			}
			if suffix != tt.wantSuffix {
				t.Errorf("stateColor(%q, %v) suffix = %q, want %q", tt.state, tt.colorEnabled, suffix, tt.wantSuffix)
			}
		})
	}
}

func TestIsTerminalState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		state    string
		expected bool
	}{
		{"SUCCESSFUL", true},
		{"success", true},
		{"FAILED", true},
		{"failure", true},
		{"STOPPED", true},
		{"stopped", true},
		{"CANCELLED", true},
		{"cancelled", true},
		{"INPROGRESS", false},
		{"in_progress", false},
		{"PENDING", false},
		{"pending", false},
		{"UNKNOWN", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := isTerminalState(tt.state)
			if got != tt.expected {
				t.Errorf("isTerminalState(%q) = %v, want %v", tt.state, got, tt.expected)
			}
		})
	}
}

func TestAllBuildsComplete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		statuses []types.CommitStatus
		expected bool
	}{
		{
			name:     "empty statuses",
			statuses: []types.CommitStatus{},
			expected: false,
		},
		{
			name: "all successful",
			statuses: []types.CommitStatus{
				{State: "SUCCESSFUL"},
				{State: "SUCCESS"},
			},
			expected: true,
		},
		{
			name: "all failed",
			statuses: []types.CommitStatus{
				{State: "FAILED"},
				{State: "FAILURE"},
			},
			expected: true,
		},
		{
			name: "mixed terminal states",
			statuses: []types.CommitStatus{
				{State: "SUCCESSFUL"},
				{State: "FAILED"},
				{State: "STOPPED"},
			},
			expected: true,
		},
		{
			name: "one in progress",
			statuses: []types.CommitStatus{
				{State: "SUCCESSFUL"},
				{State: "INPROGRESS"},
			},
			expected: false,
		},
		{
			name: "all in progress",
			statuses: []types.CommitStatus{
				{State: "INPROGRESS"},
				{State: "PENDING"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allBuildsComplete(tt.statuses)
			if got != tt.expected {
				t.Errorf("allBuildsComplete() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAnyBuildFailed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		statuses []types.CommitStatus
		expected bool
	}{
		{
			name:     "empty statuses",
			statuses: []types.CommitStatus{},
			expected: false,
		},
		{
			name: "all successful",
			statuses: []types.CommitStatus{
				{State: "SUCCESSFUL"},
				{State: "SUCCESS"},
			},
			expected: false,
		},
		{
			name: "one failed",
			statuses: []types.CommitStatus{
				{State: "SUCCESSFUL"},
				{State: "FAILED"},
			},
			expected: true,
		},
		{
			name: "one failure",
			statuses: []types.CommitStatus{
				{State: "SUCCESS"},
				{State: "FAILURE"},
			},
			expected: true,
		},
		{
			name: "in progress only",
			statuses: []types.CommitStatus{
				{State: "INPROGRESS"},
				{State: "PENDING"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := anyBuildFailed(tt.statuses)
			if got != tt.expected {
				t.Errorf("anyBuildFailed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculatePollInterval(t *testing.T) {
	t.Parallel()
	baseInterval := 10 * time.Second
	maxInterval := 2 * time.Minute

	tests := []struct {
		name        string
		iteration   int
		expectedMin time.Duration // With jitter, result should be >= this (minus jitter)
		expectedMax time.Duration // With jitter, result should be <= this (plus jitter)
	}{
		{
			name:        "iteration 0 returns base interval",
			iteration:   0,
			expectedMin: 8 * time.Second,  // 10s - 15% jitter - some margin
			expectedMax: 12 * time.Second, // 10s + 15% jitter + some margin
		},
		{
			name:        "iteration 1 applies 1.5x backoff",
			iteration:   1,
			expectedMin: 12 * time.Second, // 15s - 15% jitter - margin
			expectedMax: 18 * time.Second, // 15s + 15% jitter + margin
		},
		{
			name:        "iteration 2 applies 1.5^2 backoff",
			iteration:   2,
			expectedMin: 18 * time.Second, // 22.5s - 15% jitter - margin
			expectedMax: 27 * time.Second, // 22.5s + 15% jitter + margin
		},
		{
			name:        "iteration 5 approaches max interval",
			iteration:   5,
			expectedMin: 60 * time.Second,  // Should be close to max
			expectedMax: 140 * time.Second, // 120s + jitter + margin
		},
		{
			name:        "iteration 10 caps at max interval",
			iteration:   10,
			expectedMin: 100 * time.Second, // 120s - 15% jitter - margin
			expectedMax: 140 * time.Second, // 120s + 15% jitter + margin
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to account for jitter randomness
			for i := 0; i < 10; i++ {
				got := calculatePollInterval(baseInterval, maxInterval, tt.iteration)
				if got < tt.expectedMin {
					t.Errorf("calculatePollInterval() = %v, want >= %v", got, tt.expectedMin)
				}
				if got > tt.expectedMax {
					t.Errorf("calculatePollInterval() = %v, want <= %v", got, tt.expectedMax)
				}
			}
		})
	}
}

func TestCalculatePollIntervalCapsAtMax(t *testing.T) {
	t.Parallel()
	baseInterval := 10 * time.Second
	maxInterval := 30 * time.Second

	// After enough iterations, should cap at max (with jitter)
	for iteration := 10; iteration <= 20; iteration++ {
		got := calculatePollInterval(baseInterval, maxInterval, iteration)
		// With 15% jitter, max should be ~34.5s
		if got > 35*time.Second {
			t.Errorf("iteration %d: calculatePollInterval() = %v, should cap near %v", iteration, got, maxInterval)
		}
	}
}

func TestAddJitter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{"10 seconds", 10 * time.Second},
		{"1 minute", 1 * time.Minute},
		{"2 minutes", 2 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to verify jitter is applied
			results := make(map[time.Duration]bool)
			for i := 0; i < 100; i++ {
				got := addJitter(tt.duration)
				results[got] = true

				// Verify within expected bounds (±15% + 1s margin)
				minExpected := time.Duration(float64(tt.duration) * 0.84) // 1 - 0.15 - small margin
				maxExpected := time.Duration(float64(tt.duration) * 1.16) // 1 + 0.15 + small margin

				if got < minExpected {
					t.Errorf("addJitter(%v) = %v, want >= %v", tt.duration, got, minExpected)
				}
				if got > maxExpected {
					t.Errorf("addJitter(%v) = %v, want <= %v", tt.duration, got, maxExpected)
				}
			}

			// Verify we got some variation (jitter is working)
			if len(results) < 5 {
				t.Errorf("addJitter() produced only %d unique values in 100 runs, expected more variation", len(results))
			}
		})
	}
}

func TestAddJitterMinimum(t *testing.T) {
	t.Parallel()
	// Very small durations should not go below 1 second
	got := addJitter(500 * time.Millisecond)
	if got < time.Second {
		t.Errorf("addJitter(500ms) = %v, want >= 1s minimum", got)
	}
}

func TestAddJitterZeroAndNegative(t *testing.T) {
	t.Parallel()
	// Zero duration should return zero
	if got := addJitter(0); got != 0 {
		t.Errorf("addJitter(0) = %v, want 0", got)
	}

	// Negative duration should return unchanged
	neg := -5 * time.Second
	if got := addJitter(neg); got != neg {
		t.Errorf("addJitter(%v) = %v, want %v", neg, got, neg)
	}
}

func TestBackoffProgression(t *testing.T) {
	t.Parallel()
	// Verify the backoff progression is monotonically increasing (before hitting cap)
	baseInterval := 10 * time.Second
	maxInterval := 5 * time.Minute

	// Calculate expected values without jitter
	expectedBase := []float64{10, 15, 22.5, 33.75, 50.625, 75.9375, 113.90625, 170.859375}

	for i := 0; i < len(expectedBase)-1; i++ {
		// Run multiple times and take average to smooth out jitter
		var sum1, sum2 time.Duration
		runs := 20
		for j := 0; j < runs; j++ {
			sum1 += calculatePollInterval(baseInterval, maxInterval, i)
			sum2 += calculatePollInterval(baseInterval, maxInterval, i+1)
		}
		avg1 := sum1 / time.Duration(runs)
		avg2 := sum2 / time.Duration(runs)

		// Each iteration should be roughly 1.5x the previous (with tolerance for jitter)
		ratio := float64(avg2) / float64(avg1)
		if ratio < 1.3 || ratio > 1.7 {
			t.Errorf("backoff ratio between iteration %d and %d: got %.2f, want ~1.5", i, i+1, ratio)
		}
	}
}

// mockFetcher creates a fetch function that returns different statuses per call
type mockFetcher struct {
	calls     int
	responses []struct {
		statuses []types.CommitStatus
		err      error
	}
}

func (m *mockFetcher) fetch() ([]types.CommitStatus, error) {
	if m.calls >= len(m.responses) {
		// Return last response if we've exceeded the configured calls
		return m.responses[len(m.responses)-1].statuses, m.responses[len(m.responses)-1].err
	}
	resp := m.responses[m.calls]
	m.calls++
	return resp.statuses, resp.err
}

func TestPollUntilComplete_ImmediateSuccess(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		Interval:    10 * time.Millisecond,
		MaxInterval: 100 * time.Millisecond,
	}

	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{
				statuses: []types.CommitStatus{
					{State: "SUCCESSFUL", Name: "build-1"},
					{State: "SUCCESS", Name: "build-2"},
				},
			},
		},
	}

	ctx := context.Background()
	statuses, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}
	if fetcher.calls != 1 {
		t.Errorf("expected 1 fetch call, got %d", fetcher.calls)
	}
}

func TestPollUntilComplete_MultipleIterations(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		Interval:    1 * time.Millisecond, // Very short for testing
		MaxInterval: 5 * time.Millisecond,
	}

	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{statuses: []types.CommitStatus{{State: "INPROGRESS", Name: "build-1"}}},
			{statuses: []types.CommitStatus{{State: "INPROGRESS", Name: "build-1"}}},
			{statuses: []types.CommitStatus{{State: "SUCCESSFUL", Name: "build-1"}}},
		},
	}

	ctx := context.Background()
	statuses, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(statuses) != 1 {
		t.Errorf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != "SUCCESSFUL" {
		t.Errorf("expected SUCCESSFUL state, got %s", statuses[0].State)
	}
	if fetcher.calls != 3 {
		t.Errorf("expected 3 fetch calls, got %d", fetcher.calls)
	}
}

func TestPollUntilComplete_ContextCancellation(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		Interval:    50 * time.Millisecond,
		MaxInterval: 100 * time.Millisecond,
	}

	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{statuses: []types.CommitStatus{{State: "INPROGRESS", Name: "build-1"}}},
			{statuses: []types.CommitStatus{{State: "INPROGRESS", Name: "build-1"}}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err == nil {
		t.Fatal("expected context.Canceled error")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got %v", err)
	}
}

func TestPollUntilComplete_Timeout(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		Interval:    50 * time.Millisecond,
		MaxInterval: 100 * time.Millisecond,
	}

	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{statuses: []types.CommitStatus{{State: "INPROGRESS", Name: "build-1"}}},
			{statuses: []types.CommitStatus{{State: "INPROGRESS", Name: "build-1"}}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err == nil {
		t.Fatal("expected context.DeadlineExceeded error")
	}
	if !strings.Contains(err.Error(), "deadline exceeded") {
		t.Errorf("expected deadline exceeded error, got %v", err)
	}
}

func TestPollUntilComplete_FetchErrorRetry(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		Interval:    1 * time.Millisecond,
		MaxInterval: 5 * time.Millisecond,
	}

	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{err: fmt.Errorf("temporary network error")},
			{statuses: []types.CommitStatus{{State: "SUCCESSFUL", Name: "build-1"}}},
		},
	}

	ctx := context.Background()
	statuses, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err != nil {
		t.Fatalf("expected no error after retry, got %v", err)
	}
	if len(statuses) != 1 {
		t.Errorf("expected 1 status, got %d", len(statuses))
	}
	if fetcher.calls != 2 {
		t.Errorf("expected 2 fetch calls (1 error + 1 success), got %d", fetcher.calls)
	}
}

func TestPollUntilComplete_MaxConsecutiveErrors(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		Interval:    1 * time.Millisecond,
		MaxInterval: 5 * time.Millisecond,
	}

	testErr := fmt.Errorf("persistent error")
	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{err: testErr},
			{err: testErr},
			{err: testErr},
		},
	}

	ctx := context.Background()
	_, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err == nil {
		t.Fatal("expected error after max consecutive errors")
	}
	if !strings.Contains(err.Error(), "fetch failed after 3 attempts") {
		t.Errorf("expected 'fetch failed after 3 attempts' error, got %v", err)
	}
	if fetcher.calls != 3 {
		t.Errorf("expected 3 fetch calls, got %d", fetcher.calls)
	}
}

func TestPollUntilComplete_ErrorResetOnSuccess(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		Interval:    1 * time.Millisecond,
		MaxInterval: 5 * time.Millisecond,
	}

	testErr := fmt.Errorf("temporary error")
	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{err: testErr}, // Error 1
			{err: testErr}, // Error 2
			{statuses: []types.CommitStatus{{State: "INPROGRESS", Name: "b"}}}, // Success resets counter
			{err: testErr}, // Error 1 again
			{err: testErr}, // Error 2 again
			{statuses: []types.CommitStatus{{State: "SUCCESSFUL", Name: "b"}}}, // Final success
		},
	}

	ctx := context.Background()
	statuses, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err != nil {
		t.Fatalf("expected no error (error counter should reset), got %v", err)
	}
	if len(statuses) != 1 || statuses[0].State != "SUCCESSFUL" {
		t.Errorf("expected final successful status, got %v", statuses)
	}
	if fetcher.calls != 6 {
		t.Errorf("expected 6 fetch calls, got %d", fetcher.calls)
	}
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("ErrNoSourceCommit", func(t *testing.T) {
		t.Parallel()
		// Verify the sentinel error can be checked with errors.Is
		err := fmt.Errorf("context: %w", ErrNoSourceCommit)
		if !errors.Is(err, ErrNoSourceCommit) {
			t.Error("errors.Is should match wrapped ErrNoSourceCommit")
		}
	})

}

func TestFlagValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expectError   bool
		errorContains string
	}{
		{
			name:        "interval with wait is valid",
			args:        []string{"123", "--wait", "--interval", "5s"},
			expectError: false,
		},
		{
			name:          "interval without wait errors",
			args:          []string{"123", "--interval", "5s"},
			expectError:   true,
			errorContains: "--interval requires --wait",
		},
		{
			name:          "max-interval without wait errors",
			args:          []string{"123", "--max-interval", "1m"},
			expectError:   true,
			errorContains: "--max-interval requires --wait",
		},
		{
			name:          "timeout without wait errors",
			args:          []string{"123", "--timeout", "10m"},
			expectError:   true,
			errorContains: "--timeout requires --wait",
		},
		{
			name:          "fail-fast without wait errors",
			args:          []string{"123", "--fail-fast"},
			expectError:   true,
			errorContains: "--fail-fast requires --wait",
		},
		{
			name:        "fail-fast with wait is valid",
			args:        []string{"123", "--wait", "--fail-fast"},
			expectError: false,
		},
		{
			name:          "zero interval errors",
			args:          []string{"123", "--wait", "--interval", "0s"},
			expectError:   true,
			errorContains: "--interval must be positive",
		},
		{
			name:          "zero max-interval errors",
			args:          []string{"123", "--wait", "--max-interval", "0s"},
			expectError:   true,
			errorContains: "--max-interval must be positive",
		},
		{
			name:          "max-interval less than interval errors",
			args:          []string{"123", "--wait", "--interval", "30s", "--max-interval", "10s"},
			expectError:   true,
			errorContains: "--max-interval must be >= --interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "main",
						ProjectKey:  "PROJ",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"main": {
						Kind:    "dc",
						BaseURL: "https://bitbucket.example.com",
						Token:   "test-token",
					},
				},
			}

			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newChecksCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			}
			// Note: valid flag combinations will fail later when connecting to server
			// We're only testing flag validation here
		})
	}
}

func TestPollUntilComplete_EmptyBuildsExitsEarly(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		Interval:    10 * time.Millisecond,
		MaxInterval: 50 * time.Millisecond,
	}

	// Return empty statuses on first call
	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{statuses: []types.CommitStatus{}}, // Empty on first call
		},
	}

	ctx := context.Background()
	statuses, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("expected 0 statuses, got %d", len(statuses))
	}
	// Should exit after first call, not poll forever
	if fetcher.calls != 1 {
		t.Errorf("expected 1 fetch call (early exit), got %d", fetcher.calls)
	}
}

func TestPollUntilComplete_FailFast(t *testing.T) {
	ios := &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	opts := &checksOptions{
		ID:          123,
		Wait:        true,
		FailFast:    true,
		Interval:    1 * time.Millisecond,
		MaxInterval: 5 * time.Millisecond,
	}

	fetcher := &mockFetcher{
		responses: []struct {
			statuses []types.CommitStatus
			err      error
		}{
			{
				statuses: []types.CommitStatus{
					{State: "INPROGRESS", Name: "build-1"},
					{State: "FAILED", Name: "build-2"}, // One failed
				},
			},
		},
	}

	ctx := context.Background()
	statuses, err := pollUntilComplete(ctx, ios, opts, fetcher.fetch, false, "abc123", false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should exit immediately due to fail-fast, even though build-1 is still in progress
	if fetcher.calls != 1 {
		t.Errorf("expected 1 fetch call (fail-fast exit), got %d", fetcher.calls)
	}
	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses returned, got %d", len(statuses))
	}
}

func TestErrPendingExitCode(t *testing.T) {
	t.Parallel()
	// Verify ErrPending is distinct from ErrSilent
	if errors.Is(cmdutil.ErrPending, cmdutil.ErrSilent) {
		t.Error("ErrPending should not be equal to ErrSilent")
	}
	// Both should be sentinel errors
	if cmdutil.ErrPending == nil {
		t.Error("ErrPending should not be nil")
	}
	if cmdutil.ErrSilent == nil {
		t.Error("ErrSilent should not be nil")
	}
}

func TestEditCommandArgumentParsing(t *testing.T) {
	// Error cases: these don't need a server since they fail during arg/flag parsing
	errorTests := []struct {
		name          string
		args          []string
		errorContains string
	}{
		{
			name:          "no arguments",
			args:          []string{},
			errorContains: "accepts 1 arg(s), received 0",
		},
		{
			name:          "invalid pr id",
			args:          []string{"not-a-number", "--title", "New title"},
			errorContains: "invalid pull request id",
		},
		{
			name:          "no flags",
			args:          []string{"123"},
			errorContains: "at least one of --title, --body, --description, --reviewer, --remove-reviewer, or --with-default-reviewers is required",
		},
		{
			name:          "both body and description",
			args:          []string{"123", "--body", "body", "--description", "desc"},
			errorContains: "specify only one of --body or --description",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "main",
						ProjectKey:  "PROJ",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"main": {
						Kind:    "dc",
						BaseURL: "https://bitbucket.example.com",
						Token:   "test-token",
					},
				},
			}

			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}

	// Valid cases: use httptest server to avoid network calls and verify full execution
	validTests := []struct {
		name string
		args []string
	}{
		{name: "valid with title", args: []string{"123", "--title", "New title"}},
		{name: "valid with body", args: []string{"123", "--body", "New body"}},
		{name: "valid with description", args: []string{"123", "--description", "New desc"}},
		{name: "valid with title and body", args: []string{"123", "--title", "New title", "--body", "New body"}},
	}

	for _, tt := range validTests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// Return a valid PR for both GET and PUT with required refs
				pr := bbdc.PullRequest{
					ID: 123, Title: "Title", Description: "Desc", Version: 1,
					FromRef: bbdc.Ref{ID: "refs/heads/feature", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
					ToRef:   bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				}
				_ = json.NewEncoder(w).Encode(pr)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "main",
						ProjectKey:  "PROJ",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"main": {
						Kind:    "dc",
						BaseURL: server.URL,
						Token:   "test-token",
					},
				},
			}

			stdout := &strings.Builder{}
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("expected no error for valid args, got %v", err)
			}
			if !strings.Contains(stdout.String(), "Updated pull request #123") {
				t.Errorf("expected success output, got %q", stdout.String())
			}
		})
	}
}

func TestRunEditDataCenter(t *testing.T) {
	tests := []struct {
		name           string
		prID           int
		title          string
		body           string
		prResponse     bbdc.PullRequest
		expectPUT      bool
		putBodyCheck   func(t *testing.T, body map[string]any)
		outputContains []string
	}{
		{
			name:  "update title only",
			prID:  123,
			title: "New Title",
			prResponse: bbdc.PullRequest{
				ID:          123,
				Title:       "Old Title",
				Description: "Old Description",
				Version:     5,
				FromRef:     bbdc.Ref{ID: "refs/heads/feature", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				ToRef:       bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
			},
			expectPUT: true,
			putBodyCheck: func(t *testing.T, body map[string]any) {
				if body["title"] != "New Title" {
					t.Errorf("expected title 'New Title', got %v", body["title"])
				}
				if body["description"] != "Old Description" {
					t.Errorf("expected description 'Old Description' (unchanged), got %v", body["description"])
				}
				if int(body["version"].(float64)) != 5 {
					t.Errorf("expected version 5, got %v", body["version"])
				}
			},
			outputContains: []string{"Updated pull request #123"},
		},
		{
			name: "update body only",
			prID: 456,
			body: "New Body",
			prResponse: bbdc.PullRequest{
				ID:          456,
				Title:       "Existing Title",
				Description: "Old Body",
				Version:     3,
				FromRef:     bbdc.Ref{ID: "refs/heads/feature", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				ToRef:       bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
			},
			expectPUT: true,
			putBodyCheck: func(t *testing.T, body map[string]any) {
				if body["title"] != "Existing Title" {
					t.Errorf("expected title 'Existing Title' (unchanged), got %v", body["title"])
				}
				if body["description"] != "New Body" {
					t.Errorf("expected description 'New Body', got %v", body["description"])
				}
				if int(body["version"].(float64)) != 3 {
					t.Errorf("expected version 3, got %v", body["version"])
				}
			},
			outputContains: []string{"Updated pull request #456"},
		},
		{
			name:  "update both title and body",
			prID:  789,
			title: "New Title",
			body:  "New Body",
			prResponse: bbdc.PullRequest{
				ID:          789,
				Title:       "Old Title",
				Description: "Old Body",
				Version:     1,
				FromRef:     bbdc.Ref{ID: "refs/heads/feature", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				ToRef:       bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
			},
			expectPUT: true,
			putBodyCheck: func(t *testing.T, body map[string]any) {
				if body["title"] != "New Title" {
					t.Errorf("expected title 'New Title', got %v", body["title"])
				}
				if body["description"] != "New Body" {
					t.Errorf("expected description 'New Body', got %v", body["description"])
				}
			},
			outputContains: []string{"Updated pull request #789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var getCalled, putCalled bool
			var putBody map[string]any

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if r.Method == "GET" && strings.Contains(r.URL.Path, "/pull-requests/") {
					getCalled = true
					_ = json.NewEncoder(w).Encode(tt.prResponse)
					return
				}

				if r.Method == "PUT" && strings.Contains(r.URL.Path, "/pull-requests/") {
					putCalled = true
					_ = json.NewDecoder(r.Body).Decode(&putBody)
					// Return updated PR
					updatedPR := tt.prResponse
					if title, ok := putBody["title"].(string); ok {
						updatedPR.Title = title
					}
					if desc, ok := putBody["description"].(string); ok {
						updatedPR.Description = desc
					}
					updatedPR.Version++
					_ = json.NewEncoder(w).Encode(updatedPR)
					return
				}

				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "main",
						ProjectKey:  "PROJ",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"main": {
						Kind:     "dc",
						BaseURL:  server.URL,
						Username: "testuser",
						Token:    "test-token",
					},
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

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			args := []string{fmt.Sprintf("%d", tt.prID)}
			if tt.title != "" {
				args = append(args, "--title", tt.title)
			}
			if tt.body != "" {
				args = append(args, "--body", tt.body)
			}
			cmd.SetArgs(args)

			ctx := context.Background()
			cmd.SetContext(ctx)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !getCalled {
				t.Error("expected GET endpoint to be called")
			}

			if tt.expectPUT && !putCalled {
				t.Error("expected PUT endpoint to be called")
			}

			if tt.putBodyCheck != nil && putBody != nil {
				tt.putBodyCheck(t, putBody)
			}

			output := stdout.String()
			for _, expected := range tt.outputContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestRunEditCloud(t *testing.T) {
	tests := []struct {
		name           string
		prID           int
		title          string
		body           string
		prResponse     bbcloud.PullRequest
		putBodyCheck   func(t *testing.T, body map[string]any)
		outputContains []string
	}{
		{
			name:  "update title only",
			prID:  123,
			title: "New Title",
			prResponse: bbcloud.PullRequest{
				ID:    123,
				Title: "Old Title",
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				if body["title"] != "New Title" {
					t.Errorf("expected title 'New Title', got %v", body["title"])
				}
				// description should NOT be present (only changed fields)
				if _, ok := body["description"]; ok {
					t.Errorf("description should not be in PUT body when only title changed")
				}
			},
			outputContains: []string{"Updated pull request #123"},
		},
		{
			name: "update description only",
			prID: 456,
			body: "New Description",
			prResponse: bbcloud.PullRequest{
				ID:    456,
				Title: "Existing Title",
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				// title should NOT be present
				if _, ok := body["title"]; ok {
					t.Errorf("title should not be in PUT body when only description changed")
				}
				if body["description"] != "New Description" {
					t.Errorf("expected description 'New Description', got %v", body["description"])
				}
			},
			outputContains: []string{"Updated pull request #456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var putCalled bool
			var putBody map[string]any

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if r.Method == "PUT" && strings.Contains(r.URL.Path, "/pullrequests/") {
					putCalled = true
					_ = json.NewDecoder(r.Body).Decode(&putBody)
					// Return updated PR
					updatedPR := tt.prResponse
					if title, ok := putBody["title"].(string); ok {
						updatedPR.Title = title
					}
					_ = json.NewEncoder(w).Encode(updatedPR)
					return
				}

				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "cloud",
						Workspace:   "workspace",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"cloud": {
						Kind:     "cloud",
						BaseURL:  server.URL,
						Username: "testuser",
						Token:    "test-token",
					},
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

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			args := []string{fmt.Sprintf("%d", tt.prID)}
			if tt.title != "" {
				args = append(args, "--title", tt.title)
			}
			if tt.body != "" {
				args = append(args, "--body", tt.body)
			}
			cmd.SetArgs(args)

			ctx := context.Background()
			cmd.SetContext(ctx)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !putCalled {
				t.Error("expected PUT endpoint to be called")
			}

			if tt.putBodyCheck != nil && putBody != nil {
				tt.putBodyCheck(t, putBody)
			}

			output := stdout.String()
			for _, expected := range tt.outputContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestReviewerOverlap(t *testing.T) {
	tests := []struct {
		name   string
		add    []string
		remove []string
		want   string
	}{
		{name: "no overlap", add: []string{"alice"}, remove: []string{"bob"}, want: ""},
		{name: "overlap", add: []string{"alice", "bob"}, remove: []string{"bob"}, want: "bob"},
		{name: "both empty", add: nil, remove: nil, want: ""},
		{name: "add only", add: []string{"alice"}, remove: nil, want: ""},
		{name: "remove only", add: nil, remove: []string{"alice"}, want: ""},
		{name: "bare uuid vs braced uuid", add: []string{"550e8400-e29b-41d4-a716-446655440000"}, remove: []string{"{550e8400-e29b-41d4-a716-446655440000}"}, want: "{550e8400-e29b-41d4-a716-446655440000}"},
		{name: "braced uuid vs braced uuid", add: []string{"{550e8400-e29b-41d4-a716-446655440000}"}, remove: []string{"{550e8400-e29b-41d4-a716-446655440000}"}, want: "{550e8400-e29b-41d4-a716-446655440000}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reviewerOverlap(tt.add, tt.remove)
			if got != tt.want {
				t.Errorf("reviewerOverlap(%v, %v) = %q, want %q", tt.add, tt.remove, got, tt.want)
			}
		})
	}
}

func TestEditDCReviewers(t *testing.T) {
	tests := []struct {
		name        string
		current     []bbdc.PullRequestReviewer
		add         []string
		remove      []string
		wantNames   []string
		wantWarning string
	}{
		{
			name: "add reviewer",
			current: []bbdc.PullRequestReviewer{
				{User: bbdc.User{Name: "alice"}},
			},
			add:       []string{"bob"},
			wantNames: []string{"alice", "bob"},
		},
		{
			name: "remove reviewer",
			current: []bbdc.PullRequestReviewer{
				{User: bbdc.User{Name: "alice"}},
				{User: bbdc.User{Name: "bob"}},
			},
			remove:    []string{"bob"},
			wantNames: []string{"alice"},
		},
		{
			name: "add and remove",
			current: []bbdc.PullRequestReviewer{
				{User: bbdc.User{Name: "alice"}},
				{User: bbdc.User{Name: "bob"}},
			},
			add:       []string{"charlie"},
			remove:    []string{"bob"},
			wantNames: []string{"alice", "charlie"},
		},
		{
			name: "add already present warns",
			current: []bbdc.PullRequestReviewer{
				{User: bbdc.User{Name: "alice"}},
			},
			add:         []string{"alice"},
			wantNames:   []string{"alice"},
			wantWarning: `warning: reviewer "alice" is already on this pull request`,
		},
		{
			name:        "remove not present warns",
			current:     []bbdc.PullRequestReviewer{},
			remove:      []string{"alice"},
			wantNames:   nil,
			wantWarning: `warning: reviewer "alice" is not on this pull request`,
		},
		{
			name:      "remove all reviewers",
			current:   []bbdc.PullRequestReviewer{{User: bbdc.User{Name: "alice"}}},
			remove:    []string{"alice"},
			wantNames: nil,
		},
		{
			name:        "duplicate add deduplicates",
			current:     []bbdc.PullRequestReviewer{},
			add:         []string{"bob", "bob"},
			wantNames:   []string{"bob"},
			wantWarning: `warning: reviewer "bob" is already on this pull request`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errBuf strings.Builder
			got := editDCReviewers(&errBuf, tt.current, tt.add, tt.remove, nil)

			var gotNames []string
			for _, r := range got {
				gotNames = append(gotNames, r.User.Name)
			}
			if len(gotNames) != len(tt.wantNames) {
				t.Fatalf("got reviewers %v, want %v", gotNames, tt.wantNames)
			}
			for i, name := range gotNames {
				if name != tt.wantNames[i] {
					t.Errorf("reviewer[%d] = %q, want %q", i, name, tt.wantNames[i])
				}
			}

			if tt.wantWarning != "" && !strings.Contains(errBuf.String(), tt.wantWarning) {
				t.Errorf("expected warning containing %q, got %q", tt.wantWarning, errBuf.String())
			}
			if tt.wantWarning == "" && errBuf.String() != "" {
				t.Errorf("expected no warnings, got %q", errBuf.String())
			}
		})
	}
}

func TestEditCloudReviewers(t *testing.T) {
	const (
		aliceUUID = "{550e8400-e29b-41d4-a716-446655440000}"
		bobUUID   = "{660e8400-e29b-41d4-a716-446655440000}"
	)
	tests := []struct {
		name        string
		current     []bbcloud.User
		add         []string
		remove      []string
		want        []string
		wantWarning string
		wantErr     string
	}{
		{
			name:    "add reviewer by username",
			current: []bbcloud.User{{UUID: aliceUUID, Username: "alice"}},
			add:     []string{"bob"},
			want:    []string{aliceUUID, "bob"},
		},
		{
			name: "remove reviewer by username",
			current: []bbcloud.User{
				{UUID: aliceUUID, Username: "alice"},
				{UUID: bobUUID, Username: "bob"},
			},
			remove: []string{"bob"},
			want:   []string{aliceUUID},
		},
		{
			name:    "remove last reviewer returns empty slice",
			current: []bbcloud.User{{UUID: aliceUUID, Username: "alice"}},
			remove:  []string{aliceUUID},
			want:    []string{},
		},
		{
			name:        "add already present warns",
			current:     []bbcloud.User{{UUID: aliceUUID, Username: "alice"}},
			add:         []string{"alice"},
			want:        []string{aliceUUID},
			wantWarning: `warning: reviewer "alice" is already on this pull request`,
		},
		{
			name:        "remove not present warns",
			current:     []bbcloud.User{{UUID: aliceUUID, Username: "alice"}},
			remove:      []string{"bob"},
			want:        []string{aliceUUID},
			wantWarning: `warning: reviewer "bob" is not on this pull request`,
		},
		{
			name:    "add and remove",
			current: []bbcloud.User{{UUID: aliceUUID, Username: "alice"}},
			add:     []string{"charlie"},
			remove:  []string{"alice"},
			want:    []string{"charlie"},
		},
		{
			name:    "cross-identity overlap errors",
			current: []bbcloud.User{{UUID: aliceUUID, Username: "alice"}},
			add:     []string{"alice"},
			remove:  []string{aliceUUID},
			wantErr: "cannot be in both flags",
		},
		{
			name:        "duplicate add deduplicates",
			current:     []bbcloud.User{},
			add:         []string{"bob", "bob"},
			want:        []string{"bob"},
			wantWarning: `warning: reviewer "bob" is already on this pull request`,
		},
		{
			name:        "duplicate add by uuid deduplicates",
			current:     []bbcloud.User{},
			add:         []string{aliceUUID, aliceUUID},
			want:        []string{aliceUUID},
			wantWarning: `warning: reviewer "` + aliceUUID + `" is already on this pull request`,
		},
		{
			name:    "cross-identity overlap uuid-only user uses uuid in error",
			current: []bbcloud.User{{UUID: aliceUUID}},
			add:     []string{"550e8400-e29b-41d4-a716-446655440000"},
			remove:  []string{aliceUUID},
			wantErr: aliceUUID,
		},
		{
			name:    "remove reviewer by account id",
			current: []bbcloud.User{{AccountID: "acc-alice"}, {UUID: bobUUID, Username: "bob"}},
			remove:  []string{"acc-alice"},
			want:    []string{bobUUID},
		},
		{
			name:        "add reviewer when account-id-only user already present warns",
			current:     []bbcloud.User{{AccountID: "acc-alice"}},
			add:         []string{"acc-alice"},
			want:        []string{"acc-alice"},
			wantWarning: `warning: reviewer "acc-alice" is already on this pull request`,
		},
		{
			name:        "remove account-id-only user not present warns",
			current:     []bbcloud.User{{AccountID: "acc-alice"}},
			remove:      []string{"acc-bob"},
			want:        []string{"acc-alice"},
			wantWarning: `warning: reviewer "acc-bob" is not on this pull request`,
		},
		{
			name:    "keep account-id-only reviewer serializes account id",
			current: []bbcloud.User{{AccountID: "acc-alice"}, {UUID: bobUUID, Username: "bob"}},
			add:     []string{"charlie"},
			want:    []string{"acc-alice", bobUUID, "charlie"},
		},
		{
			name:    "cross-identity overlap by account id errors",
			current: []bbcloud.User{{UUID: aliceUUID, AccountID: "acc-alice"}},
			add:     []string{"acc-alice"},
			remove:  []string{aliceUUID},
			wantErr: "cannot be in both flags",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errBuf strings.Builder
			got, err := editCloudReviewers(&errBuf, tt.current, tt.add, tt.remove, nil)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("result[%d] = %q, want %q", i, v, tt.want[i])
				}
			}

			if tt.wantWarning != "" && !strings.Contains(errBuf.String(), tt.wantWarning) {
				t.Errorf("expected warning containing %q, got %q", tt.wantWarning, errBuf.String())
			}
			if tt.wantWarning == "" && errBuf.String() != "" {
				t.Errorf("expected no warnings, got %q", errBuf.String())
			}
		})
	}
}

func TestEditCloudReviewersSuppressesDefaultOverlapWarning(t *testing.T) {
	const bobUUID = "{660e8400-e29b-41d4-a716-446655440000}"

	// current includes bob (from defaults merge), preExisting is empty
	// → adding bob should NOT warn
	var errBuf strings.Builder
	got, err := editCloudReviewers(&errBuf, []bbcloud.User{{UUID: bobUUID, Username: "bob"}}, []string{"bob"}, nil, []bbcloud.User{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errBuf.String() != "" {
		t.Errorf("expected no warning for default-only overlap, got %q", errBuf.String())
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 reviewer, got %v", got)
	}
}

func TestEditCloudReviewersWarnsForPreExistingOverlap(t *testing.T) {
	const bobUUID = "{660e8400-e29b-41d4-a716-446655440000}"

	// bob is in preExisting → adding bob should still warn
	var errBuf strings.Builder
	preExisting := []bbcloud.User{{UUID: bobUUID, Username: "bob"}}
	_, err := editCloudReviewers(&errBuf, preExisting, []string{"bob"}, nil, preExisting)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "already on this pull request") {
		t.Errorf("expected warning for pre-existing overlap, got %q", errBuf.String())
	}
}

func TestEditDCReviewersSuppressesDefaultOverlapWarning(t *testing.T) {
	// current includes bob (from defaults merge), preExisting is empty
	var errBuf strings.Builder
	current := []bbdc.PullRequestReviewer{{User: bbdc.User{Name: "bob"}}}
	got := editDCReviewers(&errBuf, current, []string{"bob"}, nil, []bbdc.PullRequestReviewer{})
	if errBuf.String() != "" {
		t.Errorf("expected no warning for default-only overlap, got %q", errBuf.String())
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 reviewer, got %v", got)
	}
}

func TestEditDCReviewersWarnsForPreExistingOverlap(t *testing.T) {
	var errBuf strings.Builder
	preExisting := []bbdc.PullRequestReviewer{{User: bbdc.User{Name: "bob"}}}
	_ = editDCReviewers(&errBuf, preExisting, []string{"bob"}, nil, preExisting)
	if !strings.Contains(errBuf.String(), "already on this pull request") {
		t.Errorf("expected warning for pre-existing overlap, got %q", errBuf.String())
	}
}

func TestEditCommandReviewerArgumentParsing(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		errorContains string
	}{
		{
			name:          "reviewer only is valid",
			args:          []string{"123", "--reviewer", "alice"},
			errorContains: "", // no error expected
		},
		{
			name:          "remove-reviewer only is valid",
			args:          []string{"123", "--remove-reviewer", "alice"},
			errorContains: "",
		},
		{
			name:          "with-default-reviewers only is valid",
			args:          []string{"123", "--with-default-reviewers"},
			errorContains: "",
		},
		{
			name:          "with-default-reviewers false alone is invalid",
			args:          []string{"123", "--with-default-reviewers=false"},
			errorContains: "at least one of --title, --body, --description, --reviewer, --remove-reviewer, or --with-default-reviewers is required",
		},
		{
			name:          "overlap errors",
			args:          []string{"123", "--reviewer", "alice", "--remove-reviewer", "alice"},
			errorContains: `reviewer "alice" cannot be in both --reviewer and --remove-reviewer`,
		},
		{
			name:          "no flags at all",
			args:          []string{"123"},
			errorContains: "at least one of --title, --body, --description, --reviewer, --remove-reviewer, or --with-default-reviewers is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsServer := tt.errorContains == ""
			var server *httptest.Server
			if needsServer {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					pr := bbdc.PullRequest{
						ID: 123, Title: "Title", Version: 1,
						FromRef: bbdc.Ref{ID: "refs/heads/feature", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
						ToRef:   bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
					}
					if strings.Contains(r.URL.Path, "/default-reviewers/") {
						_ = json.NewEncoder(w).Encode([]map[string]any{})
						return
					}
					_ = json.NewEncoder(w).Encode(pr)
				}))
				defer server.Close()
			}

			baseURL := "https://unused.example.com"
			if needsServer {
				baseURL = server.URL
			}

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: baseURL, Token: "test-token"},
				},
			}

			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.errorContains == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			}
		})
	}
}

func TestRunEditDataCenterReviewers(t *testing.T) {
	tests := []struct {
		name            string
		withDefaults    bool
		reviewers       []string
		removeReviewers []string
		defaultUsers    []map[string]any
		prResponse      bbdc.PullRequest
		putBodyCheck    func(t *testing.T, body map[string]any)
		stderrContains  string
	}{
		{
			name:      "add reviewer",
			reviewers: []string{"charlie"},
			prResponse: bbdc.PullRequest{
				ID: 1, Title: "PR", Version: 1,
				Reviewers: []bbdc.PullRequestReviewer{{User: bbdc.User{Name: "alice"}}},
				FromRef:   bbdc.Ref{ID: "refs/heads/f", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				ToRef:     bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers, ok := body["reviewers"].([]any)
				if !ok {
					t.Fatalf("reviewers not found or wrong type in PUT body")
				}
				if len(reviewers) != 2 {
					t.Fatalf("expected 2 reviewers, got %d", len(reviewers))
				}
			},
		},
		{
			name:            "remove reviewer",
			removeReviewers: []string{"alice"},
			prResponse: bbdc.PullRequest{
				ID: 1, Title: "PR", Version: 1,
				Reviewers: []bbdc.PullRequestReviewer{
					{User: bbdc.User{Name: "alice"}},
					{User: bbdc.User{Name: "bob"}},
				},
				FromRef: bbdc.Ref{ID: "refs/heads/f", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				ToRef:   bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers, ok := body["reviewers"].([]any)
				if !ok {
					t.Fatalf("reviewers not found or wrong type in PUT body")
				}
				if len(reviewers) != 1 {
					t.Fatalf("expected 1 reviewer, got %d", len(reviewers))
				}
			},
		},
		{
			name:      "add already present warns",
			reviewers: []string{"alice"},
			prResponse: bbdc.PullRequest{
				ID: 1, Title: "PR", Version: 1,
				Reviewers: []bbdc.PullRequestReviewer{{User: bbdc.User{Name: "alice"}}},
				FromRef:   bbdc.Ref{ID: "refs/heads/f", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				ToRef:     bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers := body["reviewers"].([]any)
				if len(reviewers) != 1 {
					t.Fatalf("expected 1 reviewer (no duplicate), got %d", len(reviewers))
				}
			},
			stderrContains: "already on this pull request",
		},
		{
			name:         "add default reviewers",
			withDefaults: true,
			defaultUsers: []map[string]any{
				{"name": "bob"},
				{"name": "charlie"},
			},
			prResponse: bbdc.PullRequest{
				ID: 1, Title: "PR", Version: 1,
				Reviewers: []bbdc.PullRequestReviewer{{User: bbdc.User{Name: "alice"}}},
				FromRef:   bbdc.Ref{ID: "refs/heads/feature/auth", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				ToRef:     bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers, ok := body["reviewers"].([]any)
				if !ok {
					t.Fatalf("reviewers not found or wrong type in PUT body")
				}
				if len(reviewers) != 3 {
					t.Fatalf("expected 3 reviewers, got %d", len(reviewers))
				}
				wantNames := []string{"alice", "bob", "charlie"}
				for i, want := range wantNames {
					r := reviewers[i].(map[string]any)
					user := r["user"].(map[string]any)
					if user["name"] != want {
						t.Errorf("reviewer[%d] name = %q, want %q", i, user["name"], want)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var putBody map[string]any

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "GET" && strings.Contains(r.URL.Path, "/default-reviewers/") {
					if got := r.URL.Query().Get("sourceRefId"); got != "refs/heads/feature/auth" {
						t.Fatalf("sourceRefId = %q, want refs/heads/feature/auth", got)
					}
					if got := r.URL.Query().Get("targetRefId"); got != "refs/heads/main" {
						t.Fatalf("targetRefId = %q, want refs/heads/main", got)
					}
					_ = json.NewEncoder(w).Encode([]map[string]any{
						{
							"reviewers": []map[string]any{
								{"users": tt.defaultUsers},
							},
						},
					})
					return
				}
				if r.Method == "GET" {
					_ = json.NewEncoder(w).Encode(tt.prResponse)
					return
				}
				if r.Method == "PUT" {
					_ = json.NewDecoder(r.Body).Decode(&putBody)
					resp := tt.prResponse
					resp.Version++
					_ = json.NewEncoder(w).Encode(resp)
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: server.URL, Username: "testuser", Token: "test-token"},
				},
			}

			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: stderr},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			args := []string{"1"}
			if tt.withDefaults {
				args = append(args, "--with-default-reviewers")
			}
			for _, r := range tt.reviewers {
				args = append(args, "--reviewer", r)
			}
			for _, r := range tt.removeReviewers {
				args = append(args, "--remove-reviewer", r)
			}
			cmd.SetArgs(args)
			cmd.SetContext(context.Background())

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.putBodyCheck != nil && putBody != nil {
				tt.putBodyCheck(t, putBody)
			}

			if tt.stderrContains != "" && !strings.Contains(stderr.String(), tt.stderrContains) {
				t.Errorf("expected stderr containing %q, got %q", tt.stderrContains, stderr.String())
			}
		})
	}
}

func TestRunEditCloudReviewers(t *testing.T) {
	const (
		aliceUUID = "{550e8400-e29b-41d4-a716-446655440000}"
		bobUUID   = "{660e8400-e29b-41d4-a716-446655440000}"
		carolUUID = "{770e8400-e29b-41d4-a716-446655440000}"
	)
	tests := []struct {
		name             string
		withDefaults     bool
		reviewers        []string
		removeReviewers  []string
		defaultReviewers []map[string]any
		prResponse       bbcloud.PullRequest
		putBodyCheck     func(t *testing.T, body map[string]any)
		stderrContains   string
	}{
		{
			name:      "add reviewer by username",
			reviewers: []string{"bob"},
			prResponse: bbcloud.PullRequest{
				ID:        1,
				Title:     "PR",
				Reviewers: []bbcloud.User{{UUID: aliceUUID, Username: "alice"}},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers, ok := body["reviewers"].([]any)
				if !ok {
					t.Fatalf("reviewers not found or wrong type")
				}
				if len(reviewers) != 2 {
					t.Fatalf("expected 2 reviewers, got %d", len(reviewers))
				}
				// Second reviewer should be added by username
				r1 := reviewers[1].(map[string]any)
				if r1["username"] != "bob" {
					t.Errorf("expected second reviewer username 'bob', got %v", r1)
				}
			},
		},
		{
			name:            "remove reviewer by username",
			removeReviewers: []string{"alice"},
			prResponse: bbcloud.PullRequest{
				ID:    1,
				Title: "PR",
				Reviewers: []bbcloud.User{
					{UUID: aliceUUID, Username: "alice"},
					{UUID: bobUUID, Username: "bob"},
				},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers := body["reviewers"].([]any)
				if len(reviewers) != 1 {
					t.Fatalf("expected 1 reviewer, got %d", len(reviewers))
				}
			},
		},
		{
			name:      "add already present warns",
			reviewers: []string{"alice"},
			prResponse: bbcloud.PullRequest{
				ID:        1,
				Title:     "PR",
				Reviewers: []bbcloud.User{{UUID: aliceUUID, Username: "alice"}},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers := body["reviewers"].([]any)
				if len(reviewers) != 1 {
					t.Fatalf("expected 1 reviewer (no duplicate), got %d", len(reviewers))
				}
			},
			stderrContains: "already on this pull request",
		},
		{
			name:         "add default reviewers excludes author",
			withDefaults: true,
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "alice", "uuid": aliceUUID}},
				{"user": map[string]any{"username": "carol", "uuid": carolUUID}},
			},
			prResponse: bbcloud.PullRequest{
				ID:    1,
				Title: "PR",
				Author: struct {
					DisplayName string "json:\"display_name\""
					Username    string "json:\"username\""
					UUID        string "json:\"uuid\""
					AccountID   string "json:\"account_id\""
				}{
					DisplayName: "Alice",
					Username:    "alice",
					UUID:        aliceUUID,
				},
				Reviewers: []bbcloud.User{{UUID: bobUUID, Username: "bob"}},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers, ok := body["reviewers"].([]any)
				if !ok {
					t.Fatalf("reviewers not found or wrong type")
				}
				if len(reviewers) != 2 {
					t.Fatalf("expected 2 reviewers, got %d", len(reviewers))
				}
				r0 := reviewers[0].(map[string]any)
				if r0["uuid"] != bobUUID {
					t.Fatalf("expected first reviewer to preserve bob by uuid, got %v", r0)
				}
				r1 := reviewers[1].(map[string]any)
				if r1["uuid"] != carolUUID {
					t.Fatalf("expected second reviewer to add carol by uuid, got %v", r1)
				}
			},
		},
		{
			name:         "add default reviewers dedups and excludes author across account id",
			withDefaults: true,
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "alice", "account_id": "acc-alice"}},
				{"user": map[string]any{"username": "bob", "account_id": "acc-bob"}},
				{"user": map[string]any{"username": "carol", "account_id": "acc-carol"}},
			},
			prResponse: bbcloud.PullRequest{
				ID:    1,
				Title: "PR",
				Author: struct {
					DisplayName string "json:\"display_name\""
					Username    string "json:\"username\""
					UUID        string "json:\"uuid\""
					AccountID   string "json:\"account_id\""
				}{
					DisplayName: "Alice",
					UUID:        aliceUUID,
					AccountID:   "acc-alice",
				},
				Reviewers: []bbcloud.User{
					{UUID: bobUUID, AccountID: "acc-bob"},
				},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers, ok := body["reviewers"].([]any)
				if !ok {
					t.Fatalf("reviewers not found or wrong type")
				}
				if len(reviewers) != 2 {
					t.Fatalf("expected 2 reviewers after dedup/exclusion, got %d", len(reviewers))
				}
				r0 := reviewers[0].(map[string]any)
				if r0["uuid"] != bobUUID {
					t.Fatalf("expected existing bob reviewer preserved by uuid, got %v", r0)
				}
				r1 := reviewers[1].(map[string]any)
				if r1["username"] != "carol" {
					t.Fatalf("expected only carol to be added, got %v", r1)
				}
			},
		},
		{
			name:         "author with no identity falls back to current user",
			withDefaults: true,
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "bob", "uuid": bobUUID}},
			},
			prResponse: bbcloud.PullRequest{
				ID:    1,
				Title: "PR",
				Author: struct {
					DisplayName string "json:\"display_name\""
					Username    string "json:\"username\""
					UUID        string "json:\"uuid\""
					AccountID   string "json:\"account_id\""
				}{
					DisplayName: "Bob",
				},
				Reviewers: []bbcloud.User{},
			},
			putBodyCheck: func(t *testing.T, body map[string]any) {
				reviewers, ok := body["reviewers"].([]any)
				if !ok {
					t.Fatalf("reviewers not found or wrong type")
				}
				// bob is the current user fallback, should be excluded
				if len(reviewers) != 0 {
					t.Fatalf("expected 0 reviewers (bob excluded via current user fallback), got %d: %v", len(reviewers), reviewers)
				}
			},
			stderrContains: "no usable identity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var putBody map[string]any

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/user" && r.Method == "GET" {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"uuid":       bobUUID,
						"username":   "bob",
						"account_id": "acc-bob",
					})
					return
				}
				if strings.Contains(r.URL.Path, "effective-default-reviewers") {
					_ = json.NewEncoder(w).Encode(map[string]any{"values": tt.defaultReviewers})
					return
				}
				if r.Method == "GET" {
					_ = json.NewEncoder(w).Encode(tt.prResponse)
					return
				}
				if r.Method == "PUT" {
					_ = json.NewDecoder(r.Body).Decode(&putBody)
					_ = json.NewEncoder(w).Encode(tt.prResponse)
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"cloud": {Kind: "cloud", BaseURL: server.URL, Username: "testuser", Token: "test-token"},
				},
			}

			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: stderr},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			args := []string{"1"}
			if tt.withDefaults {
				args = append(args, "--with-default-reviewers")
			}
			for _, r := range tt.reviewers {
				args = append(args, "--reviewer", r)
			}
			for _, r := range tt.removeReviewers {
				args = append(args, "--remove-reviewer", r)
			}
			cmd.SetArgs(args)
			cmd.SetContext(context.Background())

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.putBodyCheck != nil && putBody != nil {
				tt.putBodyCheck(t, putBody)
			}

			if tt.stderrContains != "" && !strings.Contains(stderr.String(), tt.stderrContains) {
				t.Errorf("expected stderr containing %q, got %q", tt.stderrContains, stderr.String())
			}
		})
	}
}

func TestRunEditDataCenterWithDefaultReviewersFalseDoesNotFetchDefaults(t *testing.T) {
	defaultsCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/default-reviewers/") {
			defaultsCalled = true
			t.Fatalf("default reviewers endpoint should not be called when --with-default-reviewers=false")
		}
		if r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(bbdc.PullRequest{
				ID: 1, Title: "PR", Version: 1,
				Reviewers: []bbdc.PullRequestReviewer{{User: bbdc.User{Name: "alice"}}},
				FromRef:   bbdc.Ref{ID: "refs/heads/feature/auth", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				ToRef:     bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
			})
			return
		}
		if r.Method == "PUT" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			reviewers := body["reviewers"].([]any)
			if len(reviewers) != 1 {
				t.Fatalf("expected reviewers to stay unchanged, got %v", reviewers)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "title": "Updated", "version": 2})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: server.URL, Username: "testuser", Token: "test-token"},
		},
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newEditCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"1", "--title", "Updated", "--with-default-reviewers=false"})
	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defaultsCalled {
		t.Fatal("expected default reviewers endpoint not to be called")
	}
}

func TestRunEditCloudWithDefaultReviewersFalseDoesNotFetchDefaults(t *testing.T) {
	defaultsCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "effective-default-reviewers") {
			defaultsCalled = true
			t.Fatalf("effective default reviewers endpoint should not be called when --with-default-reviewers=false")
		}
		if r.Method == "PUT" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if _, ok := body["reviewers"]; ok {
				t.Fatalf("expected reviewers field to be omitted, got %v", body["reviewers"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "title": "Updated"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"cloud": {Kind: "cloud", BaseURL: server.URL, Username: "testuser", Token: "test-token"},
		},
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newEditCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"1", "--title", "Updated", "--with-default-reviewers=false"})
	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defaultsCalled {
		t.Fatal("expected effective default reviewers endpoint not to be called")
	}
}

func TestRunEditDataCenterErrors(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		args          []string
		handler       http.HandlerFunc
		errorContains string
	}{
		{
			name: "missing project and repo",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: "https://bitbucket.example.com", Token: "test-token"},
				},
			},
			args:          []string{"1", "--title", "Updated"},
			errorContains: "context must supply project and repo",
		},
		{
			name: "invalid dc client config",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: "", Token: "test-token"},
				},
			},
			args:          []string{"1", "--title", "Updated"},
			errorContains: "has no base URL configured",
		},
		{
			name: "get pull request error",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: "http://placeholder", Token: "test-token"},
				},
			},
			args: []string{"1", "--title", "Updated"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			errorContains: "500 Internal Server Error",
		},
		{
			name: "default reviewers error",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: "http://placeholder", Token: "test-token"},
				},
			},
			args: []string{"1", "--with-default-reviewers"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.Contains(r.URL.Path, "/default-reviewers/") {
					http.Error(w, "boom", http.StatusInternalServerError)
					return
				}
				_ = json.NewEncoder(w).Encode(bbdc.PullRequest{
					ID: 1, Title: "PR", Version: 1,
					FromRef: bbdc.Ref{ID: "refs/heads/feature/auth", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
					ToRef:   bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
				})
			},
			errorContains: "fetching default reviewers",
		},
		{
			name: "update error",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: "http://placeholder", Token: "test-token"},
				},
			},
			args: []string{"1", "--title", "Updated"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "GET" {
					_ = json.NewEncoder(w).Encode(bbdc.PullRequest{
						ID: 1, Title: "PR", Version: 1,
						FromRef: bbdc.Ref{ID: "refs/heads/feature/auth", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
						ToRef:   bbdc.Ref{ID: "refs/heads/main", Repository: bbdc.Repository{Slug: "repo", Project: &bbdc.Project{Key: "PROJ"}}},
					})
					return
				}
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			errorContains: "500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				server := httptest.NewServer(tt.handler)
				defer server.Close()
				tt.cfg.Hosts["main"].BaseURL = server.URL
			}

			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return tt.cfg, nil },
			}

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)
			cmd.SetContext(context.Background())

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestRunEditCloudErrors(t *testing.T) {
	const aliceUUID = "{550e8400-e29b-41d4-a716-446655440000}"

	tests := []struct {
		name          string
		cfg           *config.Config
		args          []string
		handler       http.HandlerFunc
		errorContains string
	}{
		{
			name: "missing workspace and repo",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "cloud"},
				},
				Hosts: map[string]*config.Host{
					"cloud": {Kind: "cloud", BaseURL: "https://api.bitbucket.org/2.0", Token: "test-token"},
				},
			},
			args:          []string{"1", "--title", "Updated"},
			errorContains: "context must supply workspace and repo",
		},
		{
			name: "invalid cloud client config",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"cloud": {Kind: "cloud", BaseURL: "://bad", Token: "test-token"},
				},
			},
			args:          []string{"1", "--title", "Updated"},
			errorContains: "missing protocol scheme",
		},
		{
			name: "get pull request error",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"cloud": {Kind: "cloud", BaseURL: "http://placeholder", Token: "test-token"},
				},
			},
			args: []string{"1", "--reviewer", "bob"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			errorContains: "500 Internal Server Error",
		},
		{
			name: "default reviewers error",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"cloud": {Kind: "cloud", BaseURL: "http://placeholder", Token: "test-token"},
				},
			},
			args: []string{"1", "--with-default-reviewers"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.Contains(r.URL.Path, "effective-default-reviewers") {
					http.Error(w, "boom", http.StatusInternalServerError)
					return
				}
				_ = json.NewEncoder(w).Encode(bbcloud.PullRequest{
					ID:    1,
					Title: "PR",
					Author: struct {
						DisplayName string `json:"display_name"`
						Username    string `json:"username"`
						UUID        string `json:"uuid"`
						AccountID   string `json:"account_id"`
					}{Username: "alice", UUID: aliceUUID},
				})
			},
			errorContains: "fetching default reviewers",
		},
		{
			name: "edit reviewers error",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"cloud": {Kind: "cloud", BaseURL: "http://placeholder", Token: "test-token"},
				},
			},
			args: []string{"1", "--reviewer", "alice", "--remove-reviewer", aliceUUID},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(bbcloud.PullRequest{
					ID:    1,
					Title: "PR",
					Reviewers: []bbcloud.User{
						{Username: "alice", UUID: aliceUUID},
					},
				})
			},
			errorContains: "cannot be in both flags",
		},
		{
			name: "update error",
			cfg: &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "cloud", Workspace: "ws", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"cloud": {Kind: "cloud", BaseURL: "http://placeholder", Token: "test-token"},
				},
			},
			args: []string{"1", "--title", "Updated"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "PUT" {
					http.Error(w, "boom", http.StatusInternalServerError)
					return
				}
				http.NotFound(w, r)
			},
			errorContains: "500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler != nil {
				server := httptest.NewServer(tt.handler)
				defer server.Close()
				tt.cfg.Hosts["cloud"].BaseURL = server.URL
			}

			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return tt.cfg, nil },
			}

			cmd := newEditCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)
			cmd.SetContext(context.Background())

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestRunEditResolveContextError(t *testing.T) {
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config: func() (*config.Config, error) {
			return nil, errors.New("config boom")
		},
	}

	cmd := newEditCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"1", "--title", "Updated"})
	cmd.SetContext(context.Background())

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "config boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListWorkspaceCloudUsernameFallback(t *testing.T) {
	// Change to a temp directory without a git repo to prevent
	// applyRemoteDefaults from overwriting test context values.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	tests := []struct {
		name           string
		userResponse   bbcloud.User
		hostUsername   string
		expectError    bool
		errorContains  string
		expectUsername string // The username we expect to be used in the API call
	}{
		{
			name: "uses Username when available",
			userResponse: bbcloud.User{
				UUID:      "{12345678-1234-1234-1234-123456789abc}",
				Username:  "testuser",
				AccountID: "557058:12345678-1234-1234-1234-123456789abc",
			},
			hostUsername:   "email@example.com",
			expectError:    false,
			expectUsername: "testuser",
		},
		{
			name: "falls back to AccountID when Username empty",
			userResponse: bbcloud.User{
				UUID:      "{12345678-1234-1234-1234-123456789abc}",
				Username:  "",
				AccountID: "557058:12345678-1234-1234-1234-123456789abc",
			},
			hostUsername:   "email@example.com",
			expectError:    false,
			expectUsername: "557058:12345678-1234-1234-1234-123456789abc",
		},
		{
			name: "falls back to host.Username when Username and AccountID empty and not email",
			userResponse: bbcloud.User{
				UUID:      "{12345}",
				Username:  "",
				AccountID: "",
			},
			hostUsername:   "configureduser",
			expectError:    false,
			expectUsername: "configureduser",
		},
		{
			name: "does not use host.Username if it looks like email",
			userResponse: bbcloud.User{
				UUID:      "{12345}",
				Username:  "",
				AccountID: "",
			},
			hostUsername:  "user@example.com",
			expectError:   true,
			errorContains: "could not determine username",
		},
		{
			name: "error when all fallbacks fail",
			userResponse: bbcloud.User{
				UUID:      "{12345}",
				Username:  "",
				AccountID: "",
			},
			hostUsername:  "",
			expectError:   true,
			errorContains: "could not determine username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				if r.URL.Path == "/user" {
					_ = json.NewEncoder(w).Encode(tt.userResponse)
					return
				}

				// Capture the workspace PR listing path to verify which username was used
				if strings.Contains(r.URL.Path, "/workspaces/") && strings.Contains(r.URL.Path, "/pullrequests/") {
					capturedPath = r.URL.Path
					resp := struct {
						Values []bbcloud.PullRequest `json:"values"`
						Next   string                `json:"next"`
					}{
						Values: []bbcloud.PullRequest{},
					}
					_ = json.NewEncoder(w).Encode(resp)
					return
				}

				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:      "cloud",
						Workspace: "workspace",
						// No DefaultRepo - triggers workspace mode
					},
				},
				Hosts: map[string]*config.Host{
					"cloud": {
						Kind:     "cloud",
						BaseURL:  server.URL,
						Username: tt.hostUsername,
						Token:    "test-token",
					},
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

			cmd := newListCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"--mine"})

			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the correct username was used in the API path
			expectedPath := fmt.Sprintf("/workspaces/workspace/pullrequests/%s", tt.expectUsername)
			if !strings.HasPrefix(capturedPath, expectedPath) {
				t.Errorf("expected API path to contain %q, got %q", expectedPath, capturedPath)
			}
		})
	}
}

func TestListDashboardDCForkScenario(t *testing.T) {
	// Test that fork-based PRs display the destination (ToRef) repository, not the source (FromRef)
	prs := []bbdc.PullRequest{
		{
			ID:    1,
			Title: "PR from fork",
			State: "OPEN",
			FromRef: bbdc.Ref{
				DisplayID: "feature-branch",
				// Source is from user's fork
				Repository: bbdc.Repository{
					Slug:    "user-fork",
					Project: &bbdc.Project{Key: "~USER"},
				},
			},
			ToRef: bbdc.Ref{
				DisplayID: "main",
				// Destination is the upstream repo
				Repository: bbdc.Repository{
					Slug:    "upstream-repo",
					Project: &bbdc.Project{Key: "PROJ"},
				},
			},
		},
		{
			ID:    2,
			Title: "PR same repo",
			State: "OPEN",
			FromRef: bbdc.Ref{
				DisplayID: "feature",
				Repository: bbdc.Repository{
					Slug:    "same-repo",
					Project: &bbdc.Project{Key: "PROJ"},
				},
			},
			ToRef: bbdc.Ref{
				DisplayID: "main",
				Repository: bbdc.Repository{
					Slug:    "same-repo",
					Project: &bbdc.Project{Key: "PROJ"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/dashboard/pull-requests") {
			resp := struct {
				Values     []bbdc.PullRequest `json:"values"`
				IsLastPage bool               `json:"isLastPage"`
			}{
				Values:     prs,
				IsLastPage: true,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:       "main",
				ProjectKey: "PROJ",
				// No DefaultRepo - triggers dashboard mode
			},
		},
		Hosts: map[string]*config.Host{
			"main": {
				Kind:     "dc",
				BaseURL:  server.URL,
				Username: "testuser",
				Token:    "test-token",
			},
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

	cmd := newListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--mine"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// For fork PR: should show destination repo (PROJ/upstream-repo), NOT source (~USER/user-fork)
	if strings.Contains(output, "~USER/user-fork") {
		t.Errorf("output should NOT contain source fork repo '~USER/user-fork', got:\n%s", output)
	}
	if !strings.Contains(output, "PROJ/upstream-repo") {
		t.Errorf("output should contain destination repo 'PROJ/upstream-repo', got:\n%s", output)
	}

	// For same-repo PR: should show the repo normally
	if !strings.Contains(output, "PROJ/same-repo") {
		t.Errorf("output should contain 'PROJ/same-repo', got:\n%s", output)
	}

	// Verify both PRs are listed
	if !strings.Contains(output, "PR from fork") {
		t.Errorf("output should contain 'PR from fork', got:\n%s", output)
	}
	if !strings.Contains(output, "PR same repo") {
		t.Errorf("output should contain 'PR same repo', got:\n%s", output)
	}
}

func TestListWorkspaceCloudURLFallback(t *testing.T) {
	// Change to a temp directory without a git repo to prevent
	// applyRemoteDefaults from overwriting test context values.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	// Test that URL parsing fallback works when Destination.Repository.Slug is empty
	prs := []bbcloud.PullRequest{
		{
			ID:    1,
			Title: "PR with slug",
			State: "OPEN",
		},
		{
			ID:    2,
			Title: "PR without slug",
			State: "OPEN",
		},
	}
	// First PR: has Destination.Repository.Slug set
	prs[0].Source.Branch.Name = "feature-1"
	prs[0].Destination.Branch.Name = "main"
	prs[0].Destination.Repository.Slug = "repo-from-slug"
	prs[0].Links.HTML.Href = "https://bitbucket.org/workspace/repo-from-url/pull-requests/1"
	// Second PR: Destination.Repository.Slug is empty, should fallback to URL parsing
	prs[1].Source.Branch.Name = "feature-2"
	prs[1].Destination.Branch.Name = "main"
	// prs[1].Destination.Repository.Slug is intentionally empty
	prs[1].Links.HTML.Href = "https://bitbucket.org/workspace/repo-from-url/pull-requests/2"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/user" {
			user := bbcloud.User{
				UUID:     "{12345}",
				Username: "testuser",
			}
			_ = json.NewEncoder(w).Encode(user)
			return
		}

		if strings.Contains(r.URL.Path, "/workspaces/") && strings.Contains(r.URL.Path, "/pullrequests/") {
			resp := struct {
				Values []bbcloud.PullRequest `json:"values"`
				Next   string                `json:"next"`
			}{
				Values: prs,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:      "cloud",
				Workspace: "workspace",
			},
		},
		Hosts: map[string]*config.Host{
			"cloud": {
				Kind:     "cloud",
				BaseURL:  server.URL,
				Username: "testuser",
				Token:    "test-token",
			},
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

	cmd := newListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--mine"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// First PR should use Destination.Repository.Slug
	if !strings.Contains(output, "repo-from-slug") {
		t.Errorf("PR with slug should show 'repo-from-slug', got:\n%s", output)
	}

	// Second PR should fallback to URL parsing
	if !strings.Contains(output, "repo-from-url") {
		t.Errorf("PR without slug should fallback to URL parsing and show 'repo-from-url', got:\n%s", output)
	}
}

func TestRunCreateDataCenter(t *testing.T) {
	tests := []struct {
		name           string
		draft          bool
		prResponse     bbdc.PullRequest
		outputContains []string
	}{
		{
			name: "with link",
			prResponse: bbdc.PullRequest{
				ID:    42,
				Title: "Add feature X",
				Links: struct {
					Self []struct {
						Href string `json:"href"`
					} `json:"self"`
				}{
					Self: []struct {
						Href string `json:"href"`
					}{
						{Href: "https://bitbucket.example.com/projects/PROJ/repos/repo/pull-requests/42"},
					},
				},
			},
			outputContains: []string{
				"Created pull request #42",
				"Add feature X",
				"https://bitbucket.example.com/projects/PROJ/repos/repo/pull-requests/42",
			},
		},
		{
			name: "without link",
			prResponse: bbdc.PullRequest{
				ID:    7,
				Title: "Fix bug",
			},
			outputContains: []string{
				"Created pull request #7",
				"Fix bug",
			},
		},
		{
			name:  "draft with link",
			draft: true,
			prResponse: bbdc.PullRequest{
				ID:    10,
				Title: "WIP feature",
				Links: struct {
					Self []struct {
						Href string `json:"href"`
					} `json:"self"`
				}{
					Self: []struct {
						Href string `json:"href"`
					}{
						{Href: "https://bitbucket.example.com/projects/PROJ/repos/repo/pull-requests/10"},
					},
				},
			},
			outputContains: []string{
				"Created draft pull request #10",
				"WIP feature",
				"https://bitbucket.example.com/projects/PROJ/repos/repo/pull-requests/10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "POST" && strings.Contains(r.URL.Path, "/pull-requests") {
					_ = json.NewEncoder(w).Encode(tt.prResponse)
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: server.URL, Username: "testuser", Token: "test-token"},
				},
			}

			stdout := &strings.Builder{}
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newCreateCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			args := []string{"--title", tt.prResponse.Title, "--source", "feature", "--target", "main"}
			if tt.draft {
				args = append(args, "--draft")
			}
			cmd.SetArgs(args)
			cmd.SetContext(context.Background())

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := stdout.String()
			for _, expected := range tt.outputContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestRunCreateCloud(t *testing.T) {
	tests := []struct {
		name           string
		prResponse     bbcloud.PullRequest
		outputContains []string
	}{
		{
			name: "with link",
			prResponse: bbcloud.PullRequest{
				ID:    99,
				Title: "Cloud PR",
				Links: struct {
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				}{
					HTML: struct {
						Href string `json:"href"`
					}{Href: "https://bitbucket.org/workspace/repo/pull-requests/99"},
				},
			},
			outputContains: []string{
				"Created pull request #99",
				"Cloud PR",
				"https://bitbucket.org/workspace/repo/pull-requests/99",
			},
		},
		{
			name: "without link",
			prResponse: bbcloud.PullRequest{
				ID:    3,
				Title: "No link PR",
			},
			outputContains: []string{
				"Created pull request #3",
				"No link PR",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "POST" && strings.Contains(r.URL.Path, "/pullrequests") {
					_ = json.NewEncoder(w).Encode(tt.prResponse)
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", Workspace: "ws", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "cloud", BaseURL: server.URL, Token: "test-token"},
				},
			}

			stdout := &strings.Builder{}
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newCreateCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"--title", tt.prResponse.Title, "--source", "feature", "--target", "main"})
			cmd.SetContext(context.Background())

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := stdout.String()
			for _, expected := range tt.outputContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestMergeReviewers(t *testing.T) {
	type user struct{ Name string }
	nameFunc := func(u user) string { return u.Name }

	tests := []struct {
		name     string
		explicit []string
		defaults []user
		want     []string
	}{
		{
			name:     "no defaults",
			explicit: []string{"alice"},
			defaults: nil,
			want:     []string{"alice"},
		},
		{
			name:     "no explicit",
			explicit: nil,
			defaults: []user{{"alice"}, {"bob"}},
			want:     []string{"alice", "bob"},
		},
		{
			name:     "dedup overlap",
			explicit: []string{"alice", "charlie"},
			defaults: []user{{"alice"}, {"bob"}},
			want:     []string{"alice", "charlie", "bob"},
		},
		{
			name:     "both empty",
			explicit: nil,
			defaults: nil,
			want:     nil,
		},
		{
			name:     "skip empty username",
			explicit: nil,
			defaults: []user{{"alice"}, {""}},
			want:     []string{"alice"},
		},
		{
			name:     "dedup explicit duplicates",
			explicit: []string{"alice", "alice", "bob"},
			defaults: nil,
			want:     []string{"alice", "bob"},
		},
		{
			name:     "dedup across both",
			explicit: []string{"alice", "alice"},
			defaults: []user{{"alice"}, {"bob"}, {"bob"}},
			want:     []string{"alice", "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeCreateReviewers(tt.explicit, tt.defaults, nameFunc)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("mergeCreateReviewers() = %v, want %v", got, tt.want)
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("mergeCreateReviewers()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestResolveGitBaseRefError(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	_, err = resolveGitBaseRef(context.Background(), "missing-branch", "origin")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `could not resolve base branch "missing-branch"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMergeDCPRReviewers(t *testing.T) {
	got := mergeDCEditReviewers(
		[]bbdc.PullRequestReviewer{
			{User: bbdc.User{Name: "alice"}},
			{User: bbdc.User{Name: ""}},
			{User: bbdc.User{Name: "alice"}},
		},
		[]bbdc.User{
			{Name: "alice"},
			{Name: "bob"},
			{Name: ""},
		},
	)

	var names []string
	for _, reviewer := range got {
		names = append(names, reviewer.User.Name)
	}
	want := []string{"alice", "bob"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("got %v, want %v", names, want)
		}
	}
}

func TestMergeCloudPRReviewers(t *testing.T) {
	got := mergeCloudEditReviewers(
		[]bbcloud.User{
			{Username: "alice", UUID: "{00000000-0000-0000-0000-000000000001}"},
			{},
		},
		[]bbcloud.User{
			{Username: "alice", UUID: "{00000000-0000-0000-0000-000000000001}"},
			{UUID: "{00000000-0000-0000-0000-000000000002}"},
			{Username: "bob", UUID: "{00000000-0000-0000-0000-000000000002}"},
			{},
		},
	)

	if len(got) != 2 {
		t.Fatalf("expected 2 reviewers, got %v", got)
	}
	if got[0].Username != "alice" {
		t.Fatalf("expected alice first, got %v", got[0])
	}
	if got[1].UUID != "{00000000-0000-0000-0000-000000000002}" {
		t.Fatalf("expected second reviewer UUID preserved, got %v", got[1])
	}
}

func TestMergeCloudPRReviewersDedupsAcrossAccountID(t *testing.T) {
	got := mergeCloudEditReviewers(
		[]bbcloud.User{
			{UUID: "{00000000-0000-0000-0000-000000000001}", AccountID: "acc-alice"},
		},
		[]bbcloud.User{
			{Username: "alice", AccountID: "acc-alice"},
			{Username: "bob", AccountID: "acc-bob"},
		},
	)

	if len(got) != 2 {
		t.Fatalf("expected 2 reviewers, got %v", got)
	}
	if got[0].UUID != "{00000000-0000-0000-0000-000000000001}" {
		t.Fatalf("expected existing reviewer preserved, got %v", got[0])
	}
	if got[1].Username != "bob" {
		t.Fatalf("expected bob added, got %v", got[1])
	}
}

func TestCloudReviewerIDs(t *testing.T) {
	got := cloudReviewerIDs([]bbcloud.User{
		{UUID: "{00000000-0000-0000-0000-000000000001}"},
		{Username: "bob"},
		{AccountID: "acc-charlie"},
		{},
	})
	want := []string{"{00000000-0000-0000-0000-000000000001}", "bob", "acc-charlie"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestCloudReviewerIDsDedupAcrossAccountID(t *testing.T) {
	got := cloudReviewerIDs([]bbcloud.User{
		{UUID: "{00000000-0000-0000-0000-000000000001}", AccountID: "acc-alice"},
		{Username: "alice", AccountID: "acc-alice"},
		{Username: "bob", AccountID: "acc-bob"},
	})
	want := []string{"{00000000-0000-0000-0000-000000000001}", "bob"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestGetDCDefaultReviewersError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = getDCDefaultReviewers(context.Background(), client, "PROJ", "repo", "feature", "main")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "fetching default reviewers") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSameCloudUser(t *testing.T) {
	tests := []struct {
		name string
		a    bbcloud.User
		b    bbcloud.User
		want bool
	}{
		{
			name: "same uuid",
			a:    bbcloud.User{UUID: "{00000000-0000-0000-0000-000000000001}"},
			b:    bbcloud.User{UUID: "00000000-0000-0000-0000-000000000001"},
			want: true,
		},
		{
			name: "same username",
			a:    bbcloud.User{Username: "alice"},
			b:    bbcloud.User{Username: "alice"},
			want: true,
		},
		{
			name: "same account id bridges mixed identifiers",
			a:    bbcloud.User{UUID: "{00000000-0000-0000-0000-000000000001}", AccountID: "acc-alice"},
			b:    bbcloud.User{Username: "alice", AccountID: "acc-alice"},
			want: true,
		},
		{
			name: "no shared identity",
			a:    bbcloud.User{},
			b:    bbcloud.User{Username: "alice"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameCloudUser(tt.a, tt.b); got != tt.want {
				t.Fatalf("sameCloudUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateCommandDraftFlag(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantDraft      bool
		outputContains string
	}{
		{
			name:           "with --draft flag",
			args:           []string{"--title", "WIP", "--source", "feat", "--target", "main", "--draft"},
			wantDraft:      true,
			outputContains: "Created draft pull request #1",
		},
		{
			name:           "with -d shorthand",
			args:           []string{"--title", "WIP", "--source", "feat", "--target", "main", "-d"},
			wantDraft:      true,
			outputContains: "Created draft pull request #1",
		},
		{
			name:           "without draft flag",
			args:           []string{"--title", "Ready", "--source", "feat", "--target", "main"},
			wantDraft:      false,
			outputContains: "Created pull request #1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "main",
						ProjectKey:  "PROJ",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"main": {
						Kind:    "dc",
						BaseURL: server.URL,
						Token:   "test-token",
					},
				},
			}

			stdout := &strings.Builder{}
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newCreateCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotDraft, ok := gotBody["draft"].(bool)
			if !ok {
				t.Fatal("draft field missing from request body")
			}
			if gotDraft != tt.wantDraft {
				t.Errorf("draft = %v, want %v", gotDraft, tt.wantDraft)
			}

			if !strings.Contains(stdout.String(), tt.outputContains) {
				t.Errorf("output %q does not contain %q", stdout.String(), tt.outputContains)
			}
		})
	}
}

func TestCreateCommandUsesGitDefaults(t *testing.T) {
	repoDir := initCreateDefaultsGitRepo(t, "https://bitbucket.example.com/scm/PROJ/repo.git")

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:        "main",
				ProjectKey:  "PROJ",
				DefaultRepo: "repo",
			},
		},
		Hosts: map[string]*config.Host{
			"main": {
				Kind:    "dc",
				BaseURL: server.URL,
				Token:   "test-token",
			},
		},
	}

	stdout := &strings.Builder{}
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	withWorkingDir(t, repoDir, func() {
		cmd := newCreateCmd(f)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if gotBody["title"] != "feat: add default PR behavior" {
		t.Fatalf("title = %v, want %q", gotBody["title"], "feat: add default PR behavior")
	}

	fromRef, ok := gotBody["fromRef"].(map[string]any)
	if !ok {
		t.Fatalf("fromRef missing or wrong type: %#v", gotBody["fromRef"])
	}
	if fromRef["id"] != "refs/heads/feature/default-pr" {
		t.Fatalf("fromRef.id = %v, want %q", fromRef["id"], "refs/heads/feature/default-pr")
	}

	toRef, ok := gotBody["toRef"].(map[string]any)
	if !ok {
		t.Fatalf("toRef missing or wrong type: %#v", gotBody["toRef"])
	}
	if toRef["id"] != "refs/heads/main" {
		t.Fatalf("toRef.id = %v, want %q", toRef["id"], "refs/heads/main")
	}

	if !strings.Contains(stdout.String(), "Created pull request #1") {
		t.Fatalf("output %q does not contain success message", stdout.String())
	}
}

func TestCreateCommandRequiresExplicitFlagsWhenGitDefaultsUnavailable(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:        "main",
				ProjectKey:  "PROJ",
				DefaultRepo: "repo",
			},
		},
		Hosts: map[string]*config.Host{
			"main": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	var err error
	withWorkingDir(t, tmpDir, func() {
		cmd := newCreateCmd(f)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		err = cmd.Execute()
	})

	if err == nil {
		t.Fatal("expected an error when git defaults are unavailable")
	}
	if !strings.Contains(err.Error(), "could not determine default --source from git") {
		t.Fatalf("error = %q, want source-default guidance", err.Error())
	}
}

func TestCreateCommandMultiRemoteSelectsBitbucket(t *testing.T) {
	// Set up a repo where "origin" is GitHub and "bitbucket" is the DC remote.
	// The defaults should pick "bitbucket" because it matches the active host.
	dir := t.TempDir()
	runGitPRTest(t, dir, "init", "-b", "main")
	runGitPRTest(t, dir, "config", "user.name", "Test User")
	runGitPRTest(t, dir, "config", "user.email", "test@example.com")

	writeTestFile(t, filepath.Join(dir, "README.md"), "base\n")
	runGitPRTest(t, dir, "add", "README.md")
	runGitPRTest(t, dir, "commit", "-m", "chore: base")

	// origin points to GitHub (should NOT be picked)
	runGitPRTest(t, dir, "remote", "add", "origin", "https://github.com/user/repo.git")
	// bitbucket points to the DC instance (should be picked)
	runGitPRTest(t, dir, "remote", "add", "bitbucket", "https://bitbucket.example.com/scm/PROJ/repo.git")

	mainSHA := strings.TrimSpace(runGitPRTestOutput(t, dir, "rev-parse", "HEAD"))
	runGitPRTest(t, dir, "update-ref", "refs/remotes/bitbucket/main", mainSHA)
	runGitPRTest(t, dir, "symbolic-ref", "refs/remotes/bitbucket/HEAD", "refs/remotes/bitbucket/main")

	runGitPRTest(t, dir, "checkout", "-b", "feature/multi-remote")
	writeTestFile(t, filepath.Join(dir, "README.md"), "base\nfeature\n")
	runGitPRTest(t, dir, "add", "README.md")
	runGitPRTest(t, dir, "commit", "-m", "feat: multi-remote test")

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:        "main",
				ProjectKey:  "PROJ",
				DefaultRepo: "repo",
			},
		},
		Hosts: map[string]*config.Host{
			"main": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}

	// Override the BaseURL to the test server after config is created
	cfg.Hosts["main"].BaseURL = server.URL

	// Also update the bitbucket remote to use the test server URL
	runGitPRTest(t, dir, "remote", "set-url", "bitbucket", server.URL+"/scm/PROJ/repo.git")

	stdout := &strings.Builder{}
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	withWorkingDir(t, dir, func() {
		cmd := newCreateCmd(f)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Verify the target came from the "bitbucket" remote (main), not "origin"
	toRef, ok := gotBody["toRef"].(map[string]any)
	if !ok {
		t.Fatalf("toRef missing or wrong type: %#v", gotBody["toRef"])
	}
	if toRef["id"] != "refs/heads/main" {
		t.Fatalf("toRef.id = %v, want %q (from bitbucket remote, not origin)", toRef["id"], "refs/heads/main")
	}
}

func TestCommentInlineValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "from-line without file",
			args:    []string{"42", "--text", "x", "--from-line", "10"},
			wantErr: "--file is required when --from-line or --to-line is specified",
		},
		{
			name:    "to-line without file",
			args:    []string{"42", "--text", "x", "--to-line", "25"},
			wantErr: "--file is required when --from-line or --to-line is specified",
		},
		{
			name:    "file alone",
			args:    []string{"42", "--text", "x", "--file", "src/foo.go"},
			wantErr: "--file must be used with either --from-line or --to-line",
		},
		{
			name:    "both from-line and to-line",
			args:    []string{"42", "--text", "x", "--file", "src/foo.go", "--from-line", "10", "--to-line", "25"},
			wantErr: "--from-line and --to-line are mutually exclusive",
		},
		{
			name:    "parent with inline flags",
			args:    []string{"42", "--text", "x", "--parent", "5", "--file", "src/foo.go", "--to-line", "25"},
			wantErr: "--parent cannot be combined with inline comment flags",
		},
		{
			name:    "from-line zero",
			args:    []string{"42", "--text", "x", "--file", "src/foo.go", "--from-line", "0"},
			wantErr: "--from-line must be a positive integer",
		},
		{
			name:    "to-line zero",
			args:    []string{"42", "--text", "x", "--file", "src/foo.go", "--to-line", "0"},
			wantErr: "--to-line must be a positive integer",
		},
		{
			name:    "file whitespace only with to-line",
			args:    []string{"42", "--text", "x", "--file", "   ", "--to-line", "25"},
			wantErr: "--file value must not be blank",
		},
		{
			name:    "file whitespace only alone",
			args:    []string{"42", "--text", "x", "--file", "   "},
			wantErr: "--file value must not be blank",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCommentCmd(nil)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func initCreateDefaultsGitRepo(t *testing.T, remoteURL string) string {
	t.Helper()

	dir := t.TempDir()
	runGitPRTest(t, dir, "init", "-b", "main")
	runGitPRTest(t, dir, "config", "user.name", "Test User")
	runGitPRTest(t, dir, "config", "user.email", "test@example.com")

	writeTestFile(t, filepath.Join(dir, "README.md"), "base\n")
	runGitPRTest(t, dir, "add", "README.md")
	runGitPRTest(t, dir, "commit", "-m", "chore: base")

	runGitPRTest(t, dir, "remote", "add", "origin", remoteURL)
	mainSHA := strings.TrimSpace(runGitPRTestOutput(t, dir, "rev-parse", "HEAD"))
	runGitPRTest(t, dir, "update-ref", "refs/remotes/origin/main", mainSHA)
	runGitPRTest(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	runGitPRTest(t, dir, "checkout", "-b", "feature/default-pr")
	writeTestFile(t, filepath.Join(dir, "README.md"), "base\nfeature 1\n")
	runGitPRTest(t, dir, "add", "README.md")
	runGitPRTest(t, dir, "commit", "-m", "feat: add default PR behavior")

	writeTestFile(t, filepath.Join(dir, "README.md"), "base\nfeature 1\nfeature 2\n")
	runGitPRTest(t, dir, "add", "README.md")
	runGitPRTest(t, dir, "commit", "-m", "fix: polish defaults")

	return dir
}

func withWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	fn()
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runGitPRTest(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmdArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func runGitPRTestOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}

func TestCommentPendingFlagDC(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: server.URL, Username: "u", Token: "t"},
		},
	}
	stdout := &strings.Builder{}
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newCommentCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"42", "--text", "draft feedback", "--pending"})
	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, ok := gotBody["state"].(string)
	if !ok || state != "PENDING" {
		t.Errorf("state = %v, want PENDING", gotBody["state"])
	}
	if !strings.Contains(stdout.String(), "Pending comment added") {
		t.Errorf("output = %q, want 'Pending comment added'", stdout.String())
	}
}

func TestCommentPendingFlagCloud(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", Workspace: "ws", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "cloud", BaseURL: server.URL, Username: "u", Token: "t"},
		},
	}
	stdout := &strings.Builder{}
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newCommentCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"42", "--text", "draft feedback", "--pending"})
	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pending, ok := gotBody["pending"].(bool)
	if !ok || !pending {
		t.Errorf("pending = %v, want true", gotBody["pending"])
	}
	if !strings.Contains(stdout.String(), "Pending comment added") {
		t.Errorf("output = %q, want 'Pending comment added'", stdout.String())
	}
}

func TestCommentWithoutPendingFlagDC(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: server.URL, Username: "u", Token: "t"},
		},
	}
	stdout := &strings.Builder{}
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newCommentCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"42", "--text", "regular comment"})
	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := gotBody["state"]; ok {
		t.Error("expected no state field when --pending is not set")
	}
	if !strings.Contains(stdout.String(), "Commented on pull request") {
		t.Errorf("output = %q, want 'Commented on pull request'", stdout.String())
	}
}

func TestCreateCommandBodyFlag(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantDescription string
		wantErr         string
	}{
		{
			name:            "body sets description",
			args:            []string{"--title", "T", "--source", "feat", "--target", "main", "--body", "from body"},
			wantDescription: "from body",
		},
		{
			name:            "short flag -b sets description",
			args:            []string{"--title", "T", "--source", "feat", "--target", "main", "-b", "from short"},
			wantDescription: "from short",
		},
		{
			name:            "description flag still works",
			args:            []string{"--title", "T", "--source", "feat", "--target", "main", "--description", "from desc"},
			wantDescription: "from desc",
		},
		{
			name:    "body and description together error",
			args:    []string{"--title", "T", "--source", "feat", "--target", "main", "--body", "b", "--description", "d"},
			wantErr: "specify only one of --body or --description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
				},
				Hosts: map[string]*config.Host{
					"main": {Kind: "dc", BaseURL: server.URL, Token: "test-token"},
				},
			}

			stdout := &strings.Builder{}
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			cmd := newCreateCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotDesc, _ := gotBody["description"].(string)
			if gotDesc != tt.wantDescription {
				t.Errorf("description = %q, want %q", gotDesc, tt.wantDescription)
			}
		})
	}
}

func TestCreateCloudWithDefaultReviewers(t *testing.T) {
	tests := []struct {
		name              string
		currentUser       map[string]any // response for GET /user
		explicitReviewers []string
		defaultReviewers  []map[string]any
		wantReviewers     []map[string]string
	}{
		{
			name:        "adds default reviewers",
			currentUser: map[string]any{"username": "me", "uuid": "{me}"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "alice", "uuid": "{aaa}"}},
				{"user": map[string]any{"username": "bob", "uuid": "{bbb}"}},
			},
			wantReviewers: []map[string]string{
				{"username": "alice"},
				{"username": "bob"},
			},
		},
		{
			name:        "excludes current user by UUID",
			currentUser: map[string]any{"username": "alice", "uuid": "{aaa}"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "alice", "uuid": "{aaa}"}},
				{"user": map[string]any{"username": "bob", "uuid": "{bbb}"}},
			},
			wantReviewers: []map[string]string{
				{"username": "bob"},
			},
		},
		{
			name:        "excludes current user even when host.Username is an email",
			currentUser: map[string]any{"username": "alice", "uuid": "{aaa}"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "alice", "uuid": "{aaa}"}},
				{"user": map[string]any{"username": "bob", "uuid": "{bbb}"}},
			},
			wantReviewers: []map[string]string{
				{"username": "bob"},
			},
		},
		{
			name:              "merges with explicit reviewers and deduplicates",
			currentUser:       map[string]any{"username": "me", "uuid": "{me}"},
			explicitReviewers: []string{"alice", "charlie"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "alice", "uuid": "{aaa}"}},
				{"user": map[string]any{"username": "bob", "uuid": "{bbb}"}},
			},
			wantReviewers: []map[string]string{
				{"username": "alice"},
				{"username": "charlie"},
				{"username": "bob"},
			},
		},
		{
			name:             "empty default reviewers",
			currentUser:      map[string]any{"username": "me", "uuid": "{me}"},
			defaultReviewers: []map[string]any{},
			wantReviewers:    nil,
		},
		{
			name:        "all defaults are current user",
			currentUser: map[string]any{"username": "me", "uuid": "{me}"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "me", "uuid": "{me}"}},
			},
			wantReviewers: nil,
		},
		{
			name:        "falls back to UUID when username is empty",
			currentUser: map[string]any{"username": "me", "uuid": "{00000000-0000-0000-0000-000000000001}"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "", "uuid": "{00000000-0000-0000-0000-000000000099}"}},
				{"user": map[string]any{"username": "bob", "uuid": "{00000000-0000-0000-0000-000000000002}"}},
			},
			wantReviewers: []map[string]string{
				{"uuid": "{00000000-0000-0000-0000-000000000099}"},
				{"username": "bob"},
			},
		},
		{
			name:              "dedup explicit UUID against default username for same user",
			currentUser:       map[string]any{"username": "me", "uuid": "{me}"},
			explicitReviewers: []string{"{00000000-0000-0000-0000-00000000000a}"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "alice", "uuid": "{00000000-0000-0000-0000-00000000000a}"}},
				{"user": map[string]any{"username": "bob", "uuid": "{bbb}"}},
			},
			wantReviewers: []map[string]string{
				{"uuid": "{00000000-0000-0000-0000-00000000000a}"},
				{"username": "bob"},
			},
		},
		{
			name:              "dedup bare UUID against braced UUID",
			currentUser:       map[string]any{"username": "me", "uuid": "{me}"},
			explicitReviewers: []string{"00000000-0000-0000-0000-00000000000a"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "alice", "uuid": "{00000000-0000-0000-0000-00000000000a}"}},
			},
			wantReviewers: []map[string]string{
				{"uuid": "{00000000-0000-0000-0000-00000000000a}"},
			},
		},
		{
			name:        "skips default reviewer with no username and no UUID",
			currentUser: map[string]any{"username": "me", "uuid": "{me}"},
			defaultReviewers: []map[string]any{
				{"user": map[string]any{"username": "", "uuid": ""}},
				{"user": map[string]any{"username": "bob", "uuid": "{bbb}"}},
			},
			wantReviewers: []map[string]string{
				{"username": "bob"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/user" {
					_ = json.NewEncoder(w).Encode(tt.currentUser)
					return
				}
				if strings.Contains(r.URL.Path, "effective-default-reviewers") {
					_ = json.NewEncoder(w).Encode(map[string]any{"values": tt.defaultReviewers})
					return
				}
				if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/pullrequests") {
					_ = json.NewDecoder(r.Body).Decode(&gotBody)
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "title": "test"})
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "cloud",
						Workspace:   "ws",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"cloud": {
						Kind:     "cloud",
						BaseURL:  server.URL,
						Username: "alice@example.com",
						Token:    "test-token",
					},
				},
			}

			stdout := &strings.Builder{}
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
				Config:         func() (*config.Config, error) { return cfg, nil },
			}

			args := []string{"--title", "Test PR", "--source", "feat", "--target", "main", "--with-default-reviewers"}
			for _, r := range tt.explicitReviewers {
				args = append(args, "--reviewer", r)
			}

			cmd := newCreateCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(args)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotReviewers, _ := gotBody["reviewers"].([]any)
			if len(tt.wantReviewers) == 0 {
				if len(gotReviewers) != 0 {
					t.Errorf("expected no reviewers, got %v", gotReviewers)
				}
				return
			}

			if len(gotReviewers) != len(tt.wantReviewers) {
				t.Fatalf("expected %d reviewers, got %d: %v", len(tt.wantReviewers), len(gotReviewers), gotReviewers)
			}
			for i, want := range tt.wantReviewers {
				got, ok := gotReviewers[i].(map[string]any)
				if !ok {
					t.Fatalf("reviewer[%d] is not a map", i)
				}
				for k, v := range want {
					if got[k] != v {
						t.Errorf("reviewer[%d][%q] = %v, want %v", i, k, got[k], v)
					}
				}
			}
		})
	}
}

func TestCreateCloudWithDefaultReviewersAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/user" {
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "me", "uuid": "{me}"})
			return
		}
		if strings.Contains(r.URL.Path, "effective-default-reviewers") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:        "cloud",
				Workspace:   "ws",
				DefaultRepo: "repo",
			},
		},
		Hosts: map[string]*config.Host{
			"cloud": {
				Kind:     "cloud",
				BaseURL:  server.URL,
				Username: "me",
				Token:    "test-token",
			},
		},
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newCreateCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--title", "Test", "--source", "feat", "--target", "main", "--with-default-reviewers"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when API fails")
	}
	if !strings.Contains(err.Error(), "fetching default reviewers") {
		t.Errorf("error = %q, want it to contain 'fetching default reviewers'", err.Error())
	}
}

func TestCreateCloudWithDefaultReviewersCurrentUserError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:        "cloud",
				Workspace:   "ws",
				DefaultRepo: "repo",
			},
		},
		Hosts: map[string]*config.Host{
			"cloud": {
				Kind:     "cloud",
				BaseURL:  server.URL,
				Username: "me",
				Token:    "bad-token",
			},
		},
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config:         func() (*config.Config, error) { return cfg, nil },
	}

	cmd := newCreateCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--title", "Test", "--source", "feat", "--target", "main", "--with-default-reviewers"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when CurrentUser fails")
	}
	if !strings.Contains(err.Error(), "resolving current user") {
		t.Errorf("error = %q, want it to contain 'resolving current user'", err.Error())
	}
}
