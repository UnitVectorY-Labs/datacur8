package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func csvCommand(t *testing.T, dir string, want int, args ...string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if code != want {
		t.Fatalf("%v exit=%d, want %d: %s", args, code, want, out)
	}
	return string(out)
}

func TestCSVExportFailuresPreserveDestination(t *testing.T) {
	for _, tc := range []struct {
		name, data, context      string
		validateCode, exportCode int
	}{
		{"missing", `{"price":1,"stock":3}`, `property "name": missing CSV cell`, 0, 3},
		{"extra", `{"name":"x","price":1,"stock":3,"extra":{"x":1}}`, `property "extra": undeclared CSV column`, 0, 3},
		{"integer overflow", `{"name":"x","price":1,"stock":9223372036854775808}`, `property "stock": integer must be integral`, 0, 3},
		{"null source", `{"name":null,"price":1,"stock":3}`, ``, 2, 2},
		{"source invalid", `{"name":"x","price":[],"stock":3}`, ``, 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			copyDir(t, filepath.Join(testsDir(), "example_examples_csv_export_success"), dir)
			configPath := filepath.Join(dir, ".datacur8")
			cfg, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			cfg = bytes.ReplaceAll(cfg, []byte("      required: [name, price, stock]\n"), nil)
			cfg = bytes.ReplaceAll(cfg, []byte("      additionalProperties: false\n"), nil)
			if err := os.WriteFile(configPath, cfg, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "data/1.json"), []byte(tc.data), 0600); err != nil {
				t.Fatal(err)
			}
			outPath := filepath.Join(dir, "out/products.csv")
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				t.Fatal(err)
			}
			original := []byte("existing destination\n")
			if err := os.WriteFile(outPath, original, 0600); err != nil {
				t.Fatal(err)
			}
			csvCommand(t, dir, tc.validateCode, "validate")
			out := csvCommand(t, dir, tc.exportCode, "export")
			if tc.context != "" && !strings.Contains(out, "item 0, "+tc.context) {
				t.Fatalf("missing item/property context: %s", out)
			}
			got, err := os.ReadFile(outPath)
			if err != nil || !bytes.Equal(got, original) {
				t.Fatalf("destination changed: %q, %v", got, err)
			}
		})
	}
}

func TestCSVExportRepeatAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testsDir(), "csv_export_from_formats"), dir)
	csvCommand(t, dir, 0, "export")
	first, err := os.ReadFile(filepath.Join(dir, "out/json.csv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"json", "yaml", "csv", "hcl"} {
		got, err := os.ReadFile(filepath.Join(dir, "out/"+format+".csv"))
		if err != nil || !bytes.Equal(first, got) {
			t.Fatalf("%s differs: %q, %v", format, got, err)
		}
	}
	csvCommand(t, dir, 0, "export")
	again, err := os.ReadFile(filepath.Join(dir, "out/json.csv"))
	if err != nil || !bytes.Equal(first, again) {
		t.Fatalf("repeat differs: %q, %v", again, err)
	}
	// Use precisely the same schema, now reading exported CSV instead of JSON.
	cfgPath := filepath.Join(dir, ".datacur8")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg = bytes.Replace(cfg, []byte("input: json"), []byte("input: csv"), 1)
	cfg = bytes.Replace(cfg, []byte(`^data/json/.*\.json$`), []byte(`^roundtrip\.csv$`), 1)
	if err := os.WriteFile(filepath.Join(dir, "roundtrip.csv"), first, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, cfg, 0600); err != nil {
		t.Fatal(err)
	}
	csvCommand(t, dir, 0, "validate")
	csvCommand(t, dir, 0, "export")
	got, err := os.ReadFile(filepath.Join(dir, "out/json.csv"))
	if err != nil || !bytes.Equal(first, got) {
		t.Fatalf("round trip differs: %q, %v", got, err)
	}
}

func TestCSVExportWriteFailure(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testsDir(), "example_examples_csv_export_success"), dir)
	if err := os.MkdirAll(filepath.Join(dir, "out/products.csv"), 0755); err != nil {
		t.Fatal(err)
	}
	out := csvCommand(t, dir, 3, "export")
	if !strings.Contains(out, "writing output file") {
		t.Fatal(out)
	}
}

func TestCSVSingleEmptyStringRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := `version: "0.0.0"
types:
  - name: values
    input: json
    match:
      include: ['^item\.json$']
    schema:
      type: object
      properties:
        value: {type: string}
    output:
      path: out.csv
      format: csv
`
	if err := os.WriteFile(filepath.Join(dir, ".datacur8"), []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "item.json"), []byte(`{"value":""}`), 0600); err != nil {
		t.Fatal(err)
	}
	csvCommand(t, dir, 0, "export")
	got, err := os.ReadFile(filepath.Join(dir, "out.csv"))
	if err != nil || string(got) != "value\n\"\"\n" {
		t.Fatalf("got %q, %v", got, err)
	}
	cfg = strings.Replace(cfg, "input: json", "input: csv", 1)
	cfg = strings.Replace(cfg, `^item\.json$`, `^roundtrip\.csv$`, 1)
	if err := os.WriteFile(filepath.Join(dir, ".datacur8"), []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "roundtrip.csv"), got, 0600); err != nil {
		t.Fatal(err)
	}
	csvCommand(t, dir, 0, "export")
	again, err := os.ReadFile(filepath.Join(dir, "out.csv"))
	if err != nil || !bytes.Equal(got, again) {
		t.Fatalf("round trip differs: %q, %v", again, err)
	}
}
