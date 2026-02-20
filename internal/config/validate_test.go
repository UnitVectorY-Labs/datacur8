package config

import (
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Version:    "1.0.0",
		StrictMode: "ENABLED",
		Reporting:  &Reporting{Mode: "json"},
		Types: []TypeDef{
			{
				Name:  "users",
				Input: "json",
				Match: MatchDef{Include: []string{`users/.*\.json`}},
				Schema: map[string]any{
					"type": "object",
				},
			},
		},
	}
	warnings, errs := Validate(cfg, "1.2.0")
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got: %v", warnings)
	}
}

func TestValidate_VersionNotSemver(t *testing.T) {
	cfg := &Config{Version: "abc", Types: []TypeDef{}}
	_, errs := Validate(cfg, "1.0.0")
	requireError(t, errs, "not valid semver")
}

func TestValidate_MajorVersionMismatch(t *testing.T) {
	cfg := &Config{Version: "2.0.0", Types: []TypeDef{}}
	_, errs := Validate(cfg, "1.5.0")
	requireError(t, errs, "major version mismatch")
}

func TestValidate_CLIOlderThanConfig(t *testing.T) {
	cfg := &Config{Version: "1.3.0", Types: []TypeDef{}}
	_, errs := Validate(cfg, "1.2.0")
	requireError(t, errs, "CLI version 1.2.0 is older")
}

func TestValidate_DevCLISkipsVersionCheck(t *testing.T) {
	cfg := &Config{Version: "9.9.9", Types: []TypeDef{}}
	warnings, errs := Validate(cfg, "dev")
	// no version error expected
	for _, e := range errs {
		if strings.Contains(e.Error(), "version") && strings.Contains(e.Error(), "mismatch") {
			t.Fatalf("should skip version comparison for dev CLI, got: %v", e)
		}
	}
	requireWarning(t, warnings, "dev/empty")
}

func TestValidate_EmptyCLISkipsVersionCheck(t *testing.T) {
	cfg := &Config{Version: "1.0.0", Types: []TypeDef{}}
	warnings, _ := Validate(cfg, "")
	requireWarning(t, warnings, "dev/empty")
}

func TestValidate_StrictModeInvalid(t *testing.T) {
	cfg := &Config{Version: "1.0.0", StrictMode: "INVALID", Types: []TypeDef{}}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "strict_mode")
}

func TestValidate_StrictModeValid(t *testing.T) {
	for _, mode := range []string{"", "DISABLED", "ENABLED", "FORCE"} {
		cfg := &Config{Version: "1.0.0", StrictMode: mode, Types: []TypeDef{}}
		_, errs := Validate(cfg, "dev")
		for _, e := range errs {
			if strings.Contains(e.Error(), "strict_mode") {
				t.Fatalf("strict_mode %q should be valid, got: %v", mode, e)
			}
		}
	}
}

func TestValidate_ReportingModeInvalid(t *testing.T) {
	cfg := &Config{
		Version:   "1.0.0",
		Reporting: &Reporting{Mode: "xml"},
		Types:     []TypeDef{},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "reporting.mode")
}

func TestValidate_DuplicateTypeName(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "dup", Input: "json", Match: MatchDef{Include: []string{"a"}}, Schema: map[string]any{"type": "object"}},
			{Name: "dup", Input: "json", Match: MatchDef{Include: []string{"b"}}, Schema: map[string]any{"type": "object"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "duplicate type name")
}

func TestValidate_TypeNameInvalid(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "1bad", Input: "json", Match: MatchDef{Include: []string{"a"}}, Schema: map[string]any{"type": "object"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "type name must match")
}

func TestValidate_InputInvalid(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "xml", Match: MatchDef{Include: []string{"a"}}, Schema: map[string]any{"type": "object"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "must be json, yaml, or csv")
}

func TestValidate_EmptyInclude(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{}, Schema: map[string]any{"type": "object"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "at least 1 pattern")
}

func TestValidate_InvalidRegex(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"[invalid"}}, Schema: map[string]any{"type": "object"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "invalid regex")
}

func TestValidate_SchemaTypeMissing(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"a"}}, Schema: map[string]any{"type": "array"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, `schema.type must be "object"`)
}

func TestValidate_CSVRequiresConfig(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "csv", Match: MatchDef{Include: []string{"a"}}, Schema: map[string]any{"type": "object"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "csv config is required")
}

func TestValidate_CSVDelimiterLength(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "csv", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				CSV:    &CSVDef{Delimiter: "ab"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "delimiter must be exactly 1 character")
}

func TestValidate_OutputPathConflict(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "a", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				Output: &OutputDef{Path: "out.json", Format: "json"}},
			{Name: "b", Input: "json", Match: MatchDef{Include: []string{"b"}},
				Schema: map[string]any{"type": "object"},
				Output: &OutputDef{Path: "out.json", Format: "json"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "output.path")
}

func TestValidate_OutputFormatInvalid(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "a", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				Output: &OutputDef{Path: "out.json", Format: "xml"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "output.format")
}

func TestValidate_ConstraintUnique(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "unique", Key: "$.id", Scope: "type"},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	for _, e := range errs {
		if strings.Contains(e.Error(), "constraint") {
			t.Fatalf("valid unique constraint should not error, got: %v", e)
		}
	}
}

func TestValidate_ConstraintUniqueBadScope(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "unique", Key: "$.id", Scope: "global"},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "scope")
}

func TestValidate_ConstraintForeignKeyValid(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "users", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"}},
			{Name: "orders", Input: "json", Match: MatchDef{Include: []string{"b"}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "foreign_key", Key: "$.user_id",
						References: &ReferenceDef{Type: "users", Key: "$.id"}},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	for _, e := range errs {
		if strings.Contains(e.Error(), "constraint") || strings.Contains(e.Error(), "references") {
			t.Fatalf("valid foreign_key should not error, got: %v", e)
		}
	}
}

func TestValidate_ConstraintForeignKeyBadRef(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "foreign_key", Key: "$.id",
						References: &ReferenceDef{Type: "nonexistent", Key: "$.id"}},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "does not match any defined type")
}

func TestValidate_ConstraintPathEqualsAttr(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json",
				Match: MatchDef{Include: []string{`(?P<env>[a-z]+)/.*\.json`}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "path_equals_attr", PathSelector: "path.env",
						References: &ReferenceDef{Key: "$.environment"}},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	for _, e := range errs {
		if strings.Contains(e.Error(), "constraint") || strings.Contains(e.Error(), "path_selector") {
			t.Fatalf("valid path_equals_attr should not error, got: %v", e)
		}
	}
}

func TestValidate_ConstraintPathEqualsAttrMissingCapture(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json",
				Match: MatchDef{Include: []string{`[a-z]+/.*\.json`}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "path_equals_attr", PathSelector: "path.env",
						References: &ReferenceDef{Key: "$.environment"}},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "does not define named group")
}

func TestValidate_UnknownConstraintType(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "unknown"},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "unknown constraint type")
}

func TestValidate_ConstraintForeignKeyMissingReferences(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "foreign_key", Key: "$.id"},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "references is required")
}

func TestValidate_PathEqualsAttrBuiltinSegment(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json",
				Match: MatchDef{Include: []string{`.*\.json`}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "path_equals_attr", PathSelector: "path.file",
						References: &ReferenceDef{Key: "$.name"}},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	for _, e := range errs {
		if strings.Contains(e.Error(), "capture") {
			t.Fatalf("path.file is builtin, should not require capture group, got: %v", e)
		}
	}
}

func TestValidate_CLIEqualToConfig(t *testing.T) {
	cfg := &Config{Version: "1.0.0", Types: []TypeDef{}}
	_, errs := Validate(cfg, "1.0.0")
	for _, e := range errs {
		if strings.Contains(e.Error(), "older") {
			t.Fatalf("CLI == config version should be ok, got: %v", e)
		}
	}
}

func TestValidate_InvalidSelector(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"a"}},
				Schema: map[string]any{"type": "object"},
				Constraints: []ConstraintDef{
					{Type: "unique", Key: "bad-selector"},
				}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "not a valid selector")
}

func TestValidate_SchemaRequired(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json", Match: MatchDef{Include: []string{"a"}}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "schema is required")
}

func TestValidate_InvalidExcludeRegex(t *testing.T) {
	cfg := &Config{
		Version: "1.0.0",
		Types: []TypeDef{
			{Name: "t", Input: "json",
				Match:  MatchDef{Include: []string{"a"}, Exclude: []string{"[bad"}},
				Schema: map[string]any{"type": "object"}},
		},
	}
	_, errs := Validate(cfg, "dev")
	requireError(t, errs, "match.exclude[0] invalid regex")
}

// helpers

func requireError(t *testing.T, errs []error, substr string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e.Error(), substr) {
			return
		}
	}
	t.Fatalf("expected error containing %q, got: %v", substr, errs)
}

func requireWarning(t *testing.T, warnings []string, substr string) {
	t.Helper()
	for _, w := range warnings {
		if strings.Contains(w, substr) {
			return
		}
	}
	t.Fatalf("expected warning containing %q, got: %v", substr, warnings)
}
