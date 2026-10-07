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
| `csv` | CSV files parsed as rows of objects (comma-delimited; no CSV format configuration). |
| `hcl` | HCL2 attribute files parsed as one object per file. |

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
| `path.ext` | Normalized extension without dot (for example `yaml`, `json`, `csv`, or `hcl`) |
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

```yaml
output:
  path: "out/teams.json"
  format: json
```

Export creates parent directories as needed. Input and output formats are independent: any supported input format can be exported as JSON, YAML, JSONL, HCL, or CSV (subject to CSV's schema and value restrictions).

#### CSV export

Set `output.format: csv` and an output path such as `out/products.csv`. CSV output has no type-name wrapper: one header and one row per item, in discovery/file order and then CSV source row order. Columns come from **all declared `schema.properties`**, sorted alphabetically, including optional properties. A dataset with zero items still writes the header and a final newline.

The supported schema subset is an object with a nonempty `properties` map. Each property must directly declare exactly one scalar `type`: `string`, `boolean`, `integer`, or `number`. Type arrays (even a singleton), boolean property schemas, objects, arrays, omitted types, references (`$ref`, `$dynamicRef`), composition (`allOf`, `anyOf`, `oneOf`, `not`), conditionals (`if`, `then`, `else`), `patternProperties`, and `dependentSchemas` at the root or property level are unsupported for CSV export. These configuration mistakes return exit **1**. Ordinary validation keywords such as `required`, `enum`, string patterns, and numeric bounds continue to work. `additionalProperties` may allow extra data during validation, but actual undeclared keys cannot be exported.

Every item must provide a non-null scalar value for every declared column, including optional properties. Missing cells, explicit nulls, structured values, and actual undeclared keys cannot be silently omitted, flattened, or stringified. Export conversion errors include the zero-based item index and property name and return exit **3**. Source values violating the JSON Schema return exit **2** first; choosing CSV output does not add value-level checks to `validate`.

Strings, including empty strings, Unicode, whitespace, and formula-like strings, are preserved without spreadsheet sanitization. Go's CSV writer handles commas, quotes, and embedded LF/CR characters; record endings are LF, with a final newline. A single empty string cell is explicitly quoted (`""`) so the CSV reader does not skip it as a blank line. Booleans use `true`/`false`. Integers use decimal notation without a decimal suffix and must fit signed 64-bit range; floating-point integer values are checked before conversion. Finite numbers use enough digits to preserve their existing internal value. Precision already lost by a source parser is not recovered (JSON, HCL, and CSV numeric inputs use float64). YAML native integers retain their integer precision on export.

Complete scalar records can be exported and read back using `input: csv` with the same schema. The current CSV reader converts integers to float64, so exact native YAML integer precision beyond float64's exact range is not guaranteed on re-import. Go's CSV reader also normalizes embedded CRLF to LF; the exporter itself preserves the original string bytes. Sparse and nullable exports need a future explicit encoding contract. Serialization completes before opening the destination, so conversion failure leaves an existing file untouched. Separate type outputs are not one transaction.

#### HCL export

HCL exports use the same aggregate shape as JSON and YAML, expressed as `type_name = [{ ... }]`. An empty dataset exports as `type_name = []`. Object keys are sorted for deterministic output. An aggregate export is a single object containing an array, so it is not the same shape as the individual input items.

HCL uses Unicode NFC normalization for strings. Exporting strings from JSON or YAML can therefore change their Unicode encoding while preserving their text. Object keys must already be NFC-normalized for HCL export; unsupported keys produce an export error.
