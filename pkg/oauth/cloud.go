package oauth

import (
	"errors"
	"fmt"
	"os"
)

// Cloud OAuth 2.0 configuration for Bitbucket Cloud.
//
// Bitbucket Cloud has no public-client or device grant: every
// authorization_code and refresh_token exchange needs a consumer key and
// secret. bkt is a native app, so its consumer secret cannot be kept
// confidential (RFC 8252 §8.5); it ships in source like the consumers of the
// GitHub CLI and Git Credential Manager, and every build (release, Homebrew,
// go install, Nix) can log in. The consumer is marked public on Bitbucket, so
// the secret cannot mint client_credentials tokens. bkt sends PKCE, which
// Bitbucket Cloud ignores today.
const (
	// CloudAuthorizeURL is the Bitbucket Cloud authorization endpoint.
	CloudAuthorizeURL = "https://bitbucket.org/site/oauth2/authorize"

	// CloudTokenURL is the Bitbucket Cloud token exchange endpoint.
	CloudTokenURL = "https://bitbucket.org/site/oauth2/access_token"
)

// The built-in consumer. They are variables so distributors can substitute
// their own with -ldflags -X.
var (
	cloudClientID     = "yY6jbCWzpHNbsZSd4X"
	cloudClientSecret = "UTTZHd5N8yPZmnXxvaDjJDSHt2dcA6YA" // gitleaks:allow
)

// CloudClient is a Bitbucket Cloud OAuth consumer.
type CloudClient struct {
	ID     string
	Secret string
}

// ResolveCloudClient returns the consumer for a new login: the
// BKT_OAUTH_CLIENT_ID / BKT_OAUTH_CLIENT_SECRET pair when set, otherwise the
// built-in consumer.
func ResolveCloudClient() (CloudClient, error) {
	id, secret := os.Getenv("BKT_OAUTH_CLIENT_ID"), os.Getenv("BKT_OAUTH_CLIENT_SECRET")
	switch {
	case id == "" && secret == "":
		return CloudClient{ID: cloudClientID, Secret: cloudClientSecret}, nil
	case id == "" || secret == "":
		return CloudClient{}, errors.New("set both BKT_OAUTH_CLIENT_ID and BKT_OAUTH_CLIENT_SECRET to use your own OAuth consumer, or neither to use bkt's")
	}
	return CloudClient{ID: id, Secret: secret}, nil
}

// CloudClientFor returns the consumer that can refresh tok. Bitbucket binds a
// refresh token to the consumer that issued it.
func CloudClientFor(tok *Token) (CloudClient, error) {
	c, err := ResolveCloudClient()
	if err != nil {
		return CloudClient{}, err
	}
	switch tok.ClientID {
	case "", c.ID:
		// Tokens stored before bkt recorded the consumer refresh with the
		// resolved one, as they always did.
		return c, nil
	case cloudClientID:
		return CloudClient{ID: cloudClientID, Secret: cloudClientSecret}, nil
	}
	return CloudClient{}, fmt.Errorf("this OAuth token was issued to consumer %s; set BKT_OAUTH_CLIENT_ID and BKT_OAUTH_CLIENT_SECRET to that consumer, or run `bkt auth login https://bitbucket.org --kind cloud --web` again", tok.ClientID)
}

// CloudScopes returns the OAuth scopes requested during authorization.
// Scopes cover the full Cloud command set: repos, PRs, issues, pipelines,
// pipeline variables, and webhooks.
func CloudScopes() []string {
	return []string{"account", "repository", "pullrequest", "issue", "pipeline", "webhook"}
}
