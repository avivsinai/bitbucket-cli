package pr

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/types"
)

func TestExecuteStatusCheckReturnsErrPendingOnTimeoutWithPendingBuilds(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ios := &iostreams.IOStreams{Out: &stdout, ErrOut: &stderr}
	cmd := newStatusCheckTestCommand()

	err := executeStatusCheck(&checksResult{
		ctx: context.Background(),
		ios: ios,
		cmd: cmd,
		opts: &checksOptions{
			ID:          42,
			Wait:        true,
			Interval:    time.Millisecond,
			MaxInterval: time.Millisecond,
			waitForPoll: func(context.Context, time.Duration) error { return context.DeadlineExceeded },
		},
		fetchFunc: func() ([]types.CommitStatus, error) {
			return []types.CommitStatus{{State: "INPROGRESS", Name: "build"}}, nil
		},
		commitSHA: "abc123def456",
		payload:   map[string]any{},
	})
	if !errors.Is(err, cmdutil.ErrPending) {
		t.Fatalf("err = %v, want ErrPending", err)
	}
	if got := stderr.String(); got == "" || !bytes.Contains([]byte(got), []byte("Timeout waiting for builds to complete")) {
		t.Fatalf("stderr = %q", got)
	}
	if got := stdout.String(); got == "" || !bytes.Contains([]byte(got), []byte("INPROGRESS")) {
		t.Fatalf("stdout = %q", got)
	}
}

func TestExecuteStatusCheckReturnsErrSilentOnFailedBuilds(t *testing.T) {
	var stdout bytes.Buffer
	ios := &iostreams.IOStreams{Out: &stdout, ErrOut: &bytes.Buffer{}}
	cmd := newStatusCheckTestCommand()

	err := executeStatusCheck(&checksResult{
		ctx: context.Background(),
		ios: ios,
		cmd: cmd,
		opts: &checksOptions{
			ID:          99,
			Wait:        true,
			Interval:    time.Millisecond,
			MaxInterval: time.Millisecond,
			waitForPoll: noWaitPoll,
		},
		fetchFunc: func() ([]types.CommitStatus, error) {
			return []types.CommitStatus{{State: "FAILED", Name: "ci"}}, nil
		},
		commitSHA: "abc123def456",
		payload:   map[string]any{},
	})
	if !errors.Is(err, cmdutil.ErrSilent) {
		t.Fatalf("err = %v, want ErrSilent", err)
	}
	if got := stdout.String(); got == "" || !bytes.Contains([]byte(got), []byte("FAILED")) {
		t.Fatalf("stdout = %q", got)
	}
}

func TestExecuteStatusCheckOpensBrowserForFirstStatusURL(t *testing.T) {
	var opened string
	cmd := newStatusCheckTestCommand()

	err := executeStatusCheck(&checksResult{
		ctx: context.Background(),
		ios: &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}},
		cmd: cmd,
		opts: &checksOptions{
			ID:   7,
			Web:  true,
			Wait: false,
		},
		fetchFunc: func() ([]types.CommitStatus, error) {
			return []types.CommitStatus{{State: "SUCCESSFUL", Name: "build", URL: "https://example.com/build/7"}}, nil
		},
		commitSHA: "abc123def456",
		payload:   map[string]any{},
		browserOpen: func(url string) error {
			opened = url
			return nil
		},
	})
	if err != nil {
		t.Fatalf("executeStatusCheck: %v", err)
	}
	if opened != "https://example.com/build/7" {
		t.Fatalf("opened = %q", opened)
	}
}

func TestExecuteStatusCheckReturnsBrowserError(t *testing.T) {
	cmd := newStatusCheckTestCommand()
	wantErr := errors.New("browser boom")

	err := executeStatusCheck(&checksResult{
		ctx: context.Background(),
		ios: &iostreams.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}},
		cmd: cmd,
		opts: &checksOptions{
			ID:   8,
			Web:  true,
			Wait: false,
		},
		fetchFunc: func() ([]types.CommitStatus, error) {
			return []types.CommitStatus{{State: "SUCCESSFUL", Name: "build", URL: "https://example.com/build/8"}}, nil
		},
		commitSHA: "abc123def456",
		payload:   map[string]any{},
		browserOpen: func(string) error {
			return wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wrapped %v", err, wantErr)
	}
}

func TestFindRemoteByURL(t *testing.T) {
	helperDir := t.TempDir()
	t.Setenv("PATH", helperDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PR_TEST_GIT_REMOTE_V_OUTPUT", "origin\thttps://bitbucket.org/ws/repo.git (fetch)\norigin\thttps://bitbucket.org/ws/repo.git (push)\nfork\tgit@bitbucket.org:user/repo.git (fetch)\n")

	writePRGitHelper(t, filepath.Join(helperDir, prGitHelperName()))

	if got := findRemoteByURL(context.Background(), "https://bitbucket.org/ws/repo.git"); got != "origin" {
		t.Fatalf("findRemoteByURL() = %q, want origin", got)
	}
	if got := findRemoteByURL(context.Background(), "git@bitbucket.org:user/repo.git"); got != "fork" {
		t.Fatalf("findRemoteByURL() = %q, want fork", got)
	}
	if got := findRemoteByURL(context.Background(), "https://bitbucket.org/missing/repo.git"); got != "" {
		t.Fatalf("findRemoteByURL() = %q, want empty", got)
	}
}

func TestInferProtocol(t *testing.T) {
	helperDir := t.TempDir()
	t.Setenv("PATH", helperDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PR_TEST_GIT_REMOTE_GET_URL_OUTPUT", "git@bitbucket.org:ws/repo.git")

	writePRGitHelper(t, filepath.Join(helperDir, prGitHelperName()))

	if got := inferProtocol(context.Background(), "origin"); got != "ssh" {
		t.Fatalf("inferProtocol() = %q, want ssh", got)
	}

	t.Setenv("PR_TEST_GIT_REMOTE_GET_URL_OUTPUT", "https://bitbucket.org/ws/repo.git")
	if got := inferProtocol(context.Background(), "origin"); got != "https" {
		t.Fatalf("inferProtocol() = %q, want https", got)
	}
}

func newStatusCheckTestCommand() *cobra.Command {
	root := &cobra.Command{Use: "bkt"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("yaml", false, "")
	root.PersistentFlags().String("jq", "", "")
	root.PersistentFlags().String("template", "", "")

	cmd := &cobra.Command{Use: "checks"}
	root.AddCommand(cmd)
	return cmd
}

func prGitHelperName() string {
	if runtime.GOOS == "windows" {
		return "git.bat"
	}
	return "git"
}

func writePRGitHelper(t *testing.T, target string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		script := "@echo off\r\n" +
			"if \"%1\"==\"remote\" if \"%2\"==\"-v\" (\r\n" +
			"  <nul set /p =%PR_TEST_GIT_REMOTE_V_OUTPUT%\r\n" +
			"  exit /b 0\r\n" +
			")\r\n" +
			"if \"%1\"==\"remote\" if \"%2\"==\"get-url\" (\r\n" +
			"  <nul set /p =%PR_TEST_GIT_REMOTE_GET_URL_OUTPUT%\r\n" +
			"  exit /b 0\r\n" +
			")\r\n" +
			"exit /b 1\r\n"
		if err := os.WriteFile(target, []byte(script), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", target, err)
		}
		return
	}

	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"remote\" ] && [ \"$2\" = \"-v\" ]; then\n" +
		"  printf '%s' \"$PR_TEST_GIT_REMOTE_V_OUTPUT\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"remote\" ] && [ \"$2\" = \"get-url\" ]; then\n" +
		"  printf '%s' \"$PR_TEST_GIT_REMOTE_GET_URL_OUTPUT\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(target, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%s): %v", target, err)
	}
}
