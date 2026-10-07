// Package tomldata converts TOML 1.1.0 to the common JSON-compatible model.
package tomldata

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
)

func decode(src []byte, filename string) (map[string]any, error) {
	var data map[string]any
	if err := toml.Unmarshal(src, &data); err != nil {
		if e, ok := err.(*toml.DecodeError); ok {
			line, column := e.Position()
			return nil, fmt.Errorf("%s:%d:%d: %w", filename, line, column, err)
		}
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	return data, nil
}

// Parse reads one record, recursively normalizing native temporal values to
// strings and checking integer precision before converting to float64.
func Parse(src []byte, filename string) (map[string]any, error) {
	data, err := decode(src, filename)
	if err != nil {
		return nil, err
	}
	normalized, err := normalize(data, "$")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	return normalized.(map[string]any), nil
}

func normalize(v any, path string) (any, error) {
	fail := func(message string) (any, error) { return nil, fmt.Errorf("%s: %s", path, message) }
	switch value := v.(type) {
	case nil:
		return fail("null is not representable in TOML")
	case map[string]any:
		out := make(map[string]any, len(value))
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !utf8.ValidString(key) {
				return fail("key is not valid UTF-8")
			}
			child, err := normalize(value[key], fmt.Sprintf("%s[%q]", path, key))
			if err != nil {
				return nil, err
			}
			out[key] = child
		}
		return out, nil
	case []any:
		out := make([]any, len(value))
		for i, child := range value {
			normalized, err := normalize(child, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = normalized
		}
		return out, nil
	case time.Time:
		return value.Format(time.RFC3339Nano), nil
	case toml.LocalDateTime:
		return value.LocalDate.String() + "T" + canonicalTime(value.LocalTime), nil
	case toml.LocalDate:
		return value.String(), nil
	case toml.LocalTime:
		return canonicalTime(value), nil
	case string:
		if !utf8.ValidString(value) {
			return fail("string is not valid UTF-8")
		}
		return value, nil
	case bool:
		return value, nil
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fail("non-finite float is not representable in JSON")
		}
		return value, nil
	}
	// YAML and CSV may supply any of Go's signed/unsigned integer types.
	r := reflect.ValueOf(v)
	integer := new(big.Int)
	switch r.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer.SetInt64(r.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		integer.SetUint64(r.Uint())
	default:
		return fail(fmt.Sprintf("unsupported value of type %T", v))
	}
	if !integer.IsInt64() {
		return fail("integer is outside TOML signed 64-bit range")
	}
	number, accuracy := new(big.Float).SetInt(integer).Float64()
	if accuracy != big.Exact {
		return fail("integer cannot be represented exactly as float64")
	}
	return number, nil
}

func canonicalTime(t toml.LocalTime) string {
	return time.Date(0, 1, 1, t.Hour, t.Minute, t.Second, t.Nanosecond, time.UTC).Format("15:04:05.999999999")
}

func encode(data map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	// Inline tables preserve empty objects and heterogeneous arrays, including
	// arrays mixing objects and scalars. The encoder sorts map keys recursively.
	err := toml.NewEncoder(&buf).SetTablesInline(true).Encode(data)
	return buf.Bytes(), err
}

// Marshal writes a type-name key containing the ordered records. Strings are
// never interpreted as temporal values. Conversion completes before any write.
func Marshal(typeName string, data []any) ([]byte, error) {
	if data == nil {
		data = []any{}
	}
	normalized, err := normalize(map[string]any{typeName: data}, "$")
	if err != nil {
		return nil, err
	}
	return encode(normalized.(map[string]any))
}

// Format canonicalizes validated source, removing comments and sorting keys.
// Encode native decoded values so dates/times and exact integers retain types.
func Format(src []byte, filename string) ([]byte, error) {
	data, err := decode(src, filename)
	if err != nil {
		return nil, err
	}
	if _, err := normalize(data, "$"); err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	return encode(data)
}
