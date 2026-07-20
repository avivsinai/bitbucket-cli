package mcpserver

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

func adaptDCRepository(raw bbdc.Repository) Repository {
	scope := ""
	isPrivate := true
	if raw.Project != nil {
		scope = raw.Project.Key
		isPrivate = !raw.Project.Public
	}

	return Repository{
		Scope:         scope,
		Slug:          raw.Slug,
		Name:          raw.Name,
		DefaultBranch: raw.DefaultBranch,
		IsPrivate:     isPrivate,
		URL:           stripURLQuery(firstDCLink(raw.Links.Web, raw.Links.Self)),
	}
}

func adaptCloudRepository(raw bbcloud.Repository) Repository {
	return Repository{
		Scope:         raw.Workspace.Slug,
		Slug:          raw.Slug,
		Name:          raw.Name,
		DefaultBranch: raw.MainBranch.Name,
		IsPrivate:     raw.IsPrivate,
		URL:           stripURLQuery(raw.Links.HTML.Href),
	}
}

func adaptDCPullRequest(raw bbdc.PullRequest, full bool) (PullRequest, error) {
	createdAt, err := formatDCTimestamp("createdDate", raw.CreatedDate)
	if err != nil {
		return PullRequest{}, err
	}
	updatedAt, err := formatDCTimestamp("updatedDate", raw.UpdatedDate)
	if err != nil {
		return PullRequest{}, err
	}
	reviewers, err := adaptDCReviewers(raw.Reviewers)
	if err != nil {
		return PullRequest{}, err
	}

	result := PullRequest{
		ID:           raw.ID,
		Title:        raw.Title,
		State:        raw.State,
		Author:       adaptDCUser(raw.Author.User),
		SourceBranch: raw.FromRef.DisplayID,
		TargetBranch: raw.ToRef.DisplayID,
		Repo:         adaptDCRepositoryRef(raw.ToRef.Repository),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		URL:          stripURLQuery(firstDCSelfLink(raw.Links.Self)),
		Reviewers:    reviewers,
	}
	if full {
		description := boundBitbucketText(raw.Description, PullRequestDescriptionLimit)
		result.Description = &description
	}
	return result, nil
}

func adaptCloudPullRequest(raw bbcloud.PullRequest, full bool) (PullRequest, error) {
	createdAt, err := formatCloudTimestamp("created_on", raw.CreatedOn)
	if err != nil {
		return PullRequest{}, err
	}
	updatedAt, err := formatCloudTimestamp("updated_on", raw.UpdatedOn)
	if err != nil {
		return PullRequest{}, err
	}
	reviewers, err := adaptCloudReviewers(raw.Reviewers, raw.Participants)
	if err != nil {
		return PullRequest{}, err
	}

	result := PullRequest{
		ID:    raw.ID,
		Title: raw.Title,
		State: raw.State,
		Author: User{
			Name:        firstNonEmpty(raw.AuthorNickname, raw.Author.Username, raw.Author.AccountID, raw.Author.UUID),
			DisplayName: raw.Author.DisplayName,
		},
		SourceBranch: raw.Source.Branch.Name,
		TargetBranch: raw.Destination.Branch.Name,
		Repo:         adaptCloudRepositoryRef(raw.Destination.Repository),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		URL:          stripURLQuery(raw.Links.HTML.Href),
		Reviewers:    reviewers,
	}
	if full {
		description := boundBitbucketText(raw.Description, PullRequestDescriptionLimit)
		result.Description = &description
	}
	return result, nil
}

func adaptDCComment(raw bbdc.PullRequestComment) (Comment, error) {
	createdAt := ""
	if raw.CreatedDate != nil {
		var err error
		createdAt, err = formatDCTimestamp("createdDate", *raw.CreatedDate)
		if err != nil {
			return Comment{}, err
		}
	}

	result := Comment{
		ID:        raw.ID,
		Author:    adaptDCUser(raw.Author),
		Body:      boundBitbucketText(raw.Text, CommentBodyLimit),
		CreatedAt: createdAt,
	}
	if raw.Parent != nil {
		result.ParentID = intPointer(raw.Parent.ID)
	}
	if raw.Anchor != nil {
		result.Path = raw.Anchor.Path
		result.Line = intPointer(raw.Anchor.Line)
	}
	return result, nil
}

func adaptCloudComment(raw bbcloud.PullRequestComment) (Comment, error) {
	createdAt := ""
	if raw.CreatedOn != "" {
		var err error
		createdAt, err = formatCloudTimestamp("created_on", raw.CreatedOn)
		if err != nil {
			return Comment{}, err
		}
	}

	result := Comment{
		ID:        raw.ID,
		Body:      boundBitbucketText(raw.Content.Raw, CommentBodyLimit),
		CreatedAt: createdAt,
	}
	if raw.User != nil {
		result.Author = User{
			Name:        firstNonEmpty(raw.User.Nickname, raw.User.AccountID, raw.User.UUID),
			DisplayName: raw.User.DisplayName,
		}
	}
	if raw.Parent != nil {
		result.ParentID = intPointer(raw.Parent.ID)
	}
	if raw.Inline != nil {
		result.Path = raw.Inline.Path
		switch {
		case raw.Inline.To != nil:
			result.Line = intPointer(*raw.Inline.To)
		case raw.Inline.From != nil:
			result.Line = intPointer(*raw.Inline.From)
		}
	}
	return result, nil
}

var dcCheckStates = map[string]CheckState{
	"INPROGRESS": CheckRunning,
	"SUCCESSFUL": CheckSuccessful,
	"FAILED":     CheckFailed,
	"CANCELLED":  CheckStopped,
	"UNKNOWN":    CheckUnknown,
}

var cloudCheckStates = map[string]CheckState{
	"PENDING":              CheckPending,
	"IN_PROGRESS":          CheckRunning,
	"INPROGRESS":           CheckRunning,
	"RUNNING":              CheckRunning,
	"SUCCESSFUL":           CheckSuccessful,
	"COMPLETED SUCCESSFUL": CheckSuccessful,
	"FAILED":               CheckFailed,
	"ERROR":                CheckFailed,
	"COMPLETED FAILED":     CheckFailed,
	"COMPLETED ERROR":      CheckFailed,
	"STOPPED":              CheckStopped,
	"CANCELLED":            CheckStopped,
	"COMPLETED STOPPED":    CheckStopped,
}

func adaptDCCheck(raw types.CommitStatus) Check {
	return adaptCheck(raw, stateFromMap(raw.State, dcCheckStates))
}

func adaptCloudCheck(raw types.CommitStatus) Check {
	return adaptCheck(raw, stateFromMap(raw.State, cloudCheckStates))
}

func adaptCheck(raw types.CommitStatus, state CheckState) Check {
	return Check{
		Key:   raw.Key,
		Name:  raw.Name,
		State: state,
		URL:   stripURLQuery(raw.URL),
	}
}

func adaptDiff(content, sourceCommit, targetCommit string) Diff {
	return Diff{
		Content:      boundBitbucketText(content, DiffContentLimit),
		SourceCommit: sourceCommit,
		TargetCommit: targetCommit,
	}
}

func adaptDCReviewers(raw []bbdc.PullRequestReviewer) ([]Reviewer, error) {
	reviewers := make([]Reviewer, 0, len(raw))
	for _, reviewer := range raw {
		approved, ok := approvalState(reviewer.Approved, reviewer.Status)
		if !ok {
			return nil, fmt.Errorf("reviewer %q has no approval state", firstNonEmpty(reviewer.User.Name, reviewer.User.Slug))
		}
		reviewers = append(reviewers, Reviewer{
			Name:        firstNonEmpty(reviewer.User.Name, reviewer.User.Slug),
			DisplayName: reviewer.User.FullName,
			Approved:    approved,
		})
	}
	return reviewers, nil
}

func adaptCloudReviewers(raw []bbcloud.User, participants []bbcloud.PullRequestParticipant) ([]Reviewer, error) {
	reviewers := make([]Reviewer, 0, len(raw))
	for _, reviewer := range raw {
		approved, known, matched := cloudReviewerApproval(reviewer, participants)
		if matched && !known {
			return nil, fmt.Errorf("reviewer %q has no participant approval state", cloudUserName(reviewer))
		}
		reviewers = append(reviewers, Reviewer{
			Name:        cloudUserName(reviewer),
			DisplayName: reviewer.Display,
			Approved:    approved,
		})
	}
	return reviewers, nil
}

func cloudReviewerApproval(reviewer bbcloud.User, participants []bbcloud.PullRequestParticipant) (approved, known, matched bool) {
	for _, participant := range participants {
		if sameCloudUser(reviewer, participant.User) {
			approved, known := approvalState(participant.Approved, participant.State)
			return approved, known, true
		}
	}
	return false, true, false
}

func approvalState(approved *bool, state string) (bool, bool) {
	if approved != nil {
		return *approved, true
	}
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "APPROVED":
		return true, true
	case "UNAPPROVED", "NOT_APPROVED", "CHANGES_REQUESTED", "NEEDS_WORK":
		return false, true
	default:
		return false, false
	}
}

func sameCloudUser(a, b bbcloud.User) bool {
	for _, pair := range [][2]string{
		{a.UUID, b.UUID},
		{a.AccountID, b.AccountID},
		{a.Nickname, b.Nickname},
		{a.Username, b.Username},
	} {
		if pair[0] != "" && pair[1] != "" && pair[0] == pair[1] {
			return true
		}
	}
	return false
}

func cloudUserName(user bbcloud.User) string {
	return firstNonEmpty(user.Nickname, user.Username, user.AccountID, user.UUID)
}

func adaptDCUser(user bbdc.User) User {
	return User{
		Name:        firstNonEmpty(user.Name, user.Slug),
		DisplayName: user.FullName,
	}
}

func adaptDCRepositoryRef(repo bbdc.Repository) RepositoryRef {
	scope := ""
	if repo.Project != nil {
		scope = repo.Project.Key
	}
	return RepositoryRef{Scope: scope, Slug: repo.Slug}
}

func adaptCloudRepositoryRef(repo bbcloud.RepositoryRef) RepositoryRef {
	scope, slug, _ := strings.Cut(repo.FullName, "/")
	if repo.Slug != "" {
		slug = repo.Slug
	}
	return RepositoryRef{Scope: scope, Slug: slug}
}

func formatDCTimestamp(field string, millis int64) (string, error) {
	if millis <= 0 {
		return "", fmt.Errorf("missing or invalid Bitbucket Data Center %s", field)
	}
	return time.UnixMilli(millis).UTC().Format(time.RFC3339Nano), nil
}

func formatCloudTimestamp(field, value string) (string, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", fmt.Errorf("parse Bitbucket Cloud %s: %w", field, err)
	}
	return parsed.UTC().Format(time.RFC3339Nano), nil
}

func stateFromMap(raw string, states map[string]CheckState) CheckState {
	normalized := strings.Join(strings.Fields(strings.ToUpper(raw)), " ")
	if state, ok := states[normalized]; ok {
		return state
	}
	return CheckUnknown
}

func firstDCLink(groups ...[]struct {
	Href string `json:"href"`
}) string {
	for _, group := range groups {
		if link := firstDCSelfLink(group); link != "" {
			return link
		}
	}
	return ""
}

func firstDCSelfLink(links []struct {
	Href string `json:"href"`
}) string {
	for _, link := range links {
		if link.Href != "" {
			return link.Href
		}
	}
	return ""
}

func stripURLQuery(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		before, _, _ := strings.Cut(raw, "?")
		return before
	}
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	return parsed.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func intPointer(value int) *int {
	return &value
}
