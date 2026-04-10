package branch

import "testing"

func TestMapProtectType(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no-creates lowercase", "no-creates", "NO_CREATES"},
		{"no-creates uppercase", "NO-CREATES", "NO_CREATES"},
		{"no-creates mixed case", "No-Creates", "NO_CREATES"},
		{"no-deletes", "no-deletes", "NO_DELETES"},
		{"fast-forward-only", "fast-forward-only", "FAST_FORWARD_ONLY"},
		{"require-approvals maps to PULL_REQUEST", "require-approvals", "PULL_REQUEST"},
		{"unknown type returns empty", "bogus", ""},
		{"empty string returns empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapProtectType(tc.in); got != tc.want {
				t.Errorf("mapProtectType(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEnsureBranchRef(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty becomes wildcard", "", "refs/heads/*"},
		{"bare branch gets prefixed", "main", "refs/heads/main"},
		{"slashed branch gets prefixed", "feature/login", "refs/heads/feature/login"},
		{"refs/heads left intact", "refs/heads/main", "refs/heads/main"},
		{"refs/tags left intact", "refs/tags/v1.0.0", "refs/tags/v1.0.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ensureBranchRef(tc.in); got != tc.want {
				t.Errorf("ensureBranchRef(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
