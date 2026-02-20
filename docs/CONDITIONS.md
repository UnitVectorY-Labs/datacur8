---
layout: default
title: Conditions
nav_order: 5
permalink: /conditions
---

# Conditions
{: .no_toc }

Reference for validation conditions, error messages, and how to fix them.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Overview

datacur8 validates data in phases. Each phase can produce errors that stop further processing. The exit code indicates the category of failure.

| Exit Code | Category | Description |
|-----------|----------|-------------|
| 0 | Success | All validation passed |
| 1 | Config invalid | `.datacur8` file is missing, malformed, or has semantic errors |
| 2 | Data invalid | Schema validation or constraint violations found |
| 3 | Export failure | Errors writing output files |
| 4 | Tidy failure | Errors parsing or writing files during tidy |

## Config Validation Errors (Exit Code 1)

These errors indicate problems with the `.datacur8` configuration file itself.

### Missing config file

```
error: .datacur8 not found in current directory. Run from repo root.
```

**Cause:** The `.datacur8` file does not exist in the current working directory.

**Fix:** Run datacur8 from the repository root directory that contains the `.datacur8` file.

### Invalid version format

```
error: version "abc" is not valid semver (expected major.minor.patch)
```

**Cause:** The `version` field is not a valid semantic version.

**Fix:** Use the format `major.minor.patch`, e.g., `"1.0.0"`.

### Version mismatch

```
error: major version mismatch: config requires 2.x.x but CLI is 1.0.0
error: CLI version 1.0.0 is older than config version 1.2.0
```

**Cause:** The CLI version does not satisfy the config's minimum version requirement.

**Fix:** Upgrade datacur8 to a version that meets or exceeds the config's `version` field.

### Invalid strict_mode

```
error: strict_mode "INVALID" is invalid; must be DISABLED, ENABLED, or FORCE
```

**Fix:** Set `strict_mode` to `DISABLED`, `ENABLED`, or `FORCE`.

### Invalid reporting mode

```
error: reporting.mode "xml" is invalid; must be text, json, or yaml
```

**Fix:** Set `reporting.mode` to `text`, `json`, or `yaml`.

### Duplicate type name

```
error: types[1](team): duplicate type name "team"
```

**Cause:** Two or more types have the same `name`.

**Fix:** Give each type a unique name.

### Invalid type name format

```
error: types[0](123bad): type name must match ^[a-zA-Z][a-zA-Z0-9_]*$
```

**Fix:** Type names must start with a letter and contain only letters, digits, and underscores.

### Invalid input format

```
error: types[0](mytype): input "xml" must be json, yaml, or csv
```

**Fix:** Set `input` to `json`, `yaml`, or `csv`.

### Empty match.include

```
error: types[0](mytype): match.include must have at least 1 pattern
```

**Fix:** Add at least one regex pattern to `match.include`.

### Invalid regex pattern

```
error: types[0](mytype): match.include[0] invalid regex: ...
```

**Cause:** A `match.include` or `match.exclude` pattern is not a valid regular expression.

**Fix:** Correct the regex syntax. Remember to escape special characters.

### Missing or invalid schema

```
error: types[0](mytype): schema is required
error: types[0](mytype): schema.type must be "object"
```

**Fix:** Provide a `schema` with `type: object` at the root.

### Missing CSV config

```
error: types[0](mytype): csv config is required when input is csv
```

**Cause:** A type with `input: csv` does not have a `csv` configuration block.

**Fix:** Add a `csv` block (at minimum `csv: { delimiter: "," }`).

### Invalid CSV delimiter

```
error: types[0](mytype): csv.delimiter must be exactly 1 character
```

**Fix:** Set `csv.delimiter` to a single character, e.g., `","` or `"\t"`.

### Invalid output format

```
error: types[0](mytype): output.format "xml" must be json, yaml, or jsonl
```

**Fix:** Set `output.format` to `json`, `yaml`, or `jsonl`.

### Conflicting output paths

```
error: types[1](service): output.path "out/data.json" conflicts with type "team"
```

**Cause:** Two types are configured to write to the same output path.

**Fix:** Give each type a unique `output.path`.

### Invalid constraint selector

```
error: types[0](mytype).constraints[0]: key "bad selector" is not a valid selector: ...
```

**Fix:** Use valid selector syntax: `$.field`, `$.a.b.c`, or `$.items[*].id`.

### Invalid constraint scope

```
error: types[0](mytype).constraints[0]: scope "global" must be item or type
```

**Fix:** Set `scope` to `item` or `type`.

### Missing foreign key references

```
error: types[0](mytype).constraints[0]: references is required for foreign_key
error: types[0](mytype).constraints[0]: references.type is required
```

**Fix:** Add a `references` block with `type` and `key` for `foreign_key` constraints.

### Unknown referenced type

```
error: types[0](mytype).constraints[0]: references.type "unknown" does not match any defined type
```

**Cause:** A `foreign_key` constraint references a type name that does not exist.

**Fix:** Ensure the referenced type name matches a type defined in `types`.

### Invalid path_selector

```
error: types[0](mytype).constraints[0]: path_selector "invalid" is invalid
```

**Fix:** Use a valid path selector: `path.file`, `path.parent`, `path.ext`, or `path.<capture_name>`.

### Missing named capture group

```
error: types[0](mytype).constraints[0]: path_selector uses capture "team" but match.include[0] does not define named group (?P<team>...)
```

**Cause:** A `path_equals_attr` constraint references a capture group that is not defined in all include patterns.

**Fix:** Add the named capture group `(?P<team>...)` to every pattern in `match.include`, or change the `path_selector`.

## Discovery Errors (Exit Code 1)

These errors occur during file discovery.

### File matches multiple types

```
error: file "path/to/file.yaml" matches multiple types: team, service
```

**Cause:** A file's path matches the include patterns of more than one type.

**Fix:** Adjust `match.include` or `match.exclude` patterns so each file matches exactly one type.

### Nested .datacur8 file

If a `.datacur8` file is found in a subdirectory, an error is returned because only the root config file is supported.

## Schema Validation Errors (Exit Code 2)

These errors indicate that a data file's content does not conform to the type's JSON Schema.

### Common schema errors

- **Missing required property:** A required field is absent from the data
- **Wrong type:** A field has the wrong data type (e.g., number instead of string)
- **Additional properties:** The data contains fields not defined in the schema (especially with strict mode)
- **Pattern mismatch:** A string does not match the schema's `pattern`
- **Enum violation:** A value is not in the allowed set

Error messages include the file path and the specific schema violation.

### CSV-specific schema errors

- **Unknown header:** A CSV column header is not found in `schema.properties`
- **Missing required column:** A column listed in `schema.required` is not in the header
- **Type conversion failure:** A cell value cannot be converted to the schema type

```
error: CSV header "unknown_col" not found in schema properties
error: required property "id" missing from CSV headers
error: row 3, column "price": invalid number value: "abc"
```

## Constraint Violation Errors (Exit Code 2)

These errors indicate that data passed schema validation but violates a constraint.

### Unique constraint violation

```
error: [team] teams/beta.yaml [unique] duplicate value "alpha" for key $.id (first seen in teams/alpha.yaml)
```

**Cause:** Two items of the same type have the same value for a unique key.

**Fix:** Ensure each item has a unique value for the constrained field.

### Foreign key violation

```
error: [service] services/api.yaml [foreign_key] value "unknown-team" for key $.teamId not found in type "team" key $.id
```

**Cause:** A value references a record in another type that does not exist.

**Fix:** Ensure the referenced value exists in the target type, or correct the referencing value.

### Path equals attribute violation

```
error: [team] teams/alpha.yaml [path_equals_attr] path value "alpha" does not equal attribute value "beta" for key $.id
```

**Cause:** The value derived from the file path does not match the corresponding attribute value.

**Fix:** Either rename the file/folder to match the attribute, or update the attribute to match the path.

## Export Errors (Exit Code 3)

Export errors occur when validated data cannot be written to output files.

### Common export errors

- **Write permission denied:** Cannot write to the output path
- **Directory creation failure:** Cannot create the output directory

**Fix:** Ensure the output directory is writable and the path is valid.

## Tidy Errors (Exit Code 4)

Tidy errors occur when files cannot be parsed or rewritten.

### Common tidy errors

- **Parse failure:** A file cannot be parsed in its declared format
- **Write failure:** A tidied file cannot be written back to disk

**Fix:** Ensure data files are valid for their declared format and that the directory is writable.
