package jsonldata

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseFramingAndValues(t *testing.T) {
	raw := []byte("{\"nested\":{\"array\":[null,true,{},[],\"雪\\nline\",1.5]},\"odd.key\":null}\r\n{}")
	records, err := Parse(raw)
	if err != nil || len(records) != 2 {
		t.Fatalf("records: %v, %v", records, err)
	}
	var want map[string]any
	if err := json.Unmarshal(bytes.Split(raw, []byte{'\n'})[0], &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(records[0], want) {
		t.Fatalf("values: %#v", records)
	}
	for _, bad := range []string{"\n", " \t\r\n", "\xef\xbb\xbf{}", "{\"x\":\"\xff\"}", "[]", "null", "true", "3", "\"s\"", "{} {}", "// comment", "{\n}", "{\"x\":1e400}"} {
		records, err := Parse([]byte("{}\n" + bad))
		var lineErr *Error
		if records != nil || !errors.As(err, &lineErr) || lineErr.Line != 2 {
			t.Errorf("%q: %v / %v", bad, records, err)
		}
	}
	empty, err := Parse(nil)
	if len(empty) != 0 || err != nil {
		t.Fatalf("empty: %v / %v", empty, err)
	}
}

func TestFormatLargeRecordIdempotentAndExact(t *testing.T) {
	raw := []byte(" {\"z\":9007199254740993, \"a\":{\"z\":null,\"a\":\"" + strings.Repeat("x", 70000) + "\\n雪\"}}\r\n{}")
	formatted, err := Format(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(formatted, []byte(`{"a":{"a":`)) || !bytes.Contains(formatted, []byte(`"z":9007199254740993`)) || !bytes.HasSuffix(formatted, []byte("\n{}\n")) {
		t.Fatal("format changed values/order")
	}
	again, err := Format(formatted)
	if err != nil || !bytes.Equal(formatted, again) {
		t.Fatalf("not idempotent: %v", err)
	}
	records, err := Parse(formatted)
	if err != nil || len(records) != 2 {
		t.Fatalf("large parse: %d / %v", len(records), err)
	}
	// Validation/export deliberately retain ordinary JSON input's float64 model.
	if records[0]["z"] != float64(9007199254740992) {
		t.Fatalf("number model: %v", records[0]["z"])
	}
	formatted, err = Format(nil)
	if err != nil || len(formatted) != 0 {
		t.Fatalf("empty format: %q / %v", formatted, err)
	}
}
