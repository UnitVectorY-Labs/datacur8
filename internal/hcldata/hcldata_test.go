package hcldata

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarshalRoundTrip(t *testing.T) {
	data := []any{map[string]any{
		"id":                "widget-1",
		"not an identifier": map[string]any{"quoted\"key": "${value} %{if true}\n\"quoted\"\\"},
		"values":            []any{1.25, true, nil, "text", map[string]any{"nested": []any{}}},
		"empty_object":      map[string]any{},
		"empty_array":       []any{},
		"nothing":           nil,
	}, map[string]any{"id": "widget-2"}}
	first, err := Marshal("widgets", data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Marshal("widgets", data)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("HCL output is not deterministic")
	}
	got, err := Parse(first, "widgets.hcl")
	if err != nil {
		t.Fatalf("parsing exported HCL: %v\n%s", err, first)
	}
	// Normalize Go numeric types exactly as the JSON data model does.
	expectedJSON, err := json.Marshal(map[string]any{"widgets": data})
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal(expectedJSON, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roundtrip = %#v, want %#v", got, want)
	}
}

func TestFormatIdempotent(t *testing.T) {
	src := []byte("# keep comment\nz=1 + 2\nid=\"widget-1\" # inline\n")
	formatted, err := Format(src, "widget.hcl")
	if err != nil {
		t.Fatal(err)
	}
	again, err := Format(formatted, "widget.hcl")
	if err != nil {
		t.Fatal(err)
	}
	if string(formatted) != string(again) {
		t.Fatalf("format is not idempotent:\n%s\n%s", formatted, again)
	}
	if string(formatted) != "# keep comment\nz  = 1 + 2\nid = \"widget-1\" # inline\n" {
		t.Fatalf("format changed comments, order, or expression: %s", formatted)
	}
}
