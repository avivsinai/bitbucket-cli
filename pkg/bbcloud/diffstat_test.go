package bbcloud_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestPullRequestDiffStat(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"status":        "modified",
					"lines_added":   10,
					"lines_removed": 5,
					"old":           map[string]string{"path": "src/main.go"},
					"new":           map[string]string{"path": "src/main.go"},
				},
				{
					"status":        "added",
					"lines_added":   25,
					"lines_removed": 0,
					"new":           map[string]string{"path": "src/utils.go"},
				},
				{
					"status":        "removed",
					"lines_added":   0,
					"lines_removed": 30,
					"old":           map[string]string{"path": "old/legacy.go"},
				},
			},
		})
	}))

	result, err := client.PullRequestDiffStat(context.Background(), "myworkspace", "my-repo", 7)
	if err != nil {
		t.Fatalf("PullRequestDiffStat: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7/diffstat" {
		t.Errorf("path = %q, want /repositories/myworkspace/my-repo/pullrequests/7/diffstat", gotPath)
	}
	if len(result.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(result.Entries))
	}
	if result.TotalAdded != 35 {
		t.Errorf("TotalAdded = %d, want 35", result.TotalAdded)
	}
	if result.TotalRemoved != 35 {
		t.Errorf("TotalRemoved = %d, want 35", result.TotalRemoved)
	}
	if len(result.Entries) != 3 {
		t.Fatalf("len(Entries) = %d, want 3", len(result.Entries))
	}
	if result.Entries[0].Status != "modified" {
		t.Errorf("Entries[0].Status = %q, want modified", result.Entries[0].Status)
	}
	if result.Entries[1].NewPath != "src/utils.go" {
		t.Errorf("Entries[1].NewPath = %q, want src/utils.go", result.Entries[1].NewPath)
	}
	if result.Entries[2].OldPath != "old/legacy.go" {
		t.Errorf("Entries[2].OldPath = %q, want old/legacy.go", result.Entries[2].OldPath)
	}
}

func TestPullRequestDiffStatValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
	}{
		{"empty workspace", "", "repo"},
		{"empty repo", "ws", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := client.PullRequestDiffStat(context.Background(), tt.workspace, tt.repo, 1); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestPullRequestDiffStatPagination(t *testing.T) {
	var hits int32
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{
						"status":        "modified",
						"lines_added":   10,
						"lines_removed": 2,
						"old":           map[string]string{"path": "a.go"},
						"new":           map[string]string{"path": "a.go"},
					},
				},
				"next": serverURL + "/repositories/ws/repo/pullrequests/1/diffstat?page=2",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{
						"status":        "added",
						"lines_added":   5,
						"lines_removed": 0,
						"new":           map[string]string{"path": "b.go"},
					},
				},
			})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.PullRequestDiffStat(context.Background(), "ws", "repo", 1)
	if err != nil {
		t.Fatalf("PullRequestDiffStat: %v", err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("len(Entries) = %d, want 2", len(result.Entries))
	}
	if result.TotalAdded != 15 {
		t.Errorf("TotalAdded = %d, want 15", result.TotalAdded)
	}
	if result.TotalRemoved != 2 {
		t.Errorf("TotalRemoved = %d, want 2", result.TotalRemoved)
	}
	if hits != 2 {
		t.Errorf("expected 2 requests, got %d", hits)
	}
}

func TestPullRequestDiffStatRejectsTruncatedResults(t *testing.T) {
	var hits int32
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")

		resp := map[string]any{
			"values": []map[string]any{
				{
					"status":        "modified",
					"lines_added":   1,
					"lines_removed": 1,
					"old":           map[string]string{"path": "file.go"},
					"new":           map[string]string{"path": "file.go"},
				},
			},
		}
		if count <= 50 {
			resp["next"] = serverURL + "/repositories/ws/repo/pullrequests/1/diffstat?page=overflow"
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.PullRequestDiffStat(context.Background(), "ws", "repo", 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "diffstat pagination exceeded safety limit (50 pages)" {
		t.Fatalf("error = %q", got)
	}
	if hits != 50 {
		t.Fatalf("expected 50 requests, got %d", hits)
	}
}
