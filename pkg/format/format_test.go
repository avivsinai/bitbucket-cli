package format

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestWriteAppliesJQToStruct(t *testing.T) {
	buf := new(bytes.Buffer)
	data := sample{Name: "demo", Count: 3}

	err := Write(buf, Options{Format: "json", JQ: ".name"}, data, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	var got string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}
	if got != "demo" {
		t.Fatalf("expected jq to return \"demo\", got %q", got)
	}
}

func TestWriteJQHandlesSliceResult(t *testing.T) {
	buf := new(bytes.Buffer)
	data := []sample{
		{Name: "first", Count: 1},
		{Name: "second", Count: 2},
	}

	err := Write(buf, Options{Format: "json", JQ: ".[].count"}, data, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	var counts []int
	if err := json.Unmarshal(buf.Bytes(), &counts); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}
	if len(counts) != 2 || counts[0] != 1 || counts[1] != 2 {
		t.Fatalf("expected [1,2], got %v", counts)
	}
}

func TestWriteJQPreservesLargeIntegers(t *testing.T) {
	buf := new(bytes.Buffer)
	// Simulate data that would come from an API with large integers
	// Use a map to mimic decoded JSON with json.Number
	data := map[string]any{
		"id":   json.Number("18446744073709551615"), // max uint64
		"name": "test",
	}

	err := Write(buf, Options{Format: "json", JQ: ".id"}, data, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	output := buf.String()
	// Verify the exact integer is preserved, not corrupted to 18446744073709552000
	if output != "18446744073709551615\n" {
		t.Fatalf("expected jq to preserve large integer 18446744073709551615, got %q", output)
	}
}

func TestWriteJQPreservesLargeIntegersInStructs(t *testing.T) {
	buf := new(bytes.Buffer)
	// Test with a struct that has a uint64 field (goes through marshal/unmarshal in normaliseForJQ)
	type largeIDStruct struct {
		ID   uint64 `json:"id"`
		Name string `json:"name"`
	}
	data := largeIDStruct{
		ID:   18446744073709551615, // max uint64
		Name: "test",
	}

	err := Write(buf, Options{Format: "json", JQ: ".id"}, data, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	output := buf.String()
	// Verify the exact integer is preserved through the struct marshal/unmarshal cycle
	if output != "18446744073709551615\n" {
		t.Fatalf("expected jq to preserve large integer 18446744073709551615 from struct, got %q", output)
	}
}

func TestWriteYAMLFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	data := sample{Name: "demo", Count: 3}

	err := Write(buf, Options{Format: "yaml"}, data, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name: demo") {
		t.Fatalf("expected YAML name field, got %q", output)
	}
	if !strings.Contains(output, "count: 3") {
		t.Fatalf("expected YAML count field, got %q", output)
	}
}

func TestWriteJSONFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	data := sample{Name: "demo", Count: 5}

	err := Write(buf, Options{Format: "json"}, data, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	output := buf.String()
	// JSON format should be indented
	if !strings.Contains(output, "  \"name\": \"demo\"") {
		t.Fatalf("expected indented JSON, got %q", output)
	}
}

func TestWriteTemplateFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	data := sample{Name: "demo", Count: 7}

	err := Write(buf, Options{Template: "Name={{.Name}} Count={{.Count}}"}, data, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if buf.String() != "Name=demo Count=7" {
		t.Fatalf("expected template output, got %q", buf.String())
	}
}

func TestWriteTemplateInvalidSyntax(t *testing.T) {
	buf := new(bytes.Buffer)
	err := Write(buf, Options{Template: "{{.Invalid"}, nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
	if !strings.Contains(err.Error(), "parse template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteFallback(t *testing.T) {
	called := false
	fallback := func() error {
		called = true
		return nil
	}

	buf := new(bytes.Buffer)
	err := Write(buf, Options{}, nil, fallback)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if !called {
		t.Fatal("expected fallback to be called")
	}
}

func TestWriteFallbackNil(t *testing.T) {
	buf := new(bytes.Buffer)
	err := Write(buf, Options{}, nil, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
}

func TestWriteUnsupportedFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	err := Write(buf, Options{Format: "xml"}, "data", nil)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteJQInvalidExpression(t *testing.T) {
	buf := new(bytes.Buffer)
	err := Write(buf, Options{Format: "json", JQ: ".[invalid"}, "data", nil)
	if err == nil {
		t.Fatal("expected error for invalid jq expression")
	}
}

func TestWriteJQNoResults(t *testing.T) {
	buf := new(bytes.Buffer)
	data := map[string]any{"name": "test"}

	// .missing returns null for objects, which gojq yields as nil
	err := Write(buf, Options{Format: "json", JQ: ".missing"}, data, nil)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
}

func TestNormaliseForJQBytes(t *testing.T) {
	input := []byte(`{"key":"value"}`)
	result, err := normaliseForJQ(input)
	if err != nil {
		t.Fatalf("normaliseForJQ: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["key"] != "value" {
		t.Fatalf("expected value, got %v", m["key"])
	}
}

func TestNormaliseForJQEmptyBytes(t *testing.T) {
	result, err := normaliseForJQ([]byte{})
	if err != nil {
		t.Fatalf("normaliseForJQ: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestNormaliseForJQPassthroughTypes(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"nil", nil},
		{"string", "hello"},
		{"bool", true},
		{"json.Number", json.Number("42")},
		{"float64", float64(3.14)},
		{"int", 42},
		{"int64", int64(100)},
		{"uint", uint(10)},
		{"uint64", uint64(999)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normaliseForJQ(tt.input)
			if err != nil {
				t.Fatalf("normaliseForJQ: %v", err)
			}
			if result != tt.input {
				t.Fatalf("expected passthrough, got %v", result)
			}
		})
	}
}
