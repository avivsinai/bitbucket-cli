package bbcloud_test

import (
	"encoding/json"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestPullRequestPreservesDescriptionAndParticipantApproval(t *testing.T) {
	var pr bbcloud.PullRequest
	err := json.Unmarshal([]byte(`{
		"description": "full description",
		"author": {"nickname": "author-nick"},
		"reviewers": [{"uuid": "{reviewer}", "nickname": "reviewer-nick"}],
		"participants": [{
			"user": {"uuid": "{reviewer}", "nickname": "reviewer-nick"},
			"role": "REVIEWER",
			"approved": true,
			"state": "approved"
		}]
	}`), &pr)
	if err != nil {
		t.Fatal(err)
	}
	if pr.Description != "full description" || pr.AuthorNickname != "author-nick" {
		t.Fatalf("description/author nickname = %q/%q", pr.Description, pr.AuthorNickname)
	}
	if len(pr.Reviewers) != 1 || pr.Reviewers[0].Nickname != "reviewer-nick" {
		t.Fatalf("reviewers = %+v, want retained nickname", pr.Reviewers)
	}
	if len(pr.Participants) != 1 {
		t.Fatalf("participants = %d, want 1", len(pr.Participants))
	}
	participant := pr.Participants[0]
	if participant.Role != "REVIEWER" || participant.State != "approved" {
		t.Fatalf("participant role/state = %q/%q", participant.Role, participant.State)
	}
	if participant.Approved == nil || !*participant.Approved {
		t.Fatalf("participant approved = %v, want explicit true", participant.Approved)
	}
}

func TestRepositoryPreservesMainBranch(t *testing.T) {
	var repo bbcloud.Repository
	err := json.Unmarshal([]byte(`{
		"slug": "api",
		"mainbranch": {"name": "main"}
	}`), &repo)
	if err != nil {
		t.Fatal(err)
	}
	if repo.MainBranch.Name != "main" {
		t.Fatalf("main branch = %q, want main", repo.MainBranch.Name)
	}
}
