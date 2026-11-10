package pager

import (
	"bytes"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// PAGER=" " used to crash with an index-out-of-range panic.
func TestStartWithBlankPagerWritesToStdout(t *testing.T) {
	t.Setenv("BKT_PAGER", "")
	t.Setenv("PAGER", " ")
	var out bytes.Buffer
	p := &system{ios: &iostreams.IOStreams{Out: &out}}

	w, err := p.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := w.Write([]byte("diff")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if out.String() != "diff" {
		t.Fatalf("stdout = %q, want %q", out.String(), "diff")
	}
}
