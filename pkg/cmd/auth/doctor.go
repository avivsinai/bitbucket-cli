package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// Pin to absolute paths so a poisoned $PATH cannot point these probes at a
// different binary. Both are standard-location Apple tools.
const (
	codesignPath = "/usr/bin/codesign"
	securityPath = "/usr/bin/security"
)

type doctorOptions struct {
	Host string
}

func newDoctorCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &doctorOptions{}

	cmd := &cobra.Command{
		Use:   "doctor [host]",
		Short: "Diagnose authentication and keychain issues",
		Long: `Inspect the keychain/secret store wiring and explain why prompts or
timeouts may be happening.

On macOS this reports the current bkt binary's code signature, Designated
Requirement, and whether a stored token item is present for the host. If the
stored item was created by a bkt binary with a different Designated
Requirement — which is common after ` + "`brew upgrade bkt`" + ` — macOS will
keep prompting for the Keychain password on every read. The fix in that case
is to re-run ` + "`bkt auth login`" + ` so the item is recreated with the
current binary's DR.

The command never reads the stored secret itself.`,
		Example: `  # Inspect the default host
  bkt auth doctor

  # Inspect a specific host
  bkt auth doctor bitbucket.example.com`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Host = args[0]
			}
			return runDoctor(cmd, f, opts)
		},
	}

	return cmd
}

type doctorReport struct {
	Platform       string        `json:"platform"`
	Backend        string        `json:"backend"`
	Executable     string        `json:"executable,omitempty"`
	Signature      string        `json:"signature,omitempty"`
	Identifier     string        `json:"identifier,omitempty"`
	TeamIdentifier string        `json:"team_identifier,omitempty"`
	DesignatedReq  string        `json:"designated_requirement,omitempty"`
	StableDR       bool          `json:"stable_designated_requirement"`
	TrustFlags     bool          `json:"trust_flags_enabled"`
	Hosts          []hostProbe   `json:"hosts"`
	Diagnosis      string        `json:"diagnosis"`
	NextSteps      []string      `json:"next_steps,omitempty"`
	ProbeErrors    []string      `json:"probe_errors,omitempty"`
	Elapsed        time.Duration `json:"elapsed"`
}

type hostProbe struct {
	Key        string `json:"key"`
	BaseURL    string `json:"base_url,omitempty"`
	AuthMethod string `json:"auth_method,omitempty"`
	ItemStored bool   `json:"item_stored"`
	ProbeError string `json:"probe_error,omitempty"`
}

func runDoctor(cmd *cobra.Command, f *cmdutil.Factory, opts *doctorOptions) error {
	start := time.Now()

	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	report := doctorReport{
		Platform:   runtime.GOOS,
		Backend:    describeBackend(),
		TrustFlags: runtime.GOOS == "darwin",
	}

	if exe, err := os.Executable(); err == nil {
		report.Executable = exe
		if runtime.GOOS == "darwin" {
			info, probeErr := inspectCodesign(exe)
			if probeErr != nil {
				report.ProbeErrors = append(report.ProbeErrors, fmt.Sprintf("codesign: %v", probeErr))
			}
			report.Signature = info.signature
			report.Identifier = info.identifier
			report.TeamIdentifier = info.teamIdentifier
			report.DesignatedReq = info.designatedReq
			report.StableDR = info.stableDR
		}
	} else {
		report.ProbeErrors = append(report.ProbeErrors, fmt.Sprintf("executable: %v", err))
	}

	hostKeys := chooseHostKeys(cfg, opts.Host)

	for _, key := range hostKeys {
		probe := hostProbe{Key: key}
		if host, ok := cfg.Hosts[key]; ok {
			probe.BaseURL = host.BaseURL
			probe.AuthMethod = host.AuthMethod
			if probe.AuthMethod == "" {
				probe.AuthMethod = "basic"
			}
		}
		if runtime.GOOS == "darwin" {
			present, probeErr := keychainItemPresent(key)
			probe.ItemStored = present
			if probeErr != nil {
				probe.ProbeError = probeErr.Error()
				report.ProbeErrors = append(report.ProbeErrors, fmt.Sprintf("keychain probe for %s: %v", key, probeErr))
			}
		}
		report.Hosts = append(report.Hosts, probe)
	}

	report.Diagnosis, report.NextSteps = diagnose(report, cfg)
	report.Elapsed = time.Since(start).Round(time.Millisecond)

	return cmdutil.WriteOutput(cmd, ios.Out, report, func() error {
		return writeDoctorText(ios.Out, report, f)
	})
}

func describeBackend() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS Keychain"
	case "windows":
		return "Windows Credential Manager"
	default:
		return "Secret Service / libsecret"
	}
}

func chooseHostKeys(cfg *config.Config, selector string) []string {
	if selector != "" {
		if _, ok := cfg.Hosts[selector]; ok {
			return []string{selector}
		}
		if baseURL, err := cmdutil.NormalizeBaseURL(selector); err == nil {
			if key, err := cmdutil.HostKeyFromURL(baseURL); err == nil {
				if _, ok := cfg.Hosts[key]; ok {
					return []string{key}
				}
				return []string{key}
			}
		}
		return []string{selector}
	}

	keys := make([]string, 0, len(cfg.Hosts))
	for k := range cfg.Hosts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type codesignInfo struct {
	signature      string
	identifier     string
	teamIdentifier string
	designatedReq  string
	stableDR       bool
}

// inspectCodesign shells out to /usr/bin/codesign for metadata only. It never
// prompts the user. The output is parsed permissively: missing fields are
// reported as empty strings, not errors.
func inspectCodesign(path string) (codesignInfo, error) {
	info := codesignInfo{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meta, err := runCmd(ctx, codesignPath, "-dvvv", path)
	if err != nil {
		return info, err
	}

	for _, line := range strings.Split(meta, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Signature="):
			info.signature = strings.TrimPrefix(line, "Signature=")
		case strings.HasPrefix(line, "Identifier="):
			info.identifier = strings.TrimPrefix(line, "Identifier=")
		case strings.HasPrefix(line, "TeamIdentifier="):
			info.teamIdentifier = strings.TrimPrefix(line, "TeamIdentifier=")
		case strings.Contains(line, "flags=") && strings.Contains(line, "adhoc"):
			if info.signature == "" {
				info.signature = "adhoc"
			}
		}
	}

	req, reqErr := runCmd(ctx, codesignPath, "-d", "-r-", path)
	for _, line := range strings.Split(req, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "designated => "); idx >= 0 {
			info.designatedReq = strings.TrimSpace(line[idx+len("designated => "):])
			break
		}
	}

	info.stableDR = info.designatedReq != "" && !strings.Contains(info.designatedReq, "cdhash ")

	return info, reqErr
}

// errSecItemNotFound is the exit status /usr/bin/security returns when no
// matching generic-password item exists. See Security/SecBase.h.
const errSecItemNotFound = 44

// keychainItemPresent reports whether a generic-password item exists for the
// given host key. It uses /usr/bin/security without -g, so the stored secret
// is never read and no user prompt is triggered.
//
// Returns (true, nil) when the item exists, (false, nil) when the item is
// definitively absent, and (false, err) when the probe itself could not run
// to a conclusive result — the caller must not interpret that as "missing".
func keychainItemPresent(hostKey string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out, err := runCmd(ctx, securityPath, "find-generic-password",
		"-s", "bkt",
		"-a", secret.TokenKey(hostKey),
	)
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == errSecItemNotFound {
		return false, nil
	}

	detail := strings.TrimSpace(out)
	if detail == "" {
		return false, fmt.Errorf("security find-generic-password: %w", err)
	}
	return false, fmt.Errorf("security find-generic-password: %w (%s)", err, detail)
}

// runCmd is a package-level function pointer so tests can stub the shell-out
// surface without running real codesign/security binaries.
var runCmd = func(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func diagnose(r doctorReport, cfg *config.Config) (string, []string) {
	if runtime.GOOS != "darwin" {
		if len(cfg.Hosts) == 0 {
			return "No hosts configured.", []string{"Run `bkt auth login <host>` to add one."}
		}
		return fmt.Sprintf("Backend: %s — no macOS-specific diagnostics apply.", r.Backend), nil
	}

	if len(cfg.Hosts) == 0 {
		return "No hosts configured.", []string{"Run `bkt auth login <host>` to add one."}
	}

	steps := []string{}

	if r.Signature == "" {
		steps = append(steps, "Install bkt from an official release so the binary is signed.")
	}

	if !r.StableDR && r.DesignatedReq != "" {
		// DR is cdhash-bound. Expect re-prompts after every rebuild/upgrade.
		diag := "This bkt binary has a cdhash-based Designated Requirement. " +
			"Every rebuild (including every `brew upgrade bkt`) changes the cdhash, " +
			"which invalidates any Keychain \"Always Allow\" the user has granted — " +
			"so macOS keeps re-prompting."
		steps = append(steps,
			"Re-run `bkt auth login` to recreate the stored Keychain item against the current binary's DR.",
			"For scripts/CI, set BKT_TOKEN=<token> to bypass the keychain.",
			"Long-term: upgrade to a release that pins the Designated Requirement (see issue #181).",
		)
		return diag, steps
	}

	hasStaleItem := false
	probeInconclusive := false
	for _, h := range r.Hosts {
		if h.ItemStored {
			hasStaleItem = true
		}
		if h.ProbeError != "" {
			probeInconclusive = true
		}
	}

	if probeInconclusive {
		// Don't claim absence when the probe itself failed.
		return "Could not verify whether Keychain items exist for one or more hosts " +
				"(see probe warnings). If prompts persist, re-run `bkt auth login` to " +
				"recreate the item with the current binary's DR.",
			[]string{
				"Check that the login keychain is unlocked and `/usr/bin/security` is on PATH.",
				"Re-run `bkt auth login <host>` to recreate any affected item.",
			}
	}

	if r.StableDR && hasStaleItem {
		// DR is stable now, but the stored item may have been created by an
		// older bkt binary with a different DR. Re-login refreshes it.
		return "Binary has a stable Designated Requirement. " +
				"If Keychain still prompts, the stored item was likely created by an older bkt " +
				"version with a different DR.",
			[]string{
				"Re-run `bkt auth login` once to recreate the Keychain item with the current DR.",
				"Subsequent invocations should not prompt.",
			}
	}

	if r.StableDR && !hasStaleItem {
		return "Binary has a stable Designated Requirement and no stored Keychain item was found. " +
				"A fresh `bkt auth login` should produce a persistent Keychain entry.",
			nil
	}

	return "Keychain wiring looks healthy.", nil
}

func writeDoctorText(out io.Writer, r doctorReport, f *cmdutil.Factory) error {
	w := bufio.NewWriter(out)
	defer func() { _ = w.Flush() }()

	if r.Diagnosis != "" {
		if _, err := fmt.Fprintf(w, "Diagnosis: %s\n\n", r.Diagnosis); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "Environment:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  platform: %s\n", r.Platform); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  backend:  %s\n", r.Backend); err != nil {
		return err
	}
	if r.Executable != "" {
		if _, err := fmt.Fprintf(w, "  bkt path: %s\n", r.Executable); err != nil {
			return err
		}
	}

	if r.Platform == "darwin" {
		if _, err := fmt.Fprintln(w, "\nBinary signature:"); err != nil {
			return err
		}
		printKV(w, "signature", r.Signature)
		printKV(w, "identifier", r.Identifier)
		if r.TeamIdentifier != "" {
			printKV(w, "team id", r.TeamIdentifier)
		}
		printKV(w, "designated", r.DesignatedReq)
		stable := "no (cdhash-bound; re-prompts after every upgrade)"
		if r.StableDR {
			stable = "yes"
		}
		printKV(w, "DR stable", stable)

		if _, err := fmt.Fprintln(w, "\nKeychain wiring:"); err != nil {
			return err
		}
		trust := "no"
		if r.TrustFlags {
			trust = "yes"
		}
		printKV(w, "trust flags", trust)
	}

	if len(r.Hosts) > 0 {
		if _, err := fmt.Fprintln(w, "\nHosts:"); err != nil {
			return err
		}
		for _, h := range r.Hosts {
			line := fmt.Sprintf("  - %s", h.Key)
			if h.BaseURL != "" {
				line += fmt.Sprintf(" (%s)", h.BaseURL)
			}
			if h.AuthMethod != "" {
				line += fmt.Sprintf(", auth=%s", h.AuthMethod)
			}
			if r.Platform == "darwin" {
				switch {
				case h.ProbeError != "":
					line += ", keychain=unknown"
				case h.ItemStored:
					line += ", keychain=present"
				default:
					line += ", keychain=missing"
				}
			}
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}
	}

	if len(r.NextSteps) > 0 {
		if _, err := fmt.Fprintln(w, "\nNext steps:"); err != nil {
			return err
		}
		for i, step := range r.NextSteps {
			if _, err := fmt.Fprintf(w, "  %d. %s\n", i+1, step); err != nil {
				return err
			}
		}
	}

	if len(r.ProbeErrors) > 0 {
		if _, err := fmt.Fprintln(w, "\nProbe warnings:"); err != nil {
			return err
		}
		for _, pe := range r.ProbeErrors {
			if _, err := fmt.Fprintf(w, "  - %s\n", pe); err != nil {
				return err
			}
		}
	}

	_ = f

	return nil
}

func printKV(w io.Writer, label, value string) {
	if value == "" {
		value = "(unknown)"
	}
	_, _ = fmt.Fprintf(w, "  %-11s %s\n", label+":", value)
}
