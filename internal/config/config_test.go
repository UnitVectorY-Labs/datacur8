package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

const validConfig = `
version: "1"
strict_mode: ENABLED
tidy:
  enabled: true
types:
  - name: users
    input: json
    match:
      include:
        - "data/users/*.json"
      exclude:
        - "data/users/skip.json"
    schema:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
      required:
        - id
    constraints:
      - id: unique-id
        type: unique
        key: "$.id"
        case_sensitive: false
        scope: item
      - type: foreign_key
        key: "$.role_id"
        references:
          type: roles
          key: "$.id"
    output:
      path: "output/users"
      format: jsonl
  - name: roles
    input: yaml
    match:
      include:
        - "data/roles/*.yaml"
    schema:
      type: object
      properties:
        id:
          type: string
`

func parseConfig(t *testing.T, data string) *Config {
	t.Helper()
	var cfg Config
	if err := yaml.Unmarshal([]byte(data), &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	cfg.Defaults()
	return &cfg
}

func TestLoadValidConfig(t *testing.T) {
	cfg := parseConfig(t, validConfig)

	if cfg.Version != "1" {
		t.Errorf("expected version 1, got %s", cfg.Version)
	}
	if cfg.StrictMode != "ENABLED" {
		t.Errorf("expected strict_mode ENABLED, got %s", cfg.StrictMode)
	}
	if len(cfg.Types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(cfg.Types))
	}

	users := cfg.Types[0]
	if users.Name != "users" {
		t.Errorf("expected type name users, got %s", users.Name)
	}
	if users.Input != "json" {
		t.Errorf("expected input json, got %s", users.Input)
	}
	if len(users.Match.Include) != 1 || users.Match.Include[0] != "data/users/*.json" {
		t.Errorf("unexpected include: %v", users.Match.Include)
	}
	if len(users.Match.Exclude) != 1 || users.Match.Exclude[0] != "data/users/skip.json" {
		t.Errorf("unexpected exclude: %v", users.Match.Exclude)
	}
	if users.Output == nil || users.Output.Format != "jsonl" || users.Output.Path != "output/users" {
		t.Errorf("unexpected output: %v", users.Output)
	}
	if len(users.Constraints) != 2 {
		t.Fatalf("expected 2 constraints, got %d", len(users.Constraints))
	}

	uniqueC := users.Constraints[0]
	if uniqueC.ID != "unique-id" || uniqueC.Type != "unique" {
		t.Errorf("unexpected constraint: %+v", uniqueC)
	}
	if uniqueC.IsCaseSensitive() {
		t.Error("expected case_sensitive=false to return false")
	}
	if uniqueC.Scope != "item" {
		t.Errorf("expected scope item, got %s", uniqueC.Scope)
	}

	fkC := users.Constraints[1]
	if fkC.Type != "foreign_key" {
		t.Errorf("expected foreign_key, got %s", fkC.Type)
	}
	if fkC.References == nil || fkC.References.Type != "roles" || fkC.References.Key != "$.id" {
		t.Errorf("unexpected references: %v", fkC.References)
	}
	// scope should default to "type"
	if fkC.Scope != "type" {
		t.Errorf("expected default scope type, got %s", fkC.Scope)
	}

	roles := cfg.Types[1]
	if roles.Name != "roles" || roles.Input != "yaml" {
		t.Errorf("unexpected roles type: %+v", roles)
	}
}

func TestDefaults(t *testing.T) {
	cfg := parseConfig(t, `
version: "1"
types:
  - name: t1
    input: csv
    match:
      include: ["*.csv"]
    schema:
      type: object
    constraints:
      - type: unique
        key: "$.id"
`)

	if cfg.StrictMode != "DISABLED" {
		t.Errorf("expected default strict_mode DISABLED, got %s", cfg.StrictMode)
	}
	if cfg.Types[0].Constraints[0].Scope != "type" {
		t.Errorf("expected default constraint scope type, got %s", cfg.Types[0].Constraints[0].Scope)
	}
}

func TestIsCaseSensitive(t *testing.T) {
	// nil → true
	c := ConstraintDef{}
	if !c.IsCaseSensitive() {
		t.Error("expected nil case_sensitive to be true")
	}

	// explicit true
	tr := true
	c.CaseSensitive = &tr
	if !c.IsCaseSensitive() {
		t.Error("expected explicit true to be true")
	}

	// explicit false
	f := false
	c.CaseSensitive = &f
	if c.IsCaseSensitive() {
		t.Error("expected explicit false to be false")
	}
}

func TestTidyIsEnabled(t *testing.T) {
	// nil TidyConfig
	var tc *TidyConfig
	if !tc.IsEnabled() {
		t.Error("nil TidyConfig should be enabled")
	}

	// nil Enabled field
	tc = &TidyConfig{}
	if !tc.IsEnabled() {
		t.Error("nil Enabled should be enabled")
	}

	// explicit true
	tr := true
	tc.Enabled = &tr
	if !tc.IsEnabled() {
		t.Error("explicit true should be enabled")
	}

	// explicit false
	f := false
	tc.Enabled = &f
	if tc.IsEnabled() {
		t.Error("explicit false should not be enabled")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/.datacur8")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
