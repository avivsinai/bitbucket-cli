package status

import (
	"bytes"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Without rate-limit headers the text output used to print "Limit: 0 /
// Remaining: 0", which looks like an exhausted limit.
func TestRateLimitTextWithoutHeaders(t *testing.T) {
	var out bytes.Buffer
	if err := writeRateLimitText(&out, "https://bitbucket.example.com/rest", httpx.RateLimit{}); err != nil {
		t.Fatal(err)
	}
	want := "No rate-limit headers returned by bitbucket.example.com.\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}
