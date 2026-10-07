package tomldata

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestParseTOML11AndTemporalNormalization(t *testing.T) {
	src := []byte(`offset = 1979-05-27 07:32:00.120000-07:00
utc = 1979-05-27t07:32:00z
local = 1979-05-27t07:32:00.123456789123
date = 1979-05-27
time = 07:32:00.1200
short = 07:32
# TOML 1.1: omitted seconds, extra escapes, multiline inline tables.
escapes = "\e\x41"
inline = {
 a = 1,
}
"quoted.key" = 'literal'
dotted.key = "basic"
multi = """
hello
world"""
literal = '''
hello\nworld'''
mixed = [1, true, "x", {}, []]
[[children]]
id = "a"
[[children]]
id = "b"
`)
	got, err := Parse(src, "record.toml")
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{
		"offset": "1979-05-27T07:32:00.12-07:00", "utc": "1979-05-27T07:32:00Z",
		"local": "1979-05-27T07:32:00.123456789", "date": "1979-05-27", "time": "07:32:00.12", "short": "07:32:00",
		"escapes": "\x1bA", "quoted.key": "literal", "multi": "hello\nworld", "literal": `hello\nworld`,
	} {
		if got[k] != want {
			t.Errorf("%s = %#v, want %#v", k, got[k], want)
		}
	}
	if len(got["children"].([]any)) != 2 {
		t.Fatal(got)
	}
	formatted, err := Format(src, "record.toml")
	if err != nil {
		t.Fatal(err)
	}
	again, err := Format(formatted, "record.toml")
	if err != nil || !bytes.Equal(formatted, again) {
		t.Fatalf("not idempotent: %s / %v", again, err)
	}
	roundtrip, err := Parse(formatted, "record.toml")
	if err != nil || !reflect.DeepEqual(got, roundtrip) {
		t.Fatalf("tidy changed values: %#v / %v", roundtrip, err)
	}
	if bytes.Contains(formatted, []byte("# TOML")) {
		t.Fatal("comments retained")
	}
}

func TestParseFailuresAndIntegerBoundaries(t *testing.T) {
	for _, tc := range []struct{ src, message string }{
		{"a=1\na=2", "record.toml:2:"}, {"[a]\n[a]", "record.toml:2:"}, {"a=[", "record.toml:1:"},
		{"a=nan", `$["a"]: non-finite`}, {"a=+inf", "non-finite"}, {"a=-inf", "non-finite"},
		{"a={b=[9007199254740993]}", `$["a"]["b"][0]: integer cannot`},
		{"a=9223372036854775807", "integer cannot"}, {"a=-9223372036854775807", "integer cannot"},
		{"a=9223372036854775808", "record.toml:1:"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			_, err := Parse([]byte(tc.src), "record.toml")
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("got %v, want %s", err, tc.message)
			}
		})
	}
	for _, src := range []string{"a=9007199254740991", "a=9007199254740992", "a=9007199254740994", "a=-9223372036854775808", "a=0x10"} {
		if _, err := Parse([]byte(src), "record.toml"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAggregateRoundTrip(t *testing.T) {
	data := []any{map[string]any{
		"": "empty key", "a.b": "1979-05-27", "quote\"\n": "literal ${x}",
		"é": map[string]any{}, "é": []any{}, "日本語": true,
		"mixed":         []any{float64(2), "x", false, map[string]any{}, []any{}, map[string]any{"nested": []any{true}}},
		"negative zero": math.Copysign(0, -1), "large": math.MaxFloat64, "small": math.SmallestNonzeroFloat64,
	}, map[string]any{}}
	first, err := Marshal("unusual.名前", data)
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		next, err := Marshal("unusual.名前", data)
		if err != nil || !bytes.Equal(first, next) {
			t.Fatal("nondeterministic output", err)
		}
	}
	got, err := Parse(first, "aggregate.toml")
	if err != nil || !reflect.DeepEqual(got["unusual.名前"], data) {
		t.Fatalf("roundtrip: %#v / %v\n%s", got, err, first)
	}
	if !math.Signbit(got["unusual.名前"].([]any)[0].(map[string]any)["negative zero"].(float64)) {
		t.Fatal("lost negative zero")
	}
	empty, err := Marshal("empty", nil)
	if err != nil || string(empty) != "empty = []\n" {
		t.Fatalf("empty: %q / %v", empty, err)
	}
}

func TestMarshalContextualFailures(t *testing.T) {
	for _, value := range []any{nil, math.NaN(), math.Inf(1), int64(9007199254740993), uint64(math.MaxUint64), make(chan int), "\xff"} {
		_, err := Marshal("records", []any{map[string]any{"nested": []any{value}}})
		if err == nil || !strings.Contains(err.Error(), `$["records"][0]["nested"][0]`) {
			t.Fatalf("%T: %v", value, err)
		}
	}
}
