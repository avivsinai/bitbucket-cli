package cmdutil

import "testing"

func TestValidateGitPositionalArg(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		description string
		wantErr     string
	}{
		{
			name:        "accepts normal url",
			value:       "https://bitbucket.example.com/scm/PROJ/repo.git",
			description: "clone URL",
		},
		{
			name:        "rejects empty value",
			value:       "",
			description: "repository",
			wantErr:     "repository is required",
		},
		{
			name:        "rejects option like value",
			value:       "--upload-pack=evil",
			description: "fork clone URL",
			wantErr:     `invalid fork clone URL "--upload-pack=evil": must not start with '-'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitPositionalArg(tt.value, tt.description)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateGitPositionalArg returned error: %v", err)
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
