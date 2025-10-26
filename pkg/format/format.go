package format

import (
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
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("parse jq expression: %w", err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("compile jq expression: %w", err)
	}

	iter := code.Run(value)
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
