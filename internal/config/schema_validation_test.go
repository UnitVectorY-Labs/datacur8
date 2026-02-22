package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ConfigSchemaRejectsAdditionalTopLevelProperty(t *testing.T) {
	cfgText := `
version: "0.0.0"
types: []
extra_top_level: true
`

	path := writeTempConfig(t, cfgText)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "configuration does not match schema") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_ConfigSchemaRejectsMissingVersion(t *testing.T) {
	cfgText := `
types: []
`

	path := writeTempConfig(t, cfgText)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "configuration does not match schema") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_ConfigSchemaAcceptsValidMinimalConfig(t *testing.T) {
	cfgText := `
version: "0.0.0"
types: []
`

	path := writeTempConfig(t, cfgText)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Version != "0.0.0" {
		t.Fatalf("unexpected version: %q", cfg.Version)
	}
}

func TestLoad_ConfigSchemaRejectsCSVConfigProperty(t *testing.T) {
	cfgText := `
version: "0.0.0"
types:
  - name: records
    input: csv
    match:
      include: ["^data/records\\.csv$"]
    schema:
      type: object
    csv: {}
`

	path := writeTempConfig(t, cfgText)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "configuration does not match schema") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_ConfigSchemaRejectsPerTypeTidyConfig(t *testing.T) {
	cfgText := `
version: "0.0.0"
types:
  - name: records
    input: json
    match:
      include: ["^data/records\\.json$"]
    schema:
      type: object
    tidy:
      sort_arrays_by: ["name"]
`

	path := writeTempConfig(t, cfgText)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "configuration does not match schema") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTempConfig(t *testing.T, cfgText string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".datacur8")
	if err := os.WriteFile(path, []byte(cfgText), 0o644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}
