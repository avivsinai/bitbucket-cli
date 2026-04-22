package oauth

import (
	"testing"
)

func TestCloudClientIDFromEnv(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_ID", "env-id") // gitleaks:allow

	if got := CloudClientID(); got != "env-id" {
		t.Errorf("CloudClientID() = %q, want env value %q", got, "env-id")
	}
}

func TestCloudClientIDEmpty(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_ID", "") // gitleaks:allow

	if got := CloudClientID(); got != "" {
		t.Errorf("CloudClientID() = %q, want empty", got)
	}
}

func TestCloudClientSecretFromEnv(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "env-secret") // gitleaks:allow

	if got := CloudClientSecret(); got != "env-secret" {
		t.Errorf("CloudClientSecret() = %q, want env value %q", got, "env-secret")
	}
}

func TestCloudClientSecretEmpty(t *testing.T) {
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "") // gitleaks:allow

	if got := CloudClientSecret(); got != "" {
		t.Errorf("CloudClientSecret() = %q, want empty", got)
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
