package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestCreatePullRequestRepositoryMapping(t *testing.T) {
	tests := []struct {
		name              string
		opts              bbdc.CreatePROptions
		wantSourceProject string
		wantSourceRepo    string
	}{
		{
			name: "defaults source repository to destination",
			opts: bbdc.CreatePROptions{
				Title:        "Same repository",
				SourceBranch: "feature",
				TargetBranch: "main",
			},
			wantSourceProject: "DEST",
			wantSourceRepo:    "upstream",
		},
		{
			name: "uses independent source repository",
			opts: bbdc.CreatePROptions{
				Title:            "Fork pull request",
				SourceBranch:     "feature",
				TargetBranch:     "main",
				SourceProjectKey: "fork",
				SourceRepoSlug:   "contributor-fork",
			},
			wantSourceProject: "FORK",
			wantSourceRepo:    "contributor-fork",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/rest/api/1.0/projects/dest/repos/upstream/pull-requests" {
					t.Fatalf("path = %q", r.URL.Path)
				}
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "title": tt.opts.Title})
			}))

			if _, err := client.CreatePullRequest(context.Background(), "dest", "upstream", tt.opts); err != nil {
				t.Fatalf("CreatePullRequest: %v", err)
			}

			assertRequestRepository(t, gotBody, "fromRef", tt.wantSourceProject, tt.wantSourceRepo)
			assertRequestRepository(t, gotBody, "toRef", "DEST", "upstream")
		})
	}
}

func TestGetDefaultReviewersForRepositoriesUsesBothRepositoryIDs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/1.0/projects/DEST/repos/upstream":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202, "slug": "upstream"})
		case "/rest/api/1.0/projects/FORK/repos/contributor-fork":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 101, "slug": "contributor-fork"})
		case "/rest/default-reviewers/1.0/projects/DEST/repos/upstream/reviewers":
			query := r.URL.Query()
			if got := query.Get("sourceRepoId"); got != "101" {
				t.Fatalf("sourceRepoId = %q, want 101", got)
			}
			if got := query.Get("targetRepoId"); got != "202" {
				t.Fatalf("targetRepoId = %q, want 202", got)
			}
			if got := query.Get("sourceRefId"); got != "feature" {
				t.Fatalf("sourceRefId = %q, want feature", got)
			}
			if got := query.Get("targetRefId"); got != "main" {
				t.Fatalf("targetRefId = %q, want main", got)
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "alice"}})
		default:
			http.NotFound(w, r)
		}
	}))

	reviewers, err := client.GetDefaultReviewersForRepositories(
		context.Background(),
		"DEST", "upstream",
		"FORK", "contributor-fork",
		"feature", "main",
	)
	if err != nil {
		t.Fatalf("GetDefaultReviewersForRepositories: %v", err)
	}
	if len(reviewers) != 1 || reviewers[0].Name != "alice" {
		t.Fatalf("reviewers = %#v, want alice", reviewers)
	}
}

func assertRequestRepository(t *testing.T, body map[string]any, ref, wantProject, wantRepo string) {
	t.Helper()
	refBody, ok := body[ref].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", ref, body[ref])
	}
	repository, ok := refBody["repository"].(map[string]any)
	if !ok {
		t.Fatalf("%s.repository = %#v, want object", ref, refBody["repository"])
	}
	if got := repository["slug"]; got != wantRepo {
		t.Errorf("%s.repository.slug = %v, want %q", ref, got, wantRepo)
	}
	project, ok := repository["project"].(map[string]any)
	if !ok {
		t.Fatalf("%s.repository.project = %#v, want object", ref, repository["project"])
	}
	if got := project["key"]; got != wantProject {
		t.Errorf("%s.repository.project.key = %v, want %q", ref, got, wantProject)
	}
}
