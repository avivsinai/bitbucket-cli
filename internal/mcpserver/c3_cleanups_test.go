package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

func TestCloudBackendChecksTrimsSourceRepositoryLocator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repositories/team/api/pullrequests/7":
			payload := cloudPullRequestPayload()
			payload["source"].(map[string]any)["repository"] = map[string]any{
				"full_name": "  contributor  /  fork  ",
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "/repositories/contributor/fork/commit/source-sha/statuses":
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: server.URL,
		Token:   "token",
		Retry:   httpx.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	checks, hasMore, err := (&cloudBackend{client: client}).getPullRequestChecks(
		context.Background(), RepositoryRef{Scope: "team", Slug: "api"}, 7,
	)
	if err != nil {
		t.Fatalf("getPullRequestChecks: %v", err)
	}
	if len(checks) != 0 || hasMore {
		t.Fatalf("checks=%v hasMore=%v, want empty terminal page", checks, hasMore)
	}
}
