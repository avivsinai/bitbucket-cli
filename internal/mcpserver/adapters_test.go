package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

func TestRepositoryAdapters(t *testing.T) {
	t.Run("data center", func(t *testing.T) {
		var raw bbdc.Repository
		mustUnmarshal(t, `{
			"slug":"api","name":"API","defaultBranch":"main",
			"project":{"key":"PROJ","public":false},
			"links":{"web":[{"href":"https://dc.example/projects/PROJ/repos/api?token=redacted"}]}
		}`, &raw)

		got := adaptDCRepository(raw)
		want := Repository{
			Scope: "PROJ", Slug: "api", Name: "API", DefaultBranch: "main",
			IsPrivate: true, URL: "https://dc.example/projects/PROJ/repos/api",
		}
		if got != want {
			t.Fatalf("adaptDCRepository() = %+v, want %+v", got, want)
		}
	})

	t.Run("cloud", func(t *testing.T) {
		var raw bbcloud.Repository
		mustUnmarshal(t, `{
			"slug":"api","name":"API","is_private":true,
			"workspace":{"slug":"team"},"mainbranch":{"name":"trunk"},
			"links":{"html":{"href":"https://bitbucket.org/team/api?secret=nope"}}
		}`, &raw)

		got := adaptCloudRepository(raw)
		want := Repository{
			Scope: "team", Slug: "api", Name: "API", DefaultBranch: "trunk",
			IsPrivate: true, URL: "https://bitbucket.org/team/api",
		}
		if got != want {
			t.Fatalf("adaptCloudRepository() = %+v, want %+v", got, want)
		}
	})
}

func TestDCPullRequestAdapter(t *testing.T) {
	longDescription := strings.Repeat("x", PullRequestDescriptionLimit+1)
	var raw bbdc.PullRequest
	mustUnmarshal(t, `{
		"id":42,"title":"Ship it","description":`+mustJSONString(t, longDescription)+`,"state":"OPEN",
		"createdDate":1704067200000,"updatedDate":1704067200123,
		"author":{"user":{"name":"alice","displayName":"Alice A"}},
		"fromRef":{"displayId":"feature","repository":{"slug":"fork","project":{"key":"FORK"}}},
		"toRef":{"displayId":"main","repository":{"slug":"api","project":{"key":"PROJ"}}},
		"reviewers":[
			{"user":{"name":"bob","displayName":"Bob B"},"approved":true,"role":"REVIEWER","status":"APPROVED"},
			{"user":{"name":"carol","displayName":"Carol C"},"approved":false,"role":"REVIEWER","status":"UNAPPROVED"}
		],
		"links":{"self":[{"href":"https://dc.example/pr/42?auth=drop"}]}
	}`, &raw)

	got, err := adaptDCPullRequest(raw, true)
	if err != nil {
		t.Fatalf("adaptDCPullRequest: %v", err)
	}
	if got.ID != 42 || got.Title != "Ship it" || got.State != "OPEN" {
		t.Fatalf("identity fields = %+v", got)
	}
	if got.Author != (User{Name: "alice", DisplayName: "Alice A"}) {
		t.Fatalf("author = %+v", got.Author)
	}
	if got.SourceBranch != "feature" || got.TargetBranch != "main" || got.Repo != (RepositoryRef{Scope: "PROJ", Slug: "api"}) {
		t.Fatalf("refs/repo = %q/%q %+v", got.SourceBranch, got.TargetBranch, got.Repo)
	}
	if got.CreatedAt != "2024-01-01T00:00:00Z" || got.UpdatedAt != "2024-01-01T00:00:00.123Z" {
		t.Fatalf("timestamps = %q/%q", got.CreatedAt, got.UpdatedAt)
	}
	if got.URL != "https://dc.example/pr/42" {
		t.Fatalf("URL = %q, want query-free URL", got.URL)
	}
	if len(got.Reviewers) != 2 || !got.Reviewers[0].Approved || got.Reviewers[1].Approved {
		t.Fatalf("reviewers = %+v", got.Reviewers)
	}
	assertBounded(t, got.Description, PullRequestDescriptionLimit, len(longDescription))

	list, err := adaptDCPullRequest(raw, false)
	if err != nil {
		t.Fatalf("adaptDCPullRequest list variant: %v", err)
	}
	if list.Description != nil {
		t.Fatalf("list variant description = %+v, want omitted", list.Description)
	}
}

func TestDCPullRequestAdapterRejectsMissingApproval(t *testing.T) {
	var raw bbdc.PullRequest
	mustUnmarshal(t, `{
		"createdDate":1704067200000,"updatedDate":1704067200000,
		"reviewers":[{"user":{"name":"alice"}}]
	}`, &raw)
	if _, err := adaptDCPullRequest(raw, false); err == nil || !strings.Contains(err.Error(), "approval") {
		t.Fatalf("error = %v, want missing-approval error", err)
	}
}

func TestCloudPullRequestAdapter(t *testing.T) {
	longDescription := strings.Repeat("€", PullRequestDescriptionLimit/3+1)
	var raw bbcloud.PullRequest
	mustUnmarshal(t, `{
		"id":7,"title":"Cloud PR","description":`+mustJSONString(t, longDescription)+`,"state":"OPEN",
		"created_on":"2024-01-01T02:30:00+02:00","updated_on":"2024-01-01T00:00:00.123456+00:00",
		"author":{"nickname":"alice","display_name":"Alice A"},
		"source":{"branch":{"name":"feature"},"repository":{"slug":"fork","full_name":"other/fork"}},
		"destination":{"branch":{"name":"main"},"repository":{"slug":"api","full_name":"team/api"}},
		"reviewers":[
			{"uuid":"{bob}","nickname":"bob","display_name":"Bob B"},
			{"account_id":"carol-id","nickname":"carol","display_name":"Carol C"}
		],
		"participants":[
			{"user":{"uuid":"{bob}"},"role":"REVIEWER","approved":true,"state":"approved"},
			{"user":{"account_id":"carol-id"},"role":"REVIEWER","approved":false,"state":"unapproved"}
		],
		"links":{"html":{"href":"https://bitbucket.org/team/api/pull-requests/7?token=drop"}}
	}`, &raw)

	got, err := adaptCloudPullRequest(raw, true)
	if err != nil {
		t.Fatalf("adaptCloudPullRequest: %v", err)
	}
	if got.Author != (User{Name: "alice", DisplayName: "Alice A"}) {
		t.Fatalf("author = %+v", got.Author)
	}
	if got.Repo != (RepositoryRef{Scope: "team", Slug: "api"}) {
		t.Fatalf("repo = %+v", got.Repo)
	}
	if got.CreatedAt != "2024-01-01T00:30:00Z" || got.UpdatedAt != "2024-01-01T00:00:00.123456Z" {
		t.Fatalf("timestamps = %q/%q", got.CreatedAt, got.UpdatedAt)
	}
	if got.URL != "https://bitbucket.org/team/api/pull-requests/7" {
		t.Fatalf("URL = %q", got.URL)
	}
	if len(got.Reviewers) != 2 || !got.Reviewers[0].Approved || got.Reviewers[1].Approved {
		t.Fatalf("reviewers = %+v", got.Reviewers)
	}
	assertBounded(t, got.Description, PullRequestDescriptionLimit, len(longDescription))
}

func TestCloudPullRequestAdapterErrors(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "invalid timestamp",
			raw:  `{"created_on":"not-a-time","updated_on":"2024-01-01T00:00:00Z"}`,
			want: "created_on",
		},
		{
			name: "matched participant missing approval",
			raw:  `{"created_on":"2024-01-01T00:00:00Z","updated_on":"2024-01-01T00:00:00Z","reviewers":[{"uuid":"{bob}"}],"participants":[{"user":{"uuid":"{bob}"}}]}`,
			want: "approval",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw bbcloud.PullRequest
			mustUnmarshal(t, tt.raw, &raw)
			if _, err := adaptCloudPullRequest(raw, false); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestCloudPullRequestAdapterTreatsReviewerMissingFromParticipantsAsUnapproved(t *testing.T) {
	var raw bbcloud.PullRequest
	mustUnmarshal(t, `{
		"created_on":"2024-01-01T00:00:00Z","updated_on":"2024-01-01T00:00:00Z",
		"reviewers":[{"uuid":"{bob}","nickname":"bob","display_name":"Bob B"}]
	}`, &raw)

	got, err := adaptCloudPullRequest(raw, false)
	if err != nil {
		t.Fatalf("adaptCloudPullRequest: %v", err)
	}
	if len(got.Reviewers) != 1 || got.Reviewers[0] != (Reviewer{Name: "bob", DisplayName: "Bob B", Approved: false}) {
		t.Fatalf("reviewers = %+v, want bob unapproved", got.Reviewers)
	}
}

func TestCommentAdapters(t *testing.T) {
	t.Run("data center", func(t *testing.T) {
		body := strings.Repeat("d", CommentBodyLimit+1)
		var raw bbdc.PullRequestComment
		mustUnmarshal(t, `{
			"id":9,"text":`+mustJSONString(t, body)+`,"createdDate":1704067200123,
			"author":{"name":"alice","displayName":"Alice A"},
			"parent":{"id":7},"anchor":{"path":"src/main.go","line":12}
		}`, &raw)

		got, err := adaptDCComment(raw)
		if err != nil {
			t.Fatalf("adaptDCComment: %v", err)
		}
		if got.CreatedAt != "2024-01-01T00:00:00.123Z" || got.ParentID == nil || *got.ParentID != 7 {
			t.Fatalf("date/parent = %q/%v", got.CreatedAt, got.ParentID)
		}
		if got.Path != "src/main.go" || got.Line == nil || *got.Line != 12 {
			t.Fatalf("inline = %q/%v", got.Path, got.Line)
		}
		assertTextBounded(t, got.Body, CommentBodyLimit, len(body))
	})

	t.Run("data center omits absent created date", func(t *testing.T) {
		got, err := adaptDCComment(bbdc.PullRequestComment{})
		if err != nil {
			t.Fatal(err)
		}
		if got.CreatedAt != "" {
			t.Fatalf("created_at = %q, want omitted", got.CreatedAt)
		}
	})

	t.Run("cloud", func(t *testing.T) {
		var raw bbcloud.PullRequestComment
		mustUnmarshal(t, `{
			"id":5,"content":{"raw":"hello"},
			"user":{"nickname":"alice","display_name":"Alice A"},
			"created_on":"2024-01-01T02:00:00+02:00",
			"parent":{"id":3},"inline":{"path":"README.md","from":8,"to":9}
		}`, &raw)

		got, err := adaptCloudComment(raw)
		if err != nil {
			t.Fatalf("adaptCloudComment: %v", err)
		}
		if got.CreatedAt != "2024-01-01T00:00:00Z" || got.ParentID == nil || *got.ParentID != 3 {
			t.Fatalf("date/parent = %q/%v", got.CreatedAt, got.ParentID)
		}
		if got.Path != "README.md" || got.Line == nil || *got.Line != 9 {
			t.Fatalf("inline = %q/%v, want destination line 9", got.Path, got.Line)
		}
	})
}

func TestCheckStateMappings(t *testing.T) {
	dc := []struct {
		raw  string
		want CheckState
	}{
		{"INPROGRESS", CheckRunning},
		{"SUCCESSFUL", CheckSuccessful},
		{"FAILED", CheckFailed},
		{"CANCELLED", CheckStopped},
		{"UNKNOWN", CheckUnknown},
		{"new-state", CheckUnknown},
	}
	for _, tt := range dc {
		t.Run("dc "+tt.raw, func(t *testing.T) {
			got := adaptDCCheck(types.CommitStatus{State: tt.raw, Key: "build", Name: "CI", URL: "https://ci.example/run?token=drop"})
			if got.State != tt.want || got.URL != "https://ci.example/run" {
				t.Fatalf("adaptDCCheck(%q) = %+v, want state %q and query-free URL", tt.raw, got, tt.want)
			}
		})
	}

	cloud := []struct {
		raw  string
		want CheckState
	}{
		{"PENDING", CheckPending},
		{"IN_PROGRESS", CheckRunning},
		{"INPROGRESS", CheckRunning},
		{"RUNNING", CheckRunning},
		{"SUCCESSFUL", CheckSuccessful},
		{"COMPLETED SUCCESSFUL", CheckSuccessful},
		{"FAILED", CheckFailed},
		{"ERROR", CheckFailed},
		{"COMPLETED FAILED", CheckFailed},
		{"COMPLETED ERROR", CheckFailed},
		{"STOPPED", CheckStopped},
		{"CANCELLED", CheckStopped},
		{"COMPLETED STOPPED", CheckStopped},
		{"COMPLETED", CheckUnknown},
		{"new-state", CheckUnknown},
	}
	for _, tt := range cloud {
		t.Run("cloud "+tt.raw, func(t *testing.T) {
			got := adaptCloudCheck(types.CommitStatus{State: tt.raw})
			if got.State != tt.want {
				t.Fatalf("adaptCloudCheck(%q).State = %q, want %q", tt.raw, got.State, tt.want)
			}
		})
	}
}

func TestDiffAdapterBoundsContent(t *testing.T) {
	content := strings.Repeat("x", DiffContentLimit) + "€"
	got := adaptDiff(content, "source-sha", "target-sha")
	if got.SourceCommit != "source-sha" || got.TargetCommit != "target-sha" {
		t.Fatalf("commits = %q/%q", got.SourceCommit, got.TargetCommit)
	}
	assertTextBounded(t, got.Content, DiffContentLimit, len(content))
}

func mustUnmarshal(t *testing.T, raw string, out any) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
}

func mustJSONString(t *testing.T, value string) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func assertBounded(t *testing.T, got *BoundedText, limit, originalSize int) {
	t.Helper()
	if got == nil {
		t.Fatal("bounded text is nil")
		return
	}
	assertTextBounded(t, *got, limit, originalSize)
}

func assertTextBounded(t *testing.T, got BoundedText, limit, originalSize int) {
	t.Helper()
	if !got.Truncated || got.OriginalSize == nil || *got.OriginalSize != originalSize {
		t.Fatalf("bounded text metadata = %+v, want truncated with original_size %d", got, originalSize)
	}
	if len(got.Text) > limit {
		t.Fatalf("bounded text length = %d, exceeds %d", len(got.Text), limit)
	}
	if got.Provenance.Source != ProvenanceSourceBitbucket || got.Provenance.Trust != ProvenanceTrustUntrusted {
		t.Fatalf("provenance = %+v", got.Provenance)
	}
}
