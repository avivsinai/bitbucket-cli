package oauth

import (
	"slices"
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

func TestCloudClientsForRefreshesWithIssuingConsumer(t *testing.T) {
	builtIn := CloudClient{ID: cloudClientID, Secret: cloudClientSecret}
	env := CloudClient{ID: "env-id", Secret: "env-secret"}              // gitleaks:allow
	rotated := CloudClient{ID: cloudClientID, Secret: "rotated-secret"} // gitleaks:allow

	tests := []struct {
		name     string
		env      CloudClient
		clientID string
		want     []CloudClient
		wantErr  bool
	}{
		{name: "legacy token, no env", clientID: "", want: []CloudClient{builtIn}},
		{name: "legacy token tries env then built-in", env: env, clientID: "", want: []CloudClient{env, builtIn}},
		{name: "legacy token ignores partial env", env: CloudClient{ID: "env-id"}, clientID: "", want: []CloudClient{builtIn}},
		{name: "env-issued token", env: env, clientID: env.ID, want: []CloudClient{env}},
		{name: "built-in-issued token ignores other env", env: env, clientID: builtIn.ID, want: []CloudClient{builtIn}},
		{name: "rotated built-in secret wins", env: rotated, clientID: builtIn.ID, want: []CloudClient{rotated, builtIn}},
		{name: "unknown consumer", env: env, clientID: "other-id", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BKT_OAUTH_CLIENT_ID", tt.env.ID)
			t.Setenv("BKT_OAUTH_CLIENT_SECRET", tt.env.Secret)

			got, err := CloudClientsFor(&Token{ClientID: tt.clientID})
			if (err != nil) != tt.wantErr {
				t.Fatalf("CloudClientsFor error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("CloudClientsFor = %+v, want %+v", got, tt.want)
			}
		})
	}
}
