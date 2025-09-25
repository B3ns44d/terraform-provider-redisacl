# Terraform Provider for Redis ACLs

This is a Terraform provider for managing Redis ACLs in self-hosted Redis 6+ instances. It allows you to declaratively create, update, delete, and import Redis ACL users.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23
- Redis 6+

## Building The Provider

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the Go `install` command:

```shell
go install
```

## Using the Provider

To use the provider, add the following to your Terraform configuration:

```hcl
terraform {
  required_providers {
    redisacl = {
      source  = "B3ns44d/redisacl"
      version = "0.1.0"
    }
  }
}

provider "redisacl" {
  # Configuration options
}
```

### Provider Configuration

The provider supports three connection modes: single instance, Sentinel, and Cluster.

#### Single Instance

```hcl
provider "redisacl" {
  address  = "localhost:6379"
  username = "myuser"
  password = "mypassword"
  use_tls  = false
}
```

#### Sentinel

```hcl
provider "redisacl" {
  sentinel {
    master_name = "mymaster"
    addresses   = ["localhost:26379", "localhost:26380"]
    username    = "myuser"
    password    = "mypassword"
  }
  use_tls = false
}
```

#### Cluster

```hcl
provider "redisacl" {
  cluster {
    addresses = ["localhost:7000", "localhost:7001"]
    username  = "myuser"
    password  = "mypassword"
  }
  use_tls = false
}
```

### Resource: `redisacl_user`

This resource manages a Redis ACL user.

#### Example Usage

```hcl
resource "redisacl_user" "example" {
  name      = "example-user"
  enabled   = true
  passwords = ["password123"]
  keys      = "~*"
  channels  = "&*"
  commands  = "+@all -@dangerous"
}
```

### Data Source: `redisacl_user`

This data source retrieves information about a single Redis ACL user.

#### Example Usage

```hcl
data "redisacl_user" "example" {
  name = "example-user"
}
```

### Data Source: `redisacl_users`

This data source retrieves information about all Redis ACL users.

#### Example Usage

```hcl
data "redisacl_users" "all" {}
```