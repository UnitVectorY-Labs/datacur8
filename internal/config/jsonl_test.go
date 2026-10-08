package config

import "testing"

func TestJSONLAndNDJSONSemanticOptions(t *testing.T) {
	for _, format := range []string{"jsonl", "ndjson"} {
		cfg := &Config{Version: "0.0.0", Types: []TypeDef{{Name: "records", Input: format, Match: MatchDef{Include: []string{".*"}}, Schema: map[string]any{"type": "object"}, Output: &OutputDef{Format: format, Path: "out/data"}}}}
		if _, errs := Validate(cfg, "dev"); len(errs) != 0 {
			t.Fatal(errs)
		}
		cfg.Types[0].CSV = &CSVOptions{}
		if _, errs := Validate(cfg, "dev"); len(errs) == 0 {
			t.Fatal("CSV input options accepted")
		}
		cfg.Types[0].CSV = nil
		cfg.Types[0].Output.CSV = &CSVOptions{}
		if _, errs := Validate(cfg, "dev"); len(errs) == 0 {
			t.Fatal("CSV output options accepted")
		}
	}
}
