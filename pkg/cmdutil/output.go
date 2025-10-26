package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"
)

// OutputFormat inspects the persistent json/yaml flags and returns the desired
// structured output format. Empty string means human-friendly output.
func OutputFormat(cmd *cobra.Command) (string, error) {
	jsonFlag := cmd.Root().PersistentFlags().Lookup("json")
	yamlFlag := cmd.Root().PersistentFlags().Lookup("yaml")

	jsonEnabled := jsonFlag != nil && jsonFlag.Value.String() == "true"
	yamlEnabled := yamlFlag != nil && yamlFlag.Value.String() == "true"

	if jsonEnabled && yamlEnabled {
		return "", fmt.Errorf("cannot use --json and --yaml simultaneously")
	}
	if jsonEnabled {
		return "json", nil
	}
	if yamlEnabled {
		return "yaml", nil
	}
	return "", nil
}
