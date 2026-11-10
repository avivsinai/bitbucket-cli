package bktcmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// Before the fix, Ctrl-C during `pipeline run --wait` exited 0 and any other
// command exited 1 with "Error: context canceled".
func TestExitCode(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want int
	}{
		{"cancelled, nil error", cancelled, nil, 130},
		{"cancelled, context.Canceled", cancelled, context.Canceled, 130},
		{"live, pending", context.Background(), cmdutil.ErrPending, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if got := exitCode(tt.ctx, tt.err, &out); got != tt.want {
				t.Fatalf("exitCode = %d, want %d", got, tt.want)
			}
			if tt.ctx.Err() != nil && out.Len() != 0 {
				t.Fatalf("expected no output, got %q", out.String())
			}
		})
	}
}
