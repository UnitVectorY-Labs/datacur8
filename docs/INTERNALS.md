---
layout: default
title: Internals
nav_order: 7
permalink: /internals
---

# Internals
{: .no_toc }

Internal architecture and design details for datacur8.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Package Structure

datacur8 is implemented in Go with all application logic in the `internal/` directory to enforce encapsulation. The `main.go` at the repository root handles CLI argument parsing and delegates to internal packages.

```
main.go                  # CLI entry point, flag parsing
internal/
  cli/                   # Command orchestration (validate, export, tidy)
  config/                # Config model, loading, defaults, validation
  constraints/           # Constraint evaluation engine
  discovery/             # File discovery and type matching
  export/                # Output file generation
  schema/                # JSON Schema validation with strict mode
  selector/              # JSONPath-like selector parser and evaluator
  tidy/                  # File formatting and normalization
```

### Package dependencies

```
main → cli → config, constraints, discovery, export, schema, tidy
constraints → config, selector
discovery → config
export → config
schema → (external: google/jsonschema-go)
selector → (standalone)
tidy → (standalone)
```

## Validation Phases

Validation runs in a strict sequence of phases. Each phase must succeed before the next phase runs. This ensures that errors are reported at the earliest meaningful point.

### Phase 1: Config Validation

**Package:** `config`

1. Load and parse the `.datacur8` YAML file
2. Apply default values (strict_mode, constraint scope, csv delimiter)
3. Validate the config structurally and semantically:
   - Version format and compatibility
   - Valid enum values for strict_mode, input, output.format
   - Unique type names
   - Unique output paths across types
   - Regex patterns compile successfully
   - Schemas are present with `type: object`
   - CSV config is present when input is csv
   - Constraint selectors are valid
   - Foreign key references point to defined types
   - Path capture groups exist in all include patterns

Config validation returns both warnings and errors. Warnings (e.g., version check skipped for dev builds) do not prevent further processing.

### Phase 2: File Discovery

**Package:** `discovery`

1. Walk the repository directory tree
2. Skip ignored directories (`.git`, `node_modules`, `__pycache__`, etc.) and output paths
3. For each file, test against all type include/exclude patterns
4. Extract named capture groups and built-in path values
5. Validate that each file matches exactly one type

Discovery pre-compiles all regex patterns for efficiency. The result is a sorted list of `DiscoveredFile` records, each carrying a pointer to its `TypeDef` and a map of path captures.

### Phase 3: Schema Validation

**Package:** `schema`, `cli`

1. Read and parse each discovered file according to its input format
2. For JSON and YAML: parse into a single `map[string]any`
3. For CSV: validate headers, convert each row into a typed `map[string]any`
4. Apply strict mode overlay to the schema (if configured)
5. Validate each item against its JSON Schema using `google/jsonschema-go`

CSV parsing is notable: it uses the schema to guide type conversion of cell values (string → boolean, number, integer), and validates headers against schema properties and required fields.

### Phase 4: Constraint Evaluation

**Package:** `constraints`

1. Build in-memory indexes for all items grouped by type
2. Evaluate each type's constraints:
   - **unique**: Build a set of seen values; report duplicates
   - **foreign_key**: Build a lookup index of referenced type's key values; check each owning item
   - **path_equals_attr**: Compare path capture value against item attribute value
3. Collect all errors with stable ordering (by type, then file path, then row index)

## Selectors

The selector package implements a constrained subset of JSONPath for predictable behavior.

### Supported syntax

| Syntax | Example | Meaning |
|--------|---------|---------|
| Root | `$` | The entire object |
| Field access | `$.field` | A top-level field |
| Nested access | `$.a.b.c` | Nested field traversal |
| Array projection | `$.items[*].id` | All `id` values from array items |

### Evaluation behavior

- Selectors are parsed into a sequence of segments (field names and wildcards)
- Evaluation traverses the data structure following each segment
- Missing fields return an empty result (not an error)
- The `[*]` wildcard expands across all elements of an array
- A selector is "scalar" if it contains no `[*]` wildcards

### Multi-value handling

When a selector yields multiple values for a single item:

- **unique** with `scope: type`: each value contributes to the global uniqueness set
- **unique** with `scope: item`: all values within one item must be unique
- **foreign_key**: invalid — requires a single scalar value
- **path_equals_attr**: invalid — requires a single scalar value

## Strict Mode

Strict mode is implemented as a schema overlay applied at validation time, not by modifying the config on disk.

### DISABLED

Schemas are passed to the validator exactly as defined.

### ENABLED

Before validation, the schema is traversed recursively. For every object schema that does not explicitly set `additionalProperties`, the overlay inserts `additionalProperties: false`. Explicit settings (`true` or `false`) are preserved.

### FORCE

Same as ENABLED, but `additionalProperties: true` is overridden to `false`. Only `additionalProperties` set to a schema object (not boolean) is preserved.

The overlay function walks the schema tree, visiting `properties`, `items`, `allOf`, `anyOf`, `oneOf`, `if`, `then`, `else`, and `additionalProperties` (when it is a schema object).

## CSV Parsing

CSV files are handled specially because they don't have native types — every cell is a string.

### Parsing flow

1. **Read** the entire CSV file with the configured delimiter
2. **Validate headers**: every column name must exist in `schema.properties`; every `schema.required` field must be present as a column
3. **Convert** each cell value based on the schema property type:
   - `string`: used as-is
   - `boolean`: `"true"` → `true`, `"false"` → `false` (case-insensitive)
   - `number`: parsed as float64
   - `integer`: parsed as integer, then stored as float64 for JSON compatibility
4. **Validate** each row object against the JSON Schema

If any header validation fails, no rows are processed. If any cell cannot be converted, the entire file is rejected with per-row error messages.

## Export Ordering

Export produces deterministic output through strict ordering rules:

1. Types are processed in the order they appear in the config
2. Items within a type are ordered by file path (lexicographic)
3. For CSV files, rows maintain their within-file order

### Output formats

- **JSON**: Items are wrapped in an object keyed by the type name, with the value being an array. Pretty-printed with 2-space indentation.
- **YAML**: Same structure as JSON but serialized as YAML.
- **JSONL**: One minified JSON object per line.

Output directories are created automatically if they don't exist.

## Memory Model

datacur8 uses an in-memory model for all processing:

- All discovered files are loaded into memory
- All parsed items are held in memory simultaneously
- Constraint indexes (uniqueness sets, foreign key lookup maps) are built in memory

This approach is simple and fast for the expected use case (hundreds to low thousands of files). The architecture allows for future optimizations (streaming, spill-to-disk) without changing the configuration model.

## Performance Notes

- Regex patterns are pre-compiled once during discovery setup
- File discovery skips common non-data directories early
- Schema validation uses a compiled schema evaluator
- Constraint evaluation builds indexes in a single pass, then validates in a second pass
- Export and tidy operate on already-parsed data, avoiding re-reads
