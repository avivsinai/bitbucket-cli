package cmdutil

import (
	"fmt"
	"strings"
)

// ValidateGitPositionalArg rejects values that git would parse as flags when
// passed to subcommands that do not support a "--" separator.
func ValidateGitPositionalArg(value, description string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", description)
	}
	if strings.HasPrefix(trimmed, "-") {
		return fmt.Errorf("invalid %s %q: must not start with '-'", description, value)
	}
	return nil
}
