package bbdc_test

import (
	"encoding/json"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestPullRequestReviewerPreservesApprovalFields(t *testing.T) {
	var pr bbdc.PullRequest
	err := json.Unmarshal([]byte(`{
		"reviewers": [{
			"user": {"name": "alice", "displayName": "Alice A"},
			"role": "REVIEWER",
			"approved": true,
			"status": "APPROVED"
		}]
	}`), &pr)
	if err != nil {
		t.Fatal(err)
	}
	if len(pr.Reviewers) != 1 {
		t.Fatalf("reviewers = %d, want 1", len(pr.Reviewers))
	}
	reviewer := pr.Reviewers[0]
	if reviewer.Role != "REVIEWER" || reviewer.Status != "APPROVED" {
		t.Fatalf("reviewer role/status = %q/%q, want REVIEWER/APPROVED", reviewer.Role, reviewer.Status)
	}
	if reviewer.Approved == nil || !*reviewer.Approved {
		t.Fatalf("reviewer approved = %v, want explicit true", reviewer.Approved)
	}
}

func TestPullRequestCommentPreservesCreatedDateAndParent(t *testing.T) {
	var comment bbdc.PullRequestComment
	err := json.Unmarshal([]byte(`{
		"id": 9,
		"text": "reply",
		"createdDate": 1720951200123,
		"parent": {"id": 7}
	}`), &comment)
	if err != nil {
		t.Fatal(err)
	}
	if comment.CreatedDate == nil || *comment.CreatedDate != 1720951200123 {
		t.Fatalf("created date = %v, want 1720951200123", comment.CreatedDate)
	}
	if comment.Parent == nil || comment.Parent.ID != 7 {
		t.Fatalf("parent = %+v, want id 7", comment.Parent)
	}
}
