---
layout: default
title: Configuration
nav_order: 3
has_children: true
has_toc: false
permalink: /configuration
---

# Configuration
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

**datacur8** is configured by a single YAML file named `.datacur8` placed in the repository root directory. This file defines all types, schemas, constraints, and export settings.

No additional config files are used, including in subdirectories. If a `.datacur8` file is found in a subdirectory, an error is returned.

{: .important }
The root config object is validated against `internal/config/config.schema.json` before semantic validation runs. Unknown fields are rejected for this config using `additionalProperties: false`.

---

## version

| Property | Value |
|---|---|
| Field | `version` |
| Type | `string` |
| Required | yes |
| Default | — |
| Description | Minimum **datacur8** version required by this config (semver `major.minor.patch`). |

**Schema details**

- Pattern: `^[0-9]+\.[0-9]+\.[0-9]+$`

The major version must match the CLI version, and the CLI version must be greater than or equal to the configured version.

{: .highlight }
When running a development build, the version compatibility check is skipped with a warning.

---

## strict_mode

| Property | Value |
|---|---|
| Field | `strict_mode` |
| Type | `string` |
| Required | no |
| Default | `DISABLED` |
| Description | Controls how `additionalProperties` is applied to JSON schemas used in `types[].schema`. |

**Allowed values**

| Value | Behavior |
|---|---|
| `DISABLED` | Schemas are evaluated as-is. |
| `ENABLED` | Object schemas without explicit `additionalProperties` are treated as `additionalProperties: false`. |
| `FORCE` | All object schemas are forced to `additionalProperties: false`, even if explicitly `true`. |

---

## tidy

Configuration for the `tidy` command.

| Property | Value |
|---|---|
| Field | `tidy` |
| Type | `object` |
| Required | no |

---

### enabled

| Property | Value |
|---|---|
| Field | `enabled` |
| Type | `boolean` |
| Required | no |
| Default | `true` |
| Description | Enables or disables the `tidy` command for the repository. |

{: .highlight }
If `tidy` is omitted entirely, tidy remains enabled. If `tidy` is present and `enabled: false`, the `tidy` command exits with a disabled message.

---

## types

The `types` are the different categories of data files that are represented. These could be thought of as different "tables" in a database, where each type has its own schema, constraints, and export settings.

| Property | Value |
|---|---|
| Field | `types` |
| Type | `array` of objects |
| Required | yes |
| Default | — |
| Description | List of type definitions used to discover, validate, constrain, and optionally export data files. |

---

### name

| Property | Value |
|---|---|
| Field | `name` |
| Type | `string` |
| Required | yes |
| Default | — |
| Description | Unique identifier for the type. |

**Schema details**

- `minLength`: `1`
- `maxLength`: `255`
- Pattern: `^[a-zA-Z][a-zA-Z0-9_]*$`

{: .important }
Type names must be unique across all entries in `types`. They are also used in exports and constraint references, so the name format is intentionally restricted.

---

### input

| Property | Value |
|---|---|
| Field | `input` |
| Type | `string` |
| Required | yes |
| Default | — |
| Description | Selects how files for this type are parsed. |

**Allowed values**

| Value | Description |
|---|---|
| `json` | JSON files parsed as objects. |
| `yaml` | YAML files parsed as objects. |
| `csv` | CSV-style delimited files parsed as rows of objects; comma by default, tab for TSV via `csv.delimiter`. |
| `hcl` | HCL2 attribute files parsed as one object per file. |
| `toml` | TOML 1.1.0 files parsed as one object per file. |

#### TOML input

Use `input: toml` with include patterns such as `'^data/.*\.toml$'`. datacur8 supports **TOML 1.1.0**, implemented by [go-toml v2.4.3](https://github.com/pelletier/go-toml/tree/v2.4.3). This includes 1.1 features such as omitted seconds, multiline inline tables with trailing commas, and `\e`/`\xHH` string escapes. The `.datacur8` file remains YAML; there are no TOML-specific configuration options.

One document is one record. Quoted/dotted keys, nested and inline tables, arrays of tables, heterogeneous arrays, and basic/literal/multiline strings map to objects, arrays, and scalar values. Nested tables never create additional top-level records. Strings retain their contents without variable expansion, expression evaluation, Unicode normalization, or schema-driven coercion. The existing inline JSON Schema, strict mode, and constraints apply after conversion.

Native temporal values become strings:

| TOML value | Normalized string |
|---|---|
| Offset date-time `1979-05-27 07:32:00.1200-07:00` | `1979-05-27T07:32:00.12-07:00` |
| Local date-time `1979-05-27t07:32:00.1200` | `1979-05-27T07:32:00.12` |
| Local date `1979-05-27` | `1979-05-27` |
| Local time `07:32` | `07:32:00` |

Offset date-times use RFC3339 with their numeric offset retained (zero offset uses `Z`). Local values gain no timezone or date. Seconds are explicit, `T` is uppercase, and fractional seconds have up to nine digits with trailing zeros removed; additional fractional digits are truncated to nanoseconds, never rounded. JSON Schema sees strings, including for native TOML dates/times.

Integers must fit TOML's signed 64-bit range and survive conversion to the common float64 model exactly. All integers from −2^53 through 2^53 are exact; larger magnitudes are accepted only when exactly representable (for example, 9007199254740994 and −9223372036854775808). 9007199254740993 and 9223372036854775807 are rejected instead of rounded. Finite floats are accepted; `nan`, `inf`, and their signed variants are rejected because JSON cannot represent them. TOML has no null literal.

Syntax errors, duplicate keys, and invalid table redefinitions include file, line, and column. Conversion errors include the file and a nested key/index path. Input, conversion, and schema failures return exit **2** and block export. Tidy parses and checks these same conversion limits before writing; see [TOML export](#toml-export) for output restrictions.

#### HCL input

Each HCL2 file represents one object. Top-level attributes become object properties, and attribute values retain their JSON-compatible types: strings, numbers, booleans, null, nested objects, and lists.

```hcl
id      = "widget-1"
enabled = true
metadata = {
  owner = "team-a"
}
tags = ["hardware", "featured"]
```

The parsed values are converted to a JSON-compatible object before the existing JSON Schema and constraint checks run. HCL values determine their types; the schema does not coerce strings to numbers or booleans. Nested objects and lists are supported without CSV's flat-schema restriction.

Expressions must evaluate without external context. Self-contained expressions such as `price = 10 + 2.5` are supported. Evaluated variables, references to other attributes, and function calls produce data-validation errors; unused conditional branches follow HCL's normal evaluation rules. HCL blocks such as `widget "widget-1" { ... }` are unsupported. Use an object-valued attribute (`metadata = { owner = "team-a" }`) for nested data. datacur8 does not run Terraform or load variables, providers, modules, or functions.

HCL quoted strings use template syntax. Escape a literal `${...}` as `$${...}` and a literal `%{...}` as `%%{...}`. Exports escape these sequences automatically.

Use `input: hcl` and include patterns matching the files you want to discover, typically `\.hcl$`. The `.datacur8` configuration itself remains YAML. `tidy` formats HCL while preserving comments and attribute order.

---

### match

Used to identify the files that are processed by this type. A file belongs to a type if it matches at least one `include` pattern and does not match any `exclude` pattern.

| Property | Value |
|---|---|
| Field | `match` |
| Type | `object` |
| Required | yes |
| Default | — |
| Description | File matching rules used to assign repository files to this type. |

{: .important }
Each file must match exactly one type. Matching multiple types is a validation error. Files matching no types are ignored.

Paths are matched as repository-relative paths using forward slashes.

---

#### include

| Property | Value |
|---|---|
| Field | `include` |
| Type | `array` of `string` |
| Required | yes |
| Default | — |
| Description | Regular expression patterns used to include files in this type. |

**Schema details**

- `minItems`: `1`
- Each item must be a string

Patterns are compiled as regular expressions during validation.

**Named capture groups**

Include patterns may contain named capture groups that expose path segments as metadata for constraints:

```yaml
match:
  include:
    - "^configs/(?P<team>[^/]+)/services/(?P<service>[^/]+)\\.ya?ml$"
```

This exposes selectors such as:

- `path.team`
- `path.service`

The built-in path selectors are always available:

| Selector | Description |
|---|---|
| `path.file` | File name without extension |
| `path.ext` | Normalized extension without dot (for example `yaml`, `json`, `csv`, `tsv`, or `hcl`) |
| `path.parent` | Name of the parent folder |

{: .highlight }
Avoid capture group names `file`, `ext`, or `parent` to prevent conflicts with built-in path selectors.

---

#### exclude

| Property | Value |
|---|---|
| Field | `exclude` |
| Type | `array` of `string` |
| Required | no |
| Default | `[]` |
| Description | Regular expression patterns used to exclude files after `include` matching. |

**Schema details**

- Each item must be a string

Patterns are compiled as regular expressions during validation.

---

### schema

| Property | Value |
|---|---|
| Field | `schema` |
| Type | `object` |
| Required | yes |
| Default | — |
| Description | Inline JSON Schema applied to each parsed item for this type. |

**Schema details**

- The config schema requires `schema.type` to be exactly `object`
- The `schema` object is not fully enumerated in the config schema because it is a JSON Schema document

{: .important }
The root JSON Schema type must be `object`. This is required so **datacur8** can validate and export items as objects.

```yaml
schema:
  type: object
  required: ["id", "name"]
  properties:
    id: { type: string }
    name: { type: string }
  additionalProperties: false
```

**datacur8** uses the [google/jsonschema-go](https://github.com/google/jsonschema-go) library for JSON Schema evaluation. The schema is validated as JSON Schema at config load time.

{: .highlight }
For CSV types, the schema must be a flat object (no nested objects or arrays) because CSV rows are converted into flat key-value objects before validation.

---

### constraints

| Property | Value |
|---|---|
| Field | `constraints` |
| Type | `array` of objects |
| Required | no |
| Default | `[]` |
| Description | Additional integrity rules evaluated after JSON Schema validation. |

**Schema details**

- Each item must match exactly one of the supported constraint object shapes (`unique`, `foreign_key`, or `path_equals_attr`)

{: .important }
This page documents the config structure for `constraints`. Constraint behavior, selector semantics, and examples are described in [Constraints](CONSTRAINTS.md).

**Constraint shapes**

| `type` value | Required attributes | Optional attributes |
|---|---|---|
| `unique` | `type`, `key` | `id`, `case_sensitive`, `scope` |
| `foreign_key` | `type`, `key`, `references` | `id` |
| `path_equals_attr` | `type`, `path_selector`, `references` | `id`, `case_sensitive` |

---

#### id

| Property | Value |
|---|---|
| Field | `id` |
| Type | `string` |
| Required | no |
| Default | — |
| Description | Optional stable identifier for a constraint, used in reporting and diagnostics. |

**Schema details**

- `minLength`: `1`
- Available on all constraint types

---

#### type

| Property | Value |
|---|---|
| Field | `type` |
| Type | `string` |
| Required | yes |
| Default | — |
| Description | Constraint discriminator that selects the constraint object shape. |

**Allowed values**

| Value | Meaning |
|---|---|
| `unique` | Uniqueness checks within a type or within an item |
| `foreign_key` | Cross-type referential integrity check |
| `path_equals_attr` | Compare a path-derived value to an item attribute |

{: .highlight }
In the JSON Schema, each concrete constraint shape uses `const` for `type` (for example `type: unique` for the `unique` shape).

---

#### key

| Property | Value |
|---|---|
| Field | `key` |
| Type | `string` |
| Required | yes for `unique` and `foreign_key`; not used by `path_equals_attr` |
| Default | — |
| Description | Selector that extracts the value(s) to evaluate from the owning item. |

**Schema details**

- Underlying selector schema is a non-empty string (`minLength: 1`)
- Semantic validation also checks selector syntax

Examples: `$.id`, `$.team.id`, `$.items[*].id`

---

#### scope

| Property | Value |
|---|---|
| Field | `scope` |
| Type | `string` |
| Required | no (`unique` only) |
| Default | `type` |
| Description | Controls whether `unique` checks run across the type or within each item. |

**Allowed values**

| Value | Description |
|---|---|
| `type` | Enforce uniqueness across all items in the type |
| `item` | Enforce uniqueness within each individual item |

---

#### case_sensitive

| Property | Value |
|---|---|
| Field | `case_sensitive` |
| Type | `boolean` |
| Required | no (`unique` and `path_equals_attr` only) |
| Default | `true` |
| Description | Controls case-sensitive string comparison for supported constraints. |

{: .highlight }
`case_sensitive` is not part of the `foreign_key` constraint schema.

---

#### path_selector

| Property | Value |
|---|---|
| Field | `path_selector` |
| Type | `string` |
| Required | yes (`path_equals_attr` only) |
| Default | — |
| Description | Selects a value derived from the file path (built-in path segment or named capture group). |

**Schema details**

- Pattern: `^path\\.(file|parent|ext|[a-zA-Z_][a-zA-Z0-9_]*)$`

Supported forms:

- `path.file`
- `path.parent`
- `path.ext`
- `path.<capture>` (from a named regex capture group in `match.include`)

{: .important }
If `path_selector` uses `path.<capture>`, semantic validation checks that every `match.include` regex defines that named capture group.

---

#### references

| Property | Value |
|---|---|
| Field | `references` |
| Type | `object` |
| Required | yes for `foreign_key` and `path_equals_attr`; not used by `unique` |
| Default | — |
| Description | Nested object describing the referenced type/key pair or referenced key, depending on the constraint type. |

**Schema details**

`foreign_key` uses:

```yaml
references:
  type: <type-name>
  key: <selector>
```

`path_equals_attr` uses:

```yaml
references:
  key: <selector>
```

---

##### type

| Property | Value |
|---|---|
| Field | `type` |
| Type | `string` |
| Required | yes (`foreign_key` under `references`) |
| Default | — |
| Description | Name of the referenced type in the same `.datacur8` config. |

**Schema details**

- `minLength`: `1`

{: .highlight }
Semantic validation checks that `references.type` matches a defined entry in `types[].name`.

---

##### key

| Property | Value |
|---|---|
| Field | `key` |
| Type | `string` |
| Required | yes (`foreign_key.references` and `path_equals_attr.references`) |
| Default | — |
| Description | Selector used on referenced items (`foreign_key`) or the owning item (`path_equals_attr`). |

**Schema details**

- Underlying selector schema is a non-empty string (`minLength: 1`)
- Semantic validation also checks selector syntax

---

### output

| Property | Value |
|---|---|
| Field | `output` |
| Type | `object` |
| Required | no |
| Default | — |
| Description | Per-type export configuration. If omitted, the type is validated but not exported. |

---

#### path

| Property | Value |
|---|---|
| Field | `path` |
| Type | `string` |
| Required | yes (when `output` is present) |
| Default | — |
| Description | Output file path relative to the repository root. |

**Schema details**

- `minLength`: `1`

{: .highlight }
`output.path` values must be unique across all `types[]` entries.

---

#### format

| Property | Value |
|---|---|
| Field | `format` |
| Type | `string` |
| Required | yes (when `output` is present) |
| Default | — |
| Description | Output encoding format used by `export`. |

**Allowed values**

| Value | Description |
|---|---|
| `json` | Write a JSON array/object output (depending on export shape) |
| `yaml` | Write YAML output |
| `jsonl` | Write newline-delimited JSON objects |
| `csv` | Write schema-defined scalar columns as a header and one row per item |
| `hcl` | Write an HCL attribute named after the type whose value is the exported array |
| `toml` | Write one type-name key containing the array of records |

```yaml
output:
  path: "out/teams.json"
  format: json
```

Export creates parent directories as needed. Input and output formats are independent: any supported input format can be exported as JSON, YAML, JSONL, HCL, CSV, or TOML (subject to each output format's restrictions).

#### CSV delimiters and TSV

Keep `input: csv` and `output.format: csv` for tab-delimited data. Set
`types[].csv.delimiter` for input and tidy, and
`types[].output.csv.delimiter` for export. Each defaults independently to comma,
including an omitted `delimiter` inside `csv: {}`. CSV options on other formats
are configuration errors (exit **1**). There are no `tsv` format aliases.

Copyable YAML (the double-quoted `"\t"` decodes to one actual tab):

```yaml
version: "0.0.0"
types:
  - name: items
    input: csv
    csv:
      delimiter: "\t"
    match:
      include: ['^data/.*\.tsv$']
    schema:
      type: object
      properties:
        id: {type: integer}
        title: {type: string}
      required: [id, title]
    output:
      path: out/items.csv
      format: csv
```

This reads TSV and exports comma CSV. To export TSV instead, add
`csv: {delimiter: "\t"}` inside `output`; omit the input option to convert
comma CSV to TSV. Filename extensions never select syntax: discover `.tsv`
files with an explicit regex. `path.ext` remains `tsv`.

Delimiters may be any single Unicode rune permitted by Go CSV, including comma,
tab, semicolon, and non-ASCII characters such as `界`. Empty strings, multiple
runes, quote, CR, LF, NUL, and the Unicode replacement character are invalid.
Single-quoted YAML `'\t'` means two characters and is invalid.

TSV uses CSV quoting: fields containing the separator, quotes, or newlines are
quoted, and quotes are doubled. It is not raw tab splitting. Whitespace and
empty strings are preserved; empty numeric/boolean cells fail conversion.
Input headers must be nonempty, unique, present in schema properties, and
include every required property. Unknown columns are rejected regardless of
strict mode. Logical data rows retain zero-based `row` diagnostics, even with
multiline cells. Existing schema conversion, constraints, strict mode, record
ordering, and CRLF normalization apply unchanged.

TSV export uses exactly the flat-schema, scalar, precision, empty-dataset,
and unsupported-value rules below, with the selected separator and LF record
endings. Tidy sorts headers and their cells together, preserves row order and
the input delimiter, and retains a single empty string cell as `""`.
Check mode does not write; repeated writes are idempotent. Malformed records
and ambiguous headers fail tidy before rewriting the file.

#### CSV export

Set `output.format: csv` and an output path such as `out/products.csv`. CSV output has no type-name wrapper: one header and one row per item, in discovery/file order and then CSV source row order. Columns come from **all declared `schema.properties`**, sorted alphabetically, including optional properties. A dataset with zero items still writes the header and a final newline.

The supported schema subset is an object with a nonempty `properties` map. Each property must directly declare exactly one scalar `type`: `string`, `boolean`, `integer`, or `number`. Type arrays (even a singleton), boolean property schemas, objects, arrays, omitted types, references (`$ref`, `$dynamicRef`), composition (`allOf`, `anyOf`, `oneOf`, `not`), conditionals (`if`, `then`, `else`), `patternProperties`, and `dependentSchemas` at the root or property level are unsupported for CSV export. These configuration mistakes return exit **1**. Ordinary validation keywords such as `required`, `enum`, string patterns, and numeric bounds continue to work. `additionalProperties` may allow extra data during validation, but actual undeclared keys cannot be exported.

Every item must provide a non-null scalar value for every declared column, including optional properties. Missing cells, explicit nulls, structured values, and actual undeclared keys cannot be silently omitted, flattened, or stringified. Export conversion errors include the zero-based item index and property name and return exit **3**. Source values violating the JSON Schema return exit **2** first; choosing CSV output does not add value-level checks to `validate`.

Strings, including empty strings, Unicode, whitespace, and formula-like strings, are preserved without spreadsheet sanitization. Go's CSV writer handles the configured separator, quotes, and embedded LF/CR characters; record endings are LF, with a final newline. A single empty string cell is explicitly quoted (`""`) so the CSV reader does not skip it as a blank line. Booleans use `true`/`false`. Integers use decimal notation without a decimal suffix and must fit signed 64-bit range; floating-point integer values are checked before conversion. Finite numbers use enough digits to preserve their existing internal value. Precision already lost by a source parser is not recovered (JSON, HCL, and CSV numeric inputs use float64). YAML native integers retain their integer precision on export.

Complete scalar records can be exported and read back using `input: csv` with the same schema. The current CSV reader converts integers to float64, so exact native YAML integer precision beyond float64's exact range is not guaranteed on re-import. Go's CSV reader also normalizes embedded CRLF to LF; the exporter itself preserves the original string bytes. Sparse and nullable exports need a future explicit encoding contract. Serialization completes before opening the destination, so conversion failure leaves an existing file untouched. Separate type outputs are not one transaction.

#### HCL export

HCL exports use the same aggregate shape as JSON and YAML, expressed as `type_name = [{ ... }]`. An empty dataset exports as `type_name = []`. Object keys are sorted for deterministic output. An aggregate export is a single object containing an array, so it is not the same shape as the individual input items.

HCL uses Unicode NFC normalization for strings. Exporting strings from JSON or YAML can therefore change their Unicode encoding while preserving their text. Object keys must already be NFC-normalized for HCL export; unsupported keys produce an export error.

#### TOML export

Set `output.format: toml`. Aggregate output contains one type-name key with an array of records, using inline tables: `records = [{id = 'a', owner = {team = 'platform'}}]`. Zero records emit an explicit `records = []`. Empty objects/arrays and mixed arrays remain present. Items retain discovery order; keys are sorted recursively and quoted as necessary. Unusual and Unicode keys/strings are preserved without Unicode normalization.

An aggregate export is one object containing an array, not multiple individual input records. Reading it with `input: toml` validates that wrapper as one record. JSON strings remain TOML strings even when their text resembles a date or time; native temporal types are never inferred from strings. This includes temporal strings produced by TOML input normalization.

Explicit null at any depth, including inside arrays, is rejected with a nested key/index path and zero-based record index. Missing optional properties remain absent. Non-finite floats, unsupported value types, integers outside signed 64-bit range, and native integers that cannot survive float64 normalization exactly are also rejected. Precision already lost by JSON/HCL/CSV source parsing cannot be recovered. Conversion or write failures return exit **3**; schema-invalid source data returns exit **2** first. Conversion finishes before replacing the destination. Outputs for separate types are independent.

TOML tidy reserializes validated native TOML values with sorted keys and inline tables. It removes all comments, changes ordering and spelling/quoting, and preserves values within the supported nanosecond precision. Dates/times retain native TOML types during tidy, unlike export through the JSON model. Numeric base/separator spelling can change. Formatting is idempotent. Check mode writes nothing and returns **5** for changes; `--write` applies changes. Invalid TOML or unsupported input values return **4** without rewriting that file.
