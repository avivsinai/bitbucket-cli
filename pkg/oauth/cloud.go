package oauth

// Cloud OAuth 2.0 configuration for Bitbucket Cloud.
//
// Bitbucket Cloud does not support PKCE for OAuth consumers, so the
// client_secret is embedded in the binary. This is the same trade-off made
// by tools like gh (GitHub CLI). The OAuth consumer must be registered in a
// bkt-owned Bitbucket Cloud workspace with the scopes listed in CloudScopes.
const (
	// CloudAuthorizeURL is the Bitbucket Cloud authorization endpoint.
	CloudAuthorizeURL = "https://bitbucket.org/site/oauth2/authorize"

	// CloudTokenURL is the Bitbucket Cloud token exchange endpoint.
	CloudTokenURL = "https://bitbucket.org/site/oauth2/access_token"

	// CloudClientID is the OAuth consumer key registered in the bkt workspace.
	// Replace with the actual client_id after registering the OAuth consumer.
	CloudClientID = "REPLACE_WITH_CLIENT_ID" // gitleaks:allow

	// CloudClientSecret is the OAuth consumer secret. Bitbucket Cloud does not
	// support PKCE so this ships in the binary (same trade-off as gh).
	// Replace with the actual client_secret after registering the OAuth consumer.
	CloudClientSecret = "REPLACE_WITH_CLIENT_SECRET" // gitleaks:allow
)

// CloudScopes returns the OAuth scopes requested during authorization.
// Scopes cover the full Cloud command set: repos, PRs, issues, pipelines,
// pipeline variables, and webhooks.
func CloudScopes() []string {
	return []string{"account", "repository", "pullrequest", "issue", "pipeline", "webhook"}
}
