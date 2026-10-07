package config

import (
	"fmt"
	"sort"
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
