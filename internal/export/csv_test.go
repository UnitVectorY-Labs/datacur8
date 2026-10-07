package export

import (
	"github.com/UnitVectorY-Labs/datacur8/internal/config"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCSVNumericCells(t *testing.T) {
	for _, tc := range []struct {
		name, typ string
		value     any
		want      string
		bad       bool
	}{
		{"min", "integer", int64(math.MinInt64), "-9223372036854775808", false},
		{"max", "integer", int64(math.MaxInt64), "9223372036854775807", false},
		{"float min", "integer", float64(-0x1p63), "-9223372036854775808", false},
		{"float max", "integer", math.Nextafter(0x1p63, 0), "9223372036854774784", false},
		{"overflow", "integer", float64(0x1p63), "", true},
		{"underflow", "integer", math.Nextafter(-0x1p63, math.Inf(-1)), "", true},
		{"unsigned", "integer", uint64(math.MaxUint64), "", true},
		{"fraction", "integer", 1.5, "", true},
		{"infinity", "number", math.Inf(1), "", true},
		{"nan", "number", math.NaN(), "", true},
		{"tiny", "number", math.SmallestNonzeroFloat64, "5e-324", false},
		{"large", "number", math.MaxFloat64, "1.7976931348623157e+308", false},
		{"precision", "number", 1.2345678901234567, "1.2345678901234567", false},
		{"structured", "string", []any{"x"}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := csvCell(tc.value, tc.typ)
			if (err != nil) != tc.bad || got != tc.want {
				t.Fatalf("got %q, %v; want %q, error=%v", got, err, tc.want, tc.bad)
			}
		})
	}
}

func TestCSVRejectsUnrepresentableItems(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{"cell": map[string]any{"type": "string"}}}
	for _, value := range []any{map[string]any{"cell": nil}, map[string]any{}, map[string]any{"cell": map[string]any{}}, map[string]any{"cell": []any{}}, 42} {
		if _, err := marshalCSV(schema, []any{value}); err == nil {
			t.Fatalf("accepted %v", value)
		}
	}
}

func TestCSVSingleEmptyString(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"type": "string"}}}
	got, err := marshalCSV(schema, []any{map[string]any{"value": ""}})
	if err != nil || string(got) != "value\n\"\"\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestCSVConversionFailurePreservesFile(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{"cell": map[string]any{"type": "string"}}}
	for _, tc := range []struct {
		name string
		item any
		want string
	}{
		{"null", map[string]any{"cell": nil}, "null CSV cell"},
		{"object", map[string]any{"cell": map[string]any{}}, "expected string CSV scalar"},
		{"array", map[string]any{"cell": []any{}}, "expected string CSV scalar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "out.csv")
			if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			results, errs := Export(map[string][]any{"items": {tc.item}}, []config.TypeDef{{Name: "items", Schema: schema, Output: &config.OutputDef{Format: "csv", Path: path}}}, dir)
			if len(results) != 0 || len(errs) != 1 || !strings.Contains(errs[0].Error(), `item 0, property "cell": `+tc.want) {
				t.Fatalf("results=%v errors=%v", results, errs)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != "original" {
				t.Fatalf("destination changed: %q, %v", got, err)
			}
		})
	}
}
