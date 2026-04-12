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

	t.Run("help output matches", func(t *testing.T) {
		editOut, _, err := runCLI(t, cfg, "pr", "edit", "--help")
		if err != nil {
			t.Fatalf("pr edit --help failed: %v", err)
		}
		updateOut, _, err := runCLI(t, cfg, "pr", "update", "--help")
		if err != nil {
			t.Fatalf("pr update --help failed: %v", err)
		}
		if editOut != updateOut {
			t.Errorf("help output differs between 'pr edit' and 'pr update'\nedit:\n%s\nupdate:\n%s", editOut, updateOut)
		}
	})
}
