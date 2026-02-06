package secret

import (
	"testing"
	"time"
)

func TestParseTimeoutEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want time.Duration
		ok   bool
	}{
		{name: "empty", raw: "", want: 0, ok: false},
		{name: "duration_seconds", raw: "60s", want: 60 * time.Second, ok: true},
		{name: "duration_minutes", raw: "2m", want: 2 * time.Minute, ok: true},
		{name: "plain_seconds", raw: "60", want: 60 * time.Second, ok: true},
		{name: "zero", raw: "0", want: 0, ok: false},
		{name: "negative", raw: "-1", want: 0, ok: false},
		{name: "garbage", raw: "nope", want: 0, ok: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseTimeoutEnv(tt.raw)
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v (got=%v)", ok, tt.ok, got)
			}
			if ok && got != tt.want {
				t.Fatalf("got=%v want %v", got, tt.want)
			}
		})
	}
}

func TestKeyringTimeout_EnvOverride(t *testing.T) {
	t.Setenv(envTimeout, "2m")
	if got := keyringTimeout(); got != 2*time.Minute {
		t.Fatalf("got=%v want %v", got, 2*time.Minute)
	}
}
