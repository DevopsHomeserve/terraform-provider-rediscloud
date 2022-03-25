---
layout: "rediscloud"
page_title: "Redis Cloud: rediscloud_subscription"
description: |-
  Subscription resource in the Terraform provider Redis Cloud.
---

# Resource: rediscloud_subscription

Creates a Database within your Redis Enterprise Cloud Account.
This resource is responsible for creating databases. 
This allows Redis Enterprise Cloud to provision your databases in the most efficient way.


Due to the limitations mentioned above, the differences shown by Terraform will be different from normal plan.
When an argument has been changed on a nested database - for example changing the `memory_limit_in_gb` from 1 to 2, Terraform
will display the resource as being modified with the database as being removed, and a new one added. As the resource
identifies the database based on the name, the only change that would happen would be to update the database to increase
the memory limit. Below is the Terraform output for changing the `memory_limit_in_gb` for a single database within a
subscription.


## Example Usage

```hcl

data "rediscloud_subscription" "example" {
  name = "My Example Subscription"
}

resource "random_password" "password" {
  length = 20
  upper = true
  lower = true
  number = true
  special = false
}

resource "rediscloud_database" "example" {
  subscription_id = data.rediscloud_subscription.example.id
  name = "tf-example-database"
  protocol = "redis"
  memory_limit_in_gb = 1
  data_persistence = "none"
  throughput_measurement_by = "operations-per-second"
  throughput_measurement_value = 10000
  password = random_password.password.result

  alert {
    name = "dataset-size"
    value = 40
  }
  
}

```

## Argument Reference

The following arguments are supported:

* `subscription_id` - (Required) A meaningful id to identify the subscription

* `name` - (Required) A meaningful name to identify the database. Caution should be taken when changing this value - see
the top of the page for more information.
* `protocol` - (Optional) The protocol that will be used to access the database, (either ‘redis’ or 'memcached’) Default: ‘redis’
* `memory_limit_in_gb` - (Required) Maximum memory usage for this specific database
* `support_oss_cluster_api` - (Optional) Support Redis open-source (OSS) Cluster API. Default: ‘false’
* `external_endpoint_for_oss_cluster_api` - (Optional) Should use the external endpoint for open-source (OSS) Cluster API.
Can only be enabled if OSS Cluster API support is enabled. Default: 'false'
* `client_ssl_certificate` - (Optional) SSL certificate to authenticate user connections
* `periodic_backup_path` - (Optional) Path that will be used to store database backup files
* `replica_of` - (Optional) Set of Redis database URIs, in the format `redis://user:password@host:port`, that this
database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided.
Cannot be enabled when `support_oss_cluster_api` is enabled.
* `module` - (Optional) A module object, documented below
* `alert` - (Optional) Set of alerts to enable on the database, documented below
* `data_persistence` - (Optional) Rate of database data persistence (in persistent storage). Default: ‘none’
* `password` - (Required) Password used to access the database
* `replication` - (Optional) Databases replication. Default: ‘true’
* `throughput_measurement_by` - (Required) Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)
* `throughput_measurement_value` - (Required) Throughput value (as applies to selected measurement method)
* `average_item_size_in_bytes` - (Optional) Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes)
of the items stored in the database. Default: 1000
* `source_ips` - (Optional) Set of CIDR addresses to allow access to the database. Defaults to allowing traffic.
* `hashing_policy` - (Optional) List of regular expression rules to shard the database by. See
[the documentation on clustering](https://docs.redislabs.com/latest/rc/concepts/clustering/) for more information on the
hashing policy. This cannot be set when `support_oss_cluster_api` is set to true.

The cloud_provider `region` block supports:

* `region` - (Required) Deployment region as defined by cloud provider
* `multiple_availability_zones` - (Optional) Support deployment on multiple availability zones within the selected region. Default: ‘false’
* `networking_deployment_cidr` - (Required) Deployment CIDR mask.
* `networking_vpc_id` - (Optional) Either an existing VPC Id (already exists in the specific region) or create a new VPC
(if no VPC is specified). VPC Identifier must be in a valid format (for example: ‘vpc-0125be68a4625884ad’) and existing
within the hosting account.
* `preferred_availability_zones` - (Required) Availability zones deployment preferences (for the selected provider & region).

~> **Note:** The preferred_availability_zones parameter is required for Terraform, but is optional within the Redis Enterprise Cloud UI. 
This difference in behaviour is to guarantee that a plan after an apply does not generate differences.

The database `alert` block supports:

* `name` (Required) Alert name
* `value` (Required) Alert value

The database `module` block supports:

* `name` (Required) Name of the module to enable

### Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 mins) Used when creating the subscription
* `update` - (Defaults to 30 mins) Used when updating the subscription
* `delete` - (Defaults to 10 mins) Used when destroying the subscription

## Attribute reference

The `database` block has these attributes:

* `db_id` - Identifier of the database created
* `public_endpoint` - Public endpoint to access the database
* `private_endpoint` - Private endpoint to access the database

The `region` block has these attributes:

* `networks` - List of generated network configuration

The `networks` block has these attributes:

* `networking_subnet_id` - The subnet that the subscription deploys into
* `networking_deployment_cidr` - Deployment CIDR mask for the generated
* `networking_vpc_id` - VPC id for the generated network

## Import

`rediscloud_database` can be imported using the ID of the subscription, e.g.

```
$ terraform import rediscloud_database.example 12345678
```
