package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tomlWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestTOMLExportFailuresPreserveDestination(t *testing.T) {
	for _, tc := range []struct {
		data, context string
		code          int
	}{
		{`{"value":null}`, `$["records"][0]["value"]: null`, 3},
		{`{"value":{"nested":[null]}}`, `$["records"][0]["value"]["nested"][0]: null`, 3},
		{`{"value":[null]}`, `$["records"][0]["value"][0]: null`, 3},
		{`{"value":`, "parsing JSON", 2},
	} {
		t.Run(tc.data, func(t *testing.T) {
			dir := t.TempDir()
			copyDir(t, filepath.Join(testsDir(), "toml_export_from_json"), dir)
			tomlWrite(t, dir, "data/1.json", tc.data)
			tomlWrite(t, dir, "out/nested/records.toml", "existing destination\n")
			out := csvCommand(t, dir, tc.code, "export")
			if !strings.Contains(out, tc.context) {
				t.Fatal(out)
			}
			got, err := os.ReadFile(filepath.Join(dir, "out/nested/records.toml"))
			if err != nil || string(got) != "existing destination\n" {
				t.Fatalf("destination changed: %q / %v", got, err)
			}
		})
	}
}

func TestTOMLExportWriteFailure(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testsDir(), "toml_export_from_json"), dir)
	if err := os.MkdirAll(filepath.Join(dir, "out/nested/records.toml"), 0755); err != nil {
		t.Fatal(err)
	}
	if out := csvCommand(t, dir, 3, "export"); !strings.Contains(out, "writing output file") {
		t.Fatal(out)
	}
}

func TestTOMLTidyInvalidNoWrite(t *testing.T) {
	for _, name := range []string{"toml_duplicate_key", "toml_table_redefinition", "toml_invalid_syntax", "toml_nan", "toml_inexact_integer"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			copyDir(t, filepath.Join(testsDir(), name), dir)
			p := filepath.Join(dir, "data/1.toml")
			original, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"tidy"}, {"tidy", "--write"}} {
				csvCommand(t, dir, 4, args...)
				got, err := os.ReadFile(p)
				if err != nil || !bytes.Equal(original, got) {
					t.Fatalf("rewrote invalid input: %q / %v", got, err)
				}
			}
		})
	}
}

func TestTOMLTidyIdempotent(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testsDir(), "tidy_toml"), dir)
	csvCommand(t, dir, 5, "tidy")
	csvCommand(t, dir, 0, "tidy", "--write")
	csvCommand(t, dir, 0, "tidy")
	csvCommand(t, dir, 0, "tidy", "--write")
	csvCommand(t, dir, 0, "validate")
}

func TestTOMLOutputExcludedAndRepeatable(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testsDir(), "example_examples_toml_catalog_success"), dir)
	cfg, err := os.ReadFile(filepath.Join(dir, ".datacur8"))
	if err != nil {
		t.Fatal(err)
	}
	// Broaden matching so the configured aggregate output must be excluded.
	tomlWrite(t, dir, ".datacur8", strings.Replace(string(cfg), `^data/.*\.toml$`, `.*\.toml$`, 1))
	csvCommand(t, dir, 0, "export")
	first, err := os.ReadFile(filepath.Join(dir, "out/nested/records.toml"))
	if err != nil {
		t.Fatal(err)
	}
	csvCommand(t, dir, 0, "export")
	second, err := os.ReadFile(filepath.Join(dir, "out/nested/records.toml"))
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("repeat: %q / %v", second, err)
	}
}

func TestTOMLNativeYAMLIntegerExportFailure(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testsDir(), "toml_export_from_yaml"), dir)
	tomlWrite(t, dir, "data/1.yaml", "id: a\nactive: true\ncount: 9007199254740993\n")
	tomlWrite(t, dir, "out/nested/records.toml", "existing destination\n")
	csvCommand(t, dir, 0, "validate")
	out := csvCommand(t, dir, 3, "export")
	if !strings.Contains(out, `$["records"][0]["count"]: integer cannot be represented exactly`) {
		t.Fatal(out)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out/nested/records.toml"))
	if err != nil || string(got) != "existing destination\n" {
		t.Fatalf("destination changed: %q / %v", got, err)
	}
}
