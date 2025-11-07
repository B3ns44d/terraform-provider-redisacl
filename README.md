# Terraform Provider for Redis ACL

[![Go Report Card](https://goreportcard.com/badge/github.com/B3ns44d/terraform-provider-redisacl)](https://goreportcard.com/report/github.com/B3ns44d/terraform-provider-redisacl)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

A comprehensive [Terraform](https://www.terraform.io/) provider for managing Redis Access Control Lists (ACLs). This provider supports standalone, Sentinel, and Cluster Redis deployments with full TLS support.

## Features

- **Complete ACL Management**: Create, read, update, and delete Redis ACL users
- **Multiple Deployment Types**: Supports Standalone, Sentinel, and Cluster Redis
- **Security First**: Full TLS and mutual TLS (mTLS) support
- **Data Sources**: Read individual users or list all users
- **Comprehensive Testing**: 14+ test cases with testcontainers-go integration
- **Production Ready**: Built with Terraform Plugin Framework v1.15+

## Quick Start

### Installation

#### Option 1: Terraform Registry (Recommended)

```hcl
terraform {
  required_providers {
    redisacl = {
      source  = "B3ns44d/redisacl"
      version = "~> 0.1.0"
    }
  }
}
```

#### Option 2: Local Development

```bash
# Clone the repository
git clone https://github.com/B3ns44d/terraform-provider-redisacl.git
cd terraform-provider-redisacl

# Build and install locally
make install
```

### Basic Usage

```hcl
# Configure the provider
provider "redisacl" {
  address  = "localhost:6379"
  password = "your-redis-password"
}

# Create a read-only user
resource "redisacl_user" "readonly" {
  name      = "readonly-app"
  enabled   = true
  passwords = ["secure-password"]
  keys      = "~app:*"
  commands  = "+@read -@write"
}

# Create an admin user
resource "redisacl_user" "admin" {
  name     = "admin-user"
  enabled  = true
  passwords = ["admin-password"]
  keys     = "~*"
  channels = "&*"
  commands = "+@all"
}
```

## Documentation

### Provider Configuration

The provider supports multiple Redis deployment types and connection options:

#### Standalone Redis

```hcl
provider "redisacl" {
  address  = "redis.example.com:6379"
  username = "default"
  password = "your-password"
}
```

#### Redis with TLS

```hcl
provider "redisacl" {
  address  = "secure-redis.example.com:6380"
  password = "your-password"
  use_tls  = true
  
  # Optional: For self-signed certificates (testing only)
  tls_insecure_skip_verify = true
}
```

#### Redis with Mutual TLS

```hcl
provider "redisacl" {
  address = "mtls-redis.example.com:6380"
  use_tls = true
  
  tls_ca_cert = file("ca.pem")
  tls_cert    = file("client.crt")
  tls_key     = file("client.key")
}
```

#### Redis Sentinel

```hcl
provider "redisacl" {
  username = "default"
  password = "master-password"
  
  sentinel {
    master_name = "mymaster"
    addresses   = [
      "sentinel1.example.com:26379",
      "sentinel2.example.com:26379",
      "sentinel3.example.com:26379"
    ]
    username = "sentinel-user"     # Optional
    password = "sentinel-password" # Optional
  }
}
```

> **⚠️ Important Limitation:** Redis does not automatically replicate ACL users to replica nodes. In Sentinel setups, when a failover occurs, the newly promoted master will not have the ACL users created by this provider. For high-availability scenarios requiring ACL persistence across failovers, consider using Redis Cluster instead or implementing external ACL synchronization mechanisms.

#### Redis Cluster

```hcl
provider "redisacl" {
  username = "cluster-user"
  password = "cluster-password"
  
  cluster {
    addresses = [
      "node1.example.com:6379",
      "node2.example.com:6379",
      "node3.example.com:6379"
    ]
  }
}
```

### Resources

#### `redisacl_user`

Manages a Redis ACL user with full lifecycle support.

```hcl
resource "redisacl_user" "example" {
  name     = "app-user"
  enabled  = true
  passwords = ["password1", "password2"]
  
  # Key access patterns
  keys = "~app:* ~cache:*"
  
  # Pub/Sub channel access
  channels = "&notifications:*"
  
  # Command permissions
  commands = "+@read +@write -@dangerous"
  
  # Advanced: Selectors for complex permissions
  selectors = [
    "~temp:* +@read",
    "~logs:* +@write"
  ]
  
  # Allow self-modification (use with caution)
  allow_self_mutation = false
}
```

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | ✅ | The name of the user |
| `enabled` | bool | ❌ | Whether the user is enabled (default: `true`) |
| `passwords` | list(string) | ❌ | List of passwords for the user |
| `keys` | string | ❌ | Key patterns (space-separated, default: `~*`) |
| `channels` | string | ❌ | Channel patterns (space-separated, default: `&*`) |
| `commands` | string | ❌ | Command permissions (space-separated, default: `+@all`) |
| `selectors` | list(string) | ❌ | Advanced permission selectors |
| `allow_self_mutation` | bool | ❌ | Allow modifying the currently authenticated user |

### Data Sources

#### `redisacl_user`

Read information about a specific user:

```hcl
data "redisacl_user" "existing" {
  name = "existing-user"
}

output "user_permissions" {
  value = {
    enabled  = data.redisacl_user.existing.enabled
    keys     = data.redisacl_user.existing.keys
    commands = data.redisacl_user.existing.commands
  }
}
```

#### `redisacl_users`

List all users:

```hcl
data "redisacl_users" "all" {}

output "all_usernames" {
  value = [for user in data.redisacl_users.all.users : user.name]
}

output "enabled_users" {
  value = [
    for user in data.redisacl_users.all.users : user.name
    if user.enabled
  ]
}
```

## Development

### Prerequisites

- [Go](https://golang.org/doc/install) 1.21+
- [Terraform](https://www.terraform.io/downloads.html) 1.0+
- [Docker](https://docs.docker.com/get-docker/) (for testing)
- [Make](https://www.gnu.org/software/make/)

### Setup

```bash
# Clone the repository
git clone https://github.com/B3ns44d/terraform-provider-redisacl.git
cd terraform-provider-redisacl

# Install dependencies
go mod download

# Build the provider
make build

# Install locally for development
make install
```

### Available Make Targets

```bash
make build      # Build the provider binary
make install    # Install provider locally for development
make test       # Run unit tests
make testacc    # Run acceptance tests (requires Docker)
make lint       # Run linter
make fmt        # Format code
make generate   # Generate documentation
```

## Testing

This provider includes a comprehensive test suite with **14+ test cases** covering all functionality:

### Test Categories

#### Resource Tests (`resource_acl_user_test.go`)
- ✅ **TestAccACLUserResource_Create** - Basic user creation
- ✅ **TestAccACLUserResource_Read** - User reading and import
- ✅ **TestAccACLUserResource_Update** - In-place updates
- ✅ **TestAccACLUserResource_ForceReplaceOnNameChange** - Name change handling
- ✅ **TestAccACLUserResource_Delete** - User deletion
- ✅ **TestAccACLUserResource_ImportState** - State import functionality
- ✅ **TestAccACLUserResource_WithPassword** - Password management
- ✅ **TestAccACLUserResource_InvalidConfig** - Error handling

#### Data Source Tests (`datasource_acl_user_test.go`)
- ✅ **TestAccACLUserDataSource_Read** - Individual user lookup
- ✅ **TestAccACLUserDataSource_NotFound** - Error handling
- ⏭️ **TestAccACLUserDataSource_WithSelectors** - Advanced selectors (skipped for Redis 7 compatibility)

#### Bulk Data Source Tests (`datasource_acl_users_test.go`)
- ✅ **TestAccACLUsersDataSource_ReadAll** - List all users
- ✅ **TestAccACLUsersDataSource_WithResources** - Integration with resources
- ✅ **TestAccACLUsersDataSource_Empty** - Empty state handling
- ✅ **TestAccACLUsersDataSource_UserAttributes** - Attribute validation

#### Unit Tests (`helpers_test.go`)
- ✅ **TestParseACLUser** - ACL parsing logic

### Running Tests

#### Unit Tests
```bash
make test
```

#### Acceptance Tests (with Docker)
```bash
# Requires Docker for testcontainers
make testacc

# Or run specific tests
TF_ACC=1 go test -v ./internal/provider -run TestAccACLUserResource_Create
```

### Test Infrastructure

The test suite uses **testcontainers-go** for automated Redis container management:

- **Automated Container Lifecycle** - Redis containers are automatically started/stopped
- **Test Isolation** - Each test gets a clean Redis environment
- **Automatic Cleanup** - Containers and test data are cleaned up automatically
- **Fast Execution** - Parallel test execution with proper isolation

### Test Configuration

Tests use a dedicated Redis container with:
- Redis 7 Alpine image
- Password authentication (`testpass`)
- Automatic port mapping
- Health checks and readiness verification

## Examples

Check out the [`examples/`](./examples/) directory for complete usage examples:

- **Basic Usage** - Simple user management
- **Advanced Permissions** - Complex ACL rules
- **Multiple Environments** - Different Redis deployments
- **Data Sources** - Reading existing users

### Example: Multi-Environment Setup

```hcl
# Development environment
provider "redisacl" {
  alias   = "dev"
  address = "localhost:6379"
}

# Production environment with TLS
provider "redisacl" {
  alias    = "prod"
  address  = "prod-redis.example.com:6380"
  password = var.redis_password
  use_tls  = true
}

# Create users in both environments
resource "redisacl_user" "app_user_dev" {
  provider = redisacl.dev
  name     = "app-user"
  enabled  = true
  passwords = ["dev-password"]
  commands = "+@all"  # Permissive for development
}

resource "redisacl_user" "app_user_prod" {
  provider = redisacl.prod
  name     = "app-user"
  enabled  = true
  passwords = [var.app_password]
  keys     = "~app:*"
  commands = "+@read +@write -@dangerous"  # Restrictive for production
}
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on:

- Code style and standards
- Testing requirements
- Pull request process
- Issue reporting

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Run the test suite (`make test && make testacc`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: Check the [examples](./examples/) directory
- **Bug Reports**: [Open an issue](https://github.com/B3ns44d/terraform-provider-redisacl/issues)
- **Feature Requests**: [Start a discussion](https://github.com/B3ns44d/terraform-provider-redisacl/discussions)
- **Questions**: Use [GitHub Discussions](https://github.com/B3ns44d/terraform-provider-redisacl/discussions)

## Acknowledgments

- [HashiCorp Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
- [go-redis](https://github.com/redis/go-redis) - Redis client for Go
- [testcontainers-go](https://github.com/testcontainers/testcontainers-go) - Integration testing with Docker

---

**Made with ❤️ for the Terraform and Redis communities**