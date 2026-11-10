package variable

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateVariableKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid simple key",
			key:     "MY_VAR",
			wantErr: false,
		},
		{
			name:    "valid lowercase key",
			key:     "my_var",
			wantErr: false,
		},
		{
			name:    "valid mixed case key",
			key:     "MyVar",
			wantErr: false,
		},
		{
			name:    "valid key with numbers",
			key:     "VAR123",
			wantErr: false,
		},
		{
			name:    "valid single letter",
			key:     "A",
			wantErr: false,
		},
		{
			name:    "valid key starting with underscore after letter",
			key:     "A_B_C",
			wantErr: false,
		},
		{
			name:        "empty key",
			key:         "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "key starting with number",
			key:         "123VAR",
			wantErr:     true,
			errContains: "must start with a letter",
		},
		{
			name:        "key starting with underscore",
			key:         "_MY_VAR",
			wantErr:     true,
			errContains: "must start with a letter",
		},
		{
			name:        "key with hyphen",
			key:         "MY-VAR",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "key with space",
			key:         "MY VAR",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "key with special character",
			key:         "MY$VAR",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "key with dot",
			key:         "MY.VAR",
			wantErr:     true,
			errContains: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVariableKey(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" {
					if !containsString(err.Error(), tt.errContains) {
						t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
					}
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseEnvFile(t *testing.T) {
	// Note: Test data uses obviously fake variable names and values.
	// These are NOT real credentials - they are test fixtures for env file parsing.
	tests := []struct {
		name        string
		content     string
		want        map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:    "simple key value pairs",
			content: "FOO=bar\nBAZ=qux",
			want: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			name:    "with comments",
			content: "# This is a comment\nFOO=bar\n# Another comment\nBAZ=qux",
			want: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			name:    "with empty lines",
			content: "FOO=bar\n\n\nBAZ=qux\n",
			want: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			name:    "with double quotes",
			content: "FOO=\"hello world\"",
			want: map[string]string{
				"FOO": "hello world",
			},
		},
		{
			name:    "with single quotes",
			content: "FOO='hello world'",
			want: map[string]string{
				"FOO": "hello world",
			},
		},
		{
			name:    "empty value",
			content: "FOO=",
			want: map[string]string{
				"FOO": "",
			},
		},
		{
			name:    "value with equals sign",
			content: "FOO=a=b=c",
			want: map[string]string{
				"FOO": "a=b=c",
			},
		},
		{
			name:    "whitespace around key",
			content: "  FOO  =bar",
			want: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:    "leading/trailing whitespace in line",
			content: "  FOO=bar  \n  BAZ=qux  ",
			want: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			name:        "missing equals sign",
			content:     "FOO bar",
			wantErr:     true,
			errContains: "invalid format",
		},
		{
			name:        "empty key",
			content:     "=bar",
			wantErr:     true,
			errContains: "empty key",
		},
		{
			name:    "empty file",
			content: "",
			want:    map[string]string{},
		},
		{
			name:    "only comments and empty lines",
			content: "# comment 1\n\n# comment 2\n",
			want:    map[string]string{},
		},
		{
			name:    "value with hash not a comment",
			content: "FOO=bar#baz#qux",
			want: map[string]string{
				"FOO": "bar#baz#qux",
			},
		},
		{
			name:    "quoted value with mixed chars",
			content: "FOO=\"hello 'world' and more\"",
			want: map[string]string{
				"FOO": "hello 'world' and more",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temp file with the content
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.env")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			entries, err := parseEnvFile(tmpFile)
			got := make(map[string]string, len(entries))
			for _, e := range entries {
				got[e.Key] = e.Value
			}
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" {
					if !containsString(err.Error(), tt.errContains) {
						t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Errorf("got %d entries, want %d", len(got), len(tt.want))
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParseEnvFileNotFound(t *testing.T) {
	_, err := parseEnvFile("/nonexistent/path/to/file.env")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
	if !containsString(err.Error(), "failed to open") {
		t.Errorf("expected error about opening file, got %q", err.Error())
	}
}

func TestScopeConstants(t *testing.T) {
	// Verify scope constants have expected values
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"repository", scopeRepository, "repository"},
		{"workspace", scopeWorkspace, "workspace"},
		{"deployment", scopeDeployment, "deployment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("scope constant %s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// containsString is a helper to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// `echo secret| bkt variable set TOKEN` on Windows pipes "secret\r\n"; the
// stored value used to keep the "\r".
func TestStdinValueDropsOneLineEnding(t *testing.T) {
	for in, want := range map[string]string{
		"secret\r\n": "secret",
		"secret\n":   "secret",
		"secret":     "secret",
		"secret\r":   "secret\r",
		"a\nb\n\n":   "a\nb\n",
	} {
		if got := stdinValue([]byte(in)); got != want {
			t.Errorf("stdinValue(%q) = %q, want %q", in, got, want)
		}
	}
}

// `variable set --env-file` applied variables in random map order, so a failed
// import left an unpredictable subset applied.
func TestParseEnvFileKeepsFileOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.env")
	if err := os.WriteFile(path, []byte("ZED=1\n# c\nALPHA=2\nMID=3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []envEntry{{"ZED", "1"}, {"ALPHA", "2"}, {"MID", "3"}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
