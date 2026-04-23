package auth

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
)

func TestDiagnose_NoHosts(t *testing.T) {
	cfg := &config.Config{Hosts: map[string]*config.Host{}}
	r := doctorReport{Platform: runtime.GOOS, Backend: describeBackend()}

	msg, steps := diagnose(r, cfg)
	if !strings.Contains(msg, "No hosts configured") {
		t.Errorf("diagnosis should mention no hosts, got %q", msg)
	}
	if len(steps) == 0 {
		t.Error("expected at least one next step when no hosts are configured")
	}
}

func TestDiagnose_DarwinCdhashDR(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific diagnosis")
	}

	cfg := &config.Config{Hosts: map[string]*config.Host{
		"example.com": {BaseURL: "https://example.com"},
	}}
	r := doctorReport{
		Platform:      "darwin",
		Backend:       "macOS Keychain",
		Signature:     "adhoc",
		DesignatedReq: `cdhash H"abc123"`,
		StableDR:      false,
		Hosts: []hostProbe{
			{Key: "example.com", ItemStored: true},
		},
	}

	msg, steps := diagnose(r, cfg)

	if !strings.Contains(msg, "cdhash-based") {
		t.Errorf("diagnosis should call out cdhash DR, got %q", msg)
	}
	if !strings.Contains(msg, "brew upgrade") {
		t.Errorf("diagnosis should mention brew upgrade, got %q", msg)
	}
	if len(steps) == 0 {
		t.Fatal("expected next steps for cdhash-bound DR")
	}
	joined := strings.Join(steps, "\n")
	if !strings.Contains(joined, "bkt auth login") {
		t.Errorf("next steps should suggest bkt auth login, got %q", joined)
	}
	if !strings.Contains(joined, "BKT_TOKEN") {
		t.Errorf("next steps should mention BKT_TOKEN fallback, got %q", joined)
	}
}

func TestDiagnose_DarwinStableDRWithStoredItem(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific diagnosis")
	}

	cfg := &config.Config{Hosts: map[string]*config.Host{
		"example.com": {BaseURL: "https://example.com"},
	}}
	r := doctorReport{
		Platform:      "darwin",
		Backend:       "macOS Keychain",
		Signature:     "adhoc",
		DesignatedReq: `identifier "io.github.avivsinai.bitbucket-cli"`,
		StableDR:      true,
		Hosts: []hostProbe{
			{Key: "example.com", ItemStored: true},
		},
	}

	msg, steps := diagnose(r, cfg)
	if !strings.Contains(msg, "stable Designated Requirement") {
		t.Errorf("diagnosis should affirm stable DR, got %q", msg)
	}
	joined := strings.Join(steps, "\n")
	if !strings.Contains(joined, "bkt auth login") {
		t.Errorf("should still recommend auth login when stored item may be stale, got %q", joined)
	}
}

func TestDiagnose_DarwinStableDRNoStoredItem(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific diagnosis")
	}

	cfg := &config.Config{Hosts: map[string]*config.Host{
		"example.com": {BaseURL: "https://example.com"},
	}}
	r := doctorReport{
		Platform:      "darwin",
		Backend:       "macOS Keychain",
		DesignatedReq: `identifier "io.github.avivsinai.bitbucket-cli"`,
		StableDR:      true,
		Hosts: []hostProbe{
			{Key: "example.com", ItemStored: false},
		},
	}

	msg, _ := diagnose(r, cfg)
	if !strings.Contains(msg, "stable Designated Requirement") {
		t.Errorf("diagnosis should affirm stable DR, got %q", msg)
	}
	if !strings.Contains(msg, "fresh") {
		t.Errorf("diagnosis should recommend a fresh login, got %q", msg)
	}
}

func TestDiagnose_DarwinProbeInconclusive(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific diagnosis")
	}

	cfg := &config.Config{Hosts: map[string]*config.Host{
		"example.com": {BaseURL: "https://example.com"},
	}}
	r := doctorReport{
		Platform:      "darwin",
		Backend:       "macOS Keychain",
		DesignatedReq: `identifier "io.github.avivsinai.bitbucket-cli"`,
		StableDR:      true,
		Hosts: []hostProbe{
			{Key: "example.com", ProbeError: "security find-generic-password: exit status 1"},
		},
	}

	msg, steps := diagnose(r, cfg)
	if !strings.Contains(msg, "Could not verify") {
		t.Errorf("diagnosis should surface probe uncertainty, got %q", msg)
	}
	if strings.Contains(msg, "no stored Keychain item") {
		t.Errorf("diagnosis must not claim absence when probe failed, got %q", msg)
	}
	if len(steps) == 0 {
		t.Error("expected next steps when probe is inconclusive")
	}
}

func TestChooseHostKeys(t *testing.T) {
	cfg := &config.Config{Hosts: map[string]*config.Host{
		"zzz": {BaseURL: "https://zzz"},
		"aaa": {BaseURL: "https://aaa"},
	}}

	keys := chooseHostKeys(cfg, "")
	if len(keys) != 2 || keys[0] != "aaa" || keys[1] != "zzz" {
		t.Errorf("expected sorted [aaa,zzz], got %v", keys)
	}

	keys = chooseHostKeys(cfg, "aaa")
	if len(keys) != 1 || keys[0] != "aaa" {
		t.Errorf("expected [aaa], got %v", keys)
	}

	// Selector that does not match and cannot be parsed as a URL passes through.
	keys = chooseHostKeys(cfg, "not-a-host-we-know")
	if len(keys) != 1 || keys[0] != "not-a-host-we-know" {
		t.Errorf("expected passthrough for unknown selector, got %v", keys)
	}

	// URL-style selector is normalized and resolved against known hosts.
	cfg.Hosts["bitbucket.example.com"] = &config.Host{BaseURL: "https://bitbucket.example.com"}
	keys = chooseHostKeys(cfg, "https://bitbucket.example.com")
	if len(keys) != 1 || keys[0] != "bitbucket.example.com" {
		t.Errorf("URL selector should normalize to host key, got %v", keys)
	}
}

func TestDescribeBackend(t *testing.T) {
	got := describeBackend()
	switch runtime.GOOS {
	case "darwin":
		if got != "macOS Keychain" {
			t.Errorf("darwin: got %q", got)
		}
	case "windows":
		if got != "Windows Credential Manager" {
			t.Errorf("windows: got %q", got)
		}
	default:
		if got != "Secret Service / libsecret" {
			t.Errorf("linux: got %q", got)
		}
	}
}

func TestWriteDoctorText_DarwinStableDR(t *testing.T) {
	r := doctorReport{
		Platform:      "darwin",
		Backend:       "macOS Keychain",
		Executable:    "/opt/homebrew/bin/bkt",
		Signature:     "adhoc",
		Identifier:    "io.github.avivsinai.bitbucket-cli",
		DesignatedReq: `identifier "io.github.avivsinai.bitbucket-cli"`,
		StableDR:      true,
		TrustFlags:    true,
		Hosts: []hostProbe{
			{Key: "example.com", BaseURL: "https://example.com", AuthMethod: "basic", ItemStored: true},
			{Key: "other.example", AuthMethod: "bearer", ProbeError: "exit status 1"},
		},
		Diagnosis: "something concise",
		NextSteps: []string{"do a thing", "do another thing"},
	}

	var buf bytes.Buffer
	if err := writeDoctorText(&buf, r, nil); err != nil {
		t.Fatalf("writeDoctorText: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"Diagnosis: something concise",
		"Environment:",
		"platform: darwin",
		"backend:  macOS Keychain",
		"bkt path: /opt/homebrew/bin/bkt",
		"Binary signature:",
		"signature:  adhoc",
		"identifier: io.github.avivsinai.bitbucket-cli",
		`designated: identifier "io.github.avivsinai.bitbucket-cli"`,
		"DR stable:  yes",
		"Keychain wiring:",
		"trust flags: yes",
		"Hosts:",
		"example.com (https://example.com), auth=basic, keychain=present",
		"other.example, auth=bearer, keychain=unknown",
		"Next steps:",
		"1. do a thing",
		"2. do another thing",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}

	if strings.Contains(out, "team id") {
		t.Errorf("team id line should be omitted when empty, got:\n%s", out)
	}
}

func TestWriteDoctorText_NonDarwinMinimal(t *testing.T) {
	r := doctorReport{
		Platform:  "linux",
		Backend:   "Secret Service / libsecret",
		Diagnosis: "Backend: Secret Service / libsecret — no macOS-specific diagnostics apply.",
		Hosts: []hostProbe{
			{Key: "example.com", AuthMethod: "basic"},
		},
	}

	var buf bytes.Buffer
	if err := writeDoctorText(&buf, r, nil); err != nil {
		t.Fatalf("writeDoctorText: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "Binary signature:") {
		t.Errorf("should not emit signature section off-darwin:\n%s", out)
	}
	if strings.Contains(out, "keychain=") {
		t.Errorf("should not emit keychain presence off-darwin:\n%s", out)
	}
	if !strings.Contains(out, "example.com, auth=basic") {
		t.Errorf("host line missing:\n%s", out)
	}
}
