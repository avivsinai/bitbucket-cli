package pr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
)

func TestDeclineCloseAlias(t *testing.T) {
	cfg := dcConfig("http://localhost")

	t.Run("canonical name", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "decline", "--help")
		if err != nil {
			t.Fatalf("pr decline --help failed: %v", err)
		}
	})

	t.Run("close alias", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "close", "--help")
		if err != nil {
			t.Fatalf("pr close --help failed: %v", err)
		}
	})

	t.Run("help output matches", func(t *testing.T) {
		declineOut, _, err := runCLI(t, cfg, "pr", "decline", "--help")
		if err != nil {
			t.Fatalf("pr decline --help failed: %v", err)
		}
		closeOut, _, err := runCLI(t, cfg, "pr", "close", "--help")
		if err != nil {
			t.Fatalf("pr close --help failed: %v", err)
		}
		if declineOut != closeOut {
			t.Errorf("help output differs between 'pr decline' and 'pr close'\ndecline:\n%s\nclose:\n%s", declineOut, closeOut)
		}
	})
}

func TestCommentCmd(t *testing.T) {
	cfg := dcConfig("http://localhost")

	t.Run("canonical name", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "comment", "--help")
		if err != nil {
			t.Fatalf("pr comment --help failed: %v", err)
		}
	})
}

func TestCreateDestinationFlagAlias(t *testing.T) {
	cfg := dcConfig("http://localhost")

	t.Run("--target accepted", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "create", "--target", "main", "--help")
		if err != nil {
			t.Fatalf("pr create --target --help failed: %v", err)
		}
	})

	t.Run("--destination accepted", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "create", "--destination", "main", "--help")
		if err != nil {
			t.Fatalf("pr create --destination --help failed: %v", err)
		}
	})

	t.Run("--target and --destination mutually exclusive", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "create", "--target", "main", "--destination", "develop")
		if err == nil {
			t.Fatal("expected error when both --target and --destination supplied")
		}
		if !strings.Contains(err.Error(), "specify only one") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("--destination populates target branch in API request", func(t *testing.T) {
		var (
			mu          sync.Mutex
			capturedRef string
		)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/pull-requests") {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if toRef, ok := body["toRef"].(map[string]any); ok {
					if id, ok := toRef["id"].(string); ok {
						mu.Lock()
						capturedRef = id
						mu.Unlock()
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "title": "t"})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		testCfg := &config.Config{
			ActiveContext: "test",
			Contexts: map[string]*config.Context{
				"test": {Host: "mock", ProjectKey: "PROJ", DefaultRepo: "my-repo"},
			},
			Hosts: map[string]*config.Host{
				"mock": {Kind: "dc", BaseURL: server.URL, Username: "admin", Token: "token"},
			},
		}

		_, _, err := runCLI(t, testCfg, "pr", "create",
			"--title", "t", "--source", "feat", "--destination", "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		if capturedRef != "refs/heads/main" {
			t.Errorf("expected toRef.id = refs/heads/main, got %q", capturedRef)
		}
	})
}
