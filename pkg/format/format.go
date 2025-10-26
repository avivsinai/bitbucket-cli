package format

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Write serializes data according to the chosen format. When format is empty,
// the fallback function is invoked to render human-friendly output.
func Write(w io.Writer, format string, data any, fallback func() error) error {
	switch format {
	case "":
		if fallback == nil {
			return nil
		}
		return fallback()
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "yaml":
		out, err := yaml.Marshal(data)
		if err != nil {
			return fmt.Errorf("encode yaml: %w", err)
		}
		_, err = w.Write(out)
		return err
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}
