package oauth

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Token holds an OAuth 2.0 access/refresh token pair with expiry.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// tokenRefreshGrace is subtracted from ExpiresAt when checking expiry so tokens
// are refreshed slightly before the server considers them stale.
const tokenRefreshGrace = 30 * time.Second

// FromResponse constructs a Token from OAuth token endpoint response fields.
// expiresIn is the number of seconds until the access token expires.
func FromResponse(accessToken, refreshToken string, expiresIn int) *Token {
	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
}

// IsExpired reports whether the access token should be considered expired,
// accounting for a grace period to allow proactive refresh.
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-tokenRefreshGrace))
}

// Marshal encodes the token as a JSON string suitable for keyring storage.
func (t *Token) Marshal() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Unmarshal decodes a Token from a JSON string produced by Marshal.
func Unmarshal(s string) (*Token, error) {
	var t Token
	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return nil, err
	}
	if strings.TrimSpace(t.AccessToken) == "" {
		return nil, errors.New("missing access_token")
	}
	if strings.TrimSpace(t.RefreshToken) == "" {
		return nil, errors.New("missing refresh_token")
	}
	if t.ExpiresAt.IsZero() {
		return nil, errors.New("missing expires_at")
	}
	return &t, nil
}

// IsTokenBlob reports whether a keyring value is a complete OAuth JSON token
// blob rather than a plain API token string.
func IsTokenBlob(value string) bool {
	_, err := Unmarshal(value)
	return err == nil
}
