---
layout: default
title: Configuration
nav_order: 3
permalink: /configuration
---

# Configuration
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Overview

datacur8 is configured by a single YAML file named `.datacur8` placed in the repository root directory. This file defines all types, schemas, constraints, and export settings.

No additional config files are used — including in subdirectories. If a `.datacur8` file is found in a subdirectory, an error is returned.

## Top-Level Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `version` | string | **yes** | — | Minimum datacur8 version required (semver: `major.minor.patch`) |
| `strict_mode` | string | no | `DISABLED` | One of `DISABLED`, `ENABLED`, or `FORCE` |
| `types` | array | **yes** | — | List of type definitions |
| `tidy` | object | no | `{ enabled: true }` | Global tidy configuration |

### version

The minimum version of datacur8 required to process the config, expressed as `major.minor.patch`. The major version must match the CLI version, and the CLI version must be greater than or equal to the configured version. When running a development build, this field is ignored with a warning.

### strict_mode

Controls enforcement of `additionalProperties` on JSON schemas:

| Value | Behavior |
|-------|----------|
| `DISABLED` | Schemas are evaluated as-is |
| `ENABLED` | Object schemas without an explicit `additionalProperties` are treated as `additionalProperties: false` |
| `FORCE` | All object schemas have `additionalProperties: false` applied, overriding any explicit `true` setting |

### tidy

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `tidy.enabled` | boolean | `true` | Set to `false` to disable the tidy command |

## Type Definition

Each entry in the `types` array defines a category of data files.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | **yes** | — | Unique identifier; must match `^[a-zA-Z][a-zA-Z0-9_]*$` |
| `input` | string | **yes** | — | Input format: `json`, `yaml`, or `csv` |
| `match` | object | **yes** | — | File matching rules |
| `schema` | object | **yes** | — | Inline JSON Schema; root `type` must be `object` |
| `constraints` | array | no | `[]` | List of constraints |
| `output` | object | no | — | Export configuration |
| `csv` | object | conditional | — | CSV configuration; **required** when `input` is `csv` |
| `tidy` | object | no | — | Per-type tidy configuration |

### name

Type names must be unique across all types and match the pattern `^[a-zA-Z][a-zA-Z0-9_]*$`.

### input

Each type handles exactly one file format:

| Value | Description |
|-------|-------------|
| `json` | JSON files parsed as objects |
| `yaml` | YAML files parsed as objects |
| `csv` | CSV files parsed as rows of objects |

## Match Definition

Controls which files belong to a type.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `match.include` | array of strings | **yes** | — | Regex patterns applied to repo-relative file paths (minimum 1) |
| `match.exclude` | array of strings | no | `[]` | Regex patterns for files to exclude |

A file belongs to a type if it matches at least one `include` pattern and does not match any `exclude` pattern. Each file must match **exactly one** type; matching multiple types is a validation error. Files matching no types are ignored.

Paths use forward slashes and are relative to the repository root.

### Named Capture Groups

Include patterns may contain named capture groups that expose path segments as metadata for constraints:

```yaml
match:
  include:
    - "^configs/(?P<team>[^/]+)/services/(?P<service>[^/]+)\\.ya?ml$"
```

This exposes:
- `path.team` — the captured team value
- `path.service` — the captured service value

The following path values are always available (no capture groups required):

| Selector | Description |
|----------|-------------|
| `path.file` | File name without extension |
| `path.ext` | Normalized extension without dot (`yaml`, `json`, or `csv`) |
| `path.parent` | Name of the parent folder |

## Schema

The `schema` field contains an inline JSON Schema applied to each parsed item. The root type must be `object`.

```yaml
schema:
  type: object
  required: ["id", "name"]
  properties:
    id: { type: string }
    name: { type: string }
  additionalProperties: false
```

datacur8 uses the `google/jsonschema-go` library for JSON Schema evaluation. The schema is validated to be a valid JSON Schema at config load time.

For CSV types, the schema must be a flat object (no nested objects or arrays) since CSV rows produce flat key-value objects.

## Constraints

Constraints are defined per type and enforce data integrity rules beyond JSON Schema. A constraint may reference other types but is always attached to a single owning type.

### Selectors

Constraints use JSONPath-like selectors to reference attributes:

| Syntax | Description |
|--------|-------------|
| `$` | Root object |
| `$.field` | Top-level field |
| `$.a.b.c` | Nested field access |
| `$.items[*].id` | Project values from array items |

If a selector yields multiple values:
- `unique` enforces uniqueness within each item (with `scope: item`) or across all items (with `scope: type`)
- `foreign_key` and `path_equals_attr` require a single scalar and will error if multiple values are found

### unique

Enforces uniqueness of a key across all items of a type, or within each item.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | **yes** | — | Must be `unique` |
| `key` | string | **yes** | — | Selector for the value to check |
| `id` | string | no | — | Optional identifier for the constraint |
| `case_sensitive` | boolean | no | `true` | Whether string comparison is case-sensitive |
| `scope` | string | no | `type` | `type` for cross-item uniqueness, `item` for within-item uniqueness |

```yaml
constraints:
  - type: unique
    key: "$.id"
```

### foreign_key

Ensures that a value in one type exists as a value in another type.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | **yes** | — | Must be `foreign_key` |
| `key` | string | **yes** | — | Selector on the owning item |
| `id` | string | no | — | Optional identifier |
| `references.type` | string | **yes** | — | Name of the referenced type |
| `references.key` | string | **yes** | — | Selector on the referenced type items |

```yaml
constraints:
  - type: foreign_key
    key: "$.teamId"
    references:
      type: team
      key: "$.id"
```

### path_equals_attr

Ensures that a value derived from the file path equals an attribute value.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | **yes** | — | Must be `path_equals_attr` |
| `path_selector` | string | **yes** | — | Path value: `path.file`, `path.parent`, `path.ext`, or `path.<capture>` |
| `id` | string | no | — | Optional identifier |
| `references.key` | string | **yes** | — | Selector on the item to compare against |
| `case_sensitive` | boolean | no | `true` | Whether comparison is case-sensitive |

```yaml
constraints:
  - type: path_equals_attr
    path_selector: "path.file"
    references:
      key: "$.id"
```

Note: `references.type` cannot be set for `path_equals_attr` — it always applies within the owning type.

## Output Configuration

Optional per-type configuration for export.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `output.path` | string | **yes** | Output file path relative to the repo root |
| `output.format` | string | **yes** | One of `json`, `yaml`, or `jsonl` |

```yaml
output:
  path: "out/teams.json"
  format: json
```

Output paths must be unique across all types. Export creates directories as needed.

## CSV Configuration

Required when `input` is `csv`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `csv.delimiter` | string | no | `,` | Single-character delimiter |

```yaml
csv:
  delimiter: ","
```

CSV validation:
- The header row is required
- All header names must exist in `schema.properties`
- All `schema.required` fields must be present in the header
- Values are converted to the schema-specified types (`boolean`, `number`, `integer`, `string`)

## Per-Type Tidy Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tidy.sort_arrays_by` | array of strings | no | Selectors used to sort top-level arrays for stable diffs |

```yaml
tidy:
  sort_arrays_by: ["name"]
```
