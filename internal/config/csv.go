package config

import (
	"fmt"
	"sort"
	"unicode/utf8"
)

// CSVColumn describes a scalar column declared directly in schema.properties.
type CSVColumn struct {
	Name string
	Type string
}

// CSVColumns enforces the deliberately small CSV export schema subset.
// Validation keywords (required, enum, bounds, etc.) still run in the normal pipeline.
func CSVColumns(schema map[string]any) ([]CSVColumn, error) {
	if schema["type"] != "object" {
		return nil, fmt.Errorf("CSV export requires schema.type object")
	}
	if err := csvSchemaKeywords(schema); err != nil {
		return nil, err
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok || len(props) == 0 {
		return nil, fmt.Errorf("CSV export requires nonempty schema.properties")
	}
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)
	columns := make([]CSVColumn, 0, len(names))
	for _, name := range names {
		prop, ok := props[name].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("CSV export property %q requires an explicit scalar type", name)
		}
		if err := csvSchemaKeywords(prop); err != nil {
			return nil, fmt.Errorf("property %q: %w", name, err)
		}
		typ, _ := prop["type"].(string)
		switch typ {
		case "string", "boolean", "integer", "number":
		default:
			return nil, fmt.Errorf("CSV export property %q must have one explicit type: string, boolean, integer, or number", name)
		}
		columns = append(columns, CSVColumn{Name: name, Type: typ})
	}
	return columns, nil
}

func csvSchemaKeywords(schema map[string]any) error {
	for _, keyword := range []string{"$ref", "$dynamicRef", "allOf", "anyOf", "oneOf", "not", "if", "then", "else", "patternProperties", "dependentSchemas"} {
		if _, exists := schema[keyword]; exists {
			return fmt.Errorf("CSV export does not support schema keyword %q", keyword)
		}
	}
	return nil
}

// CSVOptions applies independently to input/tidy and export.
// A pointer distinguishes an omitted delimiter (comma) from an invalid empty one.
type CSVOptions struct {
	Delimiter *string `yaml:"delimiter,omitempty"`
}

// Rune returns the configured separator, defaulting to comma.
func (o *CSVOptions) Rune() rune {
	if o == nil || o.Delimiter == nil {
		return ','
	}
	r, _ := utf8.DecodeRuneInString(*o.Delimiter)
	return r
}

func validateCSVOptions(prefix, format string, o *CSVOptions) []error {
	if o == nil {
		return nil
	}
	if format != "csv" {
		return []error{fmt.Errorf("%s: csv options require csv format", prefix)}
	}
	if o.Delimiter != nil {
		s := *o.Delimiter
		r := o.Rune()
		if !utf8.ValidString(s) || utf8.RuneCountInString(s) != 1 || r == 0 || r == '"' || r == '\r' || r == '\n' || r == utf8.RuneError {
			return []error{fmt.Errorf("%s.csv.delimiter must be one permitted Unicode rune (not quote, CR, LF, NUL, or replacement character)", prefix)}
		}
	}
	return nil
}

// CSVHeaders rejects ambiguous columns before validation or tidy can lose data.
func CSVHeaders(headers []string) error {
	seen := make(map[string]bool, len(headers))
	for _, h := range headers {
		if h == "" {
			return fmt.Errorf("CSV header must not be empty")
		}
		if seen[h] {
			return fmt.Errorf("duplicate CSV header %q", h)
		}
		seen[h] = true
	}
	return nil
}
