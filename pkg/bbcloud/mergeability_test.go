package bbcloud_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestPullRequestMergeability(t *testing.T) {
	base := `{"type":"pullrequest_state_check","status":"PASSED","required":true,"blocking":false},
	{"type":"current_user_permission_check","status":"PASSED","required":true,"blocking":false}`
	clean := `{"type":"git_mergeability_check","status":"PASSED","reason":"clean","required":true,"blocking":false}`
	for _, tt := range []struct {
		name, values, mergeable, state string
	}{
		{"clean", base + "," + clean, "MERGEABLE", "CLEAN"},
		{"conflicts", base + `,{"type":"git_mergeability_check","status":"FAILED","reason":"conflicts","required":true,"blocking":true}`, "CONFLICTING", "DIRTY"},
		{"restriction", base + "," + clean + `,{"type":"standard_merge_check","status":"FAILED","required":true,"blocking":true,"check":{"kind":"minimum_approvals","threshold":2}}`, "MERGEABLE", "BLOCKED"},
		{"optional failure", base + "," + clean + `,{"type":"standard_merge_check","status":"FAILED","required":false,"blocking":false}`, "MERGEABLE", "CLEAN"},
		{"git timeout", base + `,{"type":"git_mergeability_check","status":"UNKNOWN","required":true,"blocking":false}`, "UNKNOWN", "UNKNOWN"},
		{"impossible", base + `,{"type":"git_mergeability_check","status":"FAILED","reason":"merge_impossible","required":true,"blocking":true}`, "UNKNOWN", "BLOCKED"},
		{"missing git", base, "UNKNOWN", "UNKNOWN"},
		{"empty", "", "UNKNOWN", "UNKNOWN"},
		{"missing permissions", clean, "MERGEABLE", "UNKNOWN"},
		{"missing flags", base + `,{"type":"git_mergeability_check","status":"PASSED","reason":"clean"}`, "MERGEABLE", "UNKNOWN"},
		{"unknown restriction", base + "," + clean + `,{"type":"standard_merge_check","status":"UNKNOWN","required":true,"blocking":false}`, "MERGEABLE", "UNKNOWN"},
		{"pending check", base + "," + clean + `,{"type":"custom_pre_merge_check","status":"PENDING","required":true,"blocking":true}`, "MERGEABLE", "BLOCKED"},
		{"inconsistent git", base + `,{"type":"git_mergeability_check","status":"UNKNOWN","reason":"clean","required":true,"blocking":false}`, "UNKNOWN", "UNKNOWN"},
		{"duplicate git", base + "," + clean + "," + clean, "UNKNOWN", "UNKNOWN"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/repositories/ws/repo/pullrequests/42/mergeability/checks" {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"values":[` + tt.values + `]}`))
			}))
			got, err := client.GetPullRequestMergeability(context.Background(), "ws", "repo", 42)
			if err != nil {
				t.Fatal(err)
			}
			if got.Mergeable != tt.mergeable || got.MergeStateStatus != tt.state {
				t.Fatalf("got %s/%s, want %s/%s", got.Mergeable, got.MergeStateStatus, tt.mergeable, tt.state)
			}
			encoded, err := json.Marshal(got.Checks)
			if err != nil {
				t.Fatal(err)
			}
			if tt.name == "restriction" && !strings.Contains(string(encoded), `"threshold":2`) {
				t.Fatalf("lost restriction details: %s", encoded)
			}
		})
	}
}

func TestPullRequestMergeabilityErrorsRemainUnknown(t *testing.T) {
	for _, body := range []string{"forbidden", `{"values":`, `{}`, `{"values":null}`, `{"values":[42]}`} {
		t.Run(body, func(t *testing.T) {
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if body == "forbidden" {
					w.WriteHeader(http.StatusForbidden)
				}
				_, _ = w.Write([]byte(body))
			}))
			got, err := client.GetPullRequestMergeability(context.Background(), "ws", "repo", 42)
			if err == nil || got.Mergeable != "UNKNOWN" || got.MergeStateStatus != "UNKNOWN" {
				t.Fatalf("got %+v, error %v; want UNKNOWN and an error", got, err)
			}
		})
	}
}

func TestPullRequestMergeabilityRejectsInvalidTarget(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("invalid target must not make a request")
	}))
	for _, target := range []struct {
		workspace, repo string
		id              int
	}{{"", "repo", 1}, {"ws", "", 1}, {"ws", "repo", 0}} {
		if _, err := client.GetPullRequestMergeability(context.Background(), target.workspace, target.repo, target.id); err == nil {
			t.Fatal("expected validation error")
		}
	}
}
