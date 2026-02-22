# datacur8 Agent Notes

## What Is Unique Here
- `datacur8` is a Go CLI centered on a repository-root `.datacur8` file. Main commands are `validate`, `export`, and `tidy`.
- The `.datacur8` YAML is validated against an embedded JSON Schema (`internal/config/config.schema.json`) before semantic validation logic runs.
- Per-type data schemas are inline JSON Schema and are applied to discovered JSON/YAML/CSV inputs during `validate` and `export`.

## Test Strategy (Important)
- Tests are primarily data-driven under `tests/<case>/`.
- Each case must be complete and self-validating (the test suite now checks fixture completeness before running command assertions).
- Every `tests/<case>/` directory must contain a repository-root `.datacur8` file for that case (the config may intentionally be invalid for failure cases).
- Each case must contain `expected/validate.exit`.
- If `validate.exit` is `0` and the case config declares any `types[].output.path`, the case must include matching snapshots under `expected/export/...` for every configured output path.
- `expected/tidy/...` snapshots are required for tidy-focused cases and should contain the post-tidy file content to compare against.
- Each case typically contains:
  - `.datacur8` config
  - input files
  - `expected/` outputs (for example `validate.exit`, optional `validate.stderr`, and export/tidy snapshots)
- Documentation-linked examples should use `tests/example_*` folder names so they can be mapped back to docs clearly.
- Prefer adding or updating data-driven test cases over writing new integration harness code.
- See `docs/DATA_DRIVEN_TESTS.md` for the full fixture contract and authoring guidance.

## Documentation

- The `docs/` directory contains the markdown documentation for `datacur8` that must be kept accurate as features are added or changed.
- The documentation files include:
  - `docs/README.md`: The marketing style overview
  - `docs/COMMAND.md`: Defines commands and parameters
  - `docs/CONFIGURATION.md`: Outlines the `.datacur8` configuration structure
  - `docs/EXAMPLES.md`: Provides examples for using datacur8
  - `docs/CONSTRAINTS.md`: Describes the various constraints; must be updated as new constraints or parameters are added to constraints
  - `docs/CONDITIONS.md`: Reference for validation conditions, errors, and exit codes
  - `docs/DATA_DRIVEN_TESTS.md`: Internal guide for the data-driven test fixture contract, required files, and `example_*` documentation coverage
  - `docs/INTERNALS.md`: Documentation for internal implementation structure; must be maintained as the code evolves to ensure accuracy for future contributors
