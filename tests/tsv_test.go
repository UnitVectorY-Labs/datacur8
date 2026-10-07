package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestTSVRepeatExportAndTidy(t *testing.T) {
	for _, fixture := range []string{"example_examples_tsv_catalog_success", "tsv_export_from_formats", "csv_unicode_delimiter", "tsv_export_empty", "tsv_single_empty_cell"} {
		t.Run(fixture, func(t *testing.T) {
			dir := t.TempDir()
			copyDir(t, filepath.Join(testsDir(), fixture), dir)
			csvCommand(t, dir, 0, "export")
			snapshots := map[string][]byte{}
			err := filepath.WalkDir(filepath.Join(testsDir(), fixture, "expected/export"), func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(filepath.Join(testsDir(), fixture, "expected/export"), path)
				if err != nil {
					return err
				}
				content, err := os.ReadFile(filepath.Join(dir, rel))
				snapshots[rel] = content
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			csvCommand(t, dir, 0, "export")
			for path, want := range snapshots {
				got, err := os.ReadFile(filepath.Join(dir, path))
				if err != nil || !bytes.Equal(want, got) {
					t.Fatalf("repeat export %s: %q, %v", path, got, err)
				}
			}
			if dirExists(filepath.Join(testsDir(), fixture, "expected/tidy")) {
				csvCommand(t, dir, 0, "tidy", "--write")
				csvCommand(t, dir, 0, "tidy")
				csvCommand(t, dir, 0, "tidy", "--write")
				// Sorting columns must preserve values and row order on export.
				csvCommand(t, dir, 0, "export")
				for path, want := range snapshots {
					got, err := os.ReadFile(filepath.Join(dir, path))
					if err != nil || !bytes.Equal(want, got) {
						t.Fatalf("export after tidy %s: %q, %v", path, got, err)
					}
				}
			}
		})
	}
}

func TestTSVInvalidDataPreservesFiles(t *testing.T) {
	for _, fixture := range []string{"tsv_malformed", "tsv_record_width", "tsv_duplicate_header", "tsv_empty_header", "tsv_conversion_row", "tsv_schema_row"} {
		t.Run(fixture, func(t *testing.T) {
			dir := t.TempDir()
			copyDir(t, filepath.Join(testsDir(), fixture), dir)
			input := filepath.Join(dir, "data/items.tsv")
			original, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(dir, "out/items.csv")
			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
				t.Fatal(err)
			}
			sentinel := []byte("existing destination\n")
			if err := os.WriteFile(output, sentinel, 0600); err != nil {
				t.Fatal(err)
			}
			csvCommand(t, dir, 2, "export")
			got, err := os.ReadFile(output)
			if err != nil || !bytes.Equal(got, sentinel) {
				t.Fatalf("destination changed: %q, %v", got, err)
			}
			// Tidy only checks syntax/header ambiguity, not schema conversion.
			if fixture == "tsv_conversion_row" || fixture == "tsv_schema_row" {
				return
			}
			csvCommand(t, dir, 4, "tidy")
			csvCommand(t, dir, 4, "tidy", "--write")
			got, err = os.ReadFile(input)
			if err != nil || !bytes.Equal(got, original) {
				t.Fatalf("malformed input changed: %q, %v", got, err)
			}
		})
	}
}
