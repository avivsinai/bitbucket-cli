package lockfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readLock(t *testing.T, home string) file {
	t.Helper()
	data, err := os.ReadFile(Path(home))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	var lf file
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

func TestRecordInstallRecoversFromCorruptOrIncompatibleFile(t *testing.T) {
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
			if err := RecordInstall(home, "alpha", Entry{Source: "a/b"}); err != nil {
				t.Fatalf("RecordInstall: %v", err)
			}
			lf := readLock(t, home)
			if lf.Version != 3 || len(lf.Skills) != 1 {
				t.Fatalf("lock file = %+v, want fresh v3 with one entry", lf)
			}
		})
	}
}

func TestRecordInstallRequiresHomeDir(t *testing.T) {
	if err := RecordInstall("", "alpha", Entry{}); err == nil {
		t.Fatal("expected error for empty home directory")
	}
}
