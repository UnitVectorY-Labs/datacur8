package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
	"gopkg.in/yaml.v3"
)

func TestExportJSON(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.json")

	typeDefs := []config.TypeDef{
		{
			Name: "widgets",
			Output: &config.OutputDef{
				Path:   outPath,
				Format: "json",
			},
		},
	}

	items := map[string][]any{
		"widgets": {
			map[string]any{"name": "alpha", "count": 1},
			map[string]any{"name": "beta", "count": 2},
		},
	}

	results, errs := Export(items, typeDefs, dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Count != 2 {
		t.Errorf("expected count 2, got %d", results[0].Count)
	}
	if results[0].Format != "json" {
		t.Errorf("expected format json, got %s", results[0].Format)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var parsed map[string][]map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parsing output JSON: %v", err)
	}
	if len(parsed["widgets"]) != 2 {
		t.Fatalf("expected 2 widgets, got %d", len(parsed["widgets"]))
	}
	if parsed["widgets"][0]["name"] != "alpha" {
		t.Errorf("expected first widget name alpha, got %v", parsed["widgets"][0]["name"])
	}

	// Verify pretty-printed with 2-space indent
	content := string(data)
	if !strings.Contains(content, "  ") {
		t.Error("expected 2-space indented output")
	}
	// Verify trailing newline
	if content[len(content)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}

func TestExportYAML(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.yaml")

	typeDefs := []config.TypeDef{
		{
			Name: "gadgets",
			Output: &config.OutputDef{
				Path:   outPath,
				Format: "yaml",
			},
		},
	}

	items := map[string][]any{
		"gadgets": {
			map[string]any{"id": "g1", "label": "first"},
			map[string]any{"id": "g2", "label": "second"},
		},
	}

	results, errs := Export(items, typeDefs, dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Count != 2 {
		t.Errorf("expected count 2, got %d", results[0].Count)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var parsed map[string][]map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parsing output YAML: %v", err)
	}
	if len(parsed["gadgets"]) != 2 {
		t.Fatalf("expected 2 gadgets, got %d", len(parsed["gadgets"]))
	}
	if parsed["gadgets"][0]["id"] != "g1" {
		t.Errorf("expected first gadget id g1, got %v", parsed["gadgets"][0]["id"])
	}
}

func TestExportJSONL(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.jsonl")

	typeDefs := []config.TypeDef{
		{
			Name: "events",
			Output: &config.OutputDef{
				Path:   outPath,
				Format: "jsonl",
			},
		},
	}

	items := map[string][]any{
		"events": {
			map[string]any{"ts": "2024-01-01", "msg": "hello"},
			map[string]any{"ts": "2024-01-02", "msg": "world"},
		},
	}

	results, errs := Export(items, typeDefs, dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Count != 2 {
		t.Errorf("expected count 2, got %d", results[0].Count)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Verify each line is valid minified JSON
	for i, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
		// Verify minified (no unnecessary spaces)
		if strings.Contains(line, "  ") {
			t.Errorf("line %d should be minified", i)
		}
	}
}

func TestExportNoOutput(t *testing.T) {
	typeDefs := []config.TypeDef{
		{Name: "things"},
	}

	items := map[string][]any{
		"things": {map[string]any{"a": 1}},
	}

	results, errs := Export(items, typeDefs, t.TempDir())
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for no-output types, got %d", len(results))
	}
}

func TestExportCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "sub", "dir", "out.json")

	typeDefs := []config.TypeDef{
		{
			Name: "items",
			Output: &config.OutputDef{
				Path:   outPath,
				Format: "json",
			},
		},
	}

	items := map[string][]any{
		"items": {map[string]any{"k": "v"}},
	}

	results, errs := Export(items, typeDefs, dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

func TestExportEmptyItems(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "empty.json")

	typeDefs := []config.TypeDef{
		{
			Name: "empty",
			Output: &config.OutputDef{
				Path:   outPath,
				Format: "json",
			},
		},
	}

	items := map[string][]any{}

	results, errs := Export(items, typeDefs, dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Count != 0 {
		t.Errorf("expected count 0, got %d", results[0].Count)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var parsed map[string][]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parsing output: %v", err)
	}
	if len(parsed["empty"]) != 0 {
		t.Errorf("expected empty array, got %d items", len(parsed["empty"]))
	}
}

func TestExportRelativePath(t *testing.T) {
	dir := t.TempDir()

	typeDefs := []config.TypeDef{
		{
			Name: "rel",
			Output: &config.OutputDef{
				Path:   "output/rel.json",
				Format: "json",
			},
		},
	}

	items := map[string][]any{
		"rel": {map[string]any{"x": 1}},
	}

	results, errs := Export(items, typeDefs, dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	expected := filepath.Join(dir, "output", "rel.json")
	if results[0].Path != expected {
		t.Errorf("expected path %s, got %s", expected, results[0].Path)
	}
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("output file not created at resolved path: %v", err)
	}
}

func TestExportUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()

	typeDefs := []config.TypeDef{
		{
			Name: "bad",
			Output: &config.OutputDef{
				Path:   filepath.Join(dir, "out.xml"),
				Format: "xml",
			},
		},
	}

	items := map[string][]any{
		"bad": {map[string]any{"a": 1}},
	}

	results, errs := Export(items, typeDefs, dir)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if !strings.Contains(errs[0].Error(), "unsupported") {
		t.Errorf("expected unsupported format error, got: %v", errs[0])
	}
}
