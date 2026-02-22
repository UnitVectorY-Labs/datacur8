---
layout: default
title: Configuration
nav_order: 3
has_children: true
permalink: /configuration
---

# Configuration
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Overview

**datacur8** is configured by a single YAML file named `.datacur8` placed in the repository root directory. This file defines all types, schemas, constraints, and export settings.

No additional config files are used — including in subdirectories. If a `.datacur8` file is found in a subdirectory, an error is returned.

## Top-Level Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `version` | string | **yes** | — | Minimum **datacur8** version required (semver: `major.minor.patch`) |
| `strict_mode` | string | no | `DISABLED` | One of `DISABLED`, `ENABLED`, or `FORCE` |
| `types` | array | **yes** | — | List of type definitions |
| `tidy` | object | no | `{ enabled: true }` | Global tidy configuration |

### version

The minimum version of **datacur8** required to process the config, expressed as `major.minor.patch`. The major version must match the CLI version, and the CLI version must be greater than or equal to the configured version. 

{: .highlight }
When running a development build, this field is ignored with a warning.

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

---

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

### name

Type names must be unique across all types and match the pattern `^[a-zA-Z][a-zA-Z0-9_]*$`.


{: .important }
The type name is restricted to letters, digits, and underscores, and must start with a letter. This is to ensure that type names can be safely used as keys in export output and referenced in constraints without needing escaping.

### input

Each type handles exactly one file format:

| Value | Description |
|-------|-------------|
| `json` | JSON files parsed as objects |
| `yaml` | YAML files parsed as objects |
| `csv` | CSV files parsed as rows of objects (comma-delimited; no CSV config) |

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

{: .highlight }
Avoid using overlapping capture group names of "file", "ext", or "parent" to prevent conflicts with the default path selectors.

---

## Schema

The `schema` field contains an inline JSON Schema applied to each parsed item. 

{: .highlight }
The root type must be `object`. This is because **datacur8** is designed to export data as objects.

```yaml
schema:
  type: object
  required: ["id", "name"]
  properties:
    id: { type: string }
    name: { type: string }
  additionalProperties: false
```

**datacur8** uses the [google/jsonschema-go](https://github.com/google/jsonschema-go) library for JSON Schema evaluation. The schema is validated to be a valid JSON Schema at config load time.

{: .highlight }
For CSV types, the schema must be a flat object (no nested objects or arrays) since CSV rows produce flat key-value objects. The column names for the CSV file are used as the attribute keys to convert each row into a JSON object that is then validated against the schema.

---

## Constraints

Constraints are defined per type and enforce data integrity rules beyond JSON Schema. A constraint may reference other types but is always attached to a single owning type.

{: .important }
See [Constraints](CONSTRAINTS.md) for detailed information on each constraint type, selectors, and examples.

Supported constraint types:

- `unique` — enforce uniqueness of a key across all items of a type, or within each item
- `foreign_key` — ensure that a value in one type exists as a value in another
- `path_equals_attr` — ensure that a value derived from the file path equals an attribute value

---

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

CSV files always use a comma (`,`) delimiter.
