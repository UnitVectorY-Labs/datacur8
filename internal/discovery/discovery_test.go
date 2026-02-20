package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
)

// helper to create a file inside a temp directory tree.
func createFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverBasicMatch(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "teams/alpha.yaml", "name: alpha")
	createFile(t, root, "teams/beta.yaml", "name: beta")
	createFile(t, root, "README.md", "# readme")

	types := []config.TypeDef{
		{
			Name:  "team",
			Input: "yaml",
			Match: config.MatchDef{
				Include: []string{`^teams/[^/]+\.ya?ml$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	// Should be sorted by path.
	if files[0].Path != "teams/alpha.yaml" {
		t.Errorf("expected first file teams/alpha.yaml, got %s", files[0].Path)
	}
	if files[1].Path != "teams/beta.yaml" {
		t.Errorf("expected second file teams/beta.yaml, got %s", files[1].Path)
	}
	if files[0].TypeName != "team" {
		t.Errorf("expected type name 'team', got %q", files[0].TypeName)
	}
}

func TestDiscoverExclude(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "data/keep.json", "{}")
	createFile(t, root, "data/skip.json", "{}")

	types := []config.TypeDef{
		{
			Name:  "data",
			Input: "json",
			Match: config.MatchDef{
				Include: []string{`^data/.*\.json$`},
				Exclude: []string{`skip\.json$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "data/keep.json" {
		t.Errorf("expected data/keep.json, got %s", files[0].Path)
	}
}

func TestDiscoverMultiTypeMatch(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "overlap.yaml", "data: true")

	types := []config.TypeDef{
		{
			Name:  "typeA",
			Input: "yaml",
			Match: config.MatchDef{Include: []string{`\.yaml$`}},
		},
		{
			Name:  "typeB",
			Input: "yaml",
			Match: config.MatchDef{Include: []string{`\.yaml$`}},
		},
	}

	_, errs := Discover(root, types)
	if len(errs) == 0 {
		t.Fatal("expected error for multi-type match")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "matches multiple types") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'matches multiple types' error, got: %v", errs)
	}
}

func TestDiscoverNamedCaptures(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "configs/alpha/services/web.yaml", "id: web")

	types := []config.TypeDef{
		{
			Name:  "service",
			Input: "yaml",
			Match: config.MatchDef{
				Include: []string{`^configs/(?P<team>[^/]+)/services/(?P<service>[^/]+)\.ya?ml$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	f := files[0]
	if f.PathCaptures["team"] != "alpha" {
		t.Errorf("expected capture team=alpha, got %q", f.PathCaptures["team"])
	}
	if f.PathCaptures["service"] != "web" {
		t.Errorf("expected capture service=web, got %q", f.PathCaptures["service"])
	}
}

func TestDiscoverBuiltInCaptures(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "data/items/report.json", "{}")

	types := []config.TypeDef{
		{
			Name:  "report",
			Input: "json",
			Match: config.MatchDef{
				Include: []string{`^data/items/.*\.json$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	f := files[0]
	if f.PathCaptures["path.file"] != "report" {
		t.Errorf("expected path.file=report, got %q", f.PathCaptures["path.file"])
	}
	if f.PathCaptures["path.ext"] != "json" {
		t.Errorf("expected path.ext=json, got %q", f.PathCaptures["path.ext"])
	}
	if f.PathCaptures["path.parent"] != "items" {
		t.Errorf("expected path.parent=items, got %q", f.PathCaptures["path.parent"])
	}
}

func TestDiscoverYmlNormalizesToYaml(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "data.yml", "a: 1")

	types := []config.TypeDef{
		{
			Name:  "data",
			Input: "yaml",
			Match: config.MatchDef{
				Include: []string{`\.yml$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].PathCaptures["path.ext"] != "yaml" {
		t.Errorf("expected path.ext=yaml, got %q", files[0].PathCaptures["path.ext"])
	}
}

func TestDiscoverSkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, ".hidden/secret.yaml", "a: 1")
	createFile(t, root, ".git/config.yaml", "a: 1")
	createFile(t, root, "visible/data.yaml", "a: 1")

	types := []config.TypeDef{
		{
			Name:  "data",
			Input: "yaml",
			Match: config.MatchDef{
				Include: []string{`\.yaml$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "visible/data.yaml" {
		t.Errorf("expected visible/data.yaml, got %s", files[0].Path)
	}
}

func TestDiscoverSubdirectoryDatacur8Error(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, ".datacur8", "version: '1'")
	createFile(t, root, "sub/.datacur8", "version: '1'")

	types := []config.TypeDef{}

	_, errs := Discover(root, types)
	if len(errs) == 0 {
		t.Fatal("expected error for subdirectory .datacur8")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "subdirectory") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected subdirectory .datacur8 error, got: %v", errs)
	}
}

func TestDiscoverSkipsOutputPaths(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "data/item.json", "{}")
	createFile(t, root, "out/items.json", "[]")

	types := []config.TypeDef{
		{
			Name:  "item",
			Input: "json",
			Match: config.MatchDef{
				Include: []string{`\.json$`},
			},
			Output: &config.OutputDef{
				Path:   "out/items.json",
				Format: "json",
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file (output should be skipped), got %d", len(files))
	}
	if files[0].Path != "data/item.json" {
		t.Errorf("expected data/item.json, got %s", files[0].Path)
	}
}

func TestDiscoverNoMatchIgnored(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "readme.txt", "hello")
	createFile(t, root, "data.yaml", "a: 1")

	types := []config.TypeDef{
		{
			Name:  "data",
			Input: "yaml",
			Match: config.MatchDef{
				Include: []string{`\.yaml$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestDiscoverSortedOutput(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "c.yaml", "c: 1")
	createFile(t, root, "a.yaml", "a: 1")
	createFile(t, root, "b.yaml", "b: 1")

	types := []config.TypeDef{
		{
			Name:  "data",
			Input: "yaml",
			Match: config.MatchDef{
				Include: []string{`\.yaml$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if files[0].Path != "a.yaml" || files[1].Path != "b.yaml" || files[2].Path != "c.yaml" {
		t.Errorf("files not sorted: %v, %v, %v", files[0].Path, files[1].Path, files[2].Path)
	}
}

func TestDiscoverParentFolderRootFile(t *testing.T) {
	root := t.TempDir()
	createFile(t, root, "top.json", "{}")

	types := []config.TypeDef{
		{
			Name:  "top",
			Input: "json",
			Match: config.MatchDef{
				Include: []string{`^top\.json$`},
			},
		},
	}

	files, errs := Discover(root, types)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].PathCaptures["path.parent"] != "" {
		t.Errorf("expected empty path.parent for root file, got %q", files[0].PathCaptures["path.parent"])
	}
}


