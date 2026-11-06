package bbcloud_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestWorkspacePullRequestsPageFiltersStateWhenAPIIgnoresIt(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  []int
	}{
		{name: "open", state: "OPEN", want: []int{2, 4}},
		{name: "normalized open", state: " open ", want: []int{2, 4}},
		{name: "merged", state: "MERGED", want: []int{1}},
		{name: "declined", state: "DECLINED", want: []int{3}},
		{name: "no matches", state: "SUPERSEDED"},
		{name: "all", state: "all", want: []int{1, 2, 3, 4}},
		{name: "unspecified", want: []int{1, 2, 3, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"values": []bbcloud.PullRequest{
						{ID: 1, State: "MERGED"},
						{ID: 2, State: "OPEN"},
						{ID: 3, State: "DECLINED"},
						{ID: 4, State: "OPEN"},
					},
					"next": "/workspaces/ws/pullrequests/alice?page=2",
				})
			}))

			for _, next := range []string{"", "/workspaces/ws/pullrequests/alice?page=2"} {
				page, err := client.ListWorkspacePullRequestsPage(context.Background(), "ws", "alice", bbcloud.WorkspacePullRequestsOptions{State: tt.state}, next)
				if err != nil {
					t.Fatal(err)
				}
				var ids []int
				for _, pr := range page.Values {
					ids = append(ids, pr.ID)
				}
				if !reflect.DeepEqual(ids, tt.want) {
					t.Errorf("next=%q: IDs = %v, want %v", next, ids, tt.want)
				}
				if page.Next != "/workspaces/ws/pullrequests/alice?page=2" {
					t.Errorf("Next = %q, want the server's next reference", page.Next)
				}
			}
		})
	}
}

func TestWorkspacePullRequestsFiltersBeforeLimitAcrossPages(t *testing.T) {
	for _, limit := range []int{0, 2} {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			var pages []string
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page := r.URL.Query().Get("page")
				pages = append(pages, page)
				w.Header().Set("Content-Type", "application/json")
				switch page {
				case "":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"values": []bbcloud.PullRequest{{ID: 1, State: "MERGED"}, {ID: 2, State: "DECLINED"}},
						"next":   "/workspaces/ws/pullrequests/alice?page=2",
					})
				case "2":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"values": []bbcloud.PullRequest{{ID: 3, State: "OPEN"}, {ID: 4, State: "MERGED"}},
						"next":   "/workspaces/ws/pullrequests/alice?page=3",
					})
				case "3":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"values": []bbcloud.PullRequest{{ID: 5, State: "OPEN"}, {ID: 6, State: "OPEN"}},
					})
				default:
					t.Errorf("unexpected page %q", page)
					http.NotFound(w, r)
				}
			}))

			prs, err := client.ListWorkspacePullRequests(context.Background(), "ws", "alice", bbcloud.WorkspacePullRequestsOptions{State: "OPEN", Limit: limit})
			if err != nil {
				t.Fatal(err)
			}
			var ids []int
			for _, pr := range prs {
				ids = append(ids, pr.ID)
			}
			want := []int{3, 5, 6}
			if limit > 0 {
				want = want[:limit]
			}
			if !reflect.DeepEqual(ids, want) {
				t.Errorf("IDs = %v, want %v", ids, want)
			}
			if !reflect.DeepEqual(pages, []string{"", "2", "3"}) {
				t.Errorf("requested pages = %v, want all three pages", pages)
			}
		})
	}
}
