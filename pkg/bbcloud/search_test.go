package bbcloud_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestSearchWorkspaceSkillsPaginatesFiltersEscapesAndAuthenticates(t *testing.T) {
	var requests []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Clone(r.Context()))
		if user, password, ok := r.BasicAuth(); !ok || user != "alice" || password != "secret" {
			t.Errorf("BasicAuth = %q, %q, %v", user, password, ok)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{
				map[string]any{"content_match_count": 1, "file": map[string]any{"path": "skills/beta/SKILL.md"}},
			}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{
				map[string]any{"content_match_count": 9, "file": map[string]any{"path": "README.md"}},
				map[string]any{
					"content_match_count": 2,
					"file": map[string]any{
						"path":   "skills/alpha/SKILL.md",
						"commit": map[string]any{"repository": map[string]any{"full_name": "my team/skills"}},
					},
				},
			},
			"next": serverURL(r) + "/workspaces/my%20team/search/code?page=2&pagelen=2",
		})
	}))
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "alice", Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	results, err := client.SearchWorkspaceSkills(context.Background(), "my team", `name: "code review"`, 2)
	if err != nil {
		t.Fatalf("SearchWorkspaceSkills: %v", err)
	}
	if len(results) != 2 || results[0].File.Path != "skills/alpha/SKILL.md" || results[1].File.Path != "skills/beta/SKILL.md" {
		t.Fatalf("results = %+v", results)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	first := requests[0]
	if first.URL.EscapedPath() != "/workspaces/my%20team/search/code" {
		t.Errorf("path = %q", first.URL.EscapedPath())
	}
	if got := first.URL.Query().Get("search_query"); got != `(name: "code review") path:SKILL.md` {
		t.Errorf("search_query = %q", got)
	}
	if got := first.URL.Query().Get("fields"); got != "+values.file.commit.repository" {
		t.Errorf("fields = %q", got)
	}
	if got := first.URL.Query().Get("pagelen"); got != "2" {
		t.Errorf("pagelen = %q", got)
	}
}

func TestSearchWorkspaceSkillsRejectsForeignPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[],"next":"https://evil.example/steal"}`))
	}))
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.SearchWorkspaceSkills(context.Background(), "team", "review", 10)
	if err == nil || !strings.Contains(err.Error(), "does not target /workspaces/team/search/code") {
		t.Fatalf("error = %v", err)
	}
}

func TestSearchWorkspaceSkillsValidatesArguments(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.SearchWorkspaceSkills(context.Background(), "", "query", 1); err == nil {
		t.Fatal("expected workspace error")
	}
	if _, err := client.SearchWorkspaceSkills(context.Background(), "team", "", 1); err == nil {
		t.Fatal("expected query error")
	}
}

func TestSearchWorkspaceSkillsReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "search unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchWorkspaceSkills(context.Background(), "team", "review", 10)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("error = %v", err)
	}
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
