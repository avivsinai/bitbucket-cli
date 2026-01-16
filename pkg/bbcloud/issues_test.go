package bbcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestListIssuesBBQLQueryBuilding(t *testing.T) {
	tests := []struct {
		name           string
		opts           IssueListOptions
		wantQueryParts []string // parts that should appear in the q parameter
		wantSort       string   // expected sort parameter
		excludeQuery   []string // parts that should NOT appear
	}{
		{
			name: "filter by state",
			opts: IssueListOptions{
				State: "open",
			},
			wantQueryParts: []string{`state = "open"`},
		},
		{
			name: "state all excluded from query",
			opts: IssueListOptions{
				State: "all",
			},
			excludeQuery: []string{"state"},
		},
		{
			name: "state ALL (case insensitive) excluded from query",
			opts: IssueListOptions{
				State: "ALL",
			},
			excludeQuery: []string{"state"},
		},
		{
			name: "filter by assignee UUID",
			opts: IssueListOptions{
				Assignee: "{abc-123-def}",
			},
			wantQueryParts: []string{`assignee.uuid = "{abc-123-def}"`},
		},
		{
			name: "filter by reporter UUID",
			opts: IssueListOptions{
				Reporter: "{xyz-789}",
			},
			wantQueryParts: []string{`reporter.uuid = "{xyz-789}"`},
		},
		{
			name: "filter by kind",
			opts: IssueListOptions{
				Kind: "bug",
			},
			wantQueryParts: []string{`kind = "bug"`},
		},
		{
			name: "filter by priority",
			opts: IssueListOptions{
				Priority: "critical",
			},
			wantQueryParts: []string{`priority = "critical"`},
		},
		{
			name: "filter by milestone",
			opts: IssueListOptions{
				Milestone: "v1.0",
			},
			wantQueryParts: []string{`milestone.name = "v1.0"`},
		},
		{
			name: "multiple filters combined with AND",
			opts: IssueListOptions{
				State:    "open",
				Kind:     "bug",
				Priority: "major",
			},
			wantQueryParts: []string{
				`state = "open"`,
				`kind = "bug"`,
				`priority = "major"`,
				" AND ",
			},
		},
		{
			name: "sort by updated_on descending",
			opts: IssueListOptions{
				Sort: "-updated_on",
			},
			wantSort: "-updated_on",
		},
		{
			name: "custom query passthrough",
			opts: IssueListOptions{
				Query: `title ~ "urgent"`,
			},
			wantQueryParts: []string{`title ~ "urgent"`},
		},
		{
			name: "empty options produces no query",
			opts: IssueListOptions{},
			excludeQuery: []string{"q="},
		},
		{
			name: "whitespace-only values ignored",
			opts: IssueListOptions{
				State:    "  ",
				Assignee: "	",
			},
			excludeQuery: []string{"q="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				w.Header().Set("Content-Type", "application/json")
				resp := issueListPage{Values: []Issue{}}
				_ = json.NewEncoder(w).Encode(resp)
			}))
			t.Cleanup(server.Close)

			client, err := New(Options{BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx := context.Background()
			_, err = client.ListIssues(ctx, "workspace", "repo", tt.opts)
			if err != nil {
				t.Fatalf("ListIssues: %v", err)
			}

			// Decode the captured URL to check query parameters
			parsedURL, err := url.Parse(capturedURL)
			if err != nil {
				t.Fatalf("failed to parse captured URL: %v", err)
			}

			rawQuery, _ := url.QueryUnescape(parsedURL.RawQuery)

			// Check that expected query parts are present
			for _, part := range tt.wantQueryParts {
				if !strings.Contains(rawQuery, part) {
					t.Errorf("expected query to contain %q, got %q", part, rawQuery)
				}
			}

			// Check that excluded parts are NOT present
			for _, part := range tt.excludeQuery {
				if strings.Contains(rawQuery, part) {
					t.Errorf("expected query to NOT contain %q, got %q", part, rawQuery)
				}
			}

			// Check sort parameter
			if tt.wantSort != "" {
				sortParam := parsedURL.Query().Get("sort")
				if sortParam != tt.wantSort {
					t.Errorf("expected sort=%q, got sort=%q", tt.wantSort, sortParam)
				}
			}
		})
	}
}

func TestListIssuesValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		errorContains string
	}{
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "repo",
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing repo slug",
			workspace:     "workspace",
			repoSlug:      "",
			errorContains: "workspace and repository slug are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ListIssues(ctx, tt.workspace, tt.repoSlug, IssueListOptions{})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

func TestListIssuesPagination(t *testing.T) {
	var requestCount int
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch requestCount {
		case 1:
			resp := issueListPage{
				Values: []Issue{{ID: 1, Title: "Issue 1"}, {ID: 2, Title: "Issue 2"}},
				Next:   serverURL + "/repositories/ws/repo/issues?page=2",
			}
			_ = json.NewEncoder(w).Encode(resp)
		case 2:
			resp := issueListPage{
				Values: []Issue{{ID: 3, Title: "Issue 3"}},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	issues, err := client.ListIssues(ctx, "ws", "repo", IssueListOptions{})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}

	if len(issues) != 3 {
		t.Errorf("expected 3 issues, got %d", len(issues))
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests for pagination, got %d", requestCount)
	}
}

func TestListIssuesRespectsLimit(t *testing.T) {
	var requestCount int
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		resp := issueListPage{
			Values: []Issue{{ID: 1}, {ID: 2}, {ID: 3}},
			Next:   serverURL + "/repositories/ws/repo/issues?page=2",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	issues, err := client.ListIssues(ctx, "ws", "repo", IssueListOptions{Limit: 2})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("expected 2 issues (limit), got %d", len(issues))
	}
	if requestCount != 1 {
		t.Errorf("expected 1 request (limit satisfied), got %d", requestCount)
	}
}
