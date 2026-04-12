package oauth

import "os"

// Cloud OAuth 2.0 configuration for Bitbucket Cloud.
//
// Bitbucket Cloud does not support PKCE for OAuth consumers, so the
// client_secret is embedded in the binary at build time via ldflags.
// This is the same trade-off made by tools like gh (GitHub CLI).
const (
	// CloudAuthorizeURL is the Bitbucket Cloud authorization endpoint.
	CloudAuthorizeURL = "https://bitbucket.org/site/oauth2/authorize"

	// CloudTokenURL is the Bitbucket Cloud token exchange endpoint.
	CloudTokenURL = "https://bitbucket.org/site/oauth2/access_token"
)

// CloudClientID and CloudClientSecret are injected at build time via ldflags.
// For source installs (go install) that lack ldflags, they fall back to the
// BKT_OAUTH_CLIENT_ID / BKT_OAUTH_CLIENT_SECRET environment variables.
var (
	cloudClientID     string // set via -ldflags -X
	cloudClientSecret string // set via -ldflags -X
)

// CloudClientID returns the OAuth consumer key.
// Prefers the build-time value; falls back to BKT_OAUTH_CLIENT_ID env var.
func CloudClientID() string {
	if cloudClientID != "" {
		return cloudClientID
	}
	return os.Getenv("BKT_OAUTH_CLIENT_ID")
}

// CloudClientSecret returns the OAuth consumer secret.
// Prefers the build-time value; falls back to BKT_OAUTH_CLIENT_SECRET env var.
func CloudClientSecret() string {
	if cloudClientSecret != "" {
		return cloudClientSecret
	}
	return os.Getenv("BKT_OAUTH_CLIENT_SECRET")
}

// CloudScopes returns the OAuth scopes requested during authorization.
// Scopes cover the full Cloud command set: repos, PRs, issues, pipelines,
// pipeline variables, and webhooks.
func CloudScopes() []string {
	return []string{"account", "repository", "pullrequest", "issue", "pipeline", "webhook"}
}
