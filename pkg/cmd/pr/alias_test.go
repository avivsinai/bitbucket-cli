package pr_test

import (
	"strings"
	"testing"
)

func TestDeclineCloseAlias(t *testing.T) {
	cfg := dcConfig("http://localhost")

	t.Run("canonical name", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "decline", "--help")
		if err != nil {
			t.Fatalf("pr decline --help failed: %v", err)
		}
	})

	t.Run("close alias", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "close", "--help")
		if err != nil {
			t.Fatalf("pr close --help failed: %v", err)
		}
	})

	t.Run("help output matches", func(t *testing.T) {
		declineOut, _, err := runCLI(t, cfg, "pr", "decline", "--help")
		if err != nil {
			t.Fatalf("pr decline --help failed: %v", err)
		}
		closeOut, _, err := runCLI(t, cfg, "pr", "close", "--help")
		if err != nil {
			t.Fatalf("pr close --help failed: %v", err)
		}
		if declineOut != closeOut {
			t.Errorf("help output differs between 'pr decline' and 'pr close'\ndecline:\n%s\nclose:\n%s", declineOut, closeOut)
		}
	})
}

func TestCommentReplyAlias(t *testing.T) {
	cfg := dcConfig("http://localhost")

	t.Run("canonical name", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "comment", "--help")
		if err != nil {
			t.Fatalf("pr comment --help failed: %v", err)
		}
	})

	t.Run("reply alias", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "reply", "--help")
		if err != nil {
			t.Fatalf("pr reply --help failed: %v", err)
		}
	})

	t.Run("help output matches", func(t *testing.T) {
		commentOut, _, err := runCLI(t, cfg, "pr", "comment", "--help")
		if err != nil {
			t.Fatalf("pr comment --help failed: %v", err)
		}
		replyOut, _, err := runCLI(t, cfg, "pr", "reply", "--help")
		if err != nil {
			t.Fatalf("pr reply --help failed: %v", err)
		}
		if commentOut != replyOut {
			t.Errorf("help output differs between 'pr comment' and 'pr reply'\ncomment:\n%s\nreply:\n%s", commentOut, replyOut)
		}
	})
}

func TestCreateDestinationFlagAlias(t *testing.T) {
	cfg := dcConfig("http://localhost")

	t.Run("--target accepted", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "create", "--target", "main", "--help")
		if err != nil {
			t.Fatalf("pr create --target --help failed: %v", err)
		}
	})

	t.Run("--destination accepted", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "create", "--destination", "main", "--help")
		if err != nil {
			t.Fatalf("pr create --destination --help failed: %v", err)
		}
	})

	t.Run("--target and --destination mutually exclusive", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "create", "--target", "main", "--destination", "develop")
		if err == nil {
			t.Fatal("expected error when both --target and --destination supplied")
		}
		if !strings.Contains(err.Error(), "specify only one") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
