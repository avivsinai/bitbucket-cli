package pr

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"fits exactly", "hi", 5, "hi"},
		{"truncates with ellipsis", "hello!", 4, "h..."},
		{"newline normalised", "a\nb", 10, "a b"},
		// depth 40: max(80-2*40, 4) = 4 — must not panic
		{"depth 40 clamp short text", "x", max(80-2*40, 4), "x"},
		{"depth 40 clamp long text", "abcdefgh", max(80-2*40, 4), "a..."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncate(tc.s, tc.maxLen)
			if got != tc.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.s, tc.maxLen, got, tc.want)
			}
		})
	}
}
