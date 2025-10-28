package format

import (
	"bytes"
	"encoding/json"
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
