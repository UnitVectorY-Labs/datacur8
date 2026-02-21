---
layout: default
title: Command
nav_order: 2
permalink: /command
---

# Command Line Reference
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Usage

```
datacur8 <command> [flags]
```

datacur8 must be run from the directory that contains the `.datacur8` configuration file. If the file is not found, the CLI exits with:

```
error: .datacur8 not found in current directory. Run from repo root.
```

## Commands

### validate

Validate the configuration and all data files.

```
datacur8 validate [--config-only] [--format text|json|yaml]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--config-only` | Only validate the `.datacur8` configuration file; skip data file scanning and validation |
| `--format` | Override the output format for errors and warnings. Accepts `text`, `json`, or `yaml`. Defaults to the `reporting.mode` setting in the config |

**Behavior:**

1. Loads and validates the `.datacur8` config file
2. If `--config-only` is set, stops after config validation
3. Discovers files matching type definitions
4. Parses each file according to its input format
5. Validates each item against its JSON Schema
6. Evaluates all constraints
7. Reports all errors found

If no types are configured, validation is a no-op (config schema is still validated) and exits successfully.

### export

Export validated data to configured output files.

```
datacur8 export
```

Export runs the full validation pipeline first. If validation fails, export does not proceed and returns the validation exit code.

For each type that defines an `output` configuration, datacur8 writes a compiled output file. If no types define output, export logs a message and exits successfully.

Output formats:

| Format | Description |
|--------|-------------|
| `json` | JSON array wrapped in an object keyed by the type name |
| `yaml` | YAML array wrapped in an object keyed by the type name |
| `jsonl` | One minified JSON object per line |

Items are ordered deterministically: by type order in config, then by file path, then by within-file order (for CSV rows).

### tidy

Normalize file formatting for stable diffs.

```
datacur8 tidy [--dry-run]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--dry-run` | Show which files would be changed without writing |

**Behavior:**

- **JSON**: pretty-printed with sorted keys
- **YAML**: stable formatting with sorted keys; comments are removed
- **CSV**: sorted columns (alphabetical), optionally sorted rows via `tidy.sort_arrays_by`

Tidy does not change parsed data values. If the global `tidy.enabled` is set to `false`, tidy exits immediately.

### version

Print the datacur8 version.

```
datacur8 version
```

Prints the version string and exits with code 0.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Configuration invalid — the `.datacur8` file has errors, or the file is missing |
| `2` | Data invalid — schema validation or constraint violations found |
| `3` | Export failure — errors writing output files |
| `4` | Tidy failure — errors parsing or writing files during tidy |

## Output Formats

Error and warning output can be formatted as plain text (default), JSON, or YAML.

**Text format** (default):

```
error: [type_name] file/path.yaml message describing the problem
```

**JSON format** (`--format json`):

```json
[
  {
    "level": "error",
    "type": "team",
    "file": "teams/alpha.yaml",
    "message": "schema validation failed: ..."
  }
]
```

**YAML format** (`--format yaml`):

```yaml
- level: error
  type: team
  file: teams/alpha.yaml
  message: "schema validation failed: ..."
```

For CSV files, a `row` field is included in structured output to identify the specific row.