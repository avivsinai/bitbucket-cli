package extension

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestValidateExtensionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "accepts simple name", input: "demo"},
		{name: "accepts dots", input: "demo.v2"},
		{name: "rejects slash", input: "demo/test", wantErr: `invalid extension name "demo/test": must not contain path separators or '..'`},
		{name: "rejects backslash", input: `demo\test`, wantErr: `invalid extension name "demo\\test": must not contain path separators or '..'`},
		{name: "rejects traversal", input: "../demo", wantErr: `invalid extension name "../demo": must not contain path separators or '..'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExtensionName(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateExtensionName returned error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestRunExtensionRemoveRejectsTraversalName(t *testing.T) {
	err := runExtensionRemove(&cobra.Command{}, nil, "../demo")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `invalid extension name "../demo": must not contain path separators or '..'` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExtensionExecRejectsTraversalName(t *testing.T) {
	err := runExtensionExec(&cobra.Command{}, nil, "../demo", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `invalid extension name "../demo": must not contain path separators or '..'` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExtensionInstallUsesDoubleDash(t *testing.T) {
	f, _, stderr := newExtensionTestFactory(t)
	helperDir := t.TempDir()
	argsFile := filepath.Join(helperDir, "git-args.txt")

	t.Setenv("EXTENSION_GIT_ARGS_FILE", argsFile)
	t.Setenv("PATH", helperDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	writeGitHelperScript(t, filepath.Join(helperDir, gitHelperName()))

	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())

	if err := runExtensionInstall(cmd, f, "--upload-pack=evil"); err != nil {
		t.Fatalf("runExtensionInstall returned error: %v", err)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", argsFile, err)
	}

	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"clone", "--", "--upload-pack=evil", filepath.Join(extensionParentRootForTest(f), "--upload-pack=evil")}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("git args = %#v, want %#v", got, want)
	}
	if !strings.Contains(stderr.String(), "warning:") {
		t.Fatalf("expected missing-executable warning, got stderr:\n%s", stderr.String())
	}
}

func TestRunExtensionExecFiltersSensitiveEnv(t *testing.T) {
	f, stdout, _ := newExtensionTestFactory(t)
	cfg, err := f.ResolveConfig()
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}

	t.Setenv("GO_WANT_EXTENSION_HELPER_PROCESS", "1")
	t.Setenv("BKT_TOKEN", "secret-token")
	t.Setenv("BKT_KEYRING_PASSPHRASE", "secret-passphrase")
	t.Setenv("BKT_ALLOW_INSECURE_STORE", "1")

	helperPath := filepath.Join(extensionRootForTest(cfg), helperExecutableName("demo"))
	if err := os.MkdirAll(filepath.Dir(helperPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	copyTestBinary(t, helperPath)

	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())

	err = runExtensionExec(cmd, f, "demo", []string{"-test.run=TestExtensionHelperProcess"})
	if err != nil {
		t.Fatalf("runExtensionExec returned error: %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	if got != "|||demo" {
		t.Fatalf("stdout = %q, want %q", got, "|||demo")
	}
}

func TestExtensionHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_EXTENSION_HELPER_PROCESS") != "1" {
		return
	}

	fmt.Printf("%s|%s|%s|%s",
		os.Getenv("BKT_TOKEN"),
		os.Getenv("BKT_KEYRING_PASSPHRASE"),
		os.Getenv("BKT_ALLOW_INSECURE_STORE"),
		os.Getenv("BKT_EXTENSION_NAME"),
	)
	os.Exit(0)
}

func newExtensionTestFactory(t *testing.T) (*cmdutil.Factory, *strings.Builder, *strings.Builder) {
	t.Helper()

	cfgDir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", cfgDir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	var stdout, stderr strings.Builder
	f := &cmdutil.Factory{
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    &stdout,
			ErrOut: &stderr,
			In:     io.NopCloser(strings.NewReader("")),
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	return f, &stdout, &stderr
}

func extensionRootForTest(cfg *config.Config) string {
	return filepath.Join(filepath.Dir(cfg.Path()), "extensions", "demo")
}

func extensionParentRootForTest(f *cmdutil.Factory) string {
	cfg, err := f.ResolveConfig()
	if err != nil {
		panic(err)
	}
	return filepath.Join(filepath.Dir(cfg.Path()), "extensions")
}

func helperExecutableName(name string) string {
	if runtime.GOOS == "windows" {
		return "bkt-" + name + ".exe"
	}
	return "bkt-" + name
}

func copyTestBinary(t *testing.T, target string) {
	t.Helper()

	src, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", src, err)
	}

	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		mode = 0o644
	}
	if err := os.WriteFile(target, data, mode); err != nil {
		t.Fatalf("WriteFile(%s): %v", target, err)
	}
}

func writeGitHelperScript(t *testing.T, target string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		script := "@echo off\r\n(\r\nfor %%a in (%*) do echo %%a\r\n) > \"%EXTENSION_GIT_ARGS_FILE%\"\r\n"
		if err := os.WriteFile(target, []byte(script), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", target, err)
		}
		return
	}

	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$EXTENSION_GIT_ARGS_FILE\"\n"
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
