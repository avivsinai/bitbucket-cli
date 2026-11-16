package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
)

func TestLoginHostKeyDeploymentIdentity(t *testing.T) {
	for _, tt := range []struct {
		name, previousURL, previousKind, alias string
		wantErr                                bool
	}{
		{"same deployment", "https://BB.example.com:443/bitbucket/", "dc", "team", false},
		{"different path", "https://bb.example.com/other", "dc", "team", true},
		{"different scheme", "http://bb.example.com/bitbucket", "dc", "team", true},
		{"different port", "https://bb.example.com:8443/bitbucket", "dc", "team", true},
		{"different kind", "https://bb.example.com/bitbucket", "cloud", "team", true},
		{"encoded deployment path", "https://bb.example.com/bitbucket%2F", "dc", "team", true},
		{"malformed", "://broken", "dc", "team", true},
		{"empty", "", "dc", "team", true},
		{"canonical key occupied", "https://other.example.com", "dc", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.alias
			if key == "" {
				key = "bb.example.com"
			}
			cfg := &config.Config{Hosts: map[string]*config.Host{key: {Kind: tt.previousKind, BaseURL: tt.previousURL}}}
			got, err := loginHostKey(cfg, tt.alias, "https://bb.example.com/bitbucket", "dc")
			if (err != nil) != tt.wantErr || (!tt.wantErr && got != key) {
				t.Fatalf("key = %q, error = %v", got, err)
			}
		})
	}
}

func TestAliasCollisionRejectedBeforeVerification(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++ }))
	t.Cleanup(server.Close)
	t.Setenv(secret.EnvToken, "")
	cfg := &config.Config{Hosts: map[string]*config.Host{"team": {Kind: "dc", BaseURL: "invalid"}}}
	f, _, _ := newAuthTestFactory(cfg)
	err := runLogin(newTestCmd(), f, &loginOptions{Host: server.URL, Kind: "dc", Token: "fixture-token", Alias: "team", AllowHTTP: true})
	if err == nil || !strings.Contains(err.Error(), "already names a host") || requests != 0 || cfg.Hosts["team"].BaseURL != "invalid" {
		t.Fatalf("error = %v, verification requests = %d", err, requests)
	}
}

func TestRunLogoutAliasKeepsOtherCredentials(t *testing.T) {
	setupFileKeyring(t)
	t.Setenv(secret.EnvToken, "")
	t.Setenv("BKT_CONFIG_DIR", t.TempDir())
	store, err := secret.Open(secret.WithAllowFileFallback(true))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Hosts: map[string]*config.Host{}, Contexts: map[string]*config.Context{}}
	for _, key := range []string{"team-a", "team-b"} {
		cfg.SetHost(key, &config.Host{Kind: "cloud", BaseURL: "https://api.bitbucket.org/2.0", AllowInsecureStore: true})
		cfg.SetContext(key, &config.Context{Host: key})
		if err := store.Set(secret.TokenKey(key), key+"-fixture"); err != nil {
			t.Fatal(err)
		}
	}
	f, _, _ := newAuthTestFactory(cfg)
	if err := runLogout(newTestCmd(), f, &logoutOptions{Host: "team-b"}); err != nil {
		t.Fatal(err)
	}
	if cfg.Hosts["team-a"] == nil || cfg.Contexts["team-a"] == nil || cfg.Hosts["team-b"] != nil || cfg.Contexts["team-b"] != nil {
		t.Fatal("logout changed another identity or retained the removed one")
	}
	if got, err := store.Get(secret.TokenKey("team-a")); err != nil || got != "team-a-fixture" {
		t.Fatal("logout changed another account's credential")
	}
	if _, err := store.Get(secret.TokenKey("team-b")); err == nil {
		t.Fatal("logout retained the alias credential")
	}
}
