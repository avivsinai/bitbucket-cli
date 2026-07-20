package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

func TestNewPlatformBackendDoesNotMutateFrozenSnapshot(t *testing.T) {
	snap := &Snapshot{
		Platform:  "cloud",
		HostLabel: "cloud",
		Host:      config.Host{Kind: "cloud", Token: "token"},
	}
	want := *snap
	if _, err := newPlatformBackend(snap); err != nil {
		t.Fatalf("newPlatformBackend: %v", err)
	}
	if !reflect.DeepEqual(*snap, want) {
		t.Fatalf("snapshot mutated during client construction:\n got: %+v\nwant: %+v", *snap, want)
	}
}

func TestNewPlatformBackendCloudOAuthUsesOnlyFrozenCredential(t *testing.T) {
	t.Setenv("BKT_TOKEN", "")
	t.Setenv("BKT_OAUTH_CLIENT_ID", "")
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer startup-token" {
			t.Errorf("Authorization = %q, want startup bearer token", got)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	snap := &Snapshot{
		Platform: "cloud",
		Host: config.Host{
			Kind:           "cloud",
			BaseURL:        server.URL,
			AuthMethod:     "oauth",
			Token:          "startup-token",
			OAuthExpiresAt: time.Now().Add(-time.Minute),
		},
	}

	backend, err := newPlatformBackend(snap)
	if err != nil {
		t.Fatalf("newPlatformBackend: %v", err)
	}
	_, _, err = backend.listRepositories(context.Background(), "team", 25)
	var httpErr *httpx.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("listRepositories error = %T %v, want frozen 401 HTTPError", err, err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want one request with no refresh retry", got)
	}
	if snap.Host.Token != "startup-token" {
		t.Fatalf("snapshot token mutated to %q", snap.Host.Token)
	}
}

func TestDCBackendAppliesRepositoryRoleFiltersUpstream(t *testing.T) {
	for _, role := range []string{"AUTHOR", "REVIEWER"} {
		t.Run(strings.ToLower(role), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/api/pull-requests" {
					t.Fatalf("path = %q", r.URL.Path)
				}
				query := r.URL.Query()
				if query.Get("state") != "OPEN" || query.Get("role.1") != role || query.Get("username.1") != "alice" || query.Get("limit") != "7" {
					t.Fatalf("query = %q", r.URL.RawQuery)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"values": []map[string]any{{
						"id": 1, "title": "One", "createdDate": int64(1704067200000), "updatedDate": int64(1704067200000), "reviewers": []any{},
					}},
					"isLastPage":    false,
					"nextPageStart": 7,
				})
			}))
			t.Cleanup(server.Close)
			client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Token: "token", AuthMethod: "bearer", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
			if err != nil {
				t.Fatal(err)
			}

			items, hasMore, err := (&dcBackend{client: client, username: "alice"}).listPullRequests(
				context.Background(), RepositoryRef{Scope: "PROJ", Slug: "api"}, "OPEN", role, 7,
			)
			if err != nil {
				t.Fatalf("listPullRequests: %v", err)
			}
			if len(items) != 1 || !hasMore {
				t.Fatalf("items=%#v hasMore=%v, want one item and continuation", items, hasMore)
			}
		})
	}
}

func TestDCBackendUsesNativeDashboardRoleUpstream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/dashboard/pull-requests" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("role") != "AUTHOR" || r.URL.Query().Get("state") != "MERGED" || r.URL.Query().Get("limit") != "5" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}, "isLastPage": true})
	}))
	t.Cleanup(server.Close)
	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Token: "token", AuthMethod: "bearer", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}

	items, hasMore, err := (&dcBackend{client: client}).listMyPullRequests(context.Background(), "", "MERGED", "AUTHOR", 5)
	if err != nil {
		t.Fatalf("listMyPullRequests: %v", err)
	}
	if items == nil || hasMore {
		t.Fatalf("items=%#v hasMore=%v, want non-nil empty final page", items, hasMore)
	}
}

func TestDCBackendDashboardReviewerDoesNotNeedUsername(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/dashboard/pull-requests" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("role") != "REVIEWER" || r.URL.Query().Get("state") != "OPEN" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}, "isLastPage": true})
	}))
	t.Cleanup(server.Close)
	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Token: "token", AuthMethod: "bearer", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}

	items, hasMore, err := (&dcBackend{client: client}).listMyPullRequests(context.Background(), "", "OPEN", "REVIEWER", 5)
	if err != nil {
		t.Fatalf("listMyPullRequests: %v", err)
	}
	if items == nil || hasMore {
		t.Fatalf("items=%#v hasMore=%v, want non-nil empty final page", items, hasMore)
	}
}

func TestCloudBackendResolvesCurrentUserBeforeRepositoryRoleFilters(t *testing.T) {
	const uuid = "{123e4567-e89b-12d3-a456-426614174000}"
	tests := []struct {
		role      string
		wantField string
	}{
		{role: "AUTHOR", wantField: "author.uuid"},
		{role: "REVIEWER", wantField: "reviewers.uuid"},
	}
	for _, tt := range tests {
		t.Run(strings.ToLower(tt.role), func(t *testing.T) {
			var requests []string
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.RequestURI())
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/user":
					_ = json.NewEncoder(w).Encode(map[string]any{"uuid": uuid})
				case "/repositories/team/api/pullrequests":
					if got := r.URL.Query().Get("q"); got != tt.wantField+` = "`+uuid+`"` {
						t.Fatalf("q = %q", got)
					}
					if r.URL.Query().Get("state") != "OPEN" || r.URL.Query().Get("pagelen") != "3" {
						t.Fatalf("query = %q", r.URL.RawQuery)
					}
					_ = json.NewEncoder(w).Encode(map[string]any{
						"values": []any{},
						"next":   server.URL + "/repositories/team/api/pullrequests?page=2",
					})
				default:
					t.Fatalf("path = %q", r.URL.Path)
				}
			}))
			t.Cleanup(server.Close)
			client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Token: "token", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
			if err != nil {
				t.Fatal(err)
			}

			items, hasMore, err := (&cloudBackend{client: client}).listPullRequests(
				context.Background(), RepositoryRef{Scope: "team", Slug: "api"}, "OPEN", tt.role, 3,
			)
			if err != nil {
				t.Fatalf("listPullRequests: %v", err)
			}
			if len(requests) != 2 || items == nil || !hasMore {
				t.Fatalf("requests=%v items=%#v hasMore=%v", requests, items, hasMore)
			}
		})
	}
}

func TestCloudBackendUsesNativeWorkspaceAuthorEndpoint(t *testing.T) {
	const uuid = "{123e4567-e89b-12d3-a456-426614174000}"
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"uuid": uuid, "username": "legacy-alice", "account_id": "account-alice"})
		case "/workspaces/team/pullrequests/" + uuid:
			if r.URL.Query().Get("state") != "DECLINED" || r.URL.Query().Get("pagelen") != "4" {
				t.Fatalf("query = %q", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		default:
			t.Fatalf("path = %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Token: "token", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}

	items, hasMore, err := (&cloudBackend{client: client}).listMyPullRequests(context.Background(), "team", "DECLINED", "AUTHOR", 4)
	if err != nil {
		t.Fatalf("listMyPullRequests: %v", err)
	}
	if len(requests) != 2 || items == nil || hasMore {
		t.Fatalf("requests=%v items=%#v hasMore=%v", requests, items, hasMore)
	}
	if strings.Contains(strings.Join(requests, " "), "reviewer") {
		t.Fatalf("workspace author flow unexpectedly contains reviewer filter: %v", requests)
	}
}

func TestCloudWorkspaceUserIdentityFallsBackToAccountID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"username": "legacy-alice", "account_id": "account-alice"})
	}))
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Token: "token", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}

	identity, err := (&cloudBackend{client: client}).workspaceUserIdentity(context.Background())
	if err != nil {
		t.Fatalf("workspaceUserIdentity: %v", err)
	}
	if identity != "account-alice" {
		t.Fatalf("identity = %q, want stable account ID", identity)
	}
}
