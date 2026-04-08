package oauth

import (
	"testing"
	"time"
)

func TestFromResponse(t *testing.T) {
	before := time.Now()
	tok := FromResponse("acc", "ref", 7200)
	after := time.Now()

	if tok.AccessToken != "acc" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "acc")
	}
	if tok.RefreshToken != "ref" {
		t.Errorf("RefreshToken = %q, want %q", tok.RefreshToken, "ref")
	}
	if tok.ExpiresAt.Before(before.Add(7200 * time.Second)) {
		t.Error("ExpiresAt is before expected minimum")
	}
	if tok.ExpiresAt.After(after.Add(7200 * time.Second)) {
		t.Error("ExpiresAt is after expected maximum")
	}
}

func TestIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired — far future",
			expiresAt: time.Now().Add(time.Hour),
			want:      false,
		},
		{
			name:      "expired — past",
			expiresAt: time.Now().Add(-time.Minute),
			want:      true,
		},
		{
			name:      "expired — within 30s grace period",
			expiresAt: time.Now().Add(20 * time.Second),
			want:      true,
		},
		{
			name:      "not expired — just outside grace period",
			expiresAt: time.Now().Add(31 * time.Second),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := &Token{ExpiresAt: tt.expiresAt}
			if got := tok.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	original := &Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		ExpiresAt:    time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC),
	}

	blob, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if blob[0] != '{' {
		t.Errorf("Marshal output should start with '{', got %q", blob[:1])
	}

	decoded, err := Unmarshal(blob)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", decoded.AccessToken, original.AccessToken)
	}
	if decoded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", decoded.RefreshToken, original.RefreshToken)
	}
	if !decoded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", decoded.ExpiresAt, original.ExpiresAt)
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	_, err := Unmarshal("not-json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestIsTokenBlob(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{`{"access_token":"x"}`, true},
		{`{}`, true},
		{`plain-token-string`, false},
		{``, false},
		{`Bearer token`, false},
	}
	for _, tt := range tests {
		if got := IsTokenBlob(tt.value); got != tt.want {
			t.Errorf("IsTokenBlob(%q) = %v, want %v", tt.value, got, tt.want)
		}
	}
}
