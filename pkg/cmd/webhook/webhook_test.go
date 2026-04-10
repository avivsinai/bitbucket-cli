package webhook_test

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
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/root"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestWebhookCommandValidation(t *testing.T) {
	t.Run("list requires project and repo for data center", func(t *testing.T) {
		cfg := dcConfig("http://localhost")
		cfg.Contexts["test"].DefaultRepo = ""

		_, _, err := runCLI(t, cfg, "webhook", "list")
		if err == nil {
			t.Fatal("expected error when repo is missing")
		}
		if !strings.Contains(err.Error(), "context must supply project and repo") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("list requires workspace and repo for cloud", func(t *testing.T) {
		cfg := cloudConfig("http://localhost")
		cfg.Contexts["test"].Workspace = ""

		_, _, err := runCLI(t, cfg, "webhook", "list")
		if err == nil {
			t.Fatal("expected error when workspace is missing")
		}
		if !strings.Contains(err.Error(), "context must supply workspace and repo") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("delete rejects invalid data center webhook id", func(t *testing.T) {
		_, _, err := runCLI(t, dcConfig("http://localhost"), "webhook", "delete", "abc")
		if err == nil {
			t.Fatal("expected invalid id error")
		}
		if !strings.Contains(err.Error(), `invalid webhook id "abc"`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("test is rejected on cloud", func(t *testing.T) {
		_, _, err := runCLI(t, cloudConfig("http://localhost"), "webhook", "test", "42")
		if err == nil {
			t.Fatal("expected cloud test command to fail")
		}
		if !strings.Contains(err.Error(), "Data Center contexts only") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("test rejects invalid data center webhook id", func(t *testing.T) {
		_, _, err := runCLI(t, dcConfig("http://localhost"), "webhook", "test", "bogus")
		if err == nil {
			t.Fatal("expected invalid id error")
		}
		if !strings.Contains(err.Error(), `invalid webhook id "bogus"`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWebhookListDataCenter(t *testing.T) {
	t.Run("lists webhooks in text output", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/my-repo/webhooks" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{
						"id":     7,
						"name":   "ci-trigger",
						"url":    "https://ci.example.com/hook",
						"active": true,
					},
					{
						"id":     8,
						"name":   "slack-notify",
						"url":    "https://hooks.slack.com/abc",
						"active": false,
					},
				},
			})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "webhook", "list")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "7\tactive\tci-trigger (https://ci.example.com/hook)") {
			t.Fatalf("expected active webhook in output, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "8\tdisabled\tslack-notify (https://hooks.slack.com/abc)") {
			t.Fatalf("expected disabled webhook in output, got:\n%s", stdout)
		}
	})

	t.Run("supports json output", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{
						"id":     9,
						"name":   "json-hook",
						"url":    "https://example.com/json",
						"active": true,
						"events": []string{"repo:refs_changed"},
					},
				},
			})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "webhook", "list", "--json")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}

		var payload struct {
			Project  string `json:"project"`
			Repo     string `json:"repo"`
			Webhooks []struct {
				ID     int      `json:"id"`
				Name   string   `json:"name"`
				URL    string   `json:"url"`
				Active bool     `json:"active"`
				Events []string `json:"events"`
			} `json:"webhooks"`
		}
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("decode json output: %v\nstdout=%s", err, stdout)
		}
		if payload.Project != "PROJ" || payload.Repo != "my-repo" {
			t.Fatalf("unexpected payload header: %+v", payload)
		}
		if len(payload.Webhooks) != 1 {
			t.Fatalf("expected one webhook, got %d", len(payload.Webhooks))
		}
		if payload.Webhooks[0].Name != "json-hook" || payload.Webhooks[0].URL != "https://example.com/json" {
			t.Fatalf("unexpected webhook payload: %+v", payload.Webhooks[0])
		}
	})

	t.Run("prints empty message", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "webhook", "list")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "No webhooks configured.") {
			t.Fatalf("expected empty-state message, got: %s", stdout)
		}
	})
}

func TestWebhookListCloud(t *testing.T) {
	t.Run("lists webhooks in text output", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/repositories/myworkspace/my-repo/hooks" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{
						"uuid":   "{11111111-1111-1111-1111-111111111111}",
						"url":    "https://ci.example.com/hook",
						"active": true,
					},
					{
						"uuid":   "{22222222-2222-2222-2222-222222222222}",
						"url":    "https://hooks.slack.com/abc",
						"active": false,
					},
				},
			})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "webhook", "list")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "{11111111-1111-1111-1111-111111111111}\tactive\thttps://ci.example.com/hook") {
			t.Fatalf("expected active webhook in output, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, "{22222222-2222-2222-2222-222222222222}\tdisabled\thttps://hooks.slack.com/abc") {
			t.Fatalf("expected disabled webhook in output, got:\n%s", stdout)
		}
	})

	t.Run("prints empty message", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "webhook", "list")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "No webhooks configured.") {
			t.Fatalf("expected empty-state message, got: %s", stdout)
		}
	})
}

func TestWebhookCreate(t *testing.T) {
	t.Run("creates data center webhook", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/my-repo/webhooks" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if body["name"] != "ci-trigger" || body["url"] != "https://ci.example.com/hook" {
				t.Fatalf("unexpected request body: %+v", body)
			}
			if body["active"] != false {
				t.Fatalf("expected active=false, got %+v", body["active"])
			}
			events, ok := body["events"].([]any)
			if !ok || len(events) != 2 || events[0] != "repo:refs_changed" || events[1] != "pr:opened" {
				t.Fatalf("unexpected events payload: %+v", body["events"])
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   11,
				"name": "ci-trigger",
			})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, dcConfig(srv.URL),
			"webhook", "create",
			"--name", "ci-trigger",
			"--url", "https://ci.example.com/hook",
			"--event", "repo:refs_changed",
			"--event", "pr:opened",
			"--active=false",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "✓ Created webhook #11 (ci-trigger)") {
			t.Fatalf("unexpected output: %s", stdout)
		}
	})

	t.Run("creates cloud webhook", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/repositories/myworkspace/my-repo/hooks" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if body["description"] != "slack-notify" || body["url"] != "https://hooks.slack.com/abc" {
				t.Fatalf("unexpected request body: %+v", body)
			}
			if body["active"] != true {
				t.Fatalf("expected active=true, got %+v", body["active"])
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"uuid": "{33333333-3333-3333-3333-333333333333}",
			})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL),
			"webhook", "create",
			"--name", "slack-notify",
			"--url", "https://hooks.slack.com/abc",
			"--event", "repo:push",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "✓ Created webhook {33333333-3333-3333-3333-333333333333}") {
			t.Fatalf("unexpected output: %s", stdout)
		}
	})
}

func TestWebhookDelete(t *testing.T) {
	t.Run("deletes data center webhook", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s, want DELETE", r.Method)
			}
			if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/my-repo/webhooks/42" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "webhook", "delete", "42")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "✓ Deleted webhook #42") {
			t.Fatalf("unexpected output: %s", stdout)
		}
	})

	t.Run("deletes cloud webhook and trims braces in request path", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s, want DELETE", r.Method)
			}
			if r.URL.Path != "/repositories/myworkspace/my-repo/hooks/44444444-4444-4444-4444-444444444444" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "webhook", "delete", "{44444444-4444-4444-4444-444444444444}")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "✓ Deleted webhook {44444444-4444-4444-4444-444444444444}") {
			t.Fatalf("unexpected output: %s", stdout)
		}
	})
}

func TestWebhookTestDataCenter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/my-repo/webhooks/15/test" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "webhook", "test", "15")
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
	}
	if !strings.Contains(stdout, "✓ Triggered test delivery for webhook #15") {
		t.Fatalf("unexpected output: %s", stdout)
	}
}

func cloudConfig(baseURL string) *config.Config {
	return &config.Config{
		ActiveContext: "test",
		Contexts: map[string]*config.Context{
			"test": {
				Host:        "mock",
				Workspace:   "myworkspace",
				DefaultRepo: "my-repo",
			},
		},
		Hosts: map[string]*config.Host{
			"mock": {
				Kind:     "cloud",
				BaseURL:  baseURL,
				Username: "admin",
				Token:    "token",
			},
		},
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
	t.Chdir(t.TempDir())

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
	rootCmd.SilenceUsage = true

	t.Setenv("BKT_NO_UPDATE_CHECK", "1")
	t.Setenv("NO_COLOR", "1")

	err = rootCmd.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}
