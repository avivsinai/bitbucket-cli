package format

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"text/template"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

// Options configures structured output rendering.
type Options struct {
	Format   string
	JQ       string
	Template string
}

// Write serializes data according to the chosen options. When no structured
// output is requested the fallback function is invoked to render human-friendly
// output.
func Write(w io.Writer, opts Options, data any, fallback func() error) error {
	if opts.Format == "" && opts.JQ == "" && opts.Template == "" {
		if fallback == nil {
			return nil
		}
		return fallback()
	}

	value := data

	if opts.JQ != "" {
		var err error
		value, err = applyJQ(opts.JQ, value)
		if err != nil {
			return err
		}
	}

	if opts.Template != "" {
		tmpl, err := template.New("output").Parse(opts.Template)
		if err != nil {
			return fmt.Errorf("parse template: %w", err)
		}
		return tmpl.Execute(w, value)
	}

	switch opts.Format {
	case "", "json":
		enc := json.NewEncoder(w)
		if opts.Format != "" {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(value); err != nil {
			return fmt.Errorf("encode json: %w", err)
		}
		return nil
	case "yaml":
		out, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode yaml: %w", err)
		}
		_, err = w.Write(out)
		return err
	default:
		return fmt.Errorf("unsupported format %q", opts.Format)
	}
}

func applyJQ(expression string, value any) (any, error) {
	normalised, err := normaliseForJQ(value)
	if err != nil {
		return nil, err
	}

	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("parse jq expression: %w", err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("compile jq expression: %w", err)
	}

	iter := code.Run(normalised)
	var results []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("jq evaluation failed: %w", err)
		}
		results = append(results, v)
	}

	if len(results) == 0 {
		return nil, nil
	}
	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

func normaliseForJQ(value any) (any, error) {
	switch v := value.(type) {
	case nil,
		string,
		bool,
		json.Number,
		float64,
		int,
		int64,
		uint,
		uint64:
		return value, nil
	case []byte:
		var out any
		if len(v) == 0 {
			return nil, nil
		}
		decoder := json.NewDecoder(bytes.NewReader(v))
		decoder.UseNumber()
		if err := decoder.Decode(&out); err != nil {
			return nil, fmt.Errorf("prepare jq input: %w", err)
		}
		return out, nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("prepare jq input: %w", err)
		}
		var out any
		if len(data) == 0 {
			return nil, nil
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err := decoder.Decode(&out); err != nil {
			return nil, fmt.Errorf("prepare jq input: %w", err)
		}
		return out, nil
	}
}
