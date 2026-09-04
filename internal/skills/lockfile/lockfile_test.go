package lockfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type decodedFile struct {
	Version int              `json:"version"`
	Skills  map[string]Entry `json:"skills"`
}

func readLock(t *testing.T, home string) decodedFile {
	t.Helper()
	data, err := os.ReadFile(Path(home))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	var lf decodedFile
	if err := json.Unmarshal(data, &lf); err != nil {
		t.Fatalf("parse lock file: %v", err)
	}
	return lf
}

func TestRecordInstallCreatesAndUpdatesEntries(t *testing.T) {
	home := t.TempDir()
	entry := Entry{
		Source:          "myteam/agent-skills",
		SourceURL:       "https://bitbucket.org/myteam/agent-skills.git",
		SkillPath:       "skills/alpha/SKILL.md",
		SkillFolderHash: "commit1",
		PinnedRef:       "v1.0.0",
	}
	if err := RecordInstall(home, "alpha", entry); err != nil {
		t.Fatalf("RecordInstall: %v", err)
	}

	lf := readLock(t, home)
	if lf.Version != 3 {
		t.Fatalf("version = %d, want 3 (Vercel interop)", lf.Version)
	}
	got := lf.Skills["alpha"]
	if got.SourceType != "bitbucket" || got.Source != entry.Source || got.SourceURL != entry.SourceURL || got.SkillFolderHash != "commit1" || got.PinnedRef != "v1.0.0" {
		t.Fatalf("entry = %+v", got)
	}
	if got.InstalledAt == "" || got.InstalledAt != got.UpdatedAt {
		t.Fatalf("timestamps = %q / %q", got.InstalledAt, got.UpdatedAt)
	}

	// Re-recording keeps installedAt, refreshes the hash and drops the pin.
	entry.SkillFolderHash = "commit2"
	entry.PinnedRef = ""
	if err := RecordInstall(home, "alpha", entry); err != nil {
		t.Fatalf("RecordInstall (update): %v", err)
	}
	updated := readLock(t, home).Skills["alpha"]
	if updated.InstalledAt != got.InstalledAt {
		t.Fatalf("installedAt changed from %q to %q", got.InstalledAt, updated.InstalledAt)
	}
	if updated.SkillFolderHash != "commit2" || updated.PinnedRef != "" {
		t.Fatalf("updated entry = %+v", updated)
	}

	if err := RecordInstall(home, "acme/beta", Entry{Source: "x/y"}); err != nil {
		t.Fatal(err)
	}
	if n := len(readLock(t, home).Skills); n != 2 {
		t.Fatalf("expected 2 entries, got %d", n)
	}
}

func TestRecordInstallPreservesUnknownFields(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(home)), 0o755); err != nil {
		t.Fatal(err)
	}
	original := `{
  "version": 3,
  "skills": {
    "alpha": {"source":"old/source","installedAt":"2024-01-01T00:00:00Z","futureEntry":{"enabled":true}},
    "other": {"source":"other/source","sourceType":"github","pinnedRef":null,"futureOther":[1,2,3]}
  },
  "dismissed": {"notice": true},
  "futureTop": {"owner":"another-tool"}
}`
	if err := os.WriteFile(Path(home), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RecordInstall(home, "alpha", Entry{Source: "new/source", SkillFolderHash: "new-hash"}); err != nil {
		t.Fatalf("RecordInstall: %v", err)
	}

	data, err := os.ReadFile(Path(home))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse updated lock file: %v", err)
	}
	if got["futureTop"].(map[string]any)["owner"] != "another-tool" {
		t.Fatalf("top-level extension lost: %s", data)
	}
	skills := got["skills"].(map[string]any)
	alpha := skills["alpha"].(map[string]any)
	if alpha["futureEntry"].(map[string]any)["enabled"] != true || alpha["source"] != "new/source" {
		t.Fatalf("updated entry fields = %+v", alpha)
	}
	other := skills["other"].(map[string]any)
	if other["futureOther"].([]any)[2] != float64(3) || other["source"] != "other/source" || other["pinnedRef"] != nil {
		t.Fatalf("unrelated entry changed: %+v", other)
	}
	if _, ok := other["skillFolderHash"]; ok {
		t.Fatalf("omitted known field was added to unrelated entry: %+v", other)
	}
}

func TestRecordInstallRejectsCorruptOrIncompatibleFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "corrupt json", content: "{not json"},
		{name: "old version", content: `{"version": 1, "skills": {"old": {"source": "a/b"}}}`},
		{name: "empty file", content: ""},
		{name: "missing skills map", content: `{"version": 3}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			if err := os.MkdirAll(filepath.Dir(Path(home)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(Path(home), []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := RecordInstall(home, "alpha", Entry{Source: "a/b"}); err == nil {
				t.Fatal("RecordInstall succeeded for invalid existing file")
			}
			data, err := os.ReadFile(Path(home))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.content {
				t.Fatalf("invalid lock file was overwritten: got %q, want %q", data, tt.content)
			}
		})
	}
}

func TestRecordInstallRequiresHomeDir(t *testing.T) {
	if err := RecordInstall("", "alpha", Entry{}); err == nil {
		t.Fatal("expected error for empty home directory")
	}
}
