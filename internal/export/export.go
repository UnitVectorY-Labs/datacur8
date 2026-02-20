package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
	"gopkg.in/yaml.v3"
)

// ExportResult tracks what was exported
type ExportResult struct {
	TypeName string
	Path     string
	Format   string
	Count    int // number of items exported
}

// Export writes validated items to their configured output files.
// items is a map from type name to ordered slice of parsed data items ([]any where each is map[string]any)
// typeDefs contains the type definitions with output config
// rootDir is the base directory for resolving output paths
// Returns results and any errors
func Export(items map[string][]any, typeDefs []config.TypeDef, rootDir string) ([]ExportResult, []error) {
	var results []ExportResult
	var errs []error

	for _, td := range typeDefs {
		if td.Output == nil {
			continue
		}

		data := items[td.Name]

		outPath := td.Output.Path
		if !filepath.IsAbs(outPath) {
			outPath = filepath.Join(rootDir, outPath)
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			errs = append(errs, fmt.Errorf("creating output directory for %s: %w", td.Name, err))
			continue
		}

		format := strings.ToLower(td.Output.Format)

		var content []byte
		var err error

		switch format {
		case "json":
			content, err = marshalJSON(td.Name, data)
		case "yaml":
			content, err = marshalYAML(td.Name, data)
		case "jsonl":
			content, err = marshalJSONL(data)
		default:
			errs = append(errs, fmt.Errorf("unsupported output format %q for type %s", td.Output.Format, td.Name))
			continue
		}

		if err != nil {
			errs = append(errs, fmt.Errorf("marshaling %s output for type %s: %w", format, td.Name, err))
			continue
		}

		if err := os.WriteFile(outPath, content, 0o644); err != nil {
			errs = append(errs, fmt.Errorf("writing output file for %s: %w", td.Name, err))
			continue
		}

		results = append(results, ExportResult{
			TypeName: td.Name,
			Path:     outPath,
			Format:   format,
			Count:    len(data),
		})
	}

	return results, errs
}

func marshalJSON(typeName string, data []any) ([]byte, error) {
	if data == nil {
		data = []any{}
	}
	wrapper := map[string]any{typeName: data}
	out, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return nil, err
	}
	out = append(out, '\n')
	return out, nil
}

func marshalYAML(typeName string, data []any) ([]byte, error) {
	if data == nil {
		data = []any{}
	}
	wrapper := map[string]any{typeName: data}
	out, err := yaml.Marshal(wrapper)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func marshalJSONL(data []any) ([]byte, error) {
	var buf []byte
	for _, item := range data {
		line, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}
	return buf, nil
}
