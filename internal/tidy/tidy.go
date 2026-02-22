package tidy

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// TidyResult tracks what was tidied
type TidyResult struct {
	Path     string
	Changed  bool   // Whether the file was actually modified
	Original []byte // Original file content
	Tidied   []byte // Tidied file content
}

// TidyFile tidies a single file.
// input is the file format: "json", "yaml", "csv"
// dryRun: if true, don't write changes, just report if they would change
func TidyFile(path string, input string, dryRun bool) (TidyResult, error) {
	switch input {
	case "json":
		return tidyJSON(path, dryRun)
	case "yaml":
		return tidyYAML(path, dryRun)
	case "csv":
		return tidyCSV(path, dryRun)
	default:
		return TidyResult{Path: path}, fmt.Errorf("unsupported input format: %s", input)
	}
}

func tidyJSON(path string, dryRun bool) (TidyResult, error) {
	original, err := os.ReadFile(path)
	if err != nil {
		return TidyResult{Path: path}, fmt.Errorf("reading file: %w", err)
	}

	var data any
	if err := json.Unmarshal(original, &data); err != nil {
		return TidyResult{Path: path}, fmt.Errorf("parsing JSON: %w", err)
	}

	data = sortKeys(data)

	tidied, err := marshalJSONIndent(data)
	if err != nil {
		return TidyResult{Path: path}, fmt.Errorf("marshaling JSON: %w", err)
	}

	changed := !bytes.Equal(original, tidied)
	if changed && !dryRun {
		if err := os.WriteFile(path, tidied, 0o644); err != nil {
			return TidyResult{Path: path}, fmt.Errorf("writing file: %w", err)
		}
	}

	return TidyResult{Path: path, Changed: changed, Original: original, Tidied: tidied}, nil
}

func marshalJSONIndent(data any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func tidyYAML(path string, dryRun bool) (TidyResult, error) {
	original, err := os.ReadFile(path)
	if err != nil {
		return TidyResult{Path: path}, fmt.Errorf("reading file: %w", err)
	}

	var data any
	if err := yaml.Unmarshal(original, &data); err != nil {
		return TidyResult{Path: path}, fmt.Errorf("parsing YAML: %w", err)
	}

	data = normalizeYAML(data)
	data = sortKeys(data)

	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		return TidyResult{Path: path}, fmt.Errorf("marshaling YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return TidyResult{Path: path}, fmt.Errorf("closing YAML encoder: %w", err)
	}
	tidied := buf.Bytes()

	changed := !bytes.Equal(original, tidied)
	if changed && !dryRun {
		if err := os.WriteFile(path, tidied, 0o644); err != nil {
			return TidyResult{Path: path}, fmt.Errorf("writing file: %w", err)
		}
	}

	return TidyResult{Path: path, Changed: changed, Original: original, Tidied: tidied}, nil
}

// normalizeYAML converts YAML-decoded data to JSON-like structures (map[string]any).
// yaml.v3 Unmarshal into any produces map[string]any by default, but this
// ensures consistency for any edge cases.
func normalizeYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v := range val {
			out[k] = normalizeYAML(v)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, v := range val {
			out[i] = normalizeYAML(v)
		}
		return out
	default:
		return v
	}
}

func tidyCSV(path string, dryRun bool) (TidyResult, error) {
	original, err := os.ReadFile(path)
	if err != nil {
		return TidyResult{Path: path}, fmt.Errorf("reading file: %w", err)
	}

	reader := csv.NewReader(bytes.NewReader(original))
	records, err := reader.ReadAll()
	if err != nil {
		return TidyResult{Path: path}, fmt.Errorf("parsing CSV: %w", err)
	}

	if len(records) == 0 {
		return TidyResult{Path: path, Changed: false}, nil
	}

	headers := records[0]

	// Build sorted column index
	type colInfo struct {
		name    string
		origIdx int
	}
	cols := make([]colInfo, len(headers))
	for i, h := range headers {
		cols[i] = colInfo{name: h, origIdx: i}
	}
	sort.SliceStable(cols, func(i, j int) bool {
		return cols[i].name < cols[j].name
	})

	// Reorder all rows according to sorted columns
	sorted := make([][]string, len(records))
	for i, row := range records {
		newRow := make([]string, len(cols))
		for j, c := range cols {
			if c.origIdx < len(row) {
				newRow[j] = row[c.origIdx]
			}
		}
		sorted[i] = newRow
	}

	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if err := writer.WriteAll(sorted); err != nil {
		return TidyResult{Path: path}, fmt.Errorf("writing CSV: %w", err)
	}
	writer.Flush()
	tidied := buf.Bytes()

	changed := !bytes.Equal(original, tidied)
	if changed && !dryRun {
		if err := os.WriteFile(path, tidied, 0o644); err != nil {
			return TidyResult{Path: path}, fmt.Errorf("writing file: %w", err)
		}
	}

	return TidyResult{Path: path, Changed: changed, Original: original, Tidied: tidied}, nil
}

// sortKeys recursively sorts all object keys in the data structure.
func sortKeys(data any) any {
	switch v := data.(type) {
	case map[string]any:
		sorted := make(map[string]any, len(v))
		for k, val := range v {
			sorted[k] = sortKeys(val)
		}
		return sorted
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = sortKeys(val)
		}
		return out
	default:
		return data
	}
}
