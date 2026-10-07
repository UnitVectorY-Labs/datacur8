package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
)

func marshalCSV(schema map[string]any, data []any, options ...*config.CSVOptions) ([]byte, error) {
	columns, err := config.CSVColumns(schema)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf) // LF records; preserve CR within quoted strings.
	if len(options) > 0 {
		writer.Comma = options[0].Rune()
	}
	headers := make([]string, len(columns))
	declared := make(map[string]bool, len(columns))
	for i, col := range columns {
		headers[i] = col.Name
		declared[col.Name] = true
	}
	if err := writeCSVRecord(writer, &buf, headers); err != nil {
		return nil, err
	}
	for i, item := range data {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("item %d: expected object", i)
		}
		var extra []string
		for name := range object {
			if !declared[name] {
				extra = append(extra, name)
			}
		}
		sort.Strings(extra)
		if len(extra) > 0 {
			return nil, fmt.Errorf("item %d, property %q: undeclared CSV column", i, extra[0])
		}
		row := make([]string, len(columns))
		for j, col := range columns {
			value, exists := object[col.Name]
			if !exists {
				return nil, fmt.Errorf("item %d, property %q: missing CSV cell", i, col.Name)
			}
			if value == nil {
				return nil, fmt.Errorf("item %d, property %q: null CSV cell", i, col.Name)
			}
			cell, err := csvCell(value, col.Type)
			if err != nil {
				return nil, fmt.Errorf("item %d, property %q: %w", i, col.Name, err)
			}
			row[j] = cell
		}
		if err := writeCSVRecord(writer, &buf, row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func csvCell(value any, typ string) (string, error) {
	switch typ {
	case "string":
		if s, ok := value.(string); ok {
			return s, nil
		}
	case "boolean":
		if b, ok := value.(bool); ok {
			return strconv.FormatBool(b), nil
		}
	case "integer", "number":
		// Preserve native YAML integers without a float64 round trip. JSON/HCL and
		// CSV inputs already use float64, whose original parser precision is final.
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(v.Int(), 10), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := v.Uint()
			if typ == "integer" && n > math.MaxInt64 {
				return "", fmt.Errorf("integer outside signed 64-bit range")
			}
			return strconv.FormatUint(n, 10), nil
		case reflect.Float32, reflect.Float64:
			f := v.Float()
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return "", fmt.Errorf("non-finite CSV number")
			}
			if typ == "integer" {
				// The upper bound is exclusive: float64(MaxInt64) rounds to 2^63.
				if f < -0x1p63 || f >= 0x1p63 || math.Trunc(f) != f {
					return "", fmt.Errorf("integer must be integral and within signed 64-bit range")
				}
				return strconv.FormatInt(int64(f), 10), nil
			}
			return strconv.FormatFloat(f, 'g', -1, v.Type().Bits()), nil
		}
	}
	return "", fmt.Errorf("expected %s CSV scalar, got %T", typ, value)
}

// Go's writer emits a blank line for a single empty field, which its reader
// skips. Quote that one case explicitly so an empty string remains an item.
func writeCSVRecord(writer *csv.Writer, buf *bytes.Buffer, record []string) error {
	if len(record) == 1 && record[0] == "" {
		writer.Flush()
		if err := writer.Error(); err != nil {
			return err
		}
		_, err := buf.WriteString("\"\"\n")
		return err
	}
	return writer.Write(record)
}
