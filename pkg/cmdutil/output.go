package cmdutil

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/example/bitbucket-cli/pkg/format"
)

// OutputSettings captures structured output preferences.
type OutputSettings struct {
	Format   string
	JQ       string
	Template string
}

// OutputSettings extracts flags from the command hierarchy with validation.
func ResolveOutputSettings(cmd *cobra.Command) (OutputSettings, error) {
	root := cmd.Root()

	lookup := func(name string) string {
		flag := root.PersistentFlags().Lookup(name)
		if flag == nil {
			return ""
		}
		return flag.Value.String()
	}

	jsonEnabled := lookup("json") == "true"
	yamlEnabled := lookup("yaml") == "true"
	jqExpr := lookup("jq")
	tmpl := lookup("template")

	if jsonEnabled && yamlEnabled {
		return OutputSettings{}, fmt.Errorf("cannot use --json and --yaml simultaneously")
	}

	if jqExpr != "" && tmpl != "" {
		return OutputSettings{}, fmt.Errorf("cannot use --jq and --template simultaneously")
	}

	if jqExpr != "" && !jsonEnabled {
		return OutputSettings{}, fmt.Errorf("--jq requires --json")
	}

	format := ""
	if jsonEnabled {
		format = "json"
	} else if yamlEnabled {
		format = "yaml"
	}

	return OutputSettings{
		Format:   format,
		JQ:       jqExpr,
		Template: tmpl,
	}, nil
}

// OutputFormat preserves backwards compatibility for callers needing only the
// format value.
func OutputFormat(cmd *cobra.Command) (string, error) {
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		return "", err
	}
	return settings.Format, nil
}

// WriteOutput writes structured output according to user preferences and runs
// fallback when no structured output is requested.
func WriteOutput(cmd *cobra.Command, w io.Writer, data any, fallback func() error) error {
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		return err
	}
	opts := format.Options{Format: settings.Format, JQ: settings.JQ, Template: settings.Template}
	return format.Write(w, opts, data, fallback)
}
