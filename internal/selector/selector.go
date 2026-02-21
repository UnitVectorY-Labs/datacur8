package selector

import (
	"fmt"
	"strings"
)

// segment represents one step in a selector path.
type segment struct {
	field    string // field name to access on an object
	wildcard bool   // true when the segment is [*] (iterate array elements)
}

// Selector is a parsed JSONPath-like selector.
type Selector struct {
	raw      string
	segments []segment
}

// Parse parses a selector string into a Selector.
// Valid forms: "$", "$.field", "$.a.b.c", "$.items[*].id", "$.a[*].b[*].c".
func Parse(sel string) (*Selector, error) {
	if sel == "" {
		return nil, fmt.Errorf("selector: empty selector")
	}
	if sel[0] != '$' {
		return nil, fmt.Errorf("selector: must start with '$': %s", sel)
	}

	s := &Selector{raw: sel}

	rest := sel[1:] // consume '$'
	if rest == "" {
		return s, nil // bare "$"
	}

	for rest != "" {
		if rest[0] == '.' {
			rest = rest[1:]
			if rest == "" {
				return nil, fmt.Errorf("selector: trailing dot: %s", sel)
			}
			// read field name (up to next '.', '[', or end)
			end := strings.IndexAny(rest, ".[")
			if end == -1 {
				end = len(rest)
			}
			name := rest[:end]
			if name == "" {
				return nil, fmt.Errorf("selector: empty field name: %s", sel)
			}
			s.segments = append(s.segments, segment{field: name})
			rest = rest[end:]
		} else if strings.HasPrefix(rest, "[*]") {
			s.segments = append(s.segments, segment{wildcard: true})
			rest = rest[3:]
		} else {
			return nil, fmt.Errorf("selector: unexpected character %q in: %s", rest[0], sel)
		}
	}

	return s, nil
}

// String returns the original selector string.
func (s *Selector) String() string {
	return s.raw
}

// IsScalar returns true if the selector will always yield exactly one value
// (no [*] wildcard in the path).
func (s *Selector) IsScalar() bool {
	for _, seg := range s.segments {
		if seg.wildcard {
			return false
		}
	}
	return true
}

// Evaluate applies the selector to data and returns all matched values.
// Missing fields yield an empty slice, not an error.
func (s *Selector) Evaluate(data any) ([]any, error) {
	results := resolve([]any{data}, s.segments)
	return results, nil
}

// resolve recursively applies the remaining segments to a set of current values.
func resolve(current []any, segments []segment) []any {
	if len(segments) == 0 {
		return current
	}

	seg := segments[0]
	rest := segments[1:]

	var next []any
	for _, val := range current {
		if seg.wildcard {
			arr, ok := val.([]any)
			if !ok {
				continue // not an array — skip
			}
			next = append(next, arr...)
		} else {
			m, ok := val.(map[string]any)
			if !ok {
				continue // not an object — skip
			}
			v, exists := m[seg.field]
			if !exists {
				continue // missing field — skip
			}
			next = append(next, v)
		}
	}

	return resolve(next, rest)
}
