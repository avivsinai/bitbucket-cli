package repo

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunGitCloneUsesDoubleDash(t *testing.T) {
	helperDir := t.TempDir()
	argsFile := filepath.Join(helperDir, "git-args.txt")

	t.Setenv("REPO_GIT_ARGS_FILE", argsFile)
	t.Setenv("PATH", helperDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	writeGitHelperScript(t, filepath.Join(helperDir, gitHelperName()))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runGitClone(cmd, os.Stdout, os.Stderr, strings.NewReader(""), "https://bitbucket.example.com/scm/PROJ/repo.git", "target-dir")
	if err != nil {
		t.Fatalf("runGitClone returned error: %v", err)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", argsFile, err)
	}

	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"clone", "--", "https://bitbucket.example.com/scm/PROJ/repo.git", "target-dir"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("git args = %#v, want %#v", got, want)
	}
}

func writeGitHelperScript(t *testing.T, target string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		script := "@echo off\r\n(\r\nfor %%a in (%*) do echo %%a\r\n) > \"%REPO_GIT_ARGS_FILE%\"\r\n"
		if err := os.WriteFile(target, []byte(script), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", target, err)
		}
		return
	}

	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$REPO_GIT_ARGS_FILE\"\n"
	if err := os.WriteFile(target, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", target, err)
	}
}

func gitHelperName() string {
	if runtime.GOOS == "windows" {
		return "git.bat"
	}
	return "git"
}
