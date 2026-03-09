package cmdutil

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeBaseURL ensures the Bitbucket base URL includes a scheme and has no
// trailing slash. HTTP URLs are rejected by default; use AllowHTTP to permit them.
func NormalizeBaseURL(raw string) (string, error) {
	return normalizeBaseURL(raw, false)
}

// NormalizeBaseURLAllowHTTP is like NormalizeBaseURL but permits http:// URLs.
// Callers should warn the user that credentials will be transmitted in plaintext.
func NormalizeBaseURLAllowHTTP(raw string) (string, error) {
	return normalizeBaseURL(raw, true)
}

func normalizeBaseURL(raw string, allowHTTP bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("host is required")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Scheme == "http" && !allowHTTP {
		return "", fmt.Errorf("http:// URLs are not allowed (credentials would be sent in plaintext); use --allow-http to override")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

// HostKeyFromURL resolves the host component used as the configuration key.
func HostKeyFromURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid base URL %q", baseURL)
	}
	return u.Host, nil
}
