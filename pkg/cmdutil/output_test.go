package cmdutil

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCommand(flags map[string]string) *cobra.Command {
	root := &cobra.Command{Use: "bkt"}
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("yaml", false, "")
	root.PersistentFlags().String("jq", "", "")
	root.PersistentFlags().String("template", "", "")
	root.PersistentFlags().String("format", "", "")

	child := &cobra.Command{Use: "test"}
	root.AddCommand(child)

	for k, v := range flags {
		if err := root.PersistentFlags().Set(k, v); err != nil {
			panic(err)
		}
	}
	return child
}

func TestResolveOutputSettingsJSONFormat(t *testing.T) {
	cmd := newTestCommand(map[string]string{"json": "true"})
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		t.Fatalf("ResolveOutputSettings: %v", err)
	}
	if settings.Format != "json" {
		t.Fatalf("format = %q, want json", settings.Format)
	}
}

func TestResolveOutputSettingsYAMLFormat(t *testing.T) {
	cmd := newTestCommand(map[string]string{"yaml": "true"})
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		t.Fatalf("ResolveOutputSettings: %v", err)
	}
	if settings.Format != "yaml" {
		t.Fatalf("format = %q, want yaml", settings.Format)
	}
}

func TestResolveOutputSettingsNoFlags(t *testing.T) {
	cmd := newTestCommand(nil)
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		t.Fatalf("ResolveOutputSettings: %v", err)
	}
	if settings.Format != "" {
		t.Fatalf("format = %q, want empty", settings.Format)
	}
}

func TestResolveOutputSettingsRejectsJSONAndYAML(t *testing.T) {
	cmd := newTestCommand(map[string]string{"json": "true", "yaml": "true"})
	_, err := ResolveOutputSettings(cmd)
	if err == nil {
		t.Fatal("expected error for simultaneous --json and --yaml")
	}
	if !strings.Contains(err.Error(), "cannot use --json and --yaml") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOutputSettingsRejectsJQAndTemplate(t *testing.T) {
	cmd := newTestCommand(map[string]string{"json": "true", "jq": ".name", "template": "{{.Name}}"})
	_, err := ResolveOutputSettings(cmd)
	if err == nil {
		t.Fatal("expected error for simultaneous --jq and --template")
	}
	if !strings.Contains(err.Error(), "cannot use --jq and --template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOutputSettingsJQRequiresJSON(t *testing.T) {
	cmd := newTestCommand(map[string]string{"jq": ".name"})
	_, err := ResolveOutputSettings(cmd)
	if err == nil {
		t.Fatal("expected error for --jq without --json")
	}
	if !strings.Contains(err.Error(), "--jq requires --json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOutputSettingsJQWithJSON(t *testing.T) {
	cmd := newTestCommand(map[string]string{"json": "true", "jq": ".name"})
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		t.Fatalf("ResolveOutputSettings: %v", err)
	}
	if settings.Format != "json" || settings.JQ != ".name" {
		t.Fatalf("unexpected settings: %+v", settings)
	}
}

func TestResolveOutputSettingsFormatJSON(t *testing.T) {
	cmd := newTestCommand(map[string]string{"format": "json"})
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		t.Fatalf("ResolveOutputSettings: %v", err)
	}
	if settings.Format != "json" {
		t.Fatalf("format = %q, want json", settings.Format)
	}
}

func TestResolveOutputSettingsFormatYAML(t *testing.T) {
	cmd := newTestCommand(map[string]string{"format": "yaml"})
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		t.Fatalf("ResolveOutputSettings: %v", err)
	}
	if settings.Format != "yaml" {
		t.Fatalf("format = %q, want yaml", settings.Format)
	}
}

func TestResolveOutputSettingsFormatCaseInsensitive(t *testing.T) {
	for _, val := range []string{"JSON", "YAML", "Json", "Yaml"} {
		cmd := newTestCommand(map[string]string{"format": val})
		settings, err := ResolveOutputSettings(cmd)
		if err != nil {
			t.Fatalf("--format %q: %v", val, err)
		}
		want := strings.ToLower(val)
		if settings.Format != want {
			t.Fatalf("--format %q: got %q, want %q", val, settings.Format, want)
		}
	}
}

func TestResolveOutputSettingsFormatConflictsWithJSON(t *testing.T) {
	cmd := newTestCommand(map[string]string{"format": "json", "json": "true"})
	_, err := ResolveOutputSettings(cmd)
	if err == nil {
		t.Fatal("expected error for --format with --json")
	}
	if !strings.Contains(err.Error(), "--format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOutputSettingsFormatConflictsWithYAML(t *testing.T) {
	cmd := newTestCommand(map[string]string{"format": "yaml", "yaml": "true"})
	_, err := ResolveOutputSettings(cmd)
	if err == nil {
		t.Fatal("expected error for --format with --yaml")
	}
	if !strings.Contains(err.Error(), "--format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOutputSettingsFormatInvalidValue(t *testing.T) {
	cmd := newTestCommand(map[string]string{"format": "xml"})
	_, err := ResolveOutputSettings(cmd)
	if err == nil {
		t.Fatal("expected error for unknown --format value")
	}
	if !strings.Contains(err.Error(), "xml") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveOutputSettingsFormatEmpty(t *testing.T) {
	cmd := newTestCommand(map[string]string{"format": ""})
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		t.Fatalf("ResolveOutputSettings: %v", err)
	}
	if settings.Format != "" {
		t.Fatalf("format = %q, want empty", settings.Format)
	}
}

func TestOutputFormat(t *testing.T) {
	cmd := newTestCommand(map[string]string{"yaml": "true"})
	format, err := OutputFormat(cmd)
	if err != nil {
		t.Fatalf("OutputFormat: %v", err)
	}
	if format != "yaml" {
		t.Fatalf("format = %q, want yaml", format)
	}
}
