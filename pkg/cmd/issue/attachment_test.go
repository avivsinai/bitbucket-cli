package issue

import (
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func newTestFactory() *cmdutil.Factory {
	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:        "cloud",
				Workspace:   "testworkspace",
				DefaultRepo: "testrepo",
			},
		},
		Hosts: map[string]*config.Host{
			"cloud": {
				Kind:    "cloud",
				BaseURL: "https://api.bitbucket.org/2.0",
				Token:   "test-token",
			},
		},
	}

	var stdout, stderr strings.Builder
	return &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    &stdout,
			ErrOut: &stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}
}

// --- Attachment List Command Tests ---

func TestAttachmentListRequiresIssueID(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when issue ID not provided")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected 'accepts 1 arg' error, got %q", err.Error())
	}
}

func TestAttachmentListInvalidIssueID(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"notanumber"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-numeric issue ID")
	}
	if !strings.Contains(err.Error(), "invalid issue ID") {
		t.Errorf("expected 'invalid issue ID' error, got %q", err.Error())
	}
}

// --- Attachment Upload Command Tests ---

func TestAttachmentUploadRequiresIssueIDAndFiles(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentUploadCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when arguments not provided")
	}
	if !strings.Contains(err.Error(), "at least 2 arg") {
		t.Errorf("expected 'at least 2 arg' error, got %q", err.Error())
	}
}

func TestAttachmentUploadRequiresFiles(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentUploadCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"42"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when files not provided")
	}
	if !strings.Contains(err.Error(), "at least 2 arg") {
		t.Errorf("expected 'at least 2 arg' error, got %q", err.Error())
	}
}

func TestAttachmentUploadInvalidIssueID(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentUploadCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"notanumber", "file.txt"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-numeric issue ID")
	}
	if !strings.Contains(err.Error(), "invalid issue ID") {
		t.Errorf("expected 'invalid issue ID' error, got %q", err.Error())
	}
}

func TestAttachmentUploadNonExistentFile(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentUploadCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"42", "/nonexistent/path/file.txt"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "file not found") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("expected file not found error, got %q", err.Error())
	}
}

func TestAttachmentUploadDirectory(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentUploadCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// Use a directory that exists on the system
	cmd.SetArgs([]string{"42", "/tmp"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when trying to upload a directory")
	}
	if !strings.Contains(err.Error(), "cannot upload directory") {
		t.Errorf("expected 'cannot upload directory' error, got %q", err.Error())
	}
}

// --- Attachment Download Command Tests ---

func TestAttachmentDownloadRequiresIssueID(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentDownloadCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when issue ID not provided")
	}
}

func TestAttachmentDownloadInvalidIssueID(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentDownloadCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"notanumber"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-numeric issue ID")
	}
	if !strings.Contains(err.Error(), "invalid issue ID") {
		t.Errorf("expected 'invalid issue ID' error, got %q", err.Error())
	}
}

func TestAttachmentDownloadRequiresFilenameOrFlags(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentDownloadCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"42"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no filename or flags provided")
	}
	if !strings.Contains(err.Error(), "specify a filename") {
		t.Errorf("expected 'specify a filename' error, got %q", err.Error())
	}
}

func TestAttachmentDownloadConflictingOptions(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		errorContains string
	}{
		{
			name:          "filename with --all",
			args:          []string{"42", "file.txt", "--all"},
			errorContains: "cannot use --all",
		},
		{
			name:          "filename with --pattern",
			args:          []string{"42", "file.txt", "--pattern", "*.txt"},
			errorContains: "cannot use --all or --pattern",
		},
		{
			name:          "--output with --all",
			args:          []string{"42", "--all", "--output", "out.txt"},
			errorContains: "--output can only be used",
		},
		{
			name:          "--output with --pattern",
			args:          []string{"42", "--pattern", "*.txt", "--output", "out.txt"},
			errorContains: "--output can only be used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestFactory()
			cmd := newAttachmentDownloadCmd(f)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error for conflicting options")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
			}
		})
	}
}

// --- Attachment Delete Command Tests ---

func TestAttachmentDeleteRequiresIssueIDAndFilename(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentDeleteCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when arguments not provided")
	}
	if !strings.Contains(err.Error(), "accepts 2 arg") {
		t.Errorf("expected 'accepts 2 arg' error, got %q", err.Error())
	}
}

func TestAttachmentDeleteRequiresFilename(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentDeleteCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"42"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when filename not provided")
	}
	if !strings.Contains(err.Error(), "accepts 2 arg") {
		t.Errorf("expected 'accepts 2 arg' error, got %q", err.Error())
	}
}

func TestAttachmentDeleteInvalidIssueID(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentDeleteCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"notanumber", "file.txt"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-numeric issue ID")
	}
	if !strings.Contains(err.Error(), "invalid issue ID") {
		t.Errorf("expected 'invalid issue ID' error, got %q", err.Error())
	}
}

// --- Parent Command Tests ---

func TestAttachmentCommandHasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentCmd(f)

	subcommands := cmd.Commands()
	if len(subcommands) != 4 {
		t.Errorf("expected 4 subcommands, got %d", len(subcommands))
	}

	expectedNames := map[string]bool{
		"list":     false,
		"upload":   false,
		"download": false,
		"delete":   false,
	}

	for _, sub := range subcommands {
		if _, ok := expectedNames[sub.Name()]; ok {
			expectedNames[sub.Name()] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestAttachmentCommandHasAlias(t *testing.T) {
	f := newTestFactory()
	cmd := newAttachmentCmd(f)

	if len(cmd.Aliases) == 0 {
		t.Fatal("expected at least one alias")
	}

	hasAttachAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "attach" {
			hasAttachAlias = true
			break
		}
	}
	if !hasAttachAlias {
		t.Errorf("expected 'attach' alias, got %v", cmd.Aliases)
	}
}
