---
layout: default
title: Conditions
nav_order: 6
permalink: /conditions
---

# Conditions
{: .no_toc }

Reference for all validation conditions, error messages, and exit codes returned by datacur8.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Overview

When datacur8 encounters an error, it reports it with a structured message that includes the level, type, file path (when applicable), and a descriptive message. Use `--format json` to get machine-parsable error output.

Validation proceeds through multiple phases. Each phase must succeed before the next runs. This means the first reported error type indicates the earliest failure point.

## Exit Codes

| Code | Meaning | Phase |
|------|---------|-------|
| `0` | Success | — |
| `1` | Configuration invalid | Config validation or file discovery |
| `2` | Data invalid | Schema validation or constraint evaluation |
| `3` | Export failure | Export output |
| `4` | Tidy failure | Tidy operation |

## Configuration Errors (Exit Code 1)

These errors occur during config loading and validation.

### Missing config file

```
.datacur8 not found in current directory. Run from repo root.
```

The CLI must be run from the directory containing the `.datacur8` config file.

### Config schema validation failure

```
configuration does not match schema: ...
```

The `.datacur8` file does not conform to the expected config schema. Common causes:
- Missing required fields (`version`, `types`)
- Unknown top-level properties
- Invalid field types or enum values

### Invalid version format

```
version "X" is not valid semver (expected major.minor.patch)
```

The `version` field must be in `major.minor.patch` format (e.g., `1.0.0`).

### Major version mismatch

```
major version mismatch: config requires X.x.x but CLI is Y.Z.W
```

The config requires a different major version than the running CLI. Major versions must match exactly.

### CLI version too old

```
CLI version X.Y.Z is older than config version A.B.C
```

The running CLI version is older than the minimum version specified in the config.

### Invalid strict_mode

```
strict_mode "X" is invalid; must be DISABLED, ENABLED, or FORCE
```

### Duplicate type name

```
types[N](name): duplicate type name "name"
```

Each type must have a unique name across all type definitions.

### Invalid type name

```
types[N](name): type name must match ^[a-zA-Z][a-zA-Z0-9_]*$
```

Type names must start with a letter and contain only letters, digits, and underscores.

### Invalid input format

```
types[N](name): input "X" must be json, yaml, or csv
```

### Empty include patterns

```
types[N](name): match.include must have at least 1 pattern
```

Every type must have at least one include pattern.

### Invalid regex pattern

```
types[N](name): match.include[M] invalid regex: ...
types[N](name): match.exclude[M] invalid regex: ...
```

A regex pattern in match.include or match.exclude failed to compile.

### Missing schema

```
types[N](name): schema is required
```

### Invalid schema root type

```
types[N](name): schema.type must be "object"
```

datacur8 requires all schemas to have root type `object`.

### Missing CSV configuration

```
types[N](name): csv config is required when input is csv
```

Types with `input: csv` must include a `csv` configuration block.

### Invalid CSV delimiter

```
types[N](name): csv.delimiter must be exactly 1 character
```

### Output path conflict

```
types[N](name): output.path "path" conflicts with type "other"
```

Two types cannot write to the same output path.

### Invalid output format

```
types[N](name): output.format "X" must be json, yaml, or jsonl
```

### Invalid constraint selector

```
types[N](name).constraints[M]: key "X" is not a valid selector: ...
```

The selector syntax is invalid. Valid selectors use `$`, `$.field`, `$.a.b.c`, or `$.items[*].id`.

### Unknown constraint type

```
types[N](name).constraints[M]: unknown constraint type "X"
```

Must be one of: `unique`, `foreign_key`, `path_equals_attr`.

### Missing references for foreign_key

```
types[N](name).constraints[M]: references is required for foreign_key
```

### Foreign key references unknown type

```
types[N](name).constraints[M]: references.type "X" does not match any defined type
```

### Invalid constraint scope

```
types[N](name).constraints[M]: scope "X" must be item or type
```

### Missing path capture group

```
types[N](name).constraints[M]: path_selector uses capture "X" but match.include[P] does not define named group (?P<X>...)
```

When using `path.<capture>` in a `path_equals_attr` constraint, every include pattern must define the named capture group.

## Discovery Errors (Exit Code 1)

These errors occur during file discovery.

### File matches multiple types

```
file "path" matches multiple types: typeA, typeB
```

Each file must match exactly one type. Adjust include/exclude patterns to resolve ambiguity.

### Subdirectory config file found

```
found .datacur8 in subdirectory "dir"; only root .datacur8 is allowed
```

datacur8 only supports a single `.datacur8` file at the repository root. Subdirectory config files are not allowed.

## Data Validation Errors (Exit Code 2)

These errors occur during file parsing, schema validation, or constraint evaluation.

### JSON/YAML parse failure

```
parsing JSON: ...
parsing YAML: ...
```

The file could not be parsed as valid JSON or YAML.

### CSV parse failure

```
parsing CSV: ...
```

The file could not be parsed as valid CSV.

### CSV header not in schema

```
CSV header "X" not found in schema properties
```

Every CSV column header must correspond to a property in the schema.

### CSV missing required property

```
required property "X" missing from CSV headers
```

Properties listed in `schema.required` must appear as CSV headers.

### CSV type conversion failure

```
row N, column "X": invalid boolean value: "Y"
row N, column "X": invalid number value: "Y"
row N, column "X": invalid integer value: "Y"
```

A CSV cell value could not be converted to the schema-specified type.

### Schema validation failure

```
validating root: ...
```

An item did not pass JSON Schema validation. Common causes include type mismatches, missing required fields, and unexpected additional properties (especially with strict mode).

### Unique constraint violation

```
[unique] duplicate value "X" for key $.field
```

Two or more items in the same type have the same value for the specified key.

### Foreign key constraint violation

```
[foreign_key] foreign key "X" not found in refType.$.refKey
```

The value from the owning item does not exist in the referenced type's key set.

### Path equals attribute violation

```
[path_equals_attr] path value "X" does not match attribute value "Y"
```

The path-derived value (from file name, parent folder, or capture group) does not match the item attribute value.

## Export Errors (Exit Code 3)

### Directory creation failure

```
creating output directory for type: ...
```

### Write failure

```
writing output file for type: ...
```

### Marshaling failure

```
marshaling format output for type: ...
```

## Tidy Errors (Exit Code 4)

Tidy errors occur when a file cannot be parsed or rewritten during formatting normalization.

## Error Output Formats

Errors can be output in three formats using the `--format` flag on the `validate` command:

### Text (default)

```
error: [type_name] file/path.json message
```

Written to stderr.

### JSON (`--format json`)

```json
[
  {
    "level": "error",
    "type": "type_name",
    "file": "file/path.json",
    "message": "description"
  }
]
```

Written to stdout. For CSV files, a `row` field is included.

### YAML (`--format yaml`)

```yaml
- level: error
  type: type_name
  file: file/path.json
  message: description
```

Written to stdout.
