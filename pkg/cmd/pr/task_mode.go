package pr

import (
	"fmt"
	"strconv"
	"strings"
)

// Task API modes selectable via --task-api. "auto" probes the Data Center
// version to choose between the modern blocker-comments model (>= 7.2) and the
// legacy /tasks model (< 7.2). Cloud always uses its first-class tasks API and
// ignores this flag.
const (
	taskAPIAuto            = "auto"
	taskAPIBlockerComments = "blocker-comments"
	taskAPILegacy          = "legacy"
)

var validTaskAPIModes = []string{taskAPIAuto, taskAPIBlockerComments, taskAPILegacy}

// validateTaskAPIMode ensures the user-supplied --task-api value is recognized.
func validateTaskAPIMode(mode string) error {
	for _, m := range validTaskAPIModes {
		if mode == m {
			return nil
		}
	}
	return fmt.Errorf("invalid --task-api %q: must be one of %s", mode, strings.Join(validTaskAPIModes, ", "))
}

// dcSupportsBlockerComments reports whether a Data Center version string (e.g.
// "8.19.1") is >= 7.2, the release that introduced blocker comments and
// deprecated the legacy /tasks API.
func dcSupportsBlockerComments(version string) (bool, error) {
	major, minor, err := parseDCVersion(version)
	if err != nil {
		return false, err
	}
	if major > 7 {
		return true, nil
	}
	return major == 7 && minor >= 2, nil
}

// parseDCVersion extracts the major and minor components from a Data Center
// version string. Trailing components (patch, build metadata) are ignored.
func parseDCVersion(version string) (major, minor int, err error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return 0, 0, fmt.Errorf("empty version string")
	}
	parts := strings.SplitN(version, ".", 3)
	major, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("parse major version from %q: %w", version, err)
	}
	if len(parts) > 1 {
		// Tolerate suffixes like "2-rc1" by reading the leading digits.
		minor, err = strconv.Atoi(leadingDigits(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("parse minor version from %q: %w", version, err)
		}
	}
	return major, minor, nil
}

func leadingDigits(s string) string {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return s // let Atoi surface the error
	}
	return s[:end]
}

// resolveDCTaskMode resolves the effective Data Center task API.
//
//   - An explicit mode (blocker-comments or legacy) is honored as-is with no
//     version probe.
//   - "auto" calls lookupVersion and picks blocker-comments for >= 7.2, legacy
//     otherwise.
//   - If auto detection fails, mutating operations fail closed (returning an
//     error that tells the user to pass --task-api explicitly) rather than risk
//     writing through the wrong model; read operations fall back to the modern
//     blocker-comments path.
func resolveDCTaskMode(mode string, lookupVersion func() (string, error), mutating bool) (string, error) {
	switch mode {
	case taskAPIBlockerComments, taskAPILegacy:
		return mode, nil
	case taskAPIAuto, "":
		version, err := lookupVersion()
		if err != nil {
			if mutating {
				return "", fmt.Errorf("could not detect Data Center version (%w); pass --task-api blocker-comments or --task-api legacy", err)
			}
			return taskAPIBlockerComments, nil
		}
		supported, err := dcSupportsBlockerComments(version)
		if err != nil {
			if mutating {
				return "", fmt.Errorf("could not parse Data Center version %q (%w); pass --task-api blocker-comments or --task-api legacy", version, err)
			}
			return taskAPIBlockerComments, nil
		}
		if supported {
			return taskAPIBlockerComments, nil
		}
		return taskAPILegacy, nil
	default:
		return "", fmt.Errorf("invalid --task-api %q", mode)
	}
}
