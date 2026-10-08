// Package jsonldata implements the shared JSONL/NDJSON record framing.
package jsonldata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// Error identifies the original, one-based physical line of a bad record.
type Error struct {
	Line int
	Err  error
}

func (e *Error) Error() string { return fmt.Sprintf("line %d: %v", e.Line, e.Err) }
func (e *Error) Unwrap() error { return e.Err }

// Parse returns all records, or no records on failure. Numbers follow JSON
// input's float64 model. Input is already read in full; there is no token limit.
func Parse(raw []byte) ([]map[string]any, error) { return parse(raw, false) }

func parse(raw []byte, exactNumbers bool) ([]map[string]any, error) {
	var records []map[string]any
	for line := 1; len(raw) > 0; line++ {
		record, rest, _ := bytes.Cut(raw, []byte{'\n'})
		raw = rest
		fail := func(err error) ([]map[string]any, error) {
			return nil, &Error{Line: line, Err: err}
		}
		if !utf8.Valid(record) {
			return fail(fmt.Errorf("invalid UTF-8"))
		}
		if bytes.HasPrefix(record, []byte{0xef, 0xbb, 0xbf}) {
			return fail(fmt.Errorf("BOM is not allowed"))
		}
		if len(bytes.TrimSpace(record)) == 0 {
			return fail(fmt.Errorf("blank records are not allowed"))
		}
		// Unmarshal enforces exactly one JSON value, including trailing data.
		var data map[string]any
		if err := json.Unmarshal(record, &data); err != nil {
			return fail(err)
		}
		if data == nil {
			return fail(fmt.Errorf("record must be a JSON object"))
		}
		if exactNumbers {
			// Formatting preserves numeric values without float64 rounding.
			dec := json.NewDecoder(bytes.NewReader(record))
			dec.UseNumber()
			if err := dec.Decode(&data); err != nil {
				return fail(err)
			}
		}
		records = append(records, data)
	}
	return records, nil
}

// Format validates the entire file before producing minified, sorted records.
func Format(raw []byte) ([]byte, error) {
	records, err := parse(raw, true)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
