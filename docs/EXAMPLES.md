---
layout: default
title: Examples
nav_order: 4
permalink: /examples
---

# Examples
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

## Team and Service Registry

This example manages teams and their services in YAML files with cross-file validation.

### Directory structure

```
.datacur8
configs/
  teams/
    alpha.yaml
    beta.yaml
    alpha/
      services/
        api-gateway.yaml
        user-service.yaml
    beta/
      services/
        billing.yaml
```

### Configuration

```yaml
version: "0.0.0"
strict_mode: DISABLED

types:
  - name: team
    input: yaml
    match:
      include:
        - "^configs/teams/(?P<team>[^/]+)\\.ya?ml$"
    schema:
      type: object
      required: ["id", "name"]
      properties:
        id: { type: string }
        name: { type: string }
      additionalProperties: false
    constraints:
      - id: team_id_unique
        type: unique
        key: "$.id"
      - id: team_path_matches_id
        type: path_equals_attr
        path_selector: "path.team"
        references:
          key: "$.id"
    output:
      path: "out/teams.json"
      format: json

  - name: service
    input: yaml
    match:
      include:
        - "^configs/teams/(?P<team>[^/]+)/services/(?P<service>[^/]+)\\.ya?ml$"
    schema:
      type: object
      required: ["id", "name", "teamId"]
      properties:
        id: { type: string }
        name: { type: string }
        teamId: { type: string }
      additionalProperties: false
    constraints:
      - id: service_id_unique
        type: unique
        key: "$.id"
      - id: service_team_fk
        type: foreign_key
        key: "$.teamId"
        references:
          type: team
          key: "$.id"
      - id: service_path_team_matches_teamId
        type: path_equals_attr
        path_selector: "path.team"
        references:
          key: "$.teamId"
      - id: service_file_matches_id
        type: path_equals_attr
        path_selector: "path.file"
        references:
          key: "$.id"
    output:
      path: "out/services.jsonl"
      format: jsonl
```

### Data files

`configs/teams/alpha.yaml`:

```yaml
id: alpha
name: Team Alpha
```

`configs/teams/alpha/services/api-gateway.yaml`:

```yaml
id: api-gateway
name: API Gateway
teamId: alpha
```

### What is validated

- Each team has a unique `id`
- The team's folder name matches its `id` (e.g., `alpha.yaml` must have `id: alpha`)
- Each service has a unique `id`
- Each service's `teamId` must reference an existing team
- The service's parent team folder matches its `teamId`
- The service's file name matches its `id`

## HCL Widget Catalog

This example uses one HCL2 attribute file per widget, validates each item with JSON Schema, and exports an aggregate HCL catalog. It is covered by `tests/example_examples_hcl_catalog_success`.

### Configuration

`.datacur8`:

```yaml
version: "0.0.0"

types:
  - name: widgets
    input: hcl
    match:
      include:
        - "^data/.*\\.hcl$"
    schema:
      type: object
      required: [id, enabled, price, tags, metadata]
      properties:
        id:
          type: string
        enabled:
          type: boolean
        price:
          type: number
          minimum: 0
        tags:
          type: array
          items:
            type: string
        metadata:
          type: object
          required: [owner]
          properties:
            owner:
              type: string
      additionalProperties: false
    constraints:
      - type: unique
        key: "$.id"
    output:
      path: "out/widgets.hcl"
      format: hcl
```

### Data files

`data/widget-1.hcl`:

```hcl
id      = "widget-1"
enabled = true
price   = 12.5
tags    = ["hardware", "featured"]
metadata = {
  owner = "team-a"
}
```

`data/widget-2.hcl`:

```hcl
id      = "widget-2"
enabled = false
price   = 8
tags    = []
metadata = {
  owner = "team-b"
}
```

Run `datacur8 validate` to check the converted objects against the schema and enforce unique widget IDs. Run `datacur8 export` to create `out/widgets.hcl`:

```hcl
widgets = [{
  enabled = true
  id      = "widget-1"
  metadata = {
    owner = "team-a"
  }
  price = 12.5
  tags  = ["hardware", "featured"]
  }, {
  enabled = false
  id      = "widget-2"
  metadata = {
    owner = "team-b"
  }
  price = 8
  tags  = []
}]
```

Exports order items by input file path and sort object keys. The output wraps the items in a `widgets` array, just as JSON and YAML exports do. To use JSON output instead, set `output.format: json` and `output.path: out/widgets.json`; the input files and schema stay the same.

Top-level HCL blocks, variables, attribute references, and functions are unsupported. Use attributes with literal values or self-contained expressions; see [HCL input](/configuration#hcl-input). `datacur8 tidy --write` formats these files while preserving comments and attribute order.

## CSV Product Catalog

This example validates a CSV product catalog with schema-guided type conversion.

### Directory structure

```
.datacur8
data/
  products.csv
  categories.csv
```

### Configuration

```yaml
version: "0.0.0"

types:
  - name: category
    input: csv
    match:
      include:
        - "^data/categories\\.csv$"
    schema:
      type: object
      required: ["id", "name"]
      properties:
        id: { type: string }
        name: { type: string }
    constraints:
      - type: unique
        key: "$.id"
    output:
      path: "out/categories.json"
      format: json

  - name: product
    input: csv
    match:
      include:
        - "^data/products\\.csv$"
    schema:
      type: object
      required: ["sku", "name", "price", "category_id", "active"]
      properties:
        sku: { type: string }
        name: { type: string }
        price: { type: number }
        category_id: { type: string }
        active: { type: boolean }
    constraints:
      - type: unique
        key: "$.sku"
      - type: foreign_key
        key: "$.category_id"
        references:
          type: category
          key: "$.id"
    output:
      path: "out/products.json"
      format: json
```

### Data files

`data/categories.csv`:

```csv
id,name
electronics,Electronics
clothing,Clothing
```

`data/products.csv`:

```csv
sku,name,price,category_id,active
LAPTOP-001,Gaming Laptop,1299.99,electronics,true
TSHIRT-001,Cotton T-Shirt,19.99,clothing,true
PHONE-001,Smartphone,799.00,electronics,false
```

### What is validated

- All CSV headers match schema properties
- All required columns are present
- `price` is converted to a number
- `active` is converted to a boolean
- Each product's `category_id` references an existing category

## Strict Mode

Strict mode prevents undeclared properties from appearing in data files.

### ENABLED mode

With `strict_mode: ENABLED`, any object schema that doesn't explicitly set `additionalProperties` is treated as `additionalProperties: false`.

```yaml
version: "0.0.0"
strict_mode: ENABLED

types:
  - name: config
    input: json
    match:
      include:
        - "^settings/.*\\.json$"
    schema:
      type: object
      required: ["name"]
      properties:
        name: { type: string }
        tags:
          type: object
          properties:
            env: { type: string }
          # No additionalProperties set — strict mode adds false here
```

With `ENABLED`, the following file would **fail** because `tags.region` is not declared:

```json
{
  "name": "prod",
  "tags": {
    "env": "production",
    "region": "us-east-1"
  }
}
```

### FORCE mode

With `strict_mode: FORCE`, even schemas that explicitly allow additional properties have it overridden:

```yaml
version: "0.0.0"
strict_mode: FORCE

types:
  - name: config
    input: json
    match:
      include:
        - "^settings/.*\\.json$"
    schema:
      type: object
      required: ["name"]
      properties:
        name: { type: string }
        metadata:
          type: object
          properties:
            version: { type: string }
          additionalProperties: true  # FORCE overrides this to false
```

## Multi-Format Export

This example shows exporting the same data in different formats.

### JSON export

```yaml
output:
  path: "out/teams.json"
  format: json
```

Produces:

```json
{
  "team": [
    { "id": 1, "name": "Team Alpha" },
    { "id": 2, "name": "Team Beta" }
  ]
}
```

### YAML export

```yaml
output:
  path: "out/teams.yaml"
  format: yaml
```

Produces:

```yaml
team:
  - id: 1
    name: Team Alpha
  - id: 2
    name: Team Beta
```

### JSONL export

```yaml
output:
  path: "out/services.jsonl"
  format: jsonl
```

Produces:

```
{"id":1,"name":"API Gateway"}
{"id":2,"name":"Billing Service"}
```

Each line is a minified JSON object. Items are ordered by file path for deterministic output.

## Export JSON products to CSV

The working fixture is [`tests/example_examples_csv_export_success`](https://github.com/UnitVectorY-Labs/datacur8/tree/main/tests/example_examples_csv_export_success).

`.datacur8`:

```yaml
version: "0.0.0"
types:
  - name: products
    input: json
    match:
      include: ['^data/.*\.json$']
    schema:
      type: object
      properties:
        name: {type: string}
        price: {type: number}
        stock: {type: integer}
      required: [name, price, stock]
      additionalProperties: false
    output:
      path: out/products.csv
      format: csv
```

`data/1.json`:

```json
{"name":"Widget","price":12.5,"stock":3}
```

Run `datacur8 validate`, then `datacur8 export`. `out/products.csv` contains:

```csv
name,price,stock
Widget,12.5,3
```

Columns are alphabetically sorted from the schema rather than the first item. JSON, YAML, HCL, and CSV inputs can all produce this shape. To read it back, use `input: csv` with the same schema and a match pattern for the exported CSV, and select a different output path (configured outputs are excluded from discovery).

Every declared column must be present in every item. Empty strings are valid; absent/null cells, extra undeclared keys, and structured values cannot be represented. Unsupported schemas fail with exit 1, source validation errors with exit 2, and export conversion/write errors with exit 3. See [CSV export](/configuration#csv-export) for the supported subset, precision, and round-trip limits.

## TSV catalog with CSV export

The working fixture is [`tests/example_examples_tsv_catalog_success`](https://github.com/UnitVectorY-Labs/datacur8/tree/main/tests/example_examples_tsv_catalog_success).
It covers integers, numbers, booleans, empty strings, quoted tabs/newlines and
quotes, uniqueness, and extension matching.

```yaml
version: "0.0.0"
types:
  - name: items
    input: csv
    csv: {delimiter: "\t"}
    match:
      include: ['^data/.*\.tsv$']
    schema:
      type: object
      properties:
        title: {type: string}
        id: {type: integer}
        active: {type: boolean}
        ratio: {type: number}
        ext: {type: string}
      required: [title, id, active, ratio, ext]
      additionalProperties: false
    constraints:
      - type: unique
        key: $.id
      - type: path_equals_attr
        path_selector: path.ext
        references: {key: $.ext}
    output:
      path: out/items.csv
      format: csv
```

Place CSV-style tab-delimited records in `data/items.tsv`. For example, this
table shows the columns and scalar values (use actual tabs between fields):

| title | id | active | ratio | ext |
| --- | --- | --- | --- | --- |
| Example | 1 | true | 1.25 | tsv |

Run `datacur8 validate`, `datacur8 export`, and `datacur8 tidy --write`.
Export writes comma-delimited `out/items.csv` with alphabetically sorted
columns. Tidy keeps the input tab-delimited and aligns cells with the sorted
headers. Add `csv: {delimiter: "\t"}` under `output` for TSV export instead.
The input/output delimiters default independently to comma.
See [CSV delimiters and TSV](/configuration#csv-delimiters-and-tsv) for quoting,
permitted separators, empty data, conversion limits, and errors.

## TOML catalog

This TOML 1.1.0 example is covered by `tests/example_examples_toml_catalog_success`. Each file is one record; nested tables stay within that record.

`.datacur8`:

```yaml
version: "0.0.0"
types:
  - name: records
    input: toml
    match:
      include: ['^data/.*\.toml$']
    schema:
      type: object
      required: [id]
      properties:
        id: {type: string}
    output:
      format: toml
      path: out/nested/records.toml
    constraints:
      - type: unique
        key: "$.id"
```

`data/1.toml`:

```toml
id = "widget-1"
released = 2026-10-07
[owner]
team = "platform"
```

`data/2.toml`:

```toml
id = "widget-2"
released = 2026-10-08
[owner]
team = "platform"
```

Run `datacur8 validate`, then `datacur8 export`. The generated `out/nested/records.toml` contains:

```toml
records = [{id = 'widget-1', owner = {team = 'platform'}, released = '2026-10-07'}, {id = 'widget-2', owner = {team = 'platform'}, released = '2026-10-08'}]
```

Native dates become strings during validation/export. The aggregate is a single wrapper object, distinct from individual input records. `datacur8 tidy --write` retains native dates but removes comments and sorts keys. See [TOML conversion rules](/configuration#toml-input) for temporal precision, exact integer limits, null rejection, and formatting policy.
