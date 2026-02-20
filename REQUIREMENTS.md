## datacur8 Technical Design Specification

### Purpose

`datacur8` is a command line that manages a folder of strictly structured data files stored as **JSON**, **YAML**, or **CSV** files. It provides:

- **Validation** against **full-featured JSON Schema** per file type
- **Cross-file constraints** (unique keys, foreign keys, path-to-attribute rules)
- **Export** to compiled outputs (YAML or JSON arrays, or JSONL)
- **Tidy** to normalize formatting deterministically for stable diffs

The behavior is driven entirely by a single configuration file, `.datacur8`, that defines the types and behaviors.

---

## Goals

- One command validates everything in a folder, including sub-folders as specified, deterministically with clear errors.
- Constraints are a first-class feature, beyond JSON Schema, allowing for robust data integrity.
- A file matches **exactly one** configured type allowing for deterministic parsing and validation.
- Strong local workflow: same behavior locally and in CI for a repo.
- Minimal CLI switches. Most behavior is config-driven; simple commands are easily remembered.
- Machine-parsable outputs available for automated tests.
- Data management using files that can be managed in git, but with input validation that would normally require a website and database.

---

## Terminology

- **Type**: A configured category of data files (example: `product`, `location`, `movie`).
- **Item**: One parsed file (JSON/YAML) or one parsed row (CSV), depending on type.
- **Primary key**: Attribute(s) designated by constraints for uniqueness and referencing.
- **Constraint**: Rule evaluated across all items of a type, optionally inspecting nested structures within each item. Constraints may reference other types. Above and beyond JSON Schema validation.
- **Path segments**: Named portions of file paths captured via regex groups and used as inputs for validation.

---

## CLI Surface

This command line application follows a simple subcommand structure.

### Commands

1. `datacur8 validate [--config-only] [--format]`
2. `datacur8 export`
3. `datacur8 tidy [--dry-run]`
4. `datacur8 version`

### Invocation rules

- The CLI **must be run from the repo root directory that contains `.datacur8`**.
- If run elsewhere, it errors with a single actionable message:
  - “`.datacur8` not found in current directory. Run from repo root.”

### Exit codes

- `0` success
- `1` configuration invalid
- `2` data invalid (schema and or constraints)
- `3` export failure (for example, write errors)
- `4` tidy failure (parse or write errors)

---

## Repository Configuration

### Root config file

- Exactly one file named **`.datacur8`** in the working directory.
- This file is **YAML** and is validated against the included JSON Schema (later in this spec).
- No additional dot files are used in the initial design including in subfolders.
- The presence of a .datacur8 file in a subdirectory will return an error that it is not expected or supported.

Rationale: a single config source avoids conflicts and ambiguity. Types can be flexibly arranged in nested folders however the user sees fit.

---

## `.datacur8` Configuration Model

### High level structure

- `version`: the minimum version of datacur8 required to process the config, expressed as `major.minor.patch`. When a development version of datacur8 is running this field is ignored and a warning log message is written saying it was not checked. Semantic version comparison is performed ensuring that the major version matches and the CLI version is greater than or equal to the configured version.
- `types`: list of type definitions
  - `schema` the inline JSON Schema for the type
  - `constraints` list of constraints that apply to this type
  - `output` (optional): the configuration of where and how to export this type

(not all attributes are outlined above)

---

## Type Definition

Each type defines:

1. **Input format**: one of `json`, `yaml`, `csv`
2. **Matching rules**: include and exclude path patterns and extension rules
3. **Schema**: JSON Schema inline
4. **Constraints**: list of constraints that apply to this type
5. **Output**: where and how this type is exported

### File matching

A file belongs to a type if:

- It matches the type’s `include` rules
- It does not match the type’s `exclude` rules

Additional rule:

- A file must match **exactly one** type. If it matches multiple types, validation fails.
- If it matches no types, it is ignored.

### Capturing path segments

Type include patterns may contain **named capture groups**. Captures are exposed as immutable metadata for constraints.

Example include regex:

- `^configs/(?P<team>[^/]+)/services/(?P<service>[^/]+)\.ya?ml$`

Exposes:
- `path.team`
- `path.service`
- `path.file` (always available: file name without extension)
- `path.ext` (normalized extension without dot: `yaml`, `json`, or `csv`)
- `path.parent` (name of the parent folder containing the file)

This is the supported mechanism for “folder name matches attribute” and “file name matches attribute” without adding separate bespoke features.

---

## JSON Schema Validation

### Library

The CLI uses `google/jsonschema-go` for JSON Schema evaluation, allowing schemas to use the range of features supported by that library.

### Supported schemas

- JSON Schema is specified inline in the YAML file
- Each type has exactly one schema applied to each item.
- Root schema type must be `object` for all types. Non-object roots (`boolean`, `string`, `array`, etc.) are not supported by datacur8.
- The JSON Schema is validated to be a valid JSON Schema.

### Strict mode overlay

Problem addressed: YAML commonly accumulates extra keys, and “no additional properties” is not always enforced deeply.

`strict_mode` behavior:

- If `strict_mode: ENABLED`, datacur8 enforces a **deep no-additional-properties policy** by applying an overlay during validation:
  - For every object schema that does not explicitly set `additionalProperties`, treat it as `false`.
  - If a schema explicitly sets `additionalProperties: true`, that explicit choice stands.
  - If a schema explicitly sets `additionalProperties: false`, it stands.
- If `strict_mode: FORCE`, datacur8 enforces `additionalProperties: false` on all object schemas, even if they explicitly set it to true. This is a more aggressive mode that overrides schema authors’ choices.

This overlay is applied at validation time without rewriting the schema files on disk.

Default: `strict_mode: DISABLED`.

---

## Constraints

Constraints are defined per type. A constraint may reference other types, but it is always attached to a single owning type.

### Constraint evaluation phases

Validation runs in phases:

1. **Config validation**
   - Validate `.datacur8` against its schema.
   - Validate referential integrity of config itself:
     - Unique type names
     - Output paths unique across types that define `output`
     - Referenced types exist
     - For `path.<capture_name>` selectors, every include regex for the type must define that named capture group
     - Regex compile
     - Schemas exist and are readable
     - Include patterns are not trivially impossible
2. **File discovery and parsing**
   - Walk repo (configurable root, default `.`)
   - Collect candidate files by extension and include/exclude rules
   - Determine each file’s single matching type (or error)
3. **Schema validation pass**
   - Parse each file to a canonical internal representation
   - Validate each item against its JSON Schema
4. **Constraint collection pass**
   - Build in-memory indexes required by constraints:
     - Uniqueness sets
     - Foreign key lookup maps
     - Path capture tables
5. **Constraint validation pass**
   - Evaluate constraints deterministically
   - Emit errors with stable ordering

Memory model: default is in-memory indexing. The spec allows later optimizations (spill-to-disk indexes) without changing configuration semantics.

### Constraint reference model

Constraints refer to attributes using JSONPath-like selectors with a constrained subset for predictability.

Supported selector syntax:

- Root: `$`
- Object field: `$.field`
- Nested: `$.a.b.c`
- Array items:
  - `$.items[*].id` to project values
  - No general filtering in v1

If a selector yields multiple values for a single item:
- `foreign_key` and `path_equals_attr` are invalid because they require a single scalar; the validation will fail with an error indicating the selector is invalid for that constraint.
- `unique` treats the projected values as an in-item collection and enforces uniqueness within that item

---

## Constraint Catalog

### Summary table

| Constraint | Purpose | Applies to | Key fields |
|---|---|---|---|
| `unique` | Ensure uniqueness across items (scalar key) or within each item collection (multi-value key) | All input types | `key`, `case_sensitive` |
| `foreign_key` | Ensure referenced value exists in another type | All input types | `key`, `references.type`, `references.key` |
| `path_equals_attr` | Ensure a path-derived value equals an attribute value | File-based types | `path_selector`, `references.key`, `case_sensitive` |

Advanced features deferred to a future phase:
- Composite keys for uniqueness and foreign keys
- Cardinality constraints

---

## Constraint Details

### 1) `unique`

**Purpose**  
Across all items of a type, ensure a key is unique.

**Config fields**
- `type: unique`
- `key`: selector on the item, example `$.id`
- `case_sensitive`: default true, for string keys
- `scope`: "item" (default) or "type" - whether to enforce uniqueness within each item (for multi-value keys) or across all items of the type

**Behavior**
- If `key` resolves to one scalar per item, enforce uniqueness across all items of the type.
- If `key` resolves to multiple values for an item, enforce uniqueness within that item only.
- Track duplicates and report all collisions.

---

### 2) `foreign_key`

**Purpose**  
Ensure that values in one type exist as values in another type.

**Config fields**
- `type: foreign_key`
- `key`: selector on owning item, must resolve to a single scalar
- `references.type`: referenced type name
- `references.key`: selector on referenced type items, must resolve to a single scalar

**Behavior**
- Build lookup index of `references.type` keys.
- For each owning item key value, check presence in the referenced index.

---

### 3) `path_equals_attr`

**Purpose**  
Enforce that a path-derived value equals an attribute value.

**Config fields**
- `type: path_equals_attr`
- `path_selector`: one of:
  - `path.file` (file name without extension)
  - `path.parent` (parent folder name)
  - `path.ext` (normalized extension without dot: `yaml`, `json`, or `csv`)
  - `path.<capture_name>` (named capture from `match.include`)
- `references.key`: selector on referenced type items, must resolve to a single scalar (ex: example `$.teamId`); only valid within the current type so reference.type cannot be set.
- `case_sensitive`: default true

**Behavior**
- Reads the value resolved by `path_selector` from match metadata.
- Compares that value to `attr_selector`.

---

## Inputs

### JSON and YAML

- Parsed into a canonical representation suitable for schema validation and selector evaluation.
- YAML is parsed as YAML but validated as JSON-like data.

### CSV

CSV requires:
- Header row presence
- Unknown header names are invalid (must exist in root `schema.properties`)
- All required root properties in `schema.required` must be present in the header
- CSV input schema for a csv type must be a flat object schema (no arrays or nested objects)
- Root schema type must be `object` (this is a datacur8 limitation)

CSV validation flow:
1. Validate header row against root schema object properties and required list.
2. Convert each row into one JSON object using header names.
3. Convert values using root schema property types (especially `boolean`, `number`, and `integer`).
4. If a value cannot be converted to the schema-guided type, emit a validation error.
5. Validate each row object against the type schema.

---

## Export

### Purpose

Compile validated configs into deterministic outputs per type definition, writing:

- One output file per type that defines `output`
- Output encoding per type:
  - JSON array with root object array name of the "type" of the object. A single large file.
  - YAML array with root object array name of the "type" of the object. A single large file.
  - JSONL (one JSON object per line; minified)

### Export semantics

- Export reads the same discovered, parsed, validated items as `validate`.
- Export fails if validation fails.
- If no types define an `output`, export is a no-op, logs that no outputs are configured, and exits successfully.
- Ordering:
  - Stable, deterministic ordering by:
    1) type order in config
    2) file path
    3) within-file order for arrays if applicable
- Each type with an `output` configures:
  - `output.format: json | yaml | jsonl`
  - `output.path: output/<name>.json`

### Exported shape

For JSON and YAML types:
- Default export shape is an array of items.

For CSV types:
- Export uses the item representation produced by CSV parsing:
  - Exports an array of row objects

---

## Tidy

### Purpose

Rewrite files to conform to opinionated formatting while preserving data.

- JSON: pretty printed, stable key ordering
- YAML: stable formatting, stable key ordering, comments removed
- CSV: stable column order, stable row ordering where safely derivable

### Tidy semantics

- Tidy must not change parsed data values.
- If a file fails schema validation, tidy may still operate if parsing succeeds, but the default behavior is:
  - Parse only and re-emit canonical form
  - Do not attempt to “fix” invalid content
- Sorting rules:
  - For objects: key sort lexicographically
  - For arrays: do not reorder unless configured with `tidy.sort_arrays_by` per type
- YAML comments are lost by design.

---

## `validate --config-only`

- Validates `.datacur8` against config schema
- Validates config referential integrity and regex compilation
- Does not scan or parse data files

If `types` is empty, `validate` and `validate --config-only` are no-ops (except for validating the config against the JSON Schema), log that no types are configured, and exit successfully.

---

## Configuration JSON Schema for `.datacur8`

Below is the initial JSON Schema (draft-agnostic usage is acceptable as long as the chosen evaluator supports it). This schema is intentionally strict to keep config problems obvious.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://datacur8.dev/schemas/config.schema.json",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "types"],
  "properties": {
    "version": {
      "type": "string",
      "description": "Minimum datacur8 version required by this config.",
      "pattern": "^[0-9]+\\.[0-9]+\\.[0-9]+$"
    },
    "strict_mode": {
      "type": "string",
      "enum": ["DISABLED", "ENABLED", "FORCE"],
      "default": "DISABLED"
    },
    "reporting": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "mode": { "type": "string", "enum": ["text", "json", "yaml"], "default": "text" }
      }
    },
    "types": {
      "type": "array",
      "minItems": 0,
      "items": { "$ref": "#/$defs/typeDef" }
    },
    "tidy": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "enabled": { "type": "boolean", "default": true }
      }
    }
  },
  "$defs": {
    "selector": { "type": "string", "minLength": 1 },
    "keyRef": { "$ref": "#/$defs/selector" },
    "pathSelector": {
      "type": "string",
      "pattern": "^path\\.(file|parent|ext|[a-zA-Z_][a-zA-Z0-9_]*)$"
    },
    "typeDef": {
      "type": "object",
      "additionalProperties": false,
      "required": ["name", "input", "match", "schema"],
      "properties": {
        "name": { "type": "string", "minLength": 1, "pattern": "^[a-zA-Z][a-zA-Z0-9_]*$" },
        "input": { "type": "string", "enum": ["json", "yaml", "csv"] },
        "match": { "$ref": "#/$defs/matchDef" },
        "schema": {
          "type": "object",
          "description": "Inline JSON Schema applied to each parsed item. Root type must be object.",
          "required": ["type"],
          "properties": {
            "type": { "const": "object" }
          }
        },
        "constraints": {
          "type": "array",
          "items": { "$ref": "#/$defs/constraintDef" },
          "default": []
        },
        "output": { "$ref": "#/$defs/outputDef" },
        "csv": { "$ref": "#/$defs/csvDef" },
        "tidy": { "$ref": "#/$defs/typeTidyDef" }
      },
      "allOf": [
        {
          "if": { "properties": { "input": { "const": "csv" } } },
          "then": { "required": ["csv"] }
        }
      ]
    },
    "matchDef": {
      "type": "object",
      "additionalProperties": false,
      "required": ["include"],
      "properties": {
        "include": {
          "type": "array",
          "minItems": 1,
          "items": { "type": "string", "description": "Regex applied to repo-relative path using forward slashes." }
        },
        "exclude": {
          "type": "array",
          "items": { "type": "string" },
          "default": []
        }
      }
    },
    "csvDef": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "delimiter": { "type": "string", "minLength": 1, "maxLength": 1, "default": "," }
      }
    },
    "typeTidyDef": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "sort_arrays_by": {
          "type": "array",
          "items": { "type": "string" },
          "description": "Selectors used to sort top-level arrays for stable diffs."
        }
      }
    },
    "outputDef": {
      "type": "object",
      "additionalProperties": false,
      "required": ["path", "format"],
      "properties": {
        "path": { "type": "string", "minLength": 1 },
        "format": { "type": "string", "enum": ["json", "yaml", "jsonl"] }
      }
    },
    "constraintDef": {
      "description": "Simple, minimal constraint model.",
      "oneOf": [
        { "$ref": "#/$defs/uniqueConstraintDef" },
        { "$ref": "#/$defs/foreignKeyConstraintDef" },
        { "$ref": "#/$defs/pathEqualsAttrConstraintDef" }
      ]
    },
    "uniqueConstraintDef": {
      "type": "object",
      "additionalProperties": false,
      "required": ["type", "key"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "type": { "const": "unique" },
        "key": { "$ref": "#/$defs/keyRef" },
        "case_sensitive": { "type": "boolean", "default": true },
        "scope": { "type": "string", "enum": ["item", "type"], "default": "type" }
      }
    },
    "foreignKeyRefDef": {
      "type": "object",
      "additionalProperties": false,
      "required": ["type", "key"],
      "properties": {
        "type": { "type": "string", "minLength": 1 },
        "key": { "$ref": "#/$defs/keyRef" }
      }
    },
    "foreignKeyConstraintDef": {
      "type": "object",
      "additionalProperties": false,
      "required": ["type", "key", "references"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "type": { "const": "foreign_key" },
        "key": { "$ref": "#/$defs/keyRef" },
        "references": { "$ref": "#/$defs/foreignKeyRefDef" }
      }
    },
    "pathEqualsAttrConstraintDef": {
      "type": "object",
      "additionalProperties": false,
      "required": ["type", "path_selector", "references"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "type": { "const": "path_equals_attr" },
        "path_selector": { "$ref": "#/$defs/pathSelector" },
        "references": {
          "type": "object",
          "additionalProperties": false,
          "required": ["key"],
          "properties": {
            "key": { "$ref": "#/$defs/keyRef" }
          }
        },
        "case_sensitive": { "type": "boolean", "default": true }
      }
    }
  }
}
```

---

## Example `.datacur8` Configuration

```yaml
version: "1.0.0"
strict_mode: DISABLED

reporting:
  mode: text

types:
  - name: team
    input: yaml
    match:
      include:
        - "^configs/teams/(?P<team>[^/]+)\\.ya?ml$"
    schema:
      type: object
      required: ["id", "name"]
      properties:
        id: { type: string }
        name: { type: string }
      additionalProperties: false
    constraints:
      - id: team_id_unique
        type: unique
        key: "$.id"
      - id: team_path_matches_id
        type: path_equals_attr
        path_selector: "path.team"
        references:
          key: "$.id"
    output:
      path: "out/teams.json"
      format: json

  - name: service
    input: yaml
    match:
      include:
        - "^configs/teams/(?P<team>[^/]+)/services/(?P<service>[^/]+)\\.ya?ml$"
    schema:
      type: object
      required: ["id", "name", "teamId"]
      properties:
        id: { type: string }
        name: { type: string }
        teamId: { type: string }
      additionalProperties: false
    constraints:
      - id: service_id_unique
        type: unique
        key: "$.id"
      - id: service_team_fk
        type: foreign_key
        key: "$.teamId"
        references:
          type: team
          key: "$.id"
      - id: service_path_team_matches_teamId
        type: path_equals_attr
        path_selector: "path.team"
        references:
          key: "$.teamId"
      - id: service_file_matches_id
        type: path_equals_attr
        path_selector: "path.file"
        references:
          key: "$.id"
    output:
      path: "out/services.jsonl"
      format: jsonl
```

---

## GitHub Action Integration

The GitHub Action is configuration-based by calling the CLI:

- Checkout
- Install datacur8 binary (release artifact)
- Run `datacur8 validate` (and optionally `datacur8 export`)

The Action itself contains no validation logic beyond executing the CLI, so local and CI results match.

---

## Design Decisions Locked for v1

- Exactly one `.datacur8` at repo root, required for all commands.
- Each file matches exactly one type.
- Types are constrained to exactly one input format: json, yaml, or csv.
- No global constraints. Constraints live under a type.
- Types may define a per-type output. Export does not accept runtime output arguments.
- `validate --config-only` exists; no separate `validate-config` subcommand.

---

## Open Design Choices to Resolve During Implementation

These affect behavior but do not require user-facing config changes:

- Selector engine implementation details and error localization quality for YAML and CSV
- Stable ordering rules for error emission when multiple constraints fail
- Performance strategy for large repos (in-memory indexing first, optimizations later)

---
