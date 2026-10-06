package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestPullRequestCommentsActivityAnchors(t *testing.T) {
	readers := []struct {
		name string
		read func(*bbdc.Client) ([]bbdc.PullRequestComment, error)
	}{
		{"all pages", func(c *bbdc.Client) ([]bbdc.PullRequestComment, error) {
			return c.ListPullRequestComments(context.Background(), "PROJ", "repo", 42)
		}},
		{"single page", func(c *bbdc.Client) ([]bbdc.PullRequestComment, error) {
			page, err := c.ListPullRequestCommentsPage(context.Background(), "PROJ", "repo", 42, 100, 0)
			if err != nil {
				return nil, err
			}
			return page.Values, nil
		}},
	}
	for _, reader := range readers {
		t.Run(reader.name, func(t *testing.T) {
			for _, tc := range []struct {
				name         string
				activityPath any
				nestedPath   string
				wantPath     string
				orphaned     bool
			}{
				{name: "sibling anchor", activityPath: "src/main.go", wantPath: "src/main.go"},
				{name: "orphaned anchor", activityPath: "src/old.go", wantPath: "src/old.go", orphaned: true},
				{name: "structured path", activityPath: map[string]any{"parent": "src", "name": "main.go"}, wantPath: "src/main.go"},
				{name: "nested anchor preserved", activityPath: "other.go", nestedPath: "nested.go", wantPath: "nested.go"},
				{name: "general comment"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					comment := map[string]any{
						"id": 10, "text": "root",
						"comments": []map[string]any{{"id": 11, "text": "reply"}},
					}
					anchor := func(path any) map[string]any {
						return map[string]any{"path": path, "line": 25, "lineType": "ADDED", "fileType": "TO", "orphaned": tc.orphaned}
					}
					activity := map[string]any{"action": "COMMENTED", "comment": comment}
					if tc.activityPath != nil {
						activity["commentAnchor"] = anchor(tc.activityPath)
					}
					if tc.nestedPath != "" {
						comment["anchor"] = anchor(tc.nestedPath)
					}
					client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{
							"isLastPage": true,
							"values":     []map[string]any{{"action": "APPROVED"}, activity},
						})
					}))
					comments, err := reader.read(client)
					if err != nil {
						t.Fatal(err)
					}
					if len(comments) != 2 || comments[0].ID != 10 || comments[1].ID != 11 || comments[1].Depth != 1 {
						t.Fatalf("unexpected flattened comments: %+v", comments)
					}
					if tc.wantPath == "" {
						if comments[0].Anchor != nil {
							t.Fatalf("general comment acquired anchor: %+v", comments[0].Anchor)
						}
						return
					}
					a := comments[0].Anchor
					if a == nil || a.Path != tc.wantPath || a.Line != 25 || a.LineType != "ADDED" || a.FileType != "TO" {
						t.Fatalf("anchor = %+v, want %s:25 ADDED TO", a, tc.wantPath)
					}
					encoded, err := json.Marshal(a)
					if err != nil {
						t.Fatal(err)
					}
					var fields map[string]any
					if err := json.Unmarshal(encoded, &fields); err != nil {
						t.Fatal(err)
					}
					if fields["orphaned"] != tc.orphaned {
						t.Fatalf("orphaned = %v, want %v in JSON %s", fields["orphaned"], tc.orphaned, encoded)
					}
				})
			}
		})
	}
}
