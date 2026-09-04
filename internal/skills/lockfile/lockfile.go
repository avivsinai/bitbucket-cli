// Package lockfile maintains ~/.agents/.skill-lock.json, the cross-tool record
// of installed skills introduced by Vercel's skills CLI and also written by gh.
package lockfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/avivsinai/bitbucket-cli/internal/filelock"
)

const (
	// lockVersion must match Vercel's CURRENT_LOCK_VERSION for interop.
	lockVersion = 3
	agentsDir   = ".agents"
	lockFile    = ".skill-lock.json"
	// SourceType identifies entries written by bkt.
	SourceType = "bitbucket"
)

// Entry describes where an installed skill came from.
type Entry struct {
	Source          string `json:"source"`              // "workspace/slug" or "PROJECT/slug"
	SourceType      string `json:"sourceType"`          // always SourceType for bkt installs
	SourceURL       string `json:"sourceUrl"`           // HTTPS clone URL
	SkillPath       string `json:"skillPath,omitempty"` // "skills/name/SKILL.md"
	SkillFolderHash string `json:"skillFolderHash"`     // latest commit touching the skill directory
	InstalledAt     string `json:"installedAt"`
	UpdatedAt       string `json:"updatedAt"`
	PinnedRef       string `json:"pinnedRef,omitempty"`
}

type file struct {
	Version   int              `json:"version"`
	Skills    map[string]Entry `json:"skills"`
	Dismissed map[string]bool  `json:"dismissed,omitempty"`
}

// Path returns the lock file location for the given home directory.
func Path(homeDir string) string {
	return filepath.Join(homeDir, agentsDir, lockFile)
}

// RecordInstall adds or refreshes the entry for skillName. installedAt is
// preserved across updates; updatedAt is always now. Concurrent writers are
// serialised through a sibling lock file.
func RecordInstall(homeDir, skillName string, entry Entry) error {
	if homeDir == "" {
		return errors.New("could not determine home directory")
	}
	lockPath := Path(homeDir)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("could not create lock directory: %w", err)
	}

	return filelock.With(lockPath+".lock", func() error {
		lf, err := read(lockPath)
		if err != nil {
			return err
		}

		now := time.Now().UTC().Format(time.RFC3339)
		entry.SourceType = SourceType
		entry.InstalledAt = now
		entry.UpdatedAt = now
		if existing, ok := lf.Skills[skillName]; ok && existing.InstalledAt != "" {
			entry.InstalledAt = existing.InstalledAt
		}
		lf.Skills[skillName] = entry

		return write(lockPath, lf)
	})
}

// read loads the lock file, treating a missing, empty, corrupt, or
// incompatible file as fresh state.
func read(path string) (*file, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(data) == 0) {
		return newFile(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read lock file: %w", err)
	}

	var lf file
	if err := json.Unmarshal(data, &lf); err != nil {
		return newFile(), nil //nolint:nilerr // graceful: a corrupt file means fresh state
	}
	if lf.Version != lockVersion || lf.Skills == nil {
		return newFile(), nil
	}
	return &lf, nil
}

func write(path string, lf *file) error {
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("could not write lock file: %w", err)
	}
	return nil
}

func newFile() *file {
	return &file{Version: lockVersion, Skills: make(map[string]Entry)}
}
