# Terraform Provider for KBCloud

This is the official Terraform Provider for [KBCloud](https://apecloud.cn/)(KubeBlocks Enterprise), allowing you to manage databases, clusters, backups, and other KBCloud resources via infrastructure as code.

The provider is built using the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (to build the provider plugin)
- [Python 3](https://www.python.org/downloads/) & [Poetry](https://python-poetry.org/) (for code generation)
- [KubeBlocks Enterprise](https://kubeblocks.com/products/kubeblocks-enterprise) >= 2.2.0

## Quick Start

### 1. Build the Provider

Clone this repository and build the provider locally:

```sh
# Build the provider executable
make build
```

This will create the `terraform-provider-kbcloud` binary in your workspace.

### 2. Configure Local Development (`dev_overrides`)

To test the provider locally without publishing it to the Terraform Registry, you need to configure Terraform's `dev_overrides`.

Create or edit your `~/.terraformrc` (or `%APPDATA%\terraform.rc` on Windows) to point to the directory where you built the binary:

```hcl
provider_installation {
  dev_overrides {
    # Point this to the ABSOLUTE path of the directory containing the built binary
    "registry.terraform.io/apecloud/kbcloud" = "/Users/your_username/path/to/terraform-provider-kbcloud"
  }
  # For all other providers, install them directly from their origin provider registries as normal
  direct {}
}
```

### 3. Usage Example

Create a `main.tf` file (e.g., in the `example/mysql/` directory) and configure the provider:

```hcl
terraform {
  required_providers {
    kbcloud = {
      source = "registry.terraform.io/apecloud/kbcloud"
    }
  }
}

provider "kbcloud" {
  api_url    = "https://kb-cloud-apiserver-endpoint.com/api" # or your target API endpoint
  api_key    = "your_api_key"
  api_secret = "your_api_secret"
}

resource "kbcloud_cluster" "my_mysql" {
  name             = "my-mysql-cluster"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "mysql"
  version          = "8.0.44"
  mode             = "replication"
  cluster_type     = "Normal"
  # ... Add components and other configurations ...
}
```

Then, you can run the standard Terraform workflow:

```sh
terraform plan
terraform apply
terraform destroy
```

*(Note: Because of* *`dev_overrides`,* *`terraform init`* *is not required for the* *`kbcloud`* *provider and may show warnings).*

## Development & Code Generation

This provider heavily utilizes an OpenAPI-based Python code generator to automatically create Terraform models and schemas.

### Generating Code

When the OpenAPI spec changes or you modify the generation templates, you should run the generator:

```sh
# Generate Go models and schemas, and automatically format them
make generate
```

This command runs the Python CLI under `.generator/src/` to parse `.generator/specs/adminapi-bundle-tmp.yaml` and updates the files inside:

- `internal/types/`
- `internal/resource/`
- `internal/datasource/`

For more details on how the generator works, see the [Generator README](.generator/README.md).

### Debugging with Delve

For debugging the provider, refer to the official Terraform Plugin Framework documentation on [Debugging Providers](https://developer.hashicorp.org.cn/terraform/plugin/debugging#debugging-providers).

## License

Please see the LICENSE file for details.
