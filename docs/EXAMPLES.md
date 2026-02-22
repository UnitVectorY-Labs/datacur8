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
version: "1.0.0"
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
version: "1.0.0"

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
    csv:
      delimiter: ","
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
    csv:
      delimiter: ","
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
version: "1.0.0"
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
version: "1.0.0"
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
