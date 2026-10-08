package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestJSONLInvalidFilesNeverRewriteOrExport(t *testing.T) {
	entries, err := os.ReadDir(testsDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || !(strings.HasPrefix(name, "jsonl_") || strings.HasPrefix(name, "ndjson_")) {
			continue
		}
		source := filepath.Join(testsDir(), name)
		code, err := os.ReadFile(filepath.Join(source, "expected/validate.exit"))
		if err != nil || string(code) != "2\n" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			copyDir(t, source, dir)
			// Give every failure fixture a destination and verify validation gates export.
			cfg, err := os.ReadFile(filepath.Join(dir, ".datacur8"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(cfg, []byte("    output:")) {
				// Insert under the first type, before any subsequent type.
				cfg = bytes.Replace(cfg, []byte("    schema:"), []byte("    output: {format: json, path: out/nested/records.json}\n    schema:"), 1)
				tomlWrite(t, dir, ".datacur8", string(cfg))
			}
			tomlWrite(t, dir, "out/nested/records.json", "existing destination\n")
			csvCommand(t, dir, 2, "export")
			got, err := os.ReadFile(filepath.Join(dir, "out/nested/records.json"))
			if err != nil || string(got) != "existing destination\n" {
				t.Fatalf("destination changed: %q / %v", got, err)
			}
			// Only parse failures cause tidy exit 4; schema/constraint failures do not.
			if !strings.Contains(name, "schema") && !strings.Contains(name, "strict") && !strings.Contains(name, "unique") && !strings.Contains(name, "foreign") && !strings.Contains(name, "path") && !strings.Contains(name, "coercion") {
				original, err := os.ReadFile(filepath.Join(dir, "data/1.ndjson"))
				if err != nil {
					t.Fatal(err)
				}
				for _, args := range [][]string{{"tidy"}, {"tidy", "--write"}} {
					csvCommand(t, dir, 4, args...)
					got, err := os.ReadFile(filepath.Join(dir, "data/1.ndjson"))
					if err != nil || !bytes.Equal(got, original) {
						t.Fatalf("malformed file changed: %v", err)
					}
				}
			}
		})
	}
}

func TestJSONLTidyIdempotent(t *testing.T) {
	for _, format := range []string{"jsonl", "ndjson"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			copyDir(t, filepath.Join(testsDir(), "tidy_"+format), dir)
			csvCommand(t, dir, 5, "tidy")
			csvCommand(t, dir, 0, "tidy", "--write")
			csvCommand(t, dir, 0, "tidy")
			csvCommand(t, dir, 0, "tidy", "--write")
			csvCommand(t, dir, 0, "validate")
		})
	}
}

func TestJSONLLineReports(t *testing.T) {
	for _, format := range []string{"jsonl", "ndjson"} {
		for _, report := range []string{"text", "json", "yaml"} {
			for _, failure := range []string{"schema_line", "unique", "foreign_key_failure", "path", "malformed_later"} {
				t.Run(format+"/"+report+"/"+failure, func(t *testing.T) {
					cmd := exec.Command(binaryPath, "validate", "--format", report)
					cmd.Dir = filepath.Join(testsDir(), format+"_"+failure)
					var stdout, stderr bytes.Buffer
					cmd.Stdout, cmd.Stderr = &stdout, &stderr
					err := cmd.Run()
					out := stdout.Bytes()
					if err == nil {
						t.Fatal("expected failure")
					}
					if report == "text" {
						if !strings.Contains(stderr.String(), "data/1.ndjson (line 2)") {
							t.Fatal(stderr.String())
						}
						return
					}
					var entries []struct {
						File string `json:"file" yaml:"file"`
						Line int    `json:"line" yaml:"line"`
						Row  *int   `json:"row" yaml:"row"`
					}
					if report == "json" {
						err = json.Unmarshal(out, &entries)
					} else {
						err = yaml.Unmarshal(out, &entries)
					}
					if err != nil || len(entries) == 0 {
						t.Fatalf("report: %s / %v", out, err)
					}
					last := entries[len(entries)-1]
					if last.File != "data/1.ndjson" || last.Line != 2 || last.Row != nil {
						t.Fatalf("location: %+v", last)
					}
				})
			}
		}
	}
}

func TestJSONLExportReadBackAndOutputExclusion(t *testing.T) {
	for _, format := range []string{"jsonl", "ndjson"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			copyDir(t, filepath.Join(testsDir(), "example_examples_multi_format_export_jsonl"), dir)
			csvCommand(t, dir, 0, "export")
			cfg, err := os.ReadFile(filepath.Join(dir, ".datacur8"))
			if err != nil {
				t.Fatal(err)
			}
			// Use the existing JSONL export as a multi-record input, with no wrapper.
			var c struct {
				Types []struct {
					Output struct {
						Path string `yaml:"path"`
					} `yaml:"output"`
				} `yaml:"types"`
			}
			if err := yaml.Unmarshal(cfg, &c); err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(filepath.Join(dir, c.Types[0].Output.Path))
			if err != nil {
				t.Fatal(err)
			}
			tomlWrite(t, dir, "data/records.ndjson", string(original))
			tomlWrite(t, dir, ".datacur8", "version: \"0.0.0\"\ntypes:\n  - name: records\n    input: "+format+"\n    match: {include: ['.*\\.(jsonl|ndjson)$']}\n    schema: {type: object}\n    output: {format: "+format+", path: out/nested/roundtrip.ndjson}\n")
			// Remove the original export so it isn't counted as another input.
			if err := os.Remove(filepath.Join(dir, c.Types[0].Output.Path)); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				csvCommand(t, dir, 0, "export")
				got, err := os.ReadFile(filepath.Join(dir, "out/nested/roundtrip.ndjson"))
				if err != nil || !bytes.Equal(got, original) {
					t.Fatalf("roundtrip: %q / %v", got, err)
				}
			}
		})
	}
}

func TestJSONLTidyStructuredLineReports(t *testing.T) {
	for _, format := range []string{"jsonl", "ndjson"} {
		for _, report := range []string{"json", "yaml"} {
			t.Run(format+"/"+report, func(t *testing.T) {
				dir := t.TempDir()
				copyDir(t, filepath.Join(testsDir(), format+"_malformed_later"), dir)
				cmd := exec.Command(binaryPath, "tidy", "--write", "--format", report)
				cmd.Dir = dir
				out, err := cmd.Output()
				if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 4 {
					t.Fatalf("exit: %v", err)
				}
				var entries []map[string]any
				if report == "json" {
					err = json.Unmarshal(out, &entries)
				} else {
					err = yaml.Unmarshal(out, &entries)
				}
				if err != nil || len(entries) != 1 || entries[0]["file"] != "data/1.ndjson" {
					t.Fatalf("report: %s / %v", out, err)
				}
				line := entries[0]["line"]
				if line != 2 && line != float64(2) {
					t.Fatalf("line: %v", line)
				}
			})
		}
	}
}
