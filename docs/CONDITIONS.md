---
layout: default
title: Conditions
parent: Internals
nav_order: 1
permalink: /conditions
---

# Conditions
{: .no_toc }

Reference for validation conditions, error messages, and output formats returned by datacur8.

| Area | Exit | Condition / Format | Details |
| --- | --- | --- | --- |
| Overview | N/A | Structured error message | Includes level, type, file (when applicable), and message. Use `--format json` for machine-parsable output. |
| Overview | N/A | Validation phase order | Phases run in order: config -> discovery -> data validation -> export -> tidy. The first reported error indicates the earliest failure point. |
| Overview | N/A | CLI exit code reference | See [Command](/command#exit-codes) for command-level exit-code behavior. |
| Configuration | `1` | Missing config file | Message starts with: .datacur8 not found in current directory. Run from repo root. Run the CLI from the repository root that contains `.datacur8`. |
| Configuration | `1` | Config schema validation failure | Message starts with: configuration does not match schema: ... The `.datacur8` file fails embedded JSON Schema validation (for example missing required fields, unknown properties, invalid types/enums). |
| Configuration | `1` | Invalid version format | Message pattern: version \"X\" is not valid semver (expected major.minor.patch). `version` must be `major.minor.patch` (for example `1.0.0`). |
| Configuration | `1` | Major version mismatch | Message pattern: major version mismatch: config requires X.x.x but CLI is Y.Z.W. Config major version must match the CLI major version exactly. |
| Configuration | `1` | CLI version too old | Message pattern: CLI version X.Y.Z is older than config version A.B.C. The running CLI is older than the minimum version required by the config. |
| Configuration | `1` | Invalid `strict_mode` | Message pattern: strict_mode \"X\" is invalid; must be DISABLED, ENABLED, or FORCE. |
| Configuration | `1` | Duplicate type name | Message pattern: types[N](name): duplicate type name \"name\". Each type name must be unique. |
| Configuration | `1` | Invalid type name | Message pattern: types[N](name): type name must match ^[a-zA-Z][a-zA-Z0-9_]*$. Type names must start with a letter and use only letters, digits, and underscores. |
| Configuration | `1` | Invalid input format | Message pattern: types[N](name): input \"X\" must be json, yaml, or csv. |
| Configuration | `1` | Empty include patterns | Message pattern: types[N](name): match.include must have at least 1 pattern. Every type needs at least one `match.include` pattern. |
| Configuration | `1` | Invalid regex pattern | Message pattern: types[N](name): match.include[M] invalid regex: ... or types[N](name): match.exclude[M] invalid regex: ... A `match.include` or `match.exclude` regex failed to compile. |
| Configuration | `1` | Missing schema | Message pattern: types[N](name): schema is required. Every type must define a schema. |
| Configuration | `1` | Invalid schema root type | Message pattern: types[N](name): schema.type must be \"object\". All datacur8 schemas must have root `type: object`. |
| Configuration | `1` | Missing CSV configuration | Message pattern: types[N](name): csv config is required when input is csv. Types using `input: csv` must include a `csv` block. |
| Configuration | `1` | Invalid CSV delimiter | Message pattern: types[N](name): csv.delimiter must be exactly 1 character. |
| Configuration | `1` | Output path conflict | Message pattern: types[N](name): output.path \"path\" conflicts with type \"other\". Two types cannot write to the same output path. |
| Configuration | `1` | Invalid output format | Message pattern: types[N](name): output.format \"X\" must be json, yaml, or jsonl. |
| Configuration | `1` | Invalid constraint selector | Message pattern: types[N](name).constraints[M]: key \"X\" is not a valid selector: ... Valid selectors include `$`, `$.field`, `$.a.b.c`, and `$.items[*].id`. |
| Configuration | `1` | Unknown constraint type | Message pattern: types[N](name).constraints[M]: unknown constraint type \"X\". Supported types: `unique`, `foreign_key`, `path_equals_attr`. |
| Configuration | `1` | Missing references for `foreign_key` | Message pattern: types[N](name).constraints[M]: references is required for foreign_key. |
| Configuration | `1` | `foreign_key` references unknown type | Message pattern: types[N](name).constraints[M]: references.type \"X\" does not match any defined type. Referenced type must exist in `types`. |
| Configuration | `1` | Invalid constraint scope | Message pattern: types[N](name).constraints[M]: scope \"X\" must be item or type. |
| Configuration | `1` | Missing path capture group | Message pattern mentions path_selector capture \"X\" missing named group `(?P<X>...)` in `match.include[P]`. Required when using `path.<capture>` in `path_equals_attr`. |
| Discovery | `1` | File matches multiple types | Message pattern: file \"path\" matches multiple types: typeA, typeB. Each file must match exactly one type; adjust include/exclude patterns to remove ambiguity. |
| Discovery | `1` | Subdirectory config file found | Message pattern: found .datacur8 in subdirectory \"dir\"; only root .datacur8 is allowed. datacur8 supports a single `.datacur8` at the repository root only. |
| Data Validation | `2` | JSON/YAML parse failure | Message starts with: parsing JSON: ... or parsing YAML: ... File content is not valid JSON or YAML. |
| Data Validation | `2` | CSV parse failure | Message starts with: parsing CSV: ... File content is not valid CSV. |
| Data Validation | `2` | CSV header not in schema | Message pattern: CSV header \"X\" not found in schema properties. Every CSV header must exist in schema `properties`. |
| Data Validation | `2` | CSV missing required property | Message pattern: required property \"X\" missing from CSV headers. Every property in `schema.required` must appear in the CSV header row. |
| Data Validation | `2` | CSV type conversion failure | Message patterns include row N, column \"X\": invalid boolean/number/integer value: \"Y\". A CSV cell could not be converted to the schema-specified scalar type. |
| Data Validation | `2` | Schema validation failure | Message starts with: validating root: ... JSON Schema validation failed (for example type mismatch, missing required field, or additional property under strict mode). |
| Data Validation | `2` | Unique constraint violation | Message pattern: [unique] duplicate value \"X\" for key $.field. Two or more items in the same type share the same value for a unique key. |
| Data Validation | `2` | Foreign key constraint violation | Message pattern: [foreign_key] foreign key \"X\" not found in refType.$.refKey. The owning item references a value that does not exist in the referenced type key set. |
| Data Validation | `2` | Path equals attribute violation | Message pattern: [path_equals_attr] path value \"X\" does not match attribute value \"Y\". A path-derived value (file name, parent folder, or capture group) does not match the item attribute. |
| Export | `3` | Directory creation failure | Message starts with: creating output directory for type: ... datacur8 failed to create the output directory before writing export output. |
| Export | `3` | Write failure | Message starts with: writing output file for type: ... datacur8 failed while writing the output file. |
| Export | `3` | Marshaling failure | Message starts with: marshaling format output for type: ... datacur8 failed to encode export data in the requested output format. |
| Tidy | `4` | Parse or rewrite failure | Message varies. `tidy` errors occur when a file cannot be parsed or rewritten during formatting normalization. |
| Output Format | N/A | Text (default) | Output shape: error: [type_name] file/path.json message. Written to `stderr`. |
| Output Format | N/A | JSON (`--format json`) | Output shape: array of error objects with level, type, file, and message. CSV errors also include a `row` field. Written to `stdout`. |
| Output Format | N/A | YAML (`--format yaml`) | Output shape: YAML list of error objects with level, type, file, and message. Written to `stdout`. |
| Constraint Reference | N/A | `path_equals_attr` usage | Use when troubleshooting path-to-attribute validation failures (for example: path value X does not match attribute value Y). |
| Constraint Reference | N/A | `path_equals_attr.type` | Required string. Must be `path_equals_attr`. |
| Constraint Reference | N/A | `path_equals_attr.path_selector` | Required string. Path value source: `path.file`, `path.parent`, `path.ext`, or `path.<capture>`. |
| Constraint Reference | N/A | `path_equals_attr.references.key` | Required string. Selector on the same item to compare against. |
| Constraint Reference | N/A | `path_equals_attr.case_sensitive` | Optional boolean. Default is `true`. Controls string comparison mode. |
| Constraint Reference | N/A | `path_equals_attr.id` | Optional string identifier. |
| Constraint Reference | N/A | `path_equals_attr` example | Example shape: `match.include` uses a named capture (for example `team`), then the constraint sets `path_selector` to `path.team` and compares against `references.key` such as `$.teamId`. |
