package constraints

import (
	"testing"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
)

func boolPtr(b bool) *bool { return &b }

// --- unique constraint tests ---

func TestUnique_ScalarTypeScope_NoDuplicates(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{TypeName: "user", FilePath: "a.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
			{TypeName: "user", FilePath: "b.json", Data: map[string]any{"id": "2"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "unique-id", Type: "unique", Key: "$.id", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestUnique_ScalarTypeScope_WithDuplicates(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{TypeName: "user", FilePath: "a.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
			{TypeName: "user", FilePath: "b.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
			{TypeName: "user", FilePath: "c.json", Data: map[string]any{"id": "2"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "unique-id", Type: "unique", Key: "$.id", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors (both duplicates reported), got %d: %v", len(errs), errs)
	}
	for _, e := range errs {
		if e.ConstraintType != "unique" {
			t.Errorf("expected constraint type 'unique', got %q", e.ConstraintType)
		}
	}
}

func TestUnique_ScalarTypeScope_CaseInsensitive(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{TypeName: "user", FilePath: "a.json", Data: map[string]any{"name": "Alice"}, RowIndex: -1},
			{TypeName: "user", FilePath: "b.json", Data: map[string]any{"name": "alice"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "unique-name", Type: "unique", Key: "$.name", Scope: "type",
			CaseSensitive: boolPtr(false),
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestUnique_ScalarTypeScope_CaseSensitive(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{TypeName: "user", FilePath: "a.json", Data: map[string]any{"name": "Alice"}, RowIndex: -1},
			{TypeName: "user", FilePath: "b.json", Data: map[string]any{"name": "alice"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "unique-name", Type: "unique", Key: "$.name", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors (case-sensitive), got %d: %v", len(errs), errs)
	}
}

func TestUnique_MultiValue_ItemScope(t *testing.T) {
	items := map[string][]Item{
		"config": {
			{TypeName: "config", FilePath: "a.json", Data: map[string]any{
				"tags": []any{"a", "b", "a"},
			}, RowIndex: -1},
			{TypeName: "config", FilePath: "b.json", Data: map[string]any{
				"tags": []any{"x", "y"},
			}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "config",
		Constraints: []config.ConstraintDef{{
			ID: "unique-tags", Type: "unique", Key: "$.tags[*]", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	// Multi-value key defaults to item scope: only a.json has dup "a"
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].FilePath != "a.json" {
		t.Errorf("expected error for a.json, got %s", errs[0].FilePath)
	}
}

func TestUnique_ItemScope_Explicit(t *testing.T) {
	items := map[string][]Item{
		"config": {
			{TypeName: "config", FilePath: "a.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
			{TypeName: "config", FilePath: "b.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "config",
		Constraints: []config.ConstraintDef{{
			ID: "unique-id", Type: "unique", Key: "$.id", Scope: "item",
		}},
	}}
	errs := Evaluate(items, defs)
	// scope=item with scalar key: each item checked individually, no duplicates within item
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors (item scope, no dup within single item), got %d: %v", len(errs), errs)
	}
}

func TestUnique_InvalidSelector(t *testing.T) {
	items := map[string][]Item{
		"user": {{TypeName: "user", FilePath: "a.json", Data: map[string]any{"id": "1"}, RowIndex: -1}},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "bad", Type: "unique", Key: "bad selector", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestUnique_CSV_RowIndex(t *testing.T) {
	items := map[string][]Item{
		"data": {
			{TypeName: "data", FilePath: "data.csv", Data: map[string]any{"id": "1"}, RowIndex: 0},
			{TypeName: "data", FilePath: "data.csv", Data: map[string]any{"id": "1"}, RowIndex: 1},
			{TypeName: "data", FilePath: "data.csv", Data: map[string]any{"id": "2"}, RowIndex: 2},
		},
	}
	defs := []config.TypeDef{{
		Name: "data",
		Constraints: []config.ConstraintDef{{
			ID: "unique-id", Type: "unique", Key: "$.id", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}
	// Check row indices are preserved
	rows := map[int]bool{}
	for _, e := range errs {
		rows[e.RowIndex] = true
	}
	if !rows[0] || !rows[1] {
		t.Errorf("expected row indices 0 and 1, got %v", rows)
	}
}

// --- foreign_key constraint tests ---

func TestForeignKey_Valid(t *testing.T) {
	items := map[string][]Item{
		"order": {
			{TypeName: "order", FilePath: "o1.json", Data: map[string]any{"user_id": "u1"}, RowIndex: -1},
			{TypeName: "order", FilePath: "o2.json", Data: map[string]any{"user_id": "u2"}, RowIndex: -1},
		},
		"user": {
			{TypeName: "user", FilePath: "u1.json", Data: map[string]any{"id": "u1"}, RowIndex: -1},
			{TypeName: "user", FilePath: "u2.json", Data: map[string]any{"id": "u2"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "order",
		Constraints: []config.ConstraintDef{{
			ID: "fk-user", Type: "foreign_key", Key: "$.user_id",
			References: &config.ReferenceDef{Type: "user", Key: "$.id"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestForeignKey_Missing(t *testing.T) {
	items := map[string][]Item{
		"order": {
			{TypeName: "order", FilePath: "o1.json", Data: map[string]any{"user_id": "u1"}, RowIndex: -1},
			{TypeName: "order", FilePath: "o2.json", Data: map[string]any{"user_id": "u99"}, RowIndex: -1},
		},
		"user": {
			{TypeName: "user", FilePath: "u1.json", Data: map[string]any{"id": "u1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "order",
		Constraints: []config.ConstraintDef{{
			ID: "fk-user", Type: "foreign_key", Key: "$.user_id",
			References: &config.ReferenceDef{Type: "user", Key: "$.id"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].FilePath != "o2.json" {
		t.Errorf("expected error for o2.json, got %s", errs[0].FilePath)
	}
}

func TestForeignKey_MultipleValuesError(t *testing.T) {
	items := map[string][]Item{
		"order": {
			{TypeName: "order", FilePath: "o1.json", Data: map[string]any{
				"user_ids": []any{"u1", "u2"},
			}, RowIndex: -1},
		},
		"user": {
			{TypeName: "user", FilePath: "u1.json", Data: map[string]any{"id": "u1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "order",
		Constraints: []config.ConstraintDef{{
			ID: "fk-user", Type: "foreign_key", Key: "$.user_ids[*]",
			References: &config.ReferenceDef{Type: "user", Key: "$.id"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error (multi-value), got %d: %v", len(errs), errs)
	}
	if errs[0].Message == "" {
		t.Error("expected non-empty error message")
	}
}

func TestForeignKey_MissingReferences(t *testing.T) {
	items := map[string][]Item{
		"order": {
			{TypeName: "order", FilePath: "o1.json", Data: map[string]any{"user_id": "u1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "order",
		Constraints: []config.ConstraintDef{{
			ID: "fk-user", Type: "foreign_key", Key: "$.user_id",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestForeignKey_EmptyRefType(t *testing.T) {
	items := map[string][]Item{
		"order": {
			{TypeName: "order", FilePath: "o1.json", Data: map[string]any{"user_id": "u1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "order",
		Constraints: []config.ConstraintDef{{
			ID: "fk-user", Type: "foreign_key", Key: "$.user_id",
			References: &config.ReferenceDef{Type: "nonexistent", Key: "$.id"},
		}},
	}}
	errs := Evaluate(items, defs)
	// Referenced type has no items, so all foreign keys fail
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

// --- path_equals_attr constraint tests ---

func TestPathEqualsAttr_Match(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "users/alice.json",
				Data:         map[string]any{"name": "alice"},
				PathCaptures: map[string]string{"path.file": "alice", "path.parent": "users", "path.ext": "json"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.file",
			References: &config.ReferenceDef{Key: "$.name"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestPathEqualsAttr_Mismatch(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "users/alice.json",
				Data:         map[string]any{"name": "bob"},
				PathCaptures: map[string]string{"path.file": "alice", "path.parent": "users", "path.ext": "json"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.file",
			References: &config.ReferenceDef{Key: "$.name"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestPathEqualsAttr_CaseInsensitive(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "users/Alice.json",
				Data:         map[string]any{"name": "alice"},
				PathCaptures: map[string]string{"path.file": "Alice", "path.parent": "users", "path.ext": "json"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.file",
			References:    &config.ReferenceDef{Key: "$.name"},
			CaseSensitive: boolPtr(false),
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors (case insensitive), got %d: %v", len(errs), errs)
	}
}

func TestPathEqualsAttr_CaseSensitiveMismatch(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "users/Alice.json",
				Data:         map[string]any{"name": "alice"},
				PathCaptures: map[string]string{"path.file": "Alice", "path.parent": "users", "path.ext": "json"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.file",
			References: &config.ReferenceDef{Key: "$.name"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error (case sensitive), got %d: %v", len(errs), errs)
	}
}

func TestPathEqualsAttr_MultipleValuesError(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "users/alice.json",
				Data:         map[string]any{"names": []any{"alice", "bob"}},
				PathCaptures: map[string]string{"path.file": "alice", "path.parent": "users", "path.ext": "json"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.file",
			References: &config.ReferenceDef{Key: "$.names[*]"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error (multi-value), got %d: %v", len(errs), errs)
	}
}

func TestPathEqualsAttr_MissingPathCapture(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "users/alice.json",
				Data:         map[string]any{"name": "alice"},
				PathCaptures: map[string]string{"path.file": "alice"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.custom_field",
			References: &config.ReferenceDef{Key: "$.name"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestPathEqualsAttr_MissingReferences(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{TypeName: "user", FilePath: "a.json", Data: map[string]any{"name": "a"},
				PathCaptures: map[string]string{"path.file": "a"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.file",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestPathEqualsAttr_CustomCapture(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "regions/us/alice.json",
				Data: map[string]any{"region": "us"},
				PathCaptures: map[string]string{
					"path.file": "alice", "path.parent": "us", "path.ext": "json",
					"path.region": "us",
				},
				RowIndex: -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-region", Type: "path_equals_attr", PathSelector: "path.region",
			References: &config.ReferenceDef{Key: "$.region"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

// --- Sorting tests ---

func TestEvaluate_DeterministicSorting(t *testing.T) {
	items := map[string][]Item{
		"beta": {
			{TypeName: "beta", FilePath: "b2.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
			{TypeName: "beta", FilePath: "b1.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
		},
		"alpha": {
			{TypeName: "alpha", FilePath: "a2.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
			{TypeName: "alpha", FilePath: "a1.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{
		{
			Name: "beta",
			Constraints: []config.ConstraintDef{{
				ID: "unique-id", Type: "unique", Key: "$.id", Scope: "type",
			}},
		},
		{
			Name: "alpha",
			Constraints: []config.ConstraintDef{{
				ID: "unique-id", Type: "unique", Key: "$.id", Scope: "type",
			}},
		},
	}
	errs := Evaluate(items, defs)
	if len(errs) != 4 {
		t.Fatalf("expected 4 errors, got %d", len(errs))
	}
	// Should be sorted: alpha before beta, then by file path
	if errs[0].TypeName != "alpha" || errs[1].TypeName != "alpha" {
		t.Error("expected alpha errors first")
	}
	if errs[0].FilePath != "a1.json" {
		t.Errorf("expected a1.json first, got %s", errs[0].FilePath)
	}
	if errs[2].TypeName != "beta" || errs[3].TypeName != "beta" {
		t.Error("expected beta errors last")
	}
}

func TestEvaluate_NoItems(t *testing.T) {
	items := map[string][]Item{}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "unique-id", Type: "unique", Key: "$.id", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errs))
	}
}

func TestEvaluate_AutoGeneratedConstraintID(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{TypeName: "user", FilePath: "a.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
			{TypeName: "user", FilePath: "b.json", Data: map[string]any{"id": "1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			Type: "unique", Key: "$.id", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
	if errs[0].ConstraintID != "#0" {
		t.Errorf("expected auto-generated constraint ID '#0', got %q", errs[0].ConstraintID)
	}
}

// --- Error.Error() tests ---

func TestError_ErrorString(t *testing.T) {
	e := &Error{
		ConstraintID:   "unique-id",
		ConstraintType: "unique",
		TypeName:       "user",
		FilePath:       "a.json",
		Message:        "duplicate value",
		RowIndex:       -1,
	}
	s := e.Error()
	if s != "[user] unique a.json: duplicate value" {
		t.Errorf("unexpected error string: %s", s)
	}
}

func TestError_ErrorString_WithRow(t *testing.T) {
	e := &Error{
		ConstraintID:   "unique-id",
		ConstraintType: "unique",
		TypeName:       "data",
		FilePath:       "data.csv",
		Message:        "duplicate value",
		RowIndex:       5,
	}
	s := e.Error()
	if s != "[data] unique data.csv (row 5): duplicate value" {
		t.Errorf("unexpected error string: %s", s)
	}
}

func TestPathEqualsAttr_NoAttrValue(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "users/alice.json",
				Data:         map[string]any{},
				PathCaptures: map[string]string{"path.file": "alice", "path.parent": "users", "path.ext": "json"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.file",
			References: &config.ReferenceDef{Key: "$.name"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error (no attr value), got %d: %v", len(errs), errs)
	}
}

func TestUnique_NestedScalar(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{TypeName: "user", FilePath: "a.json", Data: map[string]any{
				"meta": map[string]any{"id": "1"},
			}, RowIndex: -1},
			{TypeName: "user", FilePath: "b.json", Data: map[string]any{
				"meta": map[string]any{"id": "1"},
			}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "unique-meta-id", Type: "unique", Key: "$.meta.id", Scope: "type",
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestForeignKey_InvalidKeySelector(t *testing.T) {
	items := map[string][]Item{
		"order": {
			{TypeName: "order", FilePath: "o1.json", Data: map[string]any{"user_id": "u1"}, RowIndex: -1},
		},
		"user": {
			{TypeName: "user", FilePath: "u1.json", Data: map[string]any{"id": "u1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "order",
		Constraints: []config.ConstraintDef{{
			ID: "fk-user", Type: "foreign_key", Key: "bad",
			References: &config.ReferenceDef{Type: "user", Key: "$.id"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestForeignKey_InvalidRefSelector(t *testing.T) {
	items := map[string][]Item{
		"order": {
			{TypeName: "order", FilePath: "o1.json", Data: map[string]any{"user_id": "u1"}, RowIndex: -1},
		},
		"user": {
			{TypeName: "user", FilePath: "u1.json", Data: map[string]any{"id": "u1"}, RowIndex: -1},
		},
	}
	defs := []config.TypeDef{{
		Name: "order",
		Constraints: []config.ConstraintDef{{
			ID: "fk-user", Type: "foreign_key", Key: "$.user_id",
			References: &config.ReferenceDef{Type: "user", Key: "bad"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestPathEqualsAttr_InvalidAttrSelector(t *testing.T) {
	items := map[string][]Item{
		"user": {
			{
				TypeName: "user", FilePath: "users/alice.json",
				Data:         map[string]any{"name": "alice"},
				PathCaptures: map[string]string{"path.file": "alice"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "user",
		Constraints: []config.ConstraintDef{{
			ID: "path-name", Type: "path_equals_attr", PathSelector: "path.file",
			References: &config.ReferenceDef{Key: "bad"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestPathEqualsAttr_ParentSelector(t *testing.T) {
	items := map[string][]Item{
		"item": {
			{
				TypeName: "item", FilePath: "categories/electronics/tv.json",
				Data:         map[string]any{"category": "electronics"},
				PathCaptures: map[string]string{"path.file": "tv", "path.parent": "electronics", "path.ext": "json"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "item",
		Constraints: []config.ConstraintDef{{
			ID: "path-cat", Type: "path_equals_attr", PathSelector: "path.parent",
			References: &config.ReferenceDef{Key: "$.category"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestPathEqualsAttr_ExtSelector(t *testing.T) {
	items := map[string][]Item{
		"doc": {
			{
				TypeName: "doc", FilePath: "docs/readme.yaml",
				Data:         map[string]any{"format": "yaml"},
				PathCaptures: map[string]string{"path.file": "readme", "path.parent": "docs", "path.ext": "yaml"},
				RowIndex:     -1,
			},
		},
	}
	defs := []config.TypeDef{{
		Name: "doc",
		Constraints: []config.ConstraintDef{{
			ID: "path-ext", Type: "path_equals_attr", PathSelector: "path.ext",
			References: &config.ReferenceDef{Key: "$.format"},
		}},
	}}
	errs := Evaluate(items, defs)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}
