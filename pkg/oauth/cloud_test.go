package oauth

import (
	"testing"
)

func TestCloudClientIDFromLdflags(t *testing.T) {
	orig := cloudClientID
	defer func() { cloudClientID = orig }()

	cloudClientID = "build-id"                // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_ID", "env-id") // gitleaks:allow

	if got := CloudClientID(); got != "build-id" {
		t.Errorf("CloudClientID() = %q, want build-time value %q", got, "build-id")
	}
}

func TestCloudClientIDFromEnv(t *testing.T) {
	orig := cloudClientID
	defer func() { cloudClientID = orig }()

	cloudClientID = ""                        // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_ID", "env-id") // gitleaks:allow

	if got := CloudClientID(); got != "env-id" {
		t.Errorf("CloudClientID() = %q, want env fallback %q", got, "env-id")
	}
}

func TestCloudClientIDEmpty(t *testing.T) {
	orig := cloudClientID
	defer func() { cloudClientID = orig }()

	cloudClientID = ""                  // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_ID", "") // gitleaks:allow

	if got := CloudClientID(); got != "" {
		t.Errorf("CloudClientID() = %q, want empty", got)
	}
}

func TestCloudClientSecretFromLdflags(t *testing.T) {
	orig := cloudClientSecret
	defer func() { cloudClientSecret = orig }()

	cloudClientSecret = "build-secret"                // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "env-secret") // gitleaks:allow

	if got := CloudClientSecret(); got != "build-secret" {
		t.Errorf("CloudClientSecret() = %q, want build-time value %q", got, "build-secret")
	}
}

func TestCloudClientSecretFromEnv(t *testing.T) {
	orig := cloudClientSecret
	defer func() { cloudClientSecret = orig }()

	cloudClientSecret = ""                            // gitleaks:allow
	t.Setenv("BKT_OAUTH_CLIENT_SECRET", "env-secret") // gitleaks:allow

	if got := CloudClientSecret(); got != "env-secret" {
		t.Errorf("CloudClientSecret() = %q, want env fallback %q", got, "env-secret")
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
