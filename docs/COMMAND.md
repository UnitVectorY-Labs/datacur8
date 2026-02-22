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
Usage: datacur8 <command> [flags]

Commands:
  validate    Validate configuration and data files
  export      Export validated data to configured outputs
  tidy        Normalize file formatting for stable diffs
  version     Print the version

Run 'datacur8 <command> --help' for more information on a command.
```

{: .important }
**datacur8** must be run from the directory that contains the `.datacur8` configuration file.

## Commands

### `validate`

Validate the configuration and all data files. This provides the ability for a human user to validate the data set and also serves as a validation step for a pipeline before a pull request with changes to the data is merged.

```bash
datacur8 validate [--config-only] [--format text|json|yaml]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--config-only` | Only validate the `.datacur8` configuration file; skip data file scanning and validation |
| `--format` | Override the output format for errors and warnings. Accepts `text`, `json`, or `yaml`.<br>Defaults to `text` format |

**Behavior:**

1. Loads and validates the `.datacur8` config file
2. If `--config-only` is set, stops after config validation
3. Discovers files matching type definitions
4. Parses each file according to its input format
5. Validates each item against its JSON Schema
6. Evaluates all constraints (uniqueness, references, etc...)
7. Reports all errors found

{: .highlight }
If no types are configured in `.datacur8`, validation is a no-op (config schema is still validated) and exits successfully.

### `export`

Export validated data to configured output files. This is intended to be used in a pipeline after a change is merged to a deployment branch (ex: `main`) to compile the source data into a more consumable format for loading into downstream systems (ex: a database).

```bash
datacur8 export
```

Export runs the full validation pipeline first. If validation fails, export does not proceed and returns the validation exit code.

For each type that defines an `output` configuration, **datacur8** writes a compiled output file. If no types define output, export logs a message and exits successfully.

Output formats:

| Format | Description |
|--------|-------------|
| `json` | JSON object with one key (the type name) whose value is the exported array |
| `yaml` | YAML object with one key (the type name) whose value is the exported array |
| `jsonl` | One minified JSON object per line |

The ordering of items within the output file is intended to be deterministic based on file path to minimize differences between sequential runs.

### `tidy`

Normalize file formatting for stable diffs. This is intended to allow for the content of the human edited files to be normalized with minimal effort to allow for the diffs to be cleaner. It can be added as a required check in the pull request pipeline to ensure that all files are tidy before allowing a change to be merged.

```bash
datacur8 tidy [--write]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--write` | Rewrite files in place. Without this flag, `tidy` runs in check mode and prints a colored diff |

**Behavior:**

- Default mode is **check-only**:
  - files are not modified
  - a colored git-like diff (with hunk line numbers and line-numbered added/removed lines) is written to the terminal for each file that would change
  - exit code is non-zero when any file needs tidying (useful for CI / merge gates)
- `--write` applies the tidy changes in place and exits non-zero only on parse/write errors
- **JSON**: pretty-printed with sorted keys
- **YAML**: stable formatting with sorted keys; comments are removed
- **CSV**: sorted columns (alphabetical)

Tidy does not change parsed data values. If the global `tidy.enabled` is set to `false`, tidy exits immediately.

### `version`

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
| `5` | Tidy check failed — one or more files need formatting (check mode only) |

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
