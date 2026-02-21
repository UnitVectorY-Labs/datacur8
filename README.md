# datacur8

A config-driven CLI that validates and enforces cross-file integrity for structured JSON, YAML, and CSV datasets in a repo, then can export compiled outputs and deterministically tidy files for stable diffs.

## Documentation

Full documentation is available at the [datacur8 documentation site](https://datacur8.unitvectorylabs.com/).

| Page | Description |
|------|-------------|
| [Overview](https://datacur8.unitvectorylabs.com/) | What datacur8 is and how it works |
| [Command Reference](https://datacur8.unitvectorylabs.com/command) | CLI commands, flags, and exit codes |
| [Configuration](https://datacur8.unitvectorylabs.com/configuration) | `.datacur8` config file specification |
| [Examples](https://datacur8.unitvectorylabs.com/examples) | Working examples with teams, CSV, strict mode |
| [Conditions](https://datacur8.unitvectorylabs.com/conditions) | Error messages and how to fix them |
| [Internals](https://datacur8.unitvectorylabs.com/internals) | Architecture and design details |

## Quick Example

Create a `.datacur8` configuration file:

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

Run validation:

```sh
datacur8 validate
```

## Installation

Download the latest release from the [releases page](https://github.com/UnitVectorY-Labs/datacur8/releases).

Or build from source:

```sh
go install github.com/UnitVectorY-Labs/datacur8@latest
```

## Commands

| Command | Description |
|---------|-------------|
| `datacur8 validate` | Validate config and data files |
| `datacur8 export` | Export validated data to output files |
| `datacur8 tidy` | Normalize file formatting |
| `datacur8 version` | Print version |

## Contributing

Contributions are welcome. Please open an issue or pull request on [GitHub](https://github.com/UnitVectorY-Labs/datacur8).

## License

See [LICENSE](LICENSE) for details.
