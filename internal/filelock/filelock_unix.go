//go:build darwin || linux || freebsd || openbsd || netbsd

package filelock

import (
	"os"
	"syscall"
)

func lock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

func unlock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
