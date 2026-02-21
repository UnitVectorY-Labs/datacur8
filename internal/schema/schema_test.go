package schema

import (
	"strings"
	"testing"
)

func TestValidateItem_BasicValid(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "integer"},
		},
		"required": []any{"name"},
	}

	data := map[string]any{
		"name": "Alice",
		"age":  float64(30),
	}

	errs := ValidateItem(s, data, "DISABLED")
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidateItem_BasicInvalid(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"name"},
	}

	data := map[string]any{
		"age": float64(30),
	}

	errs := ValidateItem(s, data, "DISABLED")
	if len(errs) == 0 {
		t.Error("expected validation errors for missing required field")
	}
}

func TestValidateItem_TypeMismatch(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"age": map[string]any{"type": "integer"},
		},
	}

	data := map[string]any{
		"age": "not a number",
	}

	errs := ValidateItem(s, data, "DISABLED")
	if len(errs) == 0 {
		t.Error("expected validation errors for type mismatch")
	}
}

func TestStrictMode_Disabled_AllowsExtraProperties(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}

	data := map[string]any{
		"name":  "Alice",
		"extra": "value",
	}

	errs := ValidateItem(s, data, "DISABLED")
	if len(errs) != 0 {
		t.Errorf("DISABLED mode should allow extra properties, got %v", errs)
	}
}

func TestStrictMode_Enabled_ForbidsExtraProperties(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}

	data := map[string]any{
		"name":  "Alice",
		"extra": "value",
	}

	errs := ValidateItem(s, data, "ENABLED")
	if len(errs) == 0 {
		t.Error("ENABLED mode should forbid extra properties when not explicitly set")
	}
}

func TestStrictMode_Enabled_RespectsExplicitTrue(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"additionalProperties": true,
	}

	data := map[string]any{
		"name":  "Alice",
		"extra": "value",
	}

	errs := ValidateItem(s, data, "ENABLED")
	if len(errs) != 0 {
		t.Errorf("ENABLED mode should respect explicit additionalProperties:true, got %v", errs)
	}
}

func TestStrictMode_Force_OverridesExplicitTrue(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"additionalProperties": true,
	}

	data := map[string]any{
		"name":  "Alice",
		"extra": "value",
	}

	errs := ValidateItem(s, data, "FORCE")
	if len(errs) == 0 {
		t.Error("FORCE mode should override explicit additionalProperties:true")
	}
}

func TestStrictMode_Enabled_NestedObjects(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"address": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string"},
				},
			},
		},
	}

	data := map[string]any{
		"address": map[string]any{
			"city":    "NYC",
			"country": "US",
		},
	}

	errs := ValidateItem(s, data, "ENABLED")
	if len(errs) == 0 {
		t.Error("ENABLED mode should forbid extra properties in nested objects")
	}
}

func TestStrictMode_Force_NestedObjects(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"address": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string"},
				},
				"additionalProperties": true,
			},
		},
	}

	data := map[string]any{
		"address": map[string]any{
			"city":    "NYC",
			"country": "US",
		},
	}

	errs := ValidateItem(s, data, "FORCE")
	if len(errs) == 0 {
		t.Error("FORCE mode should override additionalProperties:true in nested objects")
	}
}

func TestValidateItem_ArraySchema(t *testing.T) {
	s := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "string",
		},
	}

	valid := []any{"a", "b", "c"}
	errs := ValidateItem(s, valid, "DISABLED")
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid array, got %v", errs)
	}

	invalid := []any{"a", float64(1)}
	errs = ValidateItem(s, invalid, "DISABLED")
	if len(errs) == 0 {
		t.Error("expected validation errors for array with wrong item type")
	}
}

func TestValidateItem_StringSchema(t *testing.T) {
	s := map[string]any{
		"type":      "string",
		"minLength": float64(3),
	}

	errs := ValidateItem(s, "hello", "DISABLED")
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}

	errs = ValidateItem(s, "hi", "DISABLED")
	if len(errs) == 0 {
		t.Error("expected validation errors for string shorter than minLength")
	}
}

func TestValidateItem_NumberSchema(t *testing.T) {
	s := map[string]any{
		"type":    "number",
		"minimum": float64(0),
		"maximum": float64(100),
	}

	errs := ValidateItem(s, float64(50), "DISABLED")
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}

	errs = ValidateItem(s, float64(200), "DISABLED")
	if len(errs) == 0 {
		t.Error("expected validation errors for number exceeding maximum")
	}
	if strings.Contains(errs[0].Error(), "/") {
		t.Errorf("expected decimal numeric value in error, got: %s", errs[0].Error())
	}
	if !strings.Contains(errs[0].Error(), "200") {
		t.Errorf("expected original numeric value in error, got: %s", errs[0].Error())
	}
}

func TestValidateItem_NumberSchema_RationalErrorIsNormalized(t *testing.T) {
	s := map[string]any{
		"type":    "number",
		"maximum": float64(6),
	}

	errs := ValidateItem(s, float64(95.5), "DISABLED")
	if len(errs) == 0 {
		t.Fatal("expected validation errors for number exceeding maximum")
	}
	if strings.Contains(errs[0].Error(), "191/2") || strings.Contains(errs[0].Error(), "/") {
		t.Errorf("expected fraction to be normalized to decimal, got: %s", errs[0].Error())
	}
	if !strings.Contains(errs[0].Error(), "95.5") {
		t.Errorf("expected decimal value 95.5 in error, got: %s", errs[0].Error())
	}
}

func TestApplyStrictMode_DoesNotMutateOriginal(t *testing.T) {
	original := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}

	_ = ApplyStrictMode(original, "FORCE")

	if _, ok := original["additionalProperties"]; ok {
		t.Error("ApplyStrictMode should not mutate the original schema")
	}
}

func TestApplyStrictMode_Disabled(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}

	result := ApplyStrictMode(s, "DISABLED")
	if _, ok := result["additionalProperties"]; ok {
		t.Error("DISABLED mode should not add additionalProperties")
	}
}

func TestApplyStrictMode_Enabled(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}

	result := ApplyStrictMode(s, "ENABLED")
	ap, ok := result["additionalProperties"]
	if !ok {
		t.Fatal("ENABLED mode should add additionalProperties")
	}
	if ap != false {
		t.Error("ENABLED mode should set additionalProperties to false")
	}
}

func TestApplyStrictMode_Force(t *testing.T) {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"additionalProperties": true,
	}

	result := ApplyStrictMode(s, "FORCE")
	ap, ok := result["additionalProperties"]
	if !ok {
		t.Fatal("FORCE mode should set additionalProperties")
	}
	if ap != false {
		t.Error("FORCE mode should set additionalProperties to false")
	}
}

func TestStrictMode_ArrayOfObjects(t *testing.T) {
	s := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "integer"},
			},
		},
	}

	data := []any{
		map[string]any{"id": float64(1), "extra": "field"},
	}

	errs := ValidateItem(s, data, "ENABLED")
	if len(errs) == 0 {
		t.Error("ENABLED mode should forbid extra properties in array item objects")
	}
}
