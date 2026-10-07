// Package hcldata converts attribute-only native HCL documents to and from
// the JSON-compatible values used by validation, constraints, and export.
package hcldata

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// Parse reads one object from a native HCL document. Expressions are evaluated
// without variables or functions; the JSON Schema remains the data schema.
func Parse(src []byte, filename string) (map[string]any, error) {
	file, diags := hclsyntax.ParseConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("%s", diags.Error())
	}
	body := file.Body.(*hclsyntax.Body)
	if len(body.Blocks) > 0 {
		block := body.Blocks[0]
		return nil, fmt.Errorf("%s: HCL blocks are not supported; use attributes with object or list values instead", block.TypeRange.String())
	}
	// Report expression errors in source order rather than map iteration order.
	attrs := make([]*hclsyntax.Attribute, 0, len(body.Attributes))
	for _, attr := range body.Attributes {
		attrs = append(attrs, attr)
	}
	sort.Slice(attrs, func(i, j int) bool { return attrs[i].Range().Start.Byte < attrs[j].Range().Start.Byte })
	values := make(map[string]cty.Value, len(attrs))
	for _, attr := range attrs {
		value, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			return nil, fmt.Errorf("%s", diags.Error())
		}
		values[attr.Name] = value
	}
	value := cty.ObjectVal(values)
	raw, err := ctyjson.Marshal(value, value.Type())
	if err != nil {
		return nil, fmt.Errorf("converting HCL to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("converting HCL to JSON: %w", err)
	}
	return data, nil
}

// Marshal exports an aggregate using one type-name attribute whose value is a
// tuple of objects. Inferring cty types from JSON preserves heterogeneous arrays,
// empty collections, and nulls without imposing a separate HCL data schema.
func Marshal(typeName string, data []any) ([]byte, error) {
	if !hclsyntax.ValidIdentifier(typeName) {
		return nil, fmt.Errorf("type name %q is not a valid HCL attribute name", typeName)
	}
	if data == nil {
		data = []any{}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	typ, err := ctyjson.ImpliedType(raw)
	if err != nil {
		return nil, err
	}
	value, err := ctyjson.Unmarshal(raw, typ)
	if err != nil {
		return nil, err
	}
	file := hclwrite.NewEmptyFile()
	file.Body().SetAttributeValue(typeName, value)
	return file.Bytes(), nil
}

// Format validates the supported HCL dialect before formatting its source.
// Formatting retains comments, attribute order, and expression spelling.
func Format(src []byte, filename string) ([]byte, error) {
	if _, err := Parse(src, filename); err != nil {
		return nil, err
	}
	return hclwrite.Format(src), nil
}
