package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFrozenBounds(t *testing.T) {
	if DefaultListLimit != 25 || MaxListLimit != 100 {
		t.Fatalf("list bounds = default %d, max %d; want 25 and 100", DefaultListLimit, MaxListLimit)
	}
	if CommentBodyLimit != 16*1024 || PullRequestDescriptionLimit != 16*1024 {
		t.Fatalf("prose bounds = comment %d, PR description %d; want 16 KiB each", CommentBodyLimit, PullRequestDescriptionLimit)
	}
	if DiffContentLimit != 256*1024 {
		t.Fatalf("diff bound = %d; want 256 KiB", DiffContentLimit)
	}
}

func TestNewListEnvelopeEnforcesBounds(t *testing.T) {
	tests := []struct {
		name          string
		items         []int
		requested     int
		hasMore       bool
		wantItems     []int
		wantLimit     int
		wantTruncated bool
	}{
		{name: "default and non-null items", requested: 0, wantItems: []int{}, wantLimit: DefaultListLimit},
		{name: "negative uses default", requested: -1, wantItems: []int{}, wantLimit: DefaultListLimit},
		{name: "requested limit truncates", items: []int{1, 2, 3}, requested: 2, wantItems: []int{1, 2}, wantLimit: 2, wantTruncated: true},
		{name: "max clamps", requested: MaxListLimit + 1, wantItems: []int{}, wantLimit: MaxListLimit},
		{name: "upstream has more", items: []int{1}, requested: 2, hasMore: true, wantItems: []int{1}, wantLimit: 2, wantTruncated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newListEnvelope(tt.items, tt.requested, tt.hasMore)
			if got.Limit != tt.wantLimit || got.Count != len(tt.wantItems) || got.Truncated != tt.wantTruncated {
				t.Fatalf("envelope metadata = %+v, want limit=%d count=%d truncated=%v", got, tt.wantLimit, len(tt.wantItems), tt.wantTruncated)
			}
			if got.Items == nil {
				t.Fatal("items must be a non-null empty slice")
			}
			if len(got.Items) != len(tt.wantItems) {
				t.Fatalf("items = %v, want %v", got.Items, tt.wantItems)
			}
			for i := range got.Items {
				if got.Items[i] != tt.wantItems[i] {
					t.Fatalf("items = %v, want %v", got.Items, tt.wantItems)
				}
			}
		})
	}
}

func TestBoundedTextPreservesUTF8AndByteCounts(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		limit         int
		wantText      string
		wantTruncated bool
		wantOriginal  int
	}{
		{name: "empty", input: "", limit: 4, wantText: ""},
		{name: "exactly at limit", input: "abcd", limit: 4, wantText: "abcd"},
		{name: "ASCII over limit", input: "abcde", limit: 4, wantText: "abcd", wantTruncated: true, wantOriginal: 5},
		{name: "multibyte rune straddles boundary", input: "ab€cd", limit: 4, wantText: "ab", wantTruncated: true, wantOriginal: 7},
		{name: "invalid input becomes valid UTF-8", input: string([]byte{'a', 0xff, 'b'}), limit: 16, wantText: "a\ufffdb"},
		{name: "invalid input byte count follows valid replacement", input: string([]byte{'a', 0xff, 'b'}), limit: 2, wantText: "a", wantTruncated: true, wantOriginal: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boundBitbucketText(tt.input, tt.limit)
			if got.Text != tt.wantText || got.Truncated != tt.wantTruncated {
				t.Fatalf("boundBitbucketText() = %+v, want text %q truncated %v", got, tt.wantText, tt.wantTruncated)
			}
			if !utf8.ValidString(got.Text) {
				t.Fatalf("bounded text is invalid UTF-8: %q", got.Text)
			}
			if len(got.Text) > tt.limit {
				t.Fatalf("bounded text is %d bytes, exceeds limit %d", len(got.Text), tt.limit)
			}
			if got.Provenance.Source != ProvenanceSourceBitbucket || got.Provenance.Trust != ProvenanceTrustUntrusted {
				t.Fatalf("provenance = %+v, want bitbucket/untrusted", got.Provenance)
			}
			if tt.wantOriginal == 0 {
				if got.OriginalSize != nil {
					t.Fatalf("original_size = %v, want omitted when content is not truncated", *got.OriginalSize)
				}
			} else if got.OriginalSize == nil || *got.OriginalSize != tt.wantOriginal {
				t.Fatalf("original_size = %v, want %d canonical UTF-8 bytes", got.OriginalSize, tt.wantOriginal)
			}
		})
	}
}

func TestBoundedTextJSONShape(t *testing.T) {
	full, err := json.Marshal(boundBitbucketText("abcde", 4))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"text":"abcd"`,
		`"truncated":true`,
		`"original_size":5`,
		`"provenance":{"source":"bitbucket","trust":"untrusted"}`,
	} {
		if !strings.Contains(string(full), want) {
			t.Fatalf("bounded text JSON missing %s: %s", want, full)
		}
	}

	short, err := json.Marshal(boundBitbucketText("abcd", 4))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(short), "original_size") {
		t.Fatalf("untruncated JSON must omit original_size: %s", short)
	}
}

func TestFrozenCheckStates(t *testing.T) {
	want := []CheckState{
		CheckPending,
		CheckRunning,
		CheckSuccessful,
		CheckFailed,
		CheckStopped,
		CheckUnknown,
	}
	got := []string{"pending", "running", "successful", "failed", "stopped", "unknown"}
	for i := range want {
		if string(want[i]) != got[i] {
			t.Fatalf("check state %d = %q, want %q", i, want[i], got[i])
		}
	}
}

func TestFrozenErrorCodes(t *testing.T) {
	want := []ErrorCode{
		ErrorInvalidInput,
		ErrorNotFound,
		ErrorAuthFailed,
		ErrorUnsupportedOnPlatform,
		ErrorRateLimited,
		ErrorUpstream,
	}
	got := []string{"invalid_input", "not_found", "auth_failed", "unsupported_on_platform", "rate_limited", "upstream_error"}
	for i := range want {
		if string(want[i]) != got[i] {
			t.Fatalf("error code %d = %q, want %q", i, want[i], got[i])
		}
	}
}

func TestDTOOptionalFieldsAreActuallyOptional(t *testing.T) {
	doc, err := json.Marshal(struct {
		Repository  Repository        `json:"repository"`
		PullRequest PullRequest       `json:"pull_request"`
		Comment     Comment           `json:"comment"`
		Check       Check             `json:"check"`
		Diff        Diff              `json:"diff"`
		List        ListEnvelope[int] `json:"list"`
		Error       Error             `json:"error"`
	}{
		PullRequest: PullRequest{Reviewers: []Reviewer{}},
		List:        ListEnvelope[int]{Items: []int{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{"default_branch", "description", "parent_id", "path", "line", "source_commit", "target_commit", "details"} {
		if strings.Contains(string(doc), `"`+absent+`"`) {
			t.Fatalf("zero-value DTO JSON unexpectedly contains optional field %q: %s", absent, doc)
		}
	}
	for _, want := range []string{`"reviewers":[]`, `"items":[]`} {
		if !strings.Contains(string(doc), want) {
			t.Fatalf("zero-value DTO JSON missing non-null collection %s: %s", want, doc)
		}
	}
}
