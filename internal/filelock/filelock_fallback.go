//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !windows

package filelock

import "os"

func lock(_ *os.File) error {
	return nil
}

func unlock(_ *os.File) error {
	return nil
}
