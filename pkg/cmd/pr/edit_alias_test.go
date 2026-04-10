package pr_test

import "testing"

func TestEditCommandResolution(t *testing.T) {
	cfg := dcConfig("http://localhost")

	t.Run("canonical name", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "edit", "--help")
		if err != nil {
			t.Fatalf("pr edit --help failed: %v", err)
		}
	})

	t.Run("update alias", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "update", "--help")
		if err != nil {
			t.Fatalf("pr update --help failed: %v", err)
		}
	})
}
