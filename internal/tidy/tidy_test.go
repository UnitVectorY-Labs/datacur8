package tidy

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return p
}

// --- JSON tests ---

func TestTidyJSON_SortsKeys(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "test.json", `{"z":1,"a":2,"m":3}`)

	res, err := TidyFile(p, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected file to be changed")
	}

	got, _ := os.ReadFile(p)
	expected := "{\n  \"a\": 2,\n  \"m\": 3,\n  \"z\": 1\n}\n"
	if string(got) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(got))
	}
}

func TestTidyJSON_NestedKeys(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "test.json", `{"b":{"z":1,"a":2},"a":3}`)

	res, err := TidyFile(p, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected file to be changed")
	}

	got, _ := os.ReadFile(p)
	expected := "{\n  \"a\": 3,\n  \"b\": {\n    \"a\": 2,\n    \"z\": 1\n  }\n}\n"
	if string(got) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(got))
	}
}

func TestTidyJSON_AlreadyTidy(t *testing.T) {
	dir := t.TempDir()
	content := "{\n  \"a\": 1,\n  \"b\": 2\n}\n"
	p := writeTempFile(t, dir, "test.json", content)

	res, err := TidyFile(p, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Changed {
		t.Error("expected file to not be changed")
	}
}

func TestTidyJSON_DryRun(t *testing.T) {
	dir := t.TempDir()
	original := `{"z":1,"a":2}`
	p := writeTempFile(t, dir, "test.json", original)

	res, err := TidyFile(p, "json", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected Changed to be true in dry run")
	}

	got, _ := os.ReadFile(p)
	if string(got) != original {
		t.Error("file should not be modified in dry run")
	}
}

func TestTidyJSON_ArrayOrderPreserved(t *testing.T) {
	dir := t.TempDir()
	input := "[\n  {\n    \"id\": 2,\n    \"name\": \"banana\"\n  },\n  {\n    \"id\": 1,\n    \"name\": \"apple\"\n  }\n]\n"
	p := writeTempFile(t, dir, "test.json", input)

	res, err := TidyFile(p, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Changed {
		t.Error("expected array order to be preserved")
	}
}

// --- YAML tests ---

func TestTidyYAML_SortsKeys(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "test.yaml", "z: 1\na: 2\nm: 3\n")

	res, err := TidyFile(p, "yaml", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected file to be changed")
	}

	got, _ := os.ReadFile(p)
	expected := "a: 2\nm: 3\nz: 1\n"
	if string(got) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(got))
	}
}

func TestTidyYAML_StripsComments(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "test.yaml", "# This is a comment\na: 1\nb: 2 # inline comment\n")

	res, err := TidyFile(p, "yaml", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected file to be changed")
	}

	got, _ := os.ReadFile(p)
	expected := "a: 1\nb: 2\n"
	if string(got) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(got))
	}
}

func TestTidyYAML_NestedKeys(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "test.yaml", "b:\n  z: 1\n  a: 2\na: 3\n")

	res, err := TidyFile(p, "yaml", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected file to be changed")
	}

	got, _ := os.ReadFile(p)
	expected := "a: 3\nb:\n  a: 2\n  z: 1\n"
	if string(got) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(got))
	}
}

func TestTidyYAML_DryRun(t *testing.T) {
	dir := t.TempDir()
	original := "z: 1\na: 2\n"
	p := writeTempFile(t, dir, "test.yaml", original)

	res, err := TidyFile(p, "yaml", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected Changed to be true in dry run")
	}

	got, _ := os.ReadFile(p)
	if string(got) != original {
		t.Error("file should not be modified in dry run")
	}
}

// --- CSV tests ---

func TestTidyCSV_SortsColumns(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "test.csv", "z,a,m\n1,2,3\n")

	res, err := TidyFile(p, "csv", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected file to be changed")
	}

	got, _ := os.ReadFile(p)
	expected := "a,m,z\n2,3,1\n"
	if string(got) != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, string(got))
	}
}

func TestTidyCSV_AlreadyTidy(t *testing.T) {
	dir := t.TempDir()
	content := "a,b\n1,2\n"
	p := writeTempFile(t, dir, "test.csv", content)

	res, err := TidyFile(p, "csv", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Changed {
		t.Error("expected file to not be changed")
	}
}

func TestTidyCSV_DryRun(t *testing.T) {
	dir := t.TempDir()
	original := "z,a\n1,2\n"
	p := writeTempFile(t, dir, "test.csv", original)

	res, err := TidyFile(p, "csv", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Changed {
		t.Error("expected Changed to be true in dry run")
	}

	got, _ := os.ReadFile(p)
	if string(got) != original {
		t.Error("file should not be modified in dry run")
	}
}

// --- sortKeys tests ---

func TestSortKeys_Map(t *testing.T) {
	input := map[string]any{"z": 1, "a": 2, "m": 3}
	result := sortKeys(input)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any")
	}
	if m["a"] != 2 || m["z"] != 1 || m["m"] != 3 {
		t.Error("values should be preserved")
	}
}

func TestSortKeys_NestedMap(t *testing.T) {
	input := map[string]any{
		"b": map[string]any{"z": 1, "a": 2},
		"a": 3,
	}
	result := sortKeys(input)
	m := result.(map[string]any)
	inner := m["b"].(map[string]any)
	if inner["a"] != 2 || inner["z"] != 1 {
		t.Error("nested values should be preserved")
	}
}

func TestSortKeys_Slice(t *testing.T) {
	input := []any{map[string]any{"z": 1, "a": 2}}
	result := sortKeys(input)
	arr := result.([]any)
	m := arr[0].(map[string]any)
	if m["a"] != 2 || m["z"] != 1 {
		t.Error("values in slice should be preserved")
	}
}

func TestSortKeys_Scalar(t *testing.T) {
	if sortKeys(42) != 42 {
		t.Error("scalar should pass through")
	}
	if sortKeys("hello") != "hello" {
		t.Error("string should pass through")
	}
}

// --- Unsupported format ---

func TestTidyFile_UnsupportedFormat(t *testing.T) {
	_, err := TidyFile("dummy.txt", "xml", false)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

// --- Empty CSV ---

func TestTidyCSV_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "test.csv", "")

	res, err := TidyFile(p, "csv", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Changed {
		t.Error("empty file should not be changed")
	}
}

// --- Result path ---

func TestTidyResult_Path(t *testing.T) {
	dir := t.TempDir()
	p := writeTempFile(t, dir, "test.json", `{"a":1}`)

	res, err := TidyFile(p, "json", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Path != p {
		t.Errorf("expected path %s, got %s", p, res.Path)
	}
}
