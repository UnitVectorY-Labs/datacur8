package selector

import (
	"reflect"
	"testing"
)

func TestParseValid(t *testing.T) {
	cases := []struct {
		input    string
		scalar   bool
		segments int
	}{
		{"$", true, 0},
		{"$.id", true, 1},
		{"$.a.b.c", true, 3},
		{"$.items[*].id", false, 3},
		{"$.a[*].b[*].c", false, 5},
		{"$[*]", false, 1},
	}
	for _, tc := range cases {
		s, err := Parse(tc.input)
		if err != nil {
			t.Fatalf("Parse(%q) unexpected error: %v", tc.input, err)
		}
		if s.String() != tc.input {
			t.Errorf("String() = %q, want %q", s.String(), tc.input)
		}
		if s.IsScalar() != tc.scalar {
			t.Errorf("IsScalar() = %v, want %v for %q", s.IsScalar(), tc.scalar, tc.input)
		}
		if len(s.segments) != tc.segments {
			t.Errorf("segments = %d, want %d for %q", len(s.segments), tc.segments, tc.input)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []string{
		"",
		"foo",
		"$.",
		"$.a.",
		"$..a",
		"$[0]",
		"$.a[0]",
	}
	for _, input := range cases {
		_, err := Parse(input)
		if err == nil {
			t.Errorf("Parse(%q) expected error, got nil", input)
		}
	}
}

func TestEvaluateRoot(t *testing.T) {
	s := mustParse(t, "$")
	data := map[string]any{"id": "abc"}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if !reflect.DeepEqual(got[0], data) {
		t.Errorf("expected root object, got %v", got[0])
	}
}

func TestEvaluateSimpleField(t *testing.T) {
	s := mustParse(t, "$.id")
	data := map[string]any{"id": "abc", "name": "test"}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	assertResults(t, got, []any{"abc"})
}

func TestEvaluateNestedField(t *testing.T) {
	s := mustParse(t, "$.a.b.c")
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": 42,
			},
		},
	}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	assertResults(t, got, []any{42})
}

func TestEvaluateArrayWildcard(t *testing.T) {
	s := mustParse(t, "$.items[*].id")
	data := map[string]any{
		"items": []any{
			map[string]any{"id": "a"},
			map[string]any{"id": "b"},
			map[string]any{"id": "c"},
		},
	}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	assertResults(t, got, []any{"a", "b", "c"})
}

func TestEvaluateNestedArrayWildcard(t *testing.T) {
	s := mustParse(t, "$.a[*].b[*].c")
	data := map[string]any{
		"a": []any{
			map[string]any{
				"b": []any{
					map[string]any{"c": 1},
					map[string]any{"c": 2},
				},
			},
			map[string]any{
				"b": []any{
					map[string]any{"c": 3},
				},
			},
		},
	}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	assertResults(t, got, []any{1, 2, 3})
}

func TestEvaluateMissingField(t *testing.T) {
	s := mustParse(t, "$.missing")
	data := map[string]any{"id": "abc"}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty results, got %v", got)
	}
}

func TestEvaluateMissingNestedField(t *testing.T) {
	s := mustParse(t, "$.a.b.missing")
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": 42,
			},
		},
	}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty results, got %v", got)
	}
}

func TestEvaluateFieldOnNonObject(t *testing.T) {
	s := mustParse(t, "$.a.b")
	data := map[string]any{"a": "string_value"}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty results, got %v", got)
	}
}

func TestEvaluateWildcardOnNonArray(t *testing.T) {
	s := mustParse(t, "$.items[*].id")
	data := map[string]any{"items": "not_an_array"}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty results, got %v", got)
	}
}

func TestEvaluateRootArray(t *testing.T) {
	s := mustParse(t, "$[*].id")
	data := []any{
		map[string]any{"id": "x"},
		map[string]any{"id": "y"},
	}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	assertResults(t, got, []any{"x", "y"})
}

func TestEvaluateArrayPartialMissing(t *testing.T) {
	s := mustParse(t, "$.items[*].id")
	data := map[string]any{
		"items": []any{
			map[string]any{"id": "a"},
			map[string]any{"name": "no_id"},
			map[string]any{"id": "c"},
		},
	}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	assertResults(t, got, []any{"a", "c"})
}

func TestEvaluateNilData(t *testing.T) {
	s := mustParse(t, "$.id")
	got, err := s.Evaluate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty results, got %v", got)
	}
}

func TestEvaluateEmptyArray(t *testing.T) {
	s := mustParse(t, "$.items[*].id")
	data := map[string]any{"items": []any{}}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty results, got %v", got)
	}
}

func TestEvaluateNestedObject(t *testing.T) {
	s := mustParse(t, "$.config")
	data := map[string]any{
		"config": map[string]any{"key": "val"},
	}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	m, ok := got[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got[0])
	}
	if m["key"] != "val" {
		t.Errorf("expected val, got %v", m["key"])
	}
}

func TestEvaluateNumericAndBoolValues(t *testing.T) {
	s := mustParse(t, "$.values[*].v")
	data := map[string]any{
		"values": []any{
			map[string]any{"v": 1.5},
			map[string]any{"v": true},
			map[string]any{"v": nil},
		},
	}
	got, err := s.Evaluate(data)
	if err != nil {
		t.Fatal(err)
	}
	assertResults(t, got, []any{1.5, true, nil})
}

func mustParse(t *testing.T, sel string) *Selector {
	t.Helper()
	s, err := Parse(sel)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", sel, err)
	}
	return s
}

func assertResults(t *testing.T, got, want []any) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("result length = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("result[%d] = %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
		}
	}
}
