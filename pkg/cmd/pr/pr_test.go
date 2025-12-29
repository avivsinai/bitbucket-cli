package pr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			errorContains: "pull request has no source commit",
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
						Host:       "main",
						ProjectKey: "PROJ",
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
			errorContains: "pull request has no source commit",
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
			name:          "valid pr id",
			args:          []string{"123"},
			expectError:   false,
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
						Host:       "main",
						ProjectKey: "PROJ",
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
				BaseURL:  "https://api.bitbucket.org/2.0",
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
				BaseURL:  "https://api.bitbucket.org/2.0",
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
	// Very small durations should not go below 1 second
	got := addJitter(500 * time.Millisecond)
	if got < time.Second {
		t.Errorf("addJitter(500ms) = %v, want >= 1s minimum", got)
	}
}

func TestAddJitterZeroAndNegative(t *testing.T) {
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
