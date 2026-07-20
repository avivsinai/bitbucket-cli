package mcpserver

import (
	"strings"
	"unicode/utf8"
)

const (
	// DefaultListLimit and MaxListLimit freeze the v1 collection bounds.
	DefaultListLimit            = 25
	MaxListLimit                = 100
	CommentBodyLimit            = 16 * 1024
	PullRequestDescriptionLimit = 16 * 1024
	DiffContentLimit            = 256 * 1024
)

const (
	// ProvenanceSourceBitbucket and ProvenanceTrustUntrusted mark external
	// content without rewriting it.
	ProvenanceSourceBitbucket = "bitbucket"
	ProvenanceTrustUntrusted  = "untrusted"
)

// ContextInfo is the bkt_get_context result DTO. Field shapes are part of
// the frozen v1 contract; never include credentials.
type ContextInfo struct {
	Platform     string   `json:"platform" jsonschema:"the pinned Bitbucket platform: dc (Data Center) or cloud"`
	HostLabel    string   `json:"host_label" jsonschema:"the bkt config host entry this server is pinned to"`
	DefaultScope string   `json:"default_scope,omitempty" jsonschema:"default scope (DC project key or Cloud workspace) used when a tool call omits the repository locator"`
	DefaultRepo  string   `json:"default_repo,omitempty" jsonschema:"default repository slug used when a tool call omits the repository locator"`
	Capabilities []string `json:"capabilities" jsonschema:"capability identifiers for the tools and roles this server supports on the pinned platform"`
}

// Repository is the platform-neutral repository result.
type Repository struct {
	Scope         string `json:"scope"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch,omitempty"`
	IsPrivate     bool   `json:"is_private"`
	URL           string `json:"url"`
}

// User is the stable identity shape embedded in author and reviewer results.
type User struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// RepositoryRef identifies a repository without leaking platform client types.
type RepositoryRef struct {
	Scope string `json:"scope"`
	Slug  string `json:"slug"`
}

// Reviewer carries the approval state returned by the upstream platform.
type Reviewer struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Approved    bool   `json:"approved"`
}

// PullRequest is the shared DC/Cloud pull request result. Description is set
// only by full-detail adapters.
type PullRequest struct {
	ID           int           `json:"id"`
	Title        string        `json:"title"`
	State        string        `json:"state"`
	Author       User          `json:"author"`
	SourceBranch string        `json:"source_branch"`
	TargetBranch string        `json:"target_branch"`
	Repo         RepositoryRef `json:"repo"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	URL          string        `json:"url"`
	Reviewers    []Reviewer    `json:"reviewers"`
	Description  *BoundedText  `json:"description,omitempty"`
}

// Comment is a normalized pull request comment. CreatedAt is omitted when an
// upstream response genuinely has no timestamp.
type Comment struct {
	ID        int         `json:"id"`
	Author    User        `json:"author"`
	Body      BoundedText `json:"body"`
	CreatedAt string      `json:"created_at,omitempty"`
	ParentID  *int        `json:"parent_id,omitempty"`
	Path      string      `json:"path,omitempty"`
	Line      *int        `json:"line,omitempty"`
}

// CheckState is the closed set of normalized build/check outcomes.
type CheckState string

const (
	CheckPending    CheckState = "pending"
	CheckRunning    CheckState = "running"
	CheckSuccessful CheckState = "successful"
	CheckFailed     CheckState = "failed"
	CheckStopped    CheckState = "stopped"
	CheckUnknown    CheckState = "unknown"
)

// Check is a normalized commit build status.
type Check struct {
	Key   string     `json:"key"`
	Name  string     `json:"name,omitempty"`
	State CheckState `json:"state"`
	URL   string     `json:"url,omitempty"`
}

// Diff contains bounded, untrusted unified diff content.
type Diff struct {
	Content      BoundedText `json:"content"`
	SourceCommit string      `json:"source_commit,omitempty"`
	TargetCommit string      `json:"target_commit,omitempty"`
}

// TextProvenance identifies where externally authored content came from and
// how consumers must treat it.
type TextProvenance struct {
	Source string `json:"source"`
	Trust  string `json:"trust"`
}

// BoundedText is the only carrier for Bitbucket-authored prose and diff data.
type BoundedText struct {
	Text         string         `json:"text"`
	Truncated    bool           `json:"truncated"`
	OriginalSize *int           `json:"original_size,omitempty"`
	Provenance   TextProvenance `json:"provenance"`
}

// ListEnvelope is a bounded collection result. A cursor can be added later
// without changing the existing fields.
type ListEnvelope[T any] struct {
	Items     []T  `json:"items"`
	Limit     int  `json:"limit"`
	Count     int  `json:"count"`
	Truncated bool `json:"truncated"`
}

func newListEnvelope[T any](items []T, requestedLimit int, hasMore bool) ListEnvelope[T] {
	limit := normalizedListLimit(requestedLimit)
	count := min(len(items), limit)
	bounded := make([]T, count)
	copy(bounded, items[:count])
	return ListEnvelope[T]{
		Items:     bounded,
		Limit:     limit,
		Count:     count,
		Truncated: hasMore || len(items) > limit,
	}
}

func normalizedListLimit(requested int) int {
	switch {
	case requested <= 0:
		return DefaultListLimit
	case requested > MaxListLimit:
		return MaxListLimit
	default:
		return requested
	}
}

// ErrorCode is the closed set of machine-readable v1 error categories.
type ErrorCode string

const (
	ErrorInvalidInput          ErrorCode = "invalid_input"
	ErrorNotFound              ErrorCode = "not_found"
	ErrorAuthFailed            ErrorCode = "auth_failed"
	ErrorUnsupportedOnPlatform ErrorCode = "unsupported_on_platform"
	ErrorRateLimited           ErrorCode = "rate_limited"
	ErrorUpstream              ErrorCode = "upstream_error"
)

// Error is the redacted, machine-readable tool error payload.
type Error struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

func boundBitbucketText(text string, limit int) BoundedText {
	text = strings.ToValidUTF8(text, "\ufffd")
	sourceSize := len(text)
	if limit < 0 {
		limit = 0
	}

	bounded := BoundedText{
		Text: text,
		Provenance: TextProvenance{
			Source: ProvenanceSourceBitbucket,
			Trust:  ProvenanceTrustUntrusted,
		},
	}
	if len(text) <= limit {
		return bounded
	}

	cut := limit
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	bounded.Text = text[:cut]
	bounded.Truncated = true
	bounded.OriginalSize = &sourceSize
	return bounded
}
