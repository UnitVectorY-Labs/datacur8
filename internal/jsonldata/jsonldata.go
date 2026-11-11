// Package jsonldata implements the shared JSONL/NDJSON record framing.
package jsonldata

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
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
func Parse(raw []byte) ([]map[string]any, error) { return parse(raw) }

func parse(raw []byte) ([]map[string]any, error) {
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
		var value any
		if err := json.Unmarshal(record, &value); err != nil {
			return fail(err)
		}
		data, ok := value.(map[string]any)
		if !ok {
			return fail(fmt.Errorf("record must be a JSON object"))
		}

		records = append(records, data)
	}
	return records, nil
}

// Format validates the entire file before producing minified, sorted records.
func Format(raw []byte) ([]byte, error) {
	if _, err := parse(raw); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for len(raw) > 0 {
		record, rest, _ := bytes.Cut(raw, []byte{'\n'})
		raw = rest
		value := jsontext.Value(bytes.Clone(record))
		if err := value.Format(jsontext.ReorderRawObjects(true)); err != nil {
			return nil, err
		}
		buf.Write(value)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}
