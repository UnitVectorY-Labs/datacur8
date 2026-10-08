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
datacur8 export [--format text|json|yaml]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format` | Override the output format for errors. Accepts `text`, `json`, or `yaml`.<br>Defaults to `text` format |

Export runs the full validation pipeline first. If validation fails, export does not proceed and returns the validation exit code.

For each type that defines an `output` configuration, **datacur8** writes a compiled output file. If no types define output, export logs a message and exits successfully.

Output formats:

| Format | Description |
|--------|-------------|
| `json` | JSON object with one key (the type name) whose value is the exported array |
| `yaml` | YAML object with one key (the type name) whose value is the exported array |
| `jsonl`, `ndjson` | Equivalent formats: one minified JSON object per line; empty dataset is zero bytes |
| `csv` | Alphabetically sorted schema columns, one header and one row per item (no wrapper) |
| `toml` | TOML type-name key containing an array of inline-table records (`records = [{id = 'a'}]`) |
| `hcl` | HCL attribute named after the type whose value is the exported array (`widgets = [{ ... }]`) |

Input and export formats are configured independently in `.datacur8`; `--format` controls diagnostics only. HCL input uses one attribute-based object per file and is validated against the same JSON Schema as the other input formats. See [Configuration](/configuration#hcl-input) for supported HCL expressions.

TOML input is one record per document, using the same schema and constraints. TOML export wraps records under the type name; empty datasets emit `type_name = []`. Dates/times normalize to strings, non-finite input floats and inexact integers return exit 2, and explicit null anywhere in an export returns exit 3 without replacing the output. See [TOML input and conversion rules](/configuration#toml-input). `.datacur8` remains YAML and `--format` remains diagnostics-only (`text|json|yaml`).

CSV export supports complete flat scalar records; see [CSV export](/configuration#csv-export) for schema restrictions, numeric limits, and round-trip behavior. Empty datasets emit the schema-derived header. Unsupported CSV schemas return exit 1; invalid source data returns exit 2; missing cells, undeclared keys, and other conversion or write failures return exit 3. Conversion failure preserves an existing destination.

The ordering of items within the output file is intended to be deterministic based on file path to minimize differences between sequential runs.

### `tidy`

Normalize file formatting for stable diffs. This is intended to allow for the content of the human edited files to be normalized with minimal effort to allow for the diffs to be cleaner. It can be added as a required check in the pull request pipeline to ensure that all files are tidy before allowing a change to be merged.

```bash
datacur8 tidy [--write] [--format text|json|yaml]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--write` | Rewrite files in place. Without this flag, `tidy` runs in check mode and prints a colored diff |
| `--format` | Override the output format for errors. Accepts `text`, `json`, or `yaml`.<br>Defaults to `text` format |

**Behavior:**

- Default mode is **check-only**:
  - files are not modified
  - a colored git-like diff (with hunk line numbers and line-numbered added/removed lines) is written to the terminal for each file that would change
  - exit code is non-zero when any file needs tidying (useful for CI / merge gates)
- `--write` applies the tidy changes in place and exits non-zero only on parse/write errors
- **JSON**: pretty-printed with sorted keys
- **JSONL/NDJSON**: one minified object per line, recursively sorted keys, record order preserved, LF and final newline; empty files remain empty
- **YAML**: stable formatting with sorted keys; comments are removed
- **CSV/TSV**: sorted columns (alphabetical), with cells reordered together and the configured input delimiter preserved
- **TOML**: TOML 1.1.0 canonical reserialization; comments removed, keys sorted, tables inline, native temporal types retained (nanosecond precision)
- **HCL**: canonical HCL formatting; comments and attribute order are preserved

Tidy does not change parsed data values. If the global `tidy.enabled` is set to `false`, tidy exits immediately.

### `version`

Print the datacur8 version.

```
datacur8 version
```

Prints the version string and exits with code 0.

Output format:

```text
datacur8 version vX.Y.Z (goX.Y, os/arch)
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Configuration invalid — the `.datacur8` file has errors, or the file is missing |
| `2` | Data invalid — schema validation or constraint violations found |
| `3` | Export failure — errors converting or writing output files |
| `4` | Tidy failure — errors parsing or writing files during tidy |
| `5` | Tidy check failed — one or more files need formatting (check mode only) |

## Output Formats

Error and warning output can be formatted as plain text (default), JSON, or YAML using the `--format` flag on `validate`, `export`, and `tidy`.

**Text format** (default) — written to `stderr`:

```
error: [type_name] file/path.yaml message describing the problem
```

**JSON format** (`--format json`) — written to `stdout`:

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

**YAML format** (`--format yaml`) — written to `stdout`:

```yaml
- level: error
  type: team
  file: teams/alpha.yaml
  message: "schema validation failed: ..."
```

For CSV files, a `row` field is included in structured output to identify the specific row.

CSV and TSV share `input: csv` and `output.format: csv`. Input/tidy use
`types[].csv.delimiter`; export uses `types[].output.csv.delimiter`.
Both default independently to comma. Use YAML `delimiter: "\t"` for tab.
See [CSV delimiters and TSV](/configuration#csv-delimiters-and-tsv).
CLI `--format` still selects diagnostics only (`text|json|yaml`).
Invalid delimiter/options return exit 1; TSV data failures return exit 2 and
block export; export encoding/write failures return exit 3. Tidy parse/header
failures return exit 4 without rewriting that file.

### JSONL / NDJSON records

`validate` and `export` accept `input: jsonl` and `input: ndjson` as equivalent
values. Each physical line is one object checked independently against the same
schema and constraints. File discovery order precedes record order. Errors report
one-based `line` locations in text, JSON, and YAML; CSV `row` stays zero-based.
Malformed files contribute no partial records and validation failure blocks
export (exit **2**). Read failures also block export.

`tidy` checks all records before rewriting each file. It preserves nested values,
escaped newlines, Unicode, and numeric values, normalizes separators, and is
idempotent. Check mode returns **5** with a diff when formatting differs and
writes nothing. Malformed records return **4** without rewriting that file.
`.datacur8` remains YAML; `--format` selects diagnostics (`text|json|yaml`), not data
encoding. See [JSONL / NDJSON input](/configuration#jsonl--ndjson-input) for framing,
object-only restrictions, and JSON numeric precision limits.
