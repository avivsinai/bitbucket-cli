package oauth

import (
	"testing"
)

// Homebrew core, go install, and Nix build from source without release
// ldflags; Cloud OAuth login must still work there.
func TestResolveCloudClientDefaultsToBuiltIn(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_ID", "")
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "")

	got, err := ResolveCloudClient()
	if err != nil {
		t.Fatalf("ResolveCloudClient: %v", err)
	}
	if got.ID == "" || got.Secret == "" {
		t.Fatal("built-in OAuth consumer is empty")
	}
}

func TestResolveCloudClientEnv(t *testing.T) {
	tests := []struct {
		name, id, secret string
		want             CloudClient
		wantErr          bool
	}{
		{name: "pair overrides built-in", id: "env-id", secret: "env-secret", want: CloudClient{ID: "env-id", Secret: "env-secret"}}, // gitleaks:allow
		{name: "id without secret", id: "env-id", wantErr: true},
		{name: "secret without id", secret: "env-secret", wantErr: true}, // gitleaks:allow
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BKT_OAUTH_CLIENT_ID", tt.id)
			t.Setenv("BKT_OAUTH_CLIENT_SECRET", tt.secret)

			got, err := ResolveCloudClient()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveCloudClient error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ResolveCloudClient = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCloudClientForRefreshesWithIssuingConsumer(t *testing.T) {
	builtIn := CloudClient{ID: cloudClientID, Secret: cloudClientSecret}
	env := CloudClient{ID: "env-id", Secret: "env-secret"} // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_ID", env.ID)
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", env.Secret)

	tests := []struct {
		name     string
		clientID string
		want     CloudClient
		wantErr  bool
	}{
		{name: "legacy token uses resolved consumer", clientID: "", want: env},
		{name: "env-issued token", clientID: env.ID, want: env},
		{name: "built-in-issued token ignores env", clientID: builtIn.ID, want: builtIn},
		{name: "unknown consumer", clientID: "other-id", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CloudClientFor(&Token{ClientID: tt.clientID})
			if (err != nil) != tt.wantErr {
				t.Fatalf("CloudClientFor error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("CloudClientFor = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCloudScopes(t *testing.T) {
	scopes := CloudScopes()
	if len(scopes) == 0 {
		t.Fatal("CloudScopes() returned empty")
	}
	want := map[string]bool{"account": true, "repository": true, "pullrequest": true, "issue": true, "pipeline": true, "webhook": true}
	for _, s := range scopes {
		if !want[s] {
			t.Errorf("unexpected scope %q", s)
		}
		delete(want, s)
	}
	for s := range want {
		t.Errorf("missing scope %q", s)
	}
}
