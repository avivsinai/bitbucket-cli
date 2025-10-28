package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func newFactoryWithServer(t *testing.T, handler http.HandlerFunc) (*cmdutil.Factory, func()) {
	t.Helper()
	server := httptest.NewServer(handler)

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host: "main",
			},
		},
		Hosts: map[string]*config.Host{
			"main": {
				Kind:    "dc",
				BaseURL: server.URL,
			},
		},
	}

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    stdout,
			ErrOut: stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	return f, server.Close
}

func TestAPICommandRespectsYAML(t *testing.T) {
	factory, cleanup := newFactoryWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"demo","count":2}`))
	})
	defer cleanup()

	cmd := NewCmdAPI(factory)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// Simulate persistent flags present on the root command.
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.PersistentFlags().Bool("yaml", false, "")
	cmd.PersistentFlags().String("jq", "", "")
	cmd.PersistentFlags().String("template", "", "")
	cmd.PersistentFlags().String("context", "", "")

	cmd.SetArgs([]string{"--yaml", "/rest/api/1.0/projects"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := factory.IOStreams.Out.(*strings.Builder).String()
	if !strings.Contains(output, "name: demo") {
		t.Fatalf("expected YAML output, got %q", output)
	}
}

func TestAPICommandRejectsStructuredOutputOnInvalidJSON(t *testing.T) {
	factory, cleanup := newFactoryWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain text"))
	})
	defer cleanup()

	cmd := NewCmdAPI(factory)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.PersistentFlags().Bool("yaml", false, "")
	cmd.PersistentFlags().String("jq", "", "")
	cmd.PersistentFlags().String("template", "", "")
	cmd.PersistentFlags().String("context", "", "")

	cmd.SetArgs([]string{"--json", "/rest/api/1.0/projects"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error when response is not JSON")
	}
	if !strings.Contains(err.Error(), "response is not valid JSON") {
		t.Fatalf("expected JSON parse error, got %v", err)
	}
}

func TestAPICommandPreservesLargeIntegers(t *testing.T) {
	factory, cleanup := newFactoryWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Use max uint64 value which would be corrupted if converted to float64
		_, _ = w.Write([]byte(`{"value":18446744073709551615}`))
	})
	defer cleanup()

	cmd := NewCmdAPI(factory)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.PersistentFlags().Bool("yaml", false, "")
	cmd.PersistentFlags().String("jq", "", "")
	cmd.PersistentFlags().String("template", "", "")
	cmd.PersistentFlags().String("context", "", "")

	cmd.SetArgs([]string{"--json", "/rest/api/1.0/test"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := factory.IOStreams.Out.(*strings.Builder).String()
	// Verify the exact integer is preserved, not corrupted to 18446744073709552000
	if !strings.Contains(output, "18446744073709551615") {
		t.Fatalf("expected output to preserve large integer 18446744073709551615, got %q", output)
	}
	if strings.Contains(output, "18446744073709552000") {
		t.Fatalf("large integer was corrupted by float64 conversion: %q", output)
	}
}

func TestAPICommandStreamsWithoutStructuredOutput(t *testing.T) {
	factory, cleanup := newFactoryWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain text response"))
	})
	defer cleanup()

	cmd := NewCmdAPI(factory)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.PersistentFlags().Bool("yaml", false, "")
	cmd.PersistentFlags().String("jq", "", "")
	cmd.PersistentFlags().String("template", "", "")
	cmd.PersistentFlags().String("context", "", "")

	// No structured output flags, should stream directly
	cmd.SetArgs([]string{"/rest/api/1.0/test"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := factory.IOStreams.Out.(*strings.Builder).String()
	if output != "plain text response" {
		t.Fatalf("expected plain text to be streamed as-is, got %q", output)
	}
}

func TestAPICommandStreamsJSONWithoutFlags(t *testing.T) {
	factory, cleanup := newFactoryWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"value"}`))
	})
	defer cleanup()

	cmd := NewCmdAPI(factory)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.PersistentFlags().Bool("yaml", false, "")
	cmd.PersistentFlags().String("jq", "", "")
	cmd.PersistentFlags().String("template", "", "")
	cmd.PersistentFlags().String("context", "", "")

	// No structured output flags, should stream JSON as-is without parsing
	cmd.SetArgs([]string{"/rest/api/1.0/test"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := factory.IOStreams.Out.(*strings.Builder).String()
	// Should be raw JSON, not reformatted
	if output != `{"key":"value"}` {
		t.Fatalf("expected raw JSON to be streamed, got %q", output)
	}
}
