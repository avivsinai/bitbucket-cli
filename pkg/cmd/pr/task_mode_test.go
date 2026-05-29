package pr

import (
	"fmt"
	"testing"
)

func TestDCSupportsBlockerComments(t *testing.T) {
	tests := []struct {
		version string
		want    bool
		wantErr bool
	}{
		{"7.2.0", true, false},
		{"7.1.9", false, false},
		{"7.21.0", true, false},
		{"8.19.1", true, false},
		{"6.10.0", false, false},
		{"7.2", true, false},
		{"7", false, false},
		{"not-a-version", false, true},
		{"", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := dcSupportsBlockerComments(tt.version)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("dcSupportsBlockerComments(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestResolveDCTaskMode(t *testing.T) {
	ok := func(v string) func() (string, error) { return func() (string, error) { return v, nil } }
	fail := func() (string, error) { return "", fmt.Errorf("network down") }

	tests := []struct {
		name     string
		mode     string
		lookup   func() (string, error)
		mutating bool
		want     string
		wantErr  bool
	}{
		{"explicit blocker-comments skips probe", taskAPIBlockerComments, fail, true, taskAPIBlockerComments, false},
		{"explicit legacy skips probe", taskAPILegacy, fail, true, taskAPILegacy, false},
		{"auto modern", taskAPIAuto, ok("8.19.1"), true, taskAPIBlockerComments, false},
		{"auto old -> legacy", taskAPIAuto, ok("7.1.0"), true, taskAPILegacy, false},
		{"auto exactly 7.2", taskAPIAuto, ok("7.2.0"), false, taskAPIBlockerComments, false},
		{"auto probe fails + mutating -> error", taskAPIAuto, fail, true, "", true},
		{"auto probe fails + read -> blocker-comments", taskAPIAuto, fail, false, taskAPIBlockerComments, false},
		{"auto unparseable + mutating -> error", taskAPIAuto, ok("garbage"), true, "", true},
		{"auto unparseable + read -> blocker-comments", taskAPIAuto, ok("garbage"), false, taskAPIBlockerComments, false},
		{"empty defaults to auto", "", ok("8.0.0"), true, taskAPIBlockerComments, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDCTaskMode(tt.mode, tt.lookup, tt.mutating)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("resolveDCTaskMode(%q, _, %v) = %q, want %q", tt.mode, tt.mutating, got, tt.want)
			}
		})
	}
}

func TestValidateTaskAPIMode(t *testing.T) {
	for _, m := range []string{taskAPIAuto, taskAPIBlockerComments, taskAPILegacy} {
		if err := validateTaskAPIMode(m); err != nil {
			t.Errorf("validateTaskAPIMode(%q) unexpected error: %v", m, err)
		}
	}
	if err := validateTaskAPIMode("nonsense"); err == nil {
		t.Error("expected error for invalid mode")
	}
}
