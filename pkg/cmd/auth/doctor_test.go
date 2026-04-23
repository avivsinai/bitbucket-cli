package auth

import (
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
}
