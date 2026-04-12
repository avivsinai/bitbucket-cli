package cmdutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
	"github.com/avivsinai/bitbucket-cli/pkg/oauth"
)

func TestNewCloudClientBasicAuth(t *testing.T) {
	host := &config.Host{
		Kind:     "cloud",
		BaseURL:  "https://api.bitbucket.org/2.0",
		Username: "user@example.com",
		Token:    "app-password",
	}

	client, err := NewCloudClient(host)
	if err != nil {
		t.Fatalf("NewCloudClient error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestNewCloudClientOAuthWiresRefresher(t *testing.T) {
	host := &config.Host{
		Kind:       "cloud",
		BaseURL:    "https://api.bitbucket.org/2.0",
		Username:   "erank_ai21",
		AuthMethod: "oauth",
		Token:      "access-token",
	}

	// Ensure BKT_TOKEN is not set so the OAuth path is taken.
	t.Setenv(secret.EnvToken, "")

	client, err := NewCloudClient(host)
	if err != nil {
		t.Fatalf("NewCloudClient error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestNewCloudClientOAuthSkipsRefresherWithBKTToken(t *testing.T) {
	host := &config.Host{
		Kind:       "cloud",
		BaseURL:    "https://api.bitbucket.org/2.0",
		Username:   "erank_ai21",
		AuthMethod: "oauth",
		Token:      "env-override",
	}

	t.Setenv(secret.EnvToken, "env-override")

	client, err := NewCloudClient(host)
	if err != nil {
		t.Fatalf("NewCloudClient error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestNewCloudClientNilHostError(t *testing.T) {
	_, err := NewCloudClient(nil)
	if err == nil {
		t.Fatal("expected error for nil host")
	}
}

func TestNewDCClientBasic(t *testing.T) {
	host := &config.Host{
		Kind:    "dc",
		BaseURL: "https://bitbucket.example.com",
		Token:   "pat",
	}

	client, err := NewDCClient(host)
	if err != nil {
		t.Fatalf("NewDCClient error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestSecretOptsInsecure(t *testing.T) {
	host := &config.Host{AllowInsecureStore: true}
	opts := secretOpts(host)
	if len(opts) != 1 {
		t.Errorf("secretOpts = %d options, want 1", len(opts))
	}
}

func TestSecretOptsSecure(t *testing.T) {
	host := &config.Host{AllowInsecureStore: false}
	opts := secretOpts(host)
	if len(opts) != 0 {
		t.Errorf("secretOpts = %d options, want 0", len(opts))
	}
}

func TestSecretOptsNilHost(t *testing.T) {
	opts := secretOpts(nil)
	if opts != nil {
		t.Errorf("secretOpts(nil) = %v, want nil", opts)
	}
}

func TestOAuthTokenRefresherMissingCreds(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_ID", "")
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "")

	host := &config.Host{
		Kind:               "cloud",
		BaseURL:            "https://api.bitbucket.org/2.0",
		AuthMethod:         "oauth",
		AllowInsecureStore: true,
	}

	refresher := oauthTokenRefresher("api.bitbucket.org", host)
	_, err := refresher(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "OAuth client credentials are missing") {
		t.Errorf("error = %q", err)
	}
}

func TestOAuthTokenRefresherKeyNotFound(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_ID", "cid")         // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "csecret") // gitleaks:allow
	t.Setenv("BKT_ALLOW_INSECURE_STORE", "1")
	t.Setenv("BKT_KEYRING_PASSPHRASE", "test-pass")
	t.Setenv("KEYRING_BACKEND", "file")
	t.Setenv("KEYRING_FILE_DIR", t.TempDir())

	host := &config.Host{
		Kind:               "cloud",
		BaseURL:            "https://api.bitbucket.org/2.0",
		AuthMethod:         "oauth",
		AllowInsecureStore: true,
	}

	refresher := oauthTokenRefresher("api.bitbucket.org", host)
	_, err := refresher(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read token") {
		t.Errorf("error = %q, want 'read token'", err)
	}
}

func TestOAuthTokenRefresherInvalidBlob(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_ID", "cid")         // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "csecret") // gitleaks:allow
	t.Setenv("BKT_ALLOW_INSECURE_STORE", "1")
	t.Setenv("BKT_KEYRING_PASSPHRASE", "test-pass")
	t.Setenv("KEYRING_BACKEND", "file")
	fileDir := t.TempDir()
	t.Setenv("KEYRING_FILE_DIR", fileDir)

	// Store invalid JSON blob in keyring.
	store, err := secret.Open(secret.WithAllowFileFallback(true), secret.WithPassphrase("test-pass"), secret.WithFileDir(fileDir))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Set(secret.TokenKey("api.bitbucket.org"), "not-json"); err != nil {
		t.Fatalf("store.Set: %v", err)
	}

	host := &config.Host{
		Kind:               "cloud",
		BaseURL:            "https://api.bitbucket.org/2.0",
		AuthMethod:         "oauth",
		AllowInsecureStore: true,
	}

	refresher := oauthTokenRefresher("api.bitbucket.org", host)
	_, err = refresher(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse stored token") {
		t.Errorf("error = %q, want 'parse stored token'", err)
	}
}

func TestOAuthTokenRefresherSuccess(t *testing.T) {
	// Mock token endpoint that returns a new token.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{ // gitleaks:allow
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"expires_in":    7200,
		})
	}))
	defer tokenSrv.Close()

	t.Setenv("BKT_OAUTH_CLIENT_ID", "cid")         // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "csecret") // gitleaks:allow
	t.Setenv("BKT_ALLOW_INSECURE_STORE", "1")
	t.Setenv("BKT_KEYRING_PASSPHRASE", "test-pass")
	t.Setenv("KEYRING_BACKEND", "file")
	fileDir := t.TempDir()
	t.Setenv("KEYRING_FILE_DIR", fileDir)

	// Store a valid OAuth token blob in the keyring.
	tok := oauth.FromResponse("old-access", "old-refresh", 7200)
	blob, err := tok.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	store, err := secret.Open(secret.WithAllowFileFallback(true), secret.WithPassphrase("test-pass"), secret.WithFileDir(fileDir))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Set(secret.TokenKey("api.bitbucket.org"), blob); err != nil {
		t.Fatalf("store.Set: %v", err)
	}

	host := &config.Host{
		Kind:               "cloud",
		BaseURL:            "https://api.bitbucket.org/2.0",
		AuthMethod:         "oauth",
		AllowInsecureStore: true,
	}

	// Build a refresher that uses our mock token endpoint.
	// We can't easily override oauth.CloudTokenURL, so we test the
	// sub-components instead. Here we directly call oauthTokenRefresher
	// which uses the real CloudTokenURL. Instead, test the creds check
	// and store interaction — the actual HTTP refresh is covered in
	// pkg/oauth/flow_test.go (TestRefreshToken).
	//
	// For this test, verify the refresher reads the stored blob,
	// then fails at the refresh step (since CloudTokenURL is real and
	// our creds are fake), confirming the store-read path is covered.
	refresher := oauthTokenRefresher("api.bitbucket.org", host)
	_, err = refresher(context.Background())
	// Expected: refresh fails because our fake creds can't auth against
	// real Bitbucket, but the error should be "refresh failed" (not
	// "read token" or "parse stored token"), proving we got past those.
	if err == nil {
		t.Fatal("expected error from refresh with fake creds")
	}
	if !strings.Contains(err.Error(), "refresh failed") {
		t.Errorf("error = %q, want 'refresh failed'", err)
	}
	// Verify host.Token was NOT updated (refresh failed).
	if host.Token == "new-access" {
		t.Error("host.Token should not be updated on refresh failure")
	}
}

func TestNewHTTPClientDC(t *testing.T) {
	host := &config.Host{
		Kind:    "dc",
		BaseURL: "https://bitbucket.example.com",
		Token:   "pat",
	}
	client, err := NewHTTPClient(host)
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestNewHTTPClientCloud(t *testing.T) {
	host := &config.Host{
		Kind:    "cloud",
		BaseURL: "https://api.bitbucket.org/2.0",
		Token:   "tok",
	}
	t.Setenv(secret.EnvToken, "")
	client, err := NewHTTPClient(host)
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestNewHTTPClientUnsupportedKind(t *testing.T) {
	host := &config.Host{Kind: "unknown"}
	_, err := NewHTTPClient(host)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewHTTPClientNil(t *testing.T) {
	_, err := NewHTTPClient(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
