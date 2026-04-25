package filelock

import (
	"fmt"
	"os"
	"path/filepath"
)

// With holds an exclusive file lock while fn runs.
func With(path string, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := lock(f); err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	defer func() { _ = unlock(f) }()

	return fn()
}
