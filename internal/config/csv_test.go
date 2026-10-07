package config

import "testing"

func TestCSVExportSchema(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema map[string]any
		bad    bool
	}{
		{"scalar", map[string]any{"type": "object", "properties": map[string]any{"z": map[string]any{"type": "number"}, "a": map[string]any{"type": "string"}}}, false},
		{"no columns", map[string]any{"type": "object"}, true},
		{"boolean schema", map[string]any{"type": "object", "properties": map[string]any{"a": true}}, true},
		{"array", map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"type": "array"}}}, true},
		{"reference", map[string]any{"type": "object", "$ref": "#/$defs/x"}, true},
		{"composition", map[string]any{"type": "object", "allOf": []any{}}, true},
		{"property reference", map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"type": "string", "$ref": "#/$defs/x"}}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Version: "0.0.0", Types: []TypeDef{{Name: "items", Input: "json", Match: MatchDef{Include: []string{".*"}}, Schema: tc.schema, Output: &OutputDef{Format: "csv", Path: "out.csv"}}}}
			_, errs := Validate(cfg, "dev")
			if (len(errs) > 0) != tc.bad {
				t.Fatalf("errors=%v, want error=%v", errs, tc.bad)
			}
			if !tc.bad {
				cols, err := CSVColumns(tc.schema)
				if err != nil || cols[0].Name != "a" || cols[1].Name != "z" {
					t.Fatalf("columns=%v, error=%v", cols, err)
				}
			}
		})
	}
}
