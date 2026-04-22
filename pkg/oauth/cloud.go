package oauth

import "os"

// Cloud OAuth 2.0 configuration for Bitbucket Cloud.
//
// The flow uses PKCE (RFC 7636, S256) for defense-in-depth. Bitbucket Cloud
// still requires a client secret for authorization_code and refresh_token
// exchanges, so bkt reads the consumer credentials from runtime environment
// variables instead of embedding them in public release binaries.
const (
	// CloudAuthorizeURL is the Bitbucket Cloud authorization endpoint.
	CloudAuthorizeURL = "https://bitbucket.org/site/oauth2/authorize"

	// CloudTokenURL is the Bitbucket Cloud token exchange endpoint.
	CloudTokenURL = "https://bitbucket.org/site/oauth2/access_token"
)

// CloudClientID returns the OAuth consumer key.
func CloudClientID() string {
	return os.Getenv("BKT_OAUTH_CLIENT_ID")
}

// CloudClientSecret returns the OAuth consumer secret.
func CloudClientSecret() string {
	return os.Getenv("BKT_OAUTH_CLIENT_SECRET")
}

// CloudScopes returns the OAuth scopes requested during authorization.
// Scopes cover the full Cloud command set: repos, PRs, issues, pipelines,
// pipeline variables, and webhooks.
func CloudScopes() []string {
	return []string{"account", "repository", "pullrequest", "issue", "pipeline", "webhook"}
}
