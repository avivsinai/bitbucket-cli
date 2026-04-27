package oauth

import "os"

// Cloud OAuth 2.0 configuration for Bitbucket Cloud.
//
// The flow uses PKCE (RFC 7636, S256) for defense-in-depth. Bitbucket Cloud
// still requires a client secret for authorization_code and refresh_token
// exchanges. Official bkt release binaries embed the CLI's OAuth consumer
// credentials so browser login works out of the box; source and Nix builds can
// provide their own credentials through environment variables.
const (
	// CloudAuthorizeURL is the Bitbucket Cloud authorization endpoint.
	CloudAuthorizeURL = "https://bitbucket.org/site/oauth2/authorize"

	// CloudTokenURL is the Bitbucket Cloud token exchange endpoint.
	CloudTokenURL = "https://bitbucket.org/site/oauth2/access_token"
)

// CloudClientID and CloudClientSecret are injected at build time via ldflags.
// Environment variables remain the fallback for source, Nix, and development
// builds that do not carry the official release credentials.
var (
	cloudClientID     string // set via -ldflags -X
	cloudClientSecret string // set via -ldflags -X
)

// CloudClientID returns the OAuth consumer key.
func CloudClientID() string {
	if cloudClientID != "" {
		return cloudClientID
	}
	return os.Getenv("BKT_OAUTH_CLIENT_ID")
}

// CloudClientSecret returns the OAuth consumer secret.
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
