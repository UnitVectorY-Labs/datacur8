package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
)

// ValidateItem validates a single data item against the type's schema.
// strictMode is "DISABLED", "ENABLED", or "FORCE".
// Returns validation errors.
func ValidateItem(schemaMap map[string]any, data any, strictMode string) []error {
	adjusted := ApplyStrictMode(schemaMap, strictMode)

	schemaJSON, err := json.Marshal(adjusted)
	if err != nil {
		return []error{fmt.Errorf("marshaling schema: %w", err)}
	}

	var s jsonschema.Schema
	if err := json.Unmarshal(schemaJSON, &s); err != nil {
		return []error{fmt.Errorf("unmarshaling schema: %w", err)}
	}

	resolved, err := s.Resolve(nil)
	if err != nil {
		return []error{fmt.Errorf("resolving schema: %w", err)}
	}

	if err := resolved.Validate(data); err != nil {
		return []error{errors.New(normalizeValidationMessage(err.Error()))}
	}

	return nil
}

var rationalNumberPattern = regexp.MustCompile(`\b\d+/\d+\b`)

// normalizeValidationMessage makes library-generated numeric values easier to read.
// jsonschema-go can render float64 values as exact rationals (for example 191/2),
// which is correct but noisy in CLI output.
func normalizeValidationMessage(msg string) string {
	return rationalNumberPattern.ReplaceAllStringFunc(msg, func(s string) string {
		r := new(big.Rat)
		if _, ok := r.SetString(s); !ok {
			return s
		}
		f, _ := r.Float64()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return s
		}
		return strconv.FormatFloat(f, 'g', -1, 64)
	})
}

// ApplyStrictMode returns a deep copy of the schema with strict_mode overlay applied.
func ApplyStrictMode(schemaMap map[string]any, mode string) map[string]any {
	copied := deepCopyMap(schemaMap)
	if mode == "ENABLED" || mode == "FORCE" {
		applyStrict(copied, mode)
	}
	return copied
}

// applyStrict recursively walks the schema and sets additionalProperties to false
// on object schemas according to the mode.
func applyStrict(schema map[string]any, mode string) {
	schemaType, _ := schema["type"].(string)
	isObject := schemaType == "object"

	if isObject {
		_, hasAP := schema["additionalProperties"]
		switch mode {
		case "ENABLED":
			if !hasAP {
				schema["additionalProperties"] = false
			}
		case "FORCE":
			schema["additionalProperties"] = false
		}
	}

	// Recurse into properties
	if props, ok := schema["properties"].(map[string]any); ok {
		for _, v := range props {
			if sub, ok := v.(map[string]any); ok {
				applyStrict(sub, mode)
			}
		}
	}

	// Recurse into items (for array of objects)
	if items, ok := schema["items"].(map[string]any); ok {
		applyStrict(items, mode)
	}

	// Recurse into allOf, anyOf, oneOf
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		if arr, ok := schema[keyword].([]any); ok {
			for _, item := range arr {
				if sub, ok := item.(map[string]any); ok {
					applyStrict(sub, mode)
				}
			}
		}
	}

	// Recurse into additionalProperties if it's a schema object
	if ap, ok := schema["additionalProperties"].(map[string]any); ok {
		applyStrict(ap, mode)
	}

	// Recurse into if/then/else
	for _, keyword := range []string{"if", "then", "else", "not"} {
		if sub, ok := schema[keyword].(map[string]any); ok {
			applyStrict(sub, mode)
		}
	}

	// Recurse into patternProperties
	if pp, ok := schema["patternProperties"].(map[string]any); ok {
		for _, v := range pp {
			if sub, ok := v.(map[string]any); ok {
				applyStrict(sub, mode)
			}
		}
	}

	// Recurse into $defs / definitions
	for _, keyword := range []string{"$defs", "definitions"} {
		if defs, ok := schema[keyword].(map[string]any); ok {
			for _, v := range defs {
				if sub, ok := v.(map[string]any); ok {
					applyStrict(sub, mode)
				}
			}
		}
	}
}

// deepCopyMap creates a deep copy of a map[string]any.
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = deepCopyValue(v)
	}
	return result
}

func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case []any:
		cp := make([]any, len(val))
		for i, item := range val {
			cp[i] = deepCopyValue(item)
		}
		return cp
	default:
		return v
	}
}
