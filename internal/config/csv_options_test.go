package config

import "testing"

func TestCSVOptionsSemanticValidation(t *testing.T) {
	for _, delimiter := range []string{"", "ab", "\\t", "\"", "\r", "\n", "\x00", "\ufffd", string([]byte{0xff})} {
		t.Run(delimiter, func(t *testing.T) {
			options := &CSVOptions{Delimiter: &delimiter}
			cfg := &Config{Version: "0.0.0", Types: []TypeDef{{Name: "items", Input: "csv", CSV: options, Match: MatchDef{Include: []string{".*"}}, Schema: map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"type": "string"}}}}}}
			_, errs := Validate(cfg, "dev")
			if len(errs) != 1 {
				t.Fatalf("expected one input delimiter error, got %v", errs)
			}
			cfg.Types[0].CSV = nil
			cfg.Types[0].Output = &OutputDef{Format: "csv", Path: "out.csv", CSV: options}
			_, errs = Validate(cfg, "dev")
			if len(errs) != 1 {
				t.Fatalf("expected one output delimiter error, got %v", errs)
			}
		})
	}
	for _, delimiter := range []string{",", "\t", ";", "界"} {
		if errs := validateCSVOptions("items", "csv", &CSVOptions{Delimiter: &delimiter}); len(errs) > 0 {
			t.Fatal(errs)
		}
	}
	for _, format := range []string{"json", "yaml", "hcl", "jsonl"} {
		if errs := validateCSVOptions("items", format, &CSVOptions{}); len(errs) == 0 {
			t.Fatalf("accepted options on %s", format)
		}
	}
	var omitted *CSVOptions
	if omitted.Rune() != ',' || (&CSVOptions{}).Rune() != ',' {
		t.Fatal("default delimiter must be comma")
	}
}

func TestLoadCSVOptionsDefaults(t *testing.T) {
	for _, options := range []string{"", "    csv: {}\n", "    csv: {delimiter: \"界\"}\n"} {
		cfg, err := Load(writeTempConfig(t, "version: \"0.0.0\"\ntypes:\n  - name: items\n    input: csv\n"+options+"    match: {include: ['.*']}\n    schema: {type: object}\n"))
		if err != nil {
			t.Fatal(err)
		}
		want := ','
		if options == "    csv: {delimiter: \"界\"}\n" {
			want = '界'
		}
		if cfg.Types[0].CSV.Rune() != want {
			t.Fatalf("delimiter = %q", cfg.Types[0].CSV.Rune())
		}
	}
}
