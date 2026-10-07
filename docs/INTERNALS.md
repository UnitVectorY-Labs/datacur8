---
layout: default
title: Internals
nav_order: 7
has_toc: false
has_children: true
permalink: /internals
---

# Internals
{: .no_toc }

Internal architecture and design details for datacur8.

For the data-driven fixture contract used by the integration suite, see [Data-Driven Tests](/data-driven-tests).

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
  jsonldata/             # Shared JSONL/NDJSON framing, line errors, canonical tidy
  tomldata/              # TOML 1.1 parsing, checked JSON conversion, export, canonical tidy
  hcldata/               # HCL parsing, JSON conversion, export, and formatting
  schema/                # JSON Schema validation with strict mode
  selector/              # JSONPath-like selector parser and evaluator
  tidy/                  # File formatting and normalization
```

### Package dependencies

```
main → cli → config, constraints, discovery, export, hcldata, jsonldata, tomldata, schema, tidy
constraints → config, selector
discovery → config
export → config, hcldata, tomldata
hcldata → (external: hashicorp/hcl/v2, zclconf/go-cty)
schema → (external: google/jsonschema-go)
selector → (standalone)
tidy → hcldata, jsonldata, tomldata
tomldata → (external: pelletier/go-toml/v2)
```

## Validation Phases

Validation runs in a strict sequence of phases. Each phase must succeed before the next phase runs. This ensures that errors are reported at the earliest meaningful point.

### Phase 1: Config Validation

**Package:** `config`

1. Load and parse the `.datacur8` YAML file
2. Apply default values (strict_mode, constraint scope)
3. Validate the config structurally and semantically:
   - Version format and compatibility
   - Valid enum values for strict_mode, input, output.format
   - Unique type names
   - Unique output paths across types
   - Regex patterns compile successfully
   - Schemas are present with `type: object`
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
3. For HCL: parse attributes, evaluate without external context, and convert values through JSON into one `map[string]any`
4. For CSV: validate headers, convert each row into a typed `map[string]any`; for JSONL/NDJSON: parse all physical lines into ordered objects
5. Apply strict mode overlay to the schema (if configured)
6. Validate each item against its JSON Schema using `google/jsonschema-go`

CSV parsing is notable: it uses the schema to guide type conversion of cell values (string → boolean, number, integer), and rejects duplicate/empty headers through `config.CSVHeaders` and validates headers against schema properties and required fields.

### Phase 4: Constraint Evaluation

**Package:** `constraints`

1. Build in-memory indexes for all items grouped by type
2. Evaluate each type's constraints:
   - **unique**: Build a set of seen values; report duplicates
   - **foreign_key**: Build a lookup index of referenced type's key values; check each owning item
   - **path_equals_attr**: Compare path capture value against item attribute value
3. Collect all errors with stable ordering (by type, constraint ID, file path, line number, then CSV row index)

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

## CSV Parsing

CSV files are handled specially because they don't have native types — every cell is a string.

### Parsing flow

1. **Read** the entire CSV file using `TypeDef.CSV.Rune()` (comma by default)
2. **Validate headers**: every column name must exist in `schema.properties`; every `schema.required` field must be present as a column
3. **Convert** each cell value based on the schema property type:
   - `string`: used as-is
   - `boolean`: `"true"` → `true`, `"false"` → `false` (case-insensitive)
   - `number`: parsed as float64
   - `integer`: parsed as integer, then stored as float64 for JSON compatibility
4. **Validate** each row object against the JSON Schema

If any header validation fails, no rows are processed. If any cell cannot be converted, the entire file is rejected with per-row error messages.

## HCL Parsing and Formatting

HCL input uses HashiCorp HCL v2. Each file is an attribute body representing one item. Block syntax is rejected because block labels and repetition have no schema-independent JSON mapping. Attribute expressions are evaluated without variables or functions; self-contained expressions are supported. Values pass through JSON conversion before schema validation, giving the pipeline the same maps, arrays, scalar values, and numeric representation as JSON input. JSON Schema and constraint evaluation remain format-independent.

HCL export serializes the aggregate object as one type-named attribute containing an array, using deterministic key ordering. HCL tidy uses `hclwrite.Format` on source bytes to preserve comments and attribute order.

## TOML Parsing and Formatting

`tomldata` isolates go-toml v2.4.3 (TOML 1.1.0) from the format-independent pipeline. `Parse` decodes one document, preserves parser file/line/column diagnostics, and recursively converts maps/arrays/scalars. Temporal types become canonical strings with up to nanosecond precision; local values never acquire a timezone. Signed integers are checked with `math/big` for exact float64 conversion; non-finite floats are rejected. Sorted traversal makes conversion errors deterministic and includes nested key/index paths.

`Marshal` checks the aggregate recursively before serialization, rejecting null even inside nested arrays/objects. Native integer values from existing formats must fit signed 64-bit range and normalize exactly. The encoder sorts map keys and uses `SetTablesInline(true)` to preserve empty objects and heterogeneous arrays. Strings are never converted to temporal types. Empty datasets explicitly encode `type_name = []`. `Format` validates conversion limits but encodes the original decoded values, retaining native temporal/integer types. It removes comments, sorts keys, and canonicalizes spelling instead of preserving source formatting. Both parse and tidy truncate fractional digits beyond nanoseconds.

The new dependency is pure Go, requires Go 1.21, adds no runtime transitive dependencies, and supports the existing release OS/architecture matrix. No release matrix changes are required.

## Export Ordering

Export produces deterministic output through strict ordering rules:

1. Types are processed in the order they appear in the config
2. Items within a type are ordered by file path (lexicographic)
3. For CSV files, rows maintain their within-file order

### Output formats

- **JSON**: Items are wrapped in an object keyed by the type name, with the value being an array. Pretty-printed with 2-space indentation.
- **YAML**: Same structure as JSON but serialized as YAML.
- **CSV**: `config.CSVColumns` validates the export schema subset during semantic configuration validation and supplies sorted names/types to `export.marshalCSV`. The encoder checks every item for complete scalar cells and undeclared keys, formats finite numbers and signed 64-bit integers, and uses `encoding/csv` with LF records. A single empty cell is explicitly quoted to avoid the reader skipping a blank line. It flushes and checks writer errors.
- **JSONL/NDJSON**: Equivalent output names; one minified JSON object per line, zero bytes for no records.
- **TOML**: One type-name key containing inline-table records; keys sorted recursively, explicit empty dataset, checked representability.
- **HCL**: One attribute named after the type containing the item array (`type_name = [{ ... }]`); object keys are sorted.

Output directories are created automatically if they don't exist. Each output is serialized in memory before `os.WriteFile`, so conversion failures do not truncate an existing destination; outputs are independent rather than a global transaction. CSV input parses integers with `strconv.ParseInt(..., 10, 64)` before converting to the existing float64 representation, avoiding platform-dependent `int` limits. See [CSV export](/configuration#csv-export) for supported schemas and parser precision limits.

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
- Export operates on already-parsed data
- HCL tidy formats source bytes to preserve comments and attribute order

`config.CSVOptions` holds an optional delimiter pointer so omission defaults to
comma while an explicit empty string is rejected. The embedded configuration
schema checks option shape, delimiter length/characters, and placement; semantic
validation also enforces Go CSV rune rules and format compatibility for constructed
configs. Input and output options are independent. The CLI passes input options
to CSV parsing and `tidy.TidyFile`; export passes output options to
`marshalCSV`. All use `encoding/csv` reader/writer separators, preserving
existing conversion, diagnostics, and ordering. Tidy validates ambiguous headers
before sorting columns with their cells and writing. Both writers quote a single
empty string record explicitly to preserve it on re-read. No TSV parser or format
alias is introduced.

## JSONL / NDJSON parsing and formatting

`jsonldata.Parse` implements both config names without additional dependencies.
The CLI reads each file with `os.ReadFile` and surfaces read errors. `bytes.Cut`
frames LF-delimited records without Scanner's token-size limit, validating UTF-8,
BOM exclusion, nonblank lines, and exactly one non-null object via `json.Unmarshal`.
CRLF and missing final LF are accepted; an empty byte slice yields zero items.
Parsing returns no records if any later line fails. Numbers retain JSON input's
float64 behavior; there is no CSV-style coercion. Files occupy memory in full;
there is no additional explicit size limit.

`jsonldata.Error` carries the original one-based line and wraps the cause. Since
blank records are rejected, successful slice index + 1 is the physical line.
`constraints.Item.LineNumber` and `constraints.Error.LineNumber` carry this metadata
through all unique/foreign-key/path branches and deterministic error ordering.
Zero means no line; the independent `RowIndex` remains zero-based CSV metadata
with -1 for other inputs. CLI `reportEntry.Line` is optional in JSON/YAML and text.
Path captures remain shared file metadata for all records.

`jsonldata.Format` validates the entire file before producing any output. A second
JSON decode with `UseNumber` preserves exact numeric values when formatting,
while the initial float64 decode enforces the same accepted numeric range as
validation. Go's JSON encoder sorts keys recursively, minifies each object, and
writes LF. Empty files stay empty. `tidyJSONL` compares full content before writing;
parse errors cannot rewrite an earlier valid prefix. `RunTidy` extracts wrapped
line errors for structured diagnostics. Tidy checks syntax, not schema/constraints.
