---
layout: default
title: Data-Driven Tests
parent: Internals
nav_order: 2
permalink: /data-driven-tests
---

# Data-Driven Tests
{: .no_toc }

`datacur8` is primarily tested with data-driven fixtures under `tests/<case>/`. The fixtures are the test contract. A case is considered incomplete if required files or snapshots are missing, and the test suite must fail in that situation.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Philosophy

- Test behavior and test fixture completeness are both enforced.
- A case that is missing required snapshots is a broken test, even if the CLI behavior being exercised would otherwise pass.
- Failure cases are first-class test cases. An invalid `.datacur8` or invalid data file is valid test input when the expected results are captured under `expected/`.

## Required Case Structure

Every top-level folder under `tests/` is treated as a test case.

```text
tests/<case>/
  .datacur8                # required (may intentionally be invalid)
  ... input files ...
  expected/                # required
    validate.exit          # required
    validate.args          # optional
    validate.stdout        # optional
    validate.stderr        # optional
    export/...             # required when validate.exit == 0 and outputs are configured
    tidy/...               # required for tidy cases
```

## `expected/` File Reference

### `expected/validate.exit` (required)

- The expected exit code for `datacur8 validate`.
- Parsed as a single integer.
- Common values:
  - `0`: validation succeeded
  - `1`: config/discovery failure
  - `2`: data validation failure

### `expected/validate.args` (optional)

- Extra CLI args appended to `validate`.
- Typical use: `--format json` for stable machine-readable error snapshots.
- Whitespace-separated.

### `expected/validate.stdout` (optional)

- Snapshot of `validate` stdout.
- Used most often with `--format json`.
- Compared as JSON (structural equality), not raw text.

### `expected/validate.stderr` (optional)

- Snapshot of `validate` stderr.
- Compared line-by-line (order-insensitive for non-empty lines).

### `expected/export/...` (conditionally required)

- Required when:
  - `expected/validate.exit` is `0`, and
  - `.datacur8` declares one or more `types[].output.path`.
- Must include a snapshot file for every configured output path.
- Example:

```text
expected/
  export/
    out/
      teams.json
      services.jsonl
```

### `expected/tidy/...` (required for tidy cases)

- Contains the expected post-`tidy --write` file content.
- Paths are relative to the case root, mirrored under `expected/tidy/`.
- Example: `expected/tidy/data/w1.yaml`
- The integration suite also runs plain `tidy` (check mode) for the same fixture and asserts:
  - files are not rewritten in check mode
  - exit code is non-zero when the snapshot differs from the original input
  - diff output is emitted

## Fixture Completeness Rules Enforced by Tests

The integration suite includes a fixture meta-test that fails when:

- a `tests/<case>/` directory is missing `.datacur8`
- `expected/` is missing
- `expected/validate.exit` is missing
- `expected/export/` exists but contains no files
- `expected/tidy/` exists but contains no files
- `validate.exit == 0` and a configured `output.path` is missing a matching `expected/export/...` snapshot

This prevents silent skips and partial fixtures.

## Success vs Failure Cases

### Success cases

- `expected/validate.exit` is `0`
- If outputs are configured, `expected/export/...` snapshots are required
- Add `expected/tidy/...` when the case is intended to exercise `tidy` (used for both check mode and `--write`)

### Failure cases

- Non-zero `expected/validate.exit` is expected
- `.datacur8` may be invalid (schema/semantic errors) or the data may be invalid
- Prefer `expected/validate.args` with `--format json` plus `expected/validate.stdout` so failure intent is explicit and stable

## Documentation Example Fixtures (`example_*`)

Use `tests/example_*` for fixtures that back examples shown in user-facing docs. This keeps examples traceable and makes documentation coverage auditable.

Current examples include:

- `tests/example_readme_quick_start_success`
- `tests/example_readme_quick_start_unique_id_failure`
- `tests/example_readme_quick_start_path_file_mismatch_failure`
- `tests/example_readme_quick_start_foreign_key_failure`
- `tests/example_examples_team_service_registry_success`
- `tests/example_examples_team_service_registry_foreign_key_failure`
- `tests/example_examples_hcl_catalog_success`
- `tests/example_examples_csv_product_catalog_success`
- `tests/example_examples_csv_product_catalog_foreign_key_failure`
- `tests/example_examples_csv_product_catalog_type_conversion_failure`
- `tests/example_examples_strict_mode_enabled_failure`
- `tests/example_examples_strict_mode_force_failure`
- `tests/example_examples_multi_format_export_json`
- `tests/example_examples_multi_format_export_yaml`
- `tests/example_examples_multi_format_export_jsonl`

Behavior-focused condition examples can also use `example_*` naming (for example `tests/example_conditions_*`) when they exist to illustrate a specific error mode.

## Authoring Checklist

Before committing a new fixture:

- Add `.datacur8`
- Add input files for the scenario
- Add `expected/validate.exit`
- Add `expected/validate.args` and `expected/validate.stdout` for failure cases when possible
- Add `expected/export/...` for every configured `output.path` when validation succeeds
- Add `expected/tidy/...` when testing `tidy`
- Do not keep generated outputs in the case root (store snapshots under `expected/export/...` instead)
- Prefer one clearly named behavior per case

## CSV export coverage

`example_examples_csv_export_success` covers the documented JSON-to-CSV example. `csv_export_from_formats` uses equivalent JSON/YAML/CSV/HCL records across multiple files and byte-identical snapshots, including optional-column ordering, Unicode, quoting, embedded LF/CR, empty strings, booleans, and numeric precision. `csv_export_empty` requires a header even without input items. `csv_export_numeric_boundaries` covers signed 64-bit YAML integers and finite float64 extremes. `invalid_csv_export_schema_*` and `invalid_csv_export_option` exercise configuration rejection.

The fixture contract is unchanged. Focused CLI tests in `tests/csv_export_test.go` create temporary repositories for exit 3 conversion/write failures and assert existing destinations remain untouched. They also verify validation gating, repeat-export byte equality, and export→CSV-input→export with the same schema. Exporter unit tests cover explicit null/structured cells and numeric boundaries that source schema validation may reject before conversion. A successful validate fixture with configured outputs still requires every export snapshot.

## Configurable delimiters and TSV coverage

`example_examples_tsv_catalog_success` backs the TSV example and checks quoted
tabs/newlines/quotes, scalar conversion, empty strings, explicit extension
matching, constraints, TSV→CSV, and tidy snapshots. `tsv_export_from_formats`
checks JSON/YAML/HCL/comma CSV→TSV with byte snapshots; `tsv_export_empty` covers
header-only output. `csv_unicode_delimiter` checks a non-ASCII separator.
`tsv_single_empty_cell` preserves a quoted empty record through tidy/export.

`invalid_csv_delimiter_*` and `invalid_csv_options_*` cover embedded-schema
rejection; config unit tests independently check semantic validation.
`tsv_*` failures cover headers, syntax, width, conversion/schema logical row
indices, uniqueness, path matching, and cross-format foreign keys.
`tsv_strict_*` preserve successful processing under both strict overlays.
Focused tests in `tests/tsv_test.go` check repeated export/tidy, post-tidy value
alignment, validation-gated exports, and unchanged malformed inputs/destinations.
The existing fixture contract and completeness checks are unchanged.

TSV snapshots can end a record with a tab to represent an empty final cell.
The `*.tsv` rule in `.gitattributes` disables end-of-line whitespace warnings
for that data syntax; preserve these tabs when editing snapshots.
