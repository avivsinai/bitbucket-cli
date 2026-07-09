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
		exe, err := os.Executable()
		if err != nil {
			t.Fatalf("os.Executable: %v", err)
		}
		exe = strings.ReplaceAll(exe, "%", "%%")
		script := "@echo off\r\n" +
			"set GO_WANT_REPO_GIT_HELPER_PROCESS=1\r\n" +
			"\"" + exe + "\" -test.run=TestRepoGitHelperProcess -- %*\r\n" +
			"exit /b %ERRORLEVEL%\r\n"
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

func TestRepoGitHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_REPO_GIT_HELPER_PROCESS") != "1" {
		return
	}

	args := testArgsAfterDoubleDash(os.Args)
	if err := os.WriteFile(os.Getenv("REPO_GIT_ARGS_FILE"), []byte(strings.Join(args, "\n")), 0o644); err != nil {
		panic(err)
	}
	os.Exit(0)
}

func testArgsAfterDoubleDash(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}
	return nil
}
