package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// FlowOptions configures the OAuth authorization code flow.
type FlowOptions struct {
	// ClientID is the OAuth consumer key.
	ClientID string
	// ClientSecret is the OAuth consumer secret.
	ClientSecret string
	// AuthorizeURL is the authorization endpoint.
	AuthorizeURL string
	// TokenURL is the token exchange endpoint.
	TokenURL string
	// UserInfoURL is the endpoint used to fetch the authenticated username.
	// Defaults to https://api.bitbucket.org/2.0/user when empty.
	UserInfoURL string
	// Scopes to request.
	Scopes []string
	// Out is the writer for user-facing messages. Defaults to os.Stdout.
	Out io.Writer
	// OpenBrowser opens the given URL in the user's browser.
	// When nil the URL is printed but not opened.
	OpenBrowser func(string) error
}

// FlowResult contains the outcome of a successful OAuth flow.
type FlowResult struct {
	Token       *Token
	Username    string
	DisplayName string
}

// RunFlow executes the full OAuth authorization code flow:
//  1. Start localhost callback server
//  2. Build authorize URL with state
//  3. Open browser (or print URL)
//  4. Wait for callback with auth code
//  5. Exchange code for tokens
//  6. Fetch authenticated username via /2.0/user
func RunFlow(ctx context.Context, opts FlowOptions) (*FlowResult, error) {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	srv, err := NewCallbackServer()
	if err != nil {
		return nil, err
	}
	defer srv.Close()

	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	pkce, err := generatePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate pkce: %w", err)
	}

	authURL := buildAuthorizeURL(opts, srv.RedirectURI(), state, pkce.challenge)

	if opts.OpenBrowser != nil {
		if err := opts.OpenBrowser(authURL); err != nil {
			fmt.Fprintf(out, "Failed to open browser: %v\n", err)
			fmt.Fprintf(out, "Open this URL manually:\n  %s\n", authURL)
		}
	} else {
		fmt.Fprintf(out, "Open this URL in your browser:\n  %s\n", authURL)
	}

	code, callbackState, err := srv.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("authorization: %w", err)
	}

	if callbackState != state {
		return nil, fmt.Errorf("state mismatch: possible CSRF attack")
	}

	tok, err := exchangeCode(ctx, opts, code, srv.RedirectURI(), pkce.verifier)
	if err != nil {
		return nil, err
	}

	userInfoURL := opts.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = "https://api.bitbucket.org/2.0/user"
	}
	username, displayName, err := fetchUsername(ctx, tok.AccessToken, userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	return &FlowResult{Token: tok, Username: username, DisplayName: displayName}, nil
}

// RefreshToken exchanges a refresh token for a new access token.
func RefreshToken(ctx context.Context, refreshToken, clientID, clientSecret, tokenURL string) (*Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	return doTokenRequest(req)
}

func buildAuthorizeURL(opts FlowOptions, redirectURI, state, codeChallenge string) string {
	v := url.Values{
		"client_id":             {opts.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	if len(opts.Scopes) > 0 {
		v.Set("scope", strings.Join(opts.Scopes, " "))
	}
	return opts.AuthorizeURL + "?" + v.Encode()
}

func exchangeCode(ctx context.Context, opts FlowOptions, code, redirectURI, codeVerifier string) (*Token, error) {
	// Bitbucket requires redirect_uri at both authorize and token exchange
	// when it was included at authorize. Must be the exact same value.
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", opts.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(opts.ClientID, opts.ClientSecret)

	return doTokenRequest(req)
}

func doTokenRequest(req *http.Request) (*Token, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := errResp.Description
		if msg == "" {
			msg = errResp.Error
		}
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("token exchange: %s", msg)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("token exchange: empty access_token in response")
	}

	return FromResponse(tokenResp.AccessToken, tokenResp.RefreshToken, tokenResp.ExpiresIn), nil
}

// fetchUsername returns (username, displayName, error). Username is the
// API-safe identifier (username or account_id); displayName is for UX only.
func fetchUsername(ctx context.Context, accessToken, userInfoURL string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GET %s: %s", userInfoURL, resp.Status)
	}

	var user struct {
		Username    string `json:"username"`
		AccountID   string `json:"account_id"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", "", fmt.Errorf("decode user: %w", err)
	}

	username := user.Username
	if username == "" {
		username = user.AccountID
	}
	if username == "" {
		return "", "", fmt.Errorf("user response missing both username and account_id")
	}
	return username, user.DisplayName, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type pkceParams struct {
	verifier  string
	challenge string
}

// generatePKCE creates a PKCE verifier and S256 challenge per RFC 7636.
// The verifier is 43 base64url characters (ceil(32×8/6)=43, no padding).
// The challenge is base64url(SHA-256(verifier)) without padding.
func generatePKCE() (pkceParams, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return pkceParams{}, fmt.Errorf("generate pkce verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return pkceParams{verifier: verifier, challenge: challenge}, nil
}
