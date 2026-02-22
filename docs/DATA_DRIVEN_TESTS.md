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

- Contains the expected post-`tidy` file content.
- Paths are relative to the case root, mirrored under `expected/tidy/`.
- Example: `expected/tidy/data/w1.yaml`

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
- Add `expected/tidy/...` when the case is intended to exercise `tidy`

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
- `tests/example_examples_team_service_registry_success`
- `tests/example_examples_team_service_registry_foreign_key_failure`
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

