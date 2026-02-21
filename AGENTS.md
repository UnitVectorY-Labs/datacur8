# datacur8 Agent Notes

## What Is Unique Here
- `datacur8` is a Go CLI centered on a repository-root `.datacur8` file. Main commands are `validate`, `export`, and `tidy`.
- The `.datacur8` YAML is validated against an embedded JSON Schema (`internal/config/config.schema.json`) before semantic validation logic runs.
- Per-type data schemas are inline JSON Schema and are applied to discovered JSON/YAML/CSV inputs during `validate` and `export`.

## Test Strategy (Important)
- Tests are primarily data-driven under `tests/<case>/`.
- Each case typically contains:
  - `.datacur8` config
  - input files
  - `expected/` outputs (for example `validate.exit`, optional `validate.stderr`, and export/tidy snapshots)
- Prefer adding or updating data-driven test cases over writing new integration harness code.

## Documentation

- The `docs/` directory contains the markdown documentation for `datacur8` that must be kept accurate as features are added or changed.
- The documentation files include:
  - `README.md`: The marketing style overview
  - `COMMAND.md`: Defines commands and parameters
  - `CONFIGURATION.md`: Outlines the `.datacur8` configuration structure
  - `EXAMPLES.md`: Provides examples for using datacur8
  - `CONSTRAINTS.md`: Describes the various constraints
  - `INTERNALS.md`: Catchall place for internal design notes
