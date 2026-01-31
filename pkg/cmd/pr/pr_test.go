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

			if tt.prResponse.FromRef.LatestCommit != "" && len(tt.statusResponse) >= 0 && !statusCalled {
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

			if tt.prResponse.Source.Commit.Hash != "" && len(tt.statusResponse) >= 0 && !statusCalled {
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
	// Skip in CI environments where this test is flaky
	// See: https://github.com/avivsinai/bitbucket-cli/issues/40
	if os.Getenv("CI") != "" {
		t.Skip("Skipping flaky test in CI - see issue #40")
	}

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
			errorContains: "at least one of --title, --body, or --description is required",
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
