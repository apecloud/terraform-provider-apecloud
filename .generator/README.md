# Terraform Provider Code Generator

The goal of this sub-project is to automatically generate the scaffolding to create Terraform Plugin Framework models and schemas for ApeCloud resources and data sources based on an OpenAPI specification.

> \[!CAUTION]
> This code is HIGHLY experimental and should stabilize over the next weeks/months. As such this code is NOT intended for production uses.
> Any code that has been generated should and needs to be proofread by a human.

## How to use

### Requirements

- This project
- Poetry
- An OpenApi 3.0.x specification
- Go

### Install dependencies

Install the necessary python dependencies by running:

```sh
poetry install
```

Install go as we use the `goimports` and `go fmt` commands on the generated files to format them and manage imports.

### Marking the resources to be generated

The generator reads a configuration file in order to generate the appropriate resources and datasources.
The configuration file should look like the following:

```yaml
resources:
  { resource_name }:
    read:
      method: { read_method }
      path: { read_path }
    create:
      method: { create_method }
      path: { create_path }
    update:
      method: { update_method }
      path: { update_path }
    delete:
      method: { delete_method }
      path: { delete_path }
  ...

datasources:
  { datasource_name }:
    singular: { get_one_path }
    plural: { get_all_path }
  ...
```

- Resources
  - `resource_name` is the name of the resource to be generated.
  - `xxx_method` should be the HTTP method used by the relevant route
  - `xxx_path` should be the HTTP route of the resource's CRUD operation
- Datasources
  - `datasource_name` is the name of the datasource to be generated.
  - `get_one_path` should be the api route to get a singular item relevant to the datasource
  - `get_all_path` should be the api route to get a list of items relevant to the datasource

> \[!NOTE]
> An example using the `clusters` resource and datasource would look like this:
>
> ```yaml
> resources:
>   clusters:
>     read:
>       method: get
>       path: /api/v1/organizations/{orgName}/clusters/{name}
>     create:
>       method: post
>       path: /api/v1/organizations/{orgName}/clusters
>     update:
>       method: patch
>       path: /api/v1/organizations/{orgName}/clusters/{name}
>     delete:
>       method: delete
>       path: /api/v1/organizations/{orgName}/clusters/{name}
> datasources:
>   clusters:
>     singular: /api/v1/organizations/{orgName}/clusters/{name}
>     plural: /api/v1/organizations/{orgName}/clusters
> ```

### Running the generator

You can use the Makefile command from the project root to trigger the code generation automatically:

```sh
  $ make generate
```

Or run it manually from inside `.generator/src`:

```sh
  $ PYTHONPATH=. python -m generator.cli ../specs/adminapi.yaml ../configuration.yaml
```

> \[!NOTE]
> The generated Go models and schemas will be placed in `internal/types/`, `internal/resource/`, and `internal/datasource/` directories.

## OpenAPI Custom Extensions

The code generator supports several custom `x-terraform-*` extensions in the OpenAPI specification to handle complex edge cases in Terraform provider development. You can add these extensions directly to the schema definitions in your OpenAPI file.

### `x-terraform-type: dynamic`
**Purpose**: Resolves recursive schemas or deeply nested dynamic structures (e.g., self-referencing `children` arrays).
**Behavior**: When applied, the generator maps the property to Terraform's `DynamicType` instead of generating a strict nested schema. This treats the recursive node as an opaque JSON block, preventing `RecursionError` during Python code generation and keeping the Terraform schema valid.

### `x-terraform-ignore-update: true`
**Purpose**: Prevents Terraform state drift for fields that the API might modify but shouldn't trigger a Terraform plan diff.
**Behavior**: The generator emits `Optional: true` (without `Computed: true`) for the field and completely skips assigning this field in the auto-generated `RefreshFromAPI` method. This preserves the user's locally configured value (from `main.tf`) even if the remote API returns a different value.

### `x-terraform-force-computed: true`
**Purpose**: Enforces strict read-only behavior for server-generated fields.
**Behavior**: The generator emits *only* `Computed: true` for the field (removing `Optional: true` and `Required: true`). This explicitly tells Terraform that the value is entirely controlled by the API, preventing "inconsistent result after apply" errors. Users must remove these fields from their `main.tf` configuration to avoid "Invalid Configuration for Read-Only Attribute" errors.

