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
	Version int
	Skills  map[string]json.RawMessage
	raw     map[string]json.RawMessage
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
		fields := make(map[string]json.RawMessage)
		if raw, ok := lf.Skills[skillName]; ok {
			if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
				return fmt.Errorf("invalid lock file: skill %q must be an object", skillName)
			}
			var existing Entry
			if err := json.Unmarshal(raw, &existing); err != nil {
				return fmt.Errorf("invalid lock file entry %q: %w", skillName, err)
			}
			if existing.InstalledAt != "" {
				entry.InstalledAt = existing.InstalledAt
			}
		}
		for _, key := range entryKeys {
			delete(fields, key)
		}
		known, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		var knownFields map[string]json.RawMessage
		if err := json.Unmarshal(known, &knownFields); err != nil {
			return err
		}
		for key, value := range knownFields {
			fields[key] = value
		}
		lf.Skills[skillName], err = json.Marshal(fields)
		if err != nil {
			return err
		}

		return write(lockPath, lf)
	})
}

// read loads and validates the lock file. Unknown fields are retained so a
// bkt update does not discard data written by another compatible tool.
func read(path string) (*file, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newFile(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read lock file: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("could not parse lock file: %w", err)
	}
	if raw == nil {
		return nil, errors.New("invalid lock file: root must be an object")
	}
	var version int
	if err := json.Unmarshal(raw["version"], &version); err != nil {
		return nil, fmt.Errorf("invalid lock file version: %w", err)
	}
	if version != lockVersion {
		return nil, fmt.Errorf("unsupported lock file version %d (expected %d)", version, lockVersion)
	}
	var skills map[string]json.RawMessage
	if err := json.Unmarshal(raw["skills"], &skills); err != nil || skills == nil {
		return nil, errors.New("invalid lock file: skills must be an object")
	}
	for name, entry := range skills {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(entry, &fields); err != nil || fields == nil {
			return nil, fmt.Errorf("invalid lock file: skill %q must be an object", name)
		}
	}
	return &file{Version: version, Skills: skills, raw: raw}, nil
}

func write(path string, lf *file) error {
	root := cloneRawMap(lf.raw)
	version, err := json.Marshal(lf.Version)
	if err != nil {
		return err
	}
	root["version"] = version

	root["skills"], err = json.Marshal(lf.Skills)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeAtomic(path, data)
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".skill-lock-*.json")
	if err != nil {
		return fmt.Errorf("could not create temporary lock file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not write temporary lock file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not set lock file permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close temporary lock file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("could not write lock file: %w", err)
	}
	return nil
}

func cloneRawMap(src map[string]json.RawMessage) map[string]json.RawMessage {
	dst := make(map[string]json.RawMessage, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func newFile() *file {
	return &file{
		Version: lockVersion,
		Skills:  make(map[string]json.RawMessage),
		raw:     make(map[string]json.RawMessage),
	}
}

var entryKeys = []string{
	"source", "sourceType", "sourceUrl", "skillPath", "skillFolderHash",
	"installedAt", "updatedAt", "pinnedRef",
}
