---
layout: default
title: Constraints
parent: Configuration
nav_order: 1
permalink: /constraints
---

# Constraints
{: .no_toc }

Constraint rules extend JSON Schema with cross-file and path-aware integrity checks.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Overview

Constraints are configured under each type's `constraints` array in `.datacur8`.
For validation error troubleshooting, see the [Conditions](/conditions) page.

Each constraint has:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | **yes** | Constraint kind (`unique`, `foreign_key`, `path_equals_attr`) |
| `id` | string | no | Optional stable identifier used in reporting |

## Selector Basics

Constraint selectors use the same JSONPath-like selector syntax:

- `$.id`
- `$.team.id`
- `$.items[*].id`

Path-based constraints use `path_selector` with one of:

- `path.file` (filename without extension)
- `path.parent` (direct parent folder)
- `path.ext` (normalized extension)
- `path.<capture>` from named regex groups in `match.include`

## Available Constraints

| Goal | Constraint |
|------|------------|
| Ensure IDs are never duplicated | `unique` |
| Ensure a value exists in another type | `foreign_key` |
| Ensure path naming matches data fields | `path_equals_attr` |

### `unique`

Use `unique` to prevent duplicate identifiers in one type, or duplicate values inside a single item.

#### Attributes

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | **yes** | — | Must be `unique` |
| `key` | string | **yes** | — | Selector for value(s) to check |
| `scope` | string | no | `type` | `type` = across all items, `item` = within each item |
| `case_sensitive` | boolean | no | `true` | String comparison mode |
| `id` | string | no | — | Optional identifier |

#### Example

```yaml
constraints:
  - type: unique
    key: "$.id"
```

### `foreign_key`

Use `foreign_key` to enforce referential integrity between types (for example, `service.teamId` must exist in `team.id`).

#### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | **yes** | Must be `foreign_key` |
| `key` | string | **yes** | Selector on the owning item |
| `references.type` | string | **yes** | Referenced type name |
| `references.key` | string | **yes** | Selector on referenced type items |
| `id` | string | no | Optional identifier |

#### Example

```yaml
constraints:
  - type: foreign_key
    key: "$.teamId"
    references:
      type: team
      key: "$.id"
```

### `path_equals_attr`

Use `path_equals_attr` to enforce filename/folder conventions against data attributes.

Typical use cases:
- file name must match `id`
- folder name must match `teamId`
- named path capture must match an attribute

#### Attributes

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | **yes** | — | Must be `path_equals_attr` |
| `path_selector` | string | **yes** | — | Path source (`path.file`, `path.parent`, `path.ext`, or `path.<capture>`) |
| `references.key` | string | **yes** | — | Selector on the same item |
| `case_sensitive` | boolean | no | `true` | String comparison mode |
| `id` | string | no | — | Optional identifier |

#### Example

```yaml
match:
  include:
    - "^configs/teams/(?P<team>[^/]+)/services/[^/]+\\.ya?ml$"
constraints:
  - type: path_equals_attr
    path_selector: "path.team"
    references:
      key: "$.teamId"
```
