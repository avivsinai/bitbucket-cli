package cmdutil

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid https", "https://example.com", "https://example.com", false},
		{"trailing slash", "https://example.com/", "https://example.com", false},
		{"missing scheme", "example.com", "https://example.com", false},
		{"with path", "https://example.com/bitbucket", "https://example.com/bitbucket", false},
		{"with trailing slash and path", "https://example.com/bitbucket/", "https://example.com/bitbucket", false},
		{"http scheme", "http://example.com", "http://example.com", false},
		{"with query", "https://example.com?foo=bar", "https://example.com", false},
		{"with fragment", "https://example.com#section", "https://example.com", false},
		{"whitespace", "  https://example.com  ", "https://example.com", false},
		{"empty", "", "", true},
		{"only spaces", "   ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeBaseURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeBaseURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHostKeyFromURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"standard https", "https://bitbucket.example.com", "bitbucket.example.com", false},
		{"with port", "https://bitbucket.example.com:7990", "bitbucket.example.com:7990", false},
		{"with path", "https://bitbucket.example.com/context", "bitbucket.example.com", false},
		{"http", "http://localhost:7990", "localhost:7990", false},
		{"no scheme", "example.com", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HostKeyFromURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("HostKeyFromURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HostKeyFromURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
