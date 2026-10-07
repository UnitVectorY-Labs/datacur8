package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
	"github.com/UnitVectorY-Labs/datacur8/internal/discovery"
)

func TestJSONLFilesAreAtomicAndReadErrorsSurface(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "bad.ndjson")
	if err := os.WriteFile(path, []byte("{\"id\":\"valid-prefix\"}\n{"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"jsonl", "ndjson"} {
		td := &config.TypeDef{Name: "records", Input: format, Schema: map[string]any{"type": "object"}}
		files := []discovery.DiscoveredFile{{Path: "bad.ndjson", TypeName: td.Name, TypeDef: td}, {Path: "missing.ndjson", TypeName: td.Name, TypeDef: td}}
		items, parsing, validation := parseAndValidateFiles(files, &config.Config{})
		if len(items[td.Name]) != 0 || len(validation) != 0 || len(parsing) != 2 {
			t.Fatalf("partial dataset or missing errors: %v / %v / %v", items, parsing, validation)
		}
		if parsing[0].Line == nil || *parsing[0].Line != 2 || parsing[1].Line != nil {
			t.Fatalf("locations: %+v", parsing)
		}
	}
}
