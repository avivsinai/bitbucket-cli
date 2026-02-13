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
