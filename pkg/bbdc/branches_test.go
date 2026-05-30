package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestCreateBranchStartPointNormalization(t *testing.T) {
	tests := []struct {
		name           string
		startPoint     string
		wantStartPoint string
	}{
		{
			name:           "commit sha",
			startPoint:     "0123456789abcdef0123456789abcdef01234567",
			wantStartPoint: "0123456789abcdef0123456789abcdef01234567",
		},
		{
			name:           "branch name",
			startPoint:     "main",
			wantStartPoint: "refs/heads/main",
		},
		{
			name:           "non-branch ref",
			startPoint:     "refs/changes/123",
			wantStartPoint: "refs/changes/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want POST", r.Method)
				}
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":        "refs/heads/feature",
					"displayId": "feature",
				})
			}))
			t.Cleanup(server.Close)

			client, err := bbdc.New(bbdc.Options{BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			_, err = client.CreateBranch(context.Background(), "PROJ", "repo", bbdc.CreateBranchInput{
				Name:       "feature",
				StartPoint: tt.startPoint,
			})
			if err != nil {
				t.Fatalf("CreateBranch: %v", err)
			}

			if got := gotBody["name"]; got != "refs/heads/feature" {
				t.Errorf("name = %v, want refs/heads/feature", got)
			}
			if got := gotBody["startPoint"]; got != tt.wantStartPoint {
				t.Errorf("startPoint = %v, want %s", got, tt.wantStartPoint)
			}
		})
	}
}
