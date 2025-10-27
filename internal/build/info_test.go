package build

import "testing"

func TestVersionUsesLdflags(t *testing.T) {
	orig := versionFromLdflags
	defer func() { versionFromLdflags = orig }()

	versionFromLdflags = "1.2.3"
	if got := version(); got != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %q", got)
	}
}

func TestCommitUsesLdflags(t *testing.T) {
	orig := commitFromLdflags
	defer func() { commitFromLdflags = orig }()

	commitFromLdflags = "abcdef123456"
	if got := commit(); got != "abcdef123456" {
		t.Fatalf("expected commit from ldflags, got %q", got)
	}
}

func TestDateUsesLdflags(t *testing.T) {
	orig := dateFromLdflags
	defer func() { dateFromLdflags = orig }()

	dateFromLdflags = "2025-10-27T12:34:56Z"
	if got := date(); got != "2025-10-27T12:34:56Z" {
		t.Fatalf("expected date from ldflags, got %q", got)
	}
}
