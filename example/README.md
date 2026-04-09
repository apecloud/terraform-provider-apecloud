# KBCloud Terraform Provider Examples

This directory contains example Terraform configurations for deploying various types of database and message queue clusters on KubeBlocks Enterprise. 

The configurations illustrate how to map KubeBlocks concepts (such as Components, Volumes, Parameter Templates, and Backup Policies) to the `kbcloud_cluster` Terraform resource.

## Supported Engines and Modes

Currently, the following examples are provided:

*   **MySQL** (`mysql`): Replication mode
*   **PostgreSQL** (`postgresql`): Replication mode
*   **Redis** (`redis`): Standalone, Replication, Sentinel, and Cluster modes
*   **MongoDB** (`mongodb`): Standalone, Replicaset, and Sharding modes
*   **Kafka** (`kafka`): Combined, Separated, and withZookeeper modes
*   **SQL Server** (`mssql`): Cluster mode

## KubeBlocks (KB) Cluster Operations via Terraform

The `kbcloud_cluster` resource translates standard Terraform lifecycle operations into KubeBlocks OpsRequest actions.

### 1. Provisioning a Cluster (`terraform apply`)

Creating a new cluster with `terraform apply` triggers the underlying `ClusterCreate` API. 
*   **Engine & Version**: Defined by the `engine` and `version` fields.
*   **Topology**: Defined by the `mode` (e.g., `standalone`, `replication`, `sharding`) and the `components` list.
*   **Storage**: Handled within each component's `volumes` block, supporting IOPS limits and storage classes.

### 2. Vertical Scaling (VScale)

To scale compute resources (CPU/Memory) or modify storage capacities, update the component specifications in your `.tf` file:
*   **Compute**: Change the `class_code` (e.g., from `...1c2g...` to `...2c4g...`).
*   **Storage**: Increase the `storage` value within a volume block (Note: decreasing storage is generally not supported by underlying cloud providers).

Running `terraform apply` will automatically issue the appropriate **Vertical Scale** or **Volume Expand** `OpsRequest` to the KubeBlocks API.

### 3. Horizontal Scaling (HScale)

To change the number of replicas or shards:
*   **Replicas**: Modify the `replicas` attribute inside the specific `component` block.
*   **Shards**: For sharded architectures (like Redis Cluster or MongoDB Sharding), modify the `comp_num` attribute.

Running `terraform apply` will issue a **Horizontal Scale** `OpsRequest`.

### 4. Parameter Reconfiguration

To apply a new parameter template to a component:
*   Update the `param_tpl_id` or `param_tpl_name` within the `param_tpls` block. 
*   Applying this change will trigger a **Reconfigure** `OpsRequest`.

### 5. Termination / Deletion (`terraform destroy`)

Executing `terraform destroy` will delete the cluster. 
*   Ensure that the `termination_policy` is set to `"Delete"` if you want the underlying resources to be completely removed. 
*   If `termination_policy` is `"DoNotTerminate"`, the provider will refuse to delete the cluster as a safety mechanism.

## Getting Started

1.  Navigate to the directory of the engine you want to provision (e.g., `cd mysql`).
2.  Open `main.tf` and fill in your `api_key`, `api_secret`, `org_name`, and `environment_name`.
3.  Run `terraform init` to download the provider.
4.  Run `terraform plan` to preview the changes.
5.  Run `terraform apply` to provision the cluster.
