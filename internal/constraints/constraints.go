package constraints

import (
	"fmt"
	"sort"
	"strings"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
	"github.com/UnitVectorY-Labs/datacur8/internal/selector"
)

// Item represents a parsed data item with its metadata.
type Item struct {
	TypeName     string
	FilePath     string
	Data         any               // The parsed data (map[string]any)
	PathCaptures map[string]string // Captured path segments
	RowIndex     int               // For CSV, the row index; -1 for JSON/YAML
}

// Error represents a constraint violation.
type Error struct {
	ConstraintID   string
	ConstraintType string
	TypeName       string
	FilePath       string
	Message        string
	RowIndex       int // -1 if not applicable
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.RowIndex >= 0 {
		return fmt.Sprintf("[%s] %s %s (row %d): %s", e.TypeName, e.ConstraintType, e.FilePath, e.RowIndex, e.Message)
	}
	return fmt.Sprintf("[%s] %s %s: %s", e.TypeName, e.ConstraintType, e.FilePath, e.Message)
}

// Evaluate evaluates all constraints across all items.
// items is a map from type name to slice of items.
// Returns errors sorted deterministically.
func Evaluate(items map[string][]Item, typeDefs []config.TypeDef) []Error {
	var errs []Error

	for _, td := range typeDefs {
		typeItems := items[td.Name]
		for ci, cd := range td.Constraints {
			constraintID := cd.ID
			if constraintID == "" {
				constraintID = fmt.Sprintf("#%d", ci)
			}
			var ces []Error
			switch cd.Type {
			case "unique":
				ces = evalUnique(td.Name, constraintID, cd, typeItems)
			case "foreign_key":
				ces = evalForeignKey(td.Name, constraintID, cd, typeItems, items)
			case "path_equals_attr":
				ces = evalPathEqualsAttr(td.Name, constraintID, cd, typeItems)
			}
			errs = append(errs, ces...)
		}
	}

	sort.Slice(errs, func(i, j int) bool {
		if errs[i].TypeName != errs[j].TypeName {
			return errs[i].TypeName < errs[j].TypeName
		}
		if errs[i].ConstraintID != errs[j].ConstraintID {
			return errs[i].ConstraintID < errs[j].ConstraintID
		}
		if errs[i].FilePath != errs[j].FilePath {
			return errs[i].FilePath < errs[j].FilePath
		}
		return errs[i].RowIndex < errs[j].RowIndex
	})

	return errs
}

// normalizeKey converts a value to a string key for comparison.
func normalizeKey(v any, caseSensitive bool) string {
	s := fmt.Sprintf("%v", v)
	if !caseSensitive {
		s = strings.ToLower(s)
	}
	return s
}

// evalUnique checks the "unique" constraint.
func evalUnique(typeName, constraintID string, cd config.ConstraintDef, items []Item) []Error {
	sel, err := selector.Parse(cd.Key)
	if err != nil {
		return []Error{{
			ConstraintID:   constraintID,
			ConstraintType: "unique",
			TypeName:       typeName,
			FilePath:       "",
			Message:        fmt.Sprintf("invalid selector %q: %v", cd.Key, err),
			RowIndex:       -1,
		}}
	}

	caseSensitive := cd.IsCaseSensitive()
	isScalar := sel.IsScalar()

	if isScalar && cd.Scope == "type" {
		return evalUniqueTypeScope(typeName, constraintID, cd, sel, caseSensitive, items)
	}

	// Multi-value or scope=="item": uniqueness within each item
	return evalUniqueItemScope(typeName, constraintID, cd, sel, caseSensitive, items)
}

// evalUniqueTypeScope enforces uniqueness of a scalar key across all items of the type.
func evalUniqueTypeScope(typeName, constraintID string, cd config.ConstraintDef, sel *selector.Selector, caseSensitive bool, items []Item) []Error {
	type seen struct {
		filePath string
		rowIndex int
	}
	index := make(map[string][]seen)

	for _, item := range items {
		vals, _ := sel.Evaluate(item.Data)
		if len(vals) == 0 {
			continue
		}
		key := normalizeKey(vals[0], caseSensitive)
		index[key] = append(index[key], seen{filePath: item.FilePath, rowIndex: item.RowIndex})
	}

	var errs []Error
	for key, entries := range index {
		if len(entries) < 2 {
			continue
		}
		for _, e := range entries {
			errs = append(errs, Error{
				ConstraintID:   constraintID,
				ConstraintType: "unique",
				TypeName:       typeName,
				FilePath:       e.filePath,
				Message:        fmt.Sprintf("duplicate value %q for key %s", key, cd.Key),
				RowIndex:       e.rowIndex,
			})
		}
	}

	return errs
}

// evalUniqueItemScope enforces uniqueness within each individual item.
func evalUniqueItemScope(typeName, constraintID string, cd config.ConstraintDef, sel *selector.Selector, caseSensitive bool, items []Item) []Error {
	var errs []Error

	for _, item := range items {
		vals, _ := sel.Evaluate(item.Data)
		seen := make(map[string]bool)
		for _, v := range vals {
			key := normalizeKey(v, caseSensitive)
			if seen[key] {
				errs = append(errs, Error{
					ConstraintID:   constraintID,
					ConstraintType: "unique",
					TypeName:       typeName,
					FilePath:       item.FilePath,
					Message:        fmt.Sprintf("duplicate value %q for key %s within item", key, cd.Key),
					RowIndex:       item.RowIndex,
				})
			}
			seen[key] = true
		}
	}

	return errs
}

// evalForeignKey checks the "foreign_key" constraint.
func evalForeignKey(typeName, constraintID string, cd config.ConstraintDef, items []Item, allItems map[string][]Item) []Error {
	if cd.References == nil {
		return []Error{{
			ConstraintID:   constraintID,
			ConstraintType: "foreign_key",
			TypeName:       typeName,
			FilePath:       "",
			Message:        "missing references definition",
			RowIndex:       -1,
		}}
	}

	keySel, err := selector.Parse(cd.Key)
	if err != nil {
		return []Error{{
			ConstraintID:   constraintID,
			ConstraintType: "foreign_key",
			TypeName:       typeName,
			FilePath:       "",
			Message:        fmt.Sprintf("invalid key selector %q: %v", cd.Key, err),
			RowIndex:       -1,
		}}
	}

	refSel, err := selector.Parse(cd.References.Key)
	if err != nil {
		return []Error{{
			ConstraintID:   constraintID,
			ConstraintType: "foreign_key",
			TypeName:       typeName,
			FilePath:       "",
			Message:        fmt.Sprintf("invalid references.key selector %q: %v", cd.References.Key, err),
			RowIndex:       -1,
		}}
	}

	// Build lookup index from referenced type
	refItems := allItems[cd.References.Type]
	refIndex := make(map[string]bool)
	for _, ri := range refItems {
		vals, _ := refSel.Evaluate(ri.Data)
		if len(vals) == 1 {
			refIndex[normalizeKey(vals[0], true)] = true
		}
	}

	var errs []Error
	for _, item := range items {
		vals, _ := keySel.Evaluate(item.Data)
		if len(vals) == 0 {
			continue
		}
		if len(vals) > 1 {
			errs = append(errs, Error{
				ConstraintID:   constraintID,
				ConstraintType: "foreign_key",
				TypeName:       typeName,
				FilePath:       item.FilePath,
				Message:        fmt.Sprintf("key selector %s resolved to multiple values; expected scalar", cd.Key),
				RowIndex:       item.RowIndex,
			})
			continue
		}
		key := normalizeKey(vals[0], true)
		if !refIndex[key] {
			errs = append(errs, Error{
				ConstraintID:   constraintID,
				ConstraintType: "foreign_key",
				TypeName:       typeName,
				FilePath:       item.FilePath,
				Message:        fmt.Sprintf("foreign key %q not found in %s.%s", key, cd.References.Type, cd.References.Key),
				RowIndex:       item.RowIndex,
			})
		}
	}

	return errs
}

// evalPathEqualsAttr checks the "path_equals_attr" constraint.
func evalPathEqualsAttr(typeName, constraintID string, cd config.ConstraintDef, items []Item) []Error {
	if cd.References == nil {
		return []Error{{
			ConstraintID:   constraintID,
			ConstraintType: "path_equals_attr",
			TypeName:       typeName,
			FilePath:       "",
			Message:        "missing references definition",
			RowIndex:       -1,
		}}
	}

	attrSel, err := selector.Parse(cd.References.Key)
	if err != nil {
		return []Error{{
			ConstraintID:   constraintID,
			ConstraintType: "path_equals_attr",
			TypeName:       typeName,
			FilePath:       "",
			Message:        fmt.Sprintf("invalid references.key selector %q: %v", cd.References.Key, err),
			RowIndex:       -1,
		}}
	}

	caseSensitive := cd.IsCaseSensitive()

	var errs []Error
	for _, item := range items {
		pathVal, ok := resolvePathSelector(cd.PathSelector, item.PathCaptures)
		if !ok {
			errs = append(errs, Error{
				ConstraintID:   constraintID,
				ConstraintType: "path_equals_attr",
				TypeName:       typeName,
				FilePath:       item.FilePath,
				Message:        fmt.Sprintf("path_selector %q not found in path captures", cd.PathSelector),
				RowIndex:       item.RowIndex,
			})
			continue
		}

		vals, _ := attrSel.Evaluate(item.Data)
		if len(vals) == 0 {
			errs = append(errs, Error{
				ConstraintID:   constraintID,
				ConstraintType: "path_equals_attr",
				TypeName:       typeName,
				FilePath:       item.FilePath,
				Message:        fmt.Sprintf("attribute selector %s resolved to no values", cd.References.Key),
				RowIndex:       item.RowIndex,
			})
			continue
		}
		if len(vals) > 1 {
			errs = append(errs, Error{
				ConstraintID:   constraintID,
				ConstraintType: "path_equals_attr",
				TypeName:       typeName,
				FilePath:       item.FilePath,
				Message:        fmt.Sprintf("attribute selector %s resolved to multiple values; expected scalar", cd.References.Key),
				RowIndex:       item.RowIndex,
			})
			continue
		}

		attrVal := normalizeKey(vals[0], caseSensitive)
		pv := pathVal
		if !caseSensitive {
			pv = strings.ToLower(pv)
		}

		if pv != attrVal {
			errs = append(errs, Error{
				ConstraintID:   constraintID,
				ConstraintType: "path_equals_attr",
				TypeName:       typeName,
				FilePath:       item.FilePath,
				Message:        fmt.Sprintf("path value %q does not match attribute value %q", pathVal, vals[0]),
				RowIndex:       item.RowIndex,
			})
		}
	}

	return errs
}

// resolvePathSelector extracts the value from path captures for the given path_selector.
func resolvePathSelector(pathSelector string, captures map[string]string) (string, bool) {
	// Built-in selectors: path.file, path.parent, path.ext
	// Custom captures: path.<capture_name> maps to captures[<capture_name>]
	// All are stored in captures with the full key (e.g., "path.file")
	v, ok := captures[pathSelector]
	return v, ok
}
