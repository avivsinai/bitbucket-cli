// Package testenv keeps package tests away from the developer's real home
// and configuration directories.
package testenv

import (
	"fmt"
	"os"
	"testing"
)

// Main runs m with the home directory and the per-OS configuration
// directory variables pointed at a temporary directory. Lock files and
// file-backed secrets that the code under test derives from os.UserHomeDir
// or os.UserConfigDir then land there, not in the user's real directories.
// Call it from TestMain.
func Main(m *testing.M) {
	dir, err := os.MkdirTemp("", "bkt-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "testenv:", err)
		os.Exit(1)
	}
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "AppData"} {
		if err := os.Setenv(key, dir); err != nil {
			fmt.Fprintln(os.Stderr, "testenv:", err)
			os.Exit(1)
		}
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
