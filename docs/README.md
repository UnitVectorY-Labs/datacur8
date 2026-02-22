---
layout: default
title: datacur8
nav_order: 1
permalink: /
---

# datacur8
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## What is datacur8?

datacur8 is a config-driven command-line tool that validates, exports, and tidies structured data files (JSON, YAML, and CSV) stored in a repository. It brings database-level data integrity to file-based datasets managed in git.

**The problem:** Teams often manage configuration, catalog, or reference data as files in a git repository. Without tooling, there is no way to enforce schemas, uniqueness, foreign key relationships, or naming conventions. Errors accumulate silently until they cause production incidents.

**The solution:** datacur8 reads a single `.datacur8` configuration file that declares your data types, schemas, and constraints. It validates every file deterministically, produces clear error messages, and runs identically locally and in CI.

![datacur8 diagram](overview.excalidraw.svg)

## Key Features

- **JSON Schema validation** — validate every data file against an inline JSON Schema with full-featured support via `google/jsonschema-go`
- **Cross-file constraints** — enforce uniqueness, foreign keys, and path-to-attribute rules across files and types
- **Export** — compile validated data into deterministic JSON, YAML, or JSONL output files
- **Tidy** — normalize formatting (sorted keys, stable ordering) for clean diffs
- **Strict mode** — optionally enforce `additionalProperties: false` on all object schemas
- **CSV support** — schema-guided type conversion with header validation
- **Git-friendly** — designed for CI pipelines; identical results locally and in automation

## Quick Start

1. Create a `.datacur8` file in your repository root:

   ```yaml
   version: "1.0.0"
   types:
     - name: team
       input: yaml
       match:
         include:
           - "^teams/.*\\.ya?ml$"
       schema:
         type: object
         required: ["id", "name"]
         properties:
           id: { type: string }
           name: { type: string }
         additionalProperties: false
       constraints:
         - type: unique
           key: "$.id"
   ```

2. Run validation:

   ```bash
   datacur8 validate
   ```

3. Export compiled outputs:

   ```bash
   datacur8 export
   ```

4. Normalize formatting:

   ```bash
   datacur8 tidy
   ```
