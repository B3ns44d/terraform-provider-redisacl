# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.3] - 2025-12-03

### Added
- **Valkey Backend Support**: Provider now supports Valkey as an alternative backend to Redis
  - New `use_valkey` boolean configuration option to enable Valkey backend (default: `false`)
  - Full compatibility with all existing ACL management operations (create, read, update, delete)
  - Support for Valkey standalone and cluster deployments
  - TLS support for secure Valkey connections
  - Unified client interface abstracts differences between Redis and Valkey backends
  - Comprehensive acceptance test suite for Valkey backend validation
- **User Existence Check**: Added validation to prevent accidental overwrite of existing ACL users
  - Provider now checks if a user already exists before creation
  - Prevents unintentional modification of manually created or externally managed users
  - Improves safety when managing ACL users in shared Redis/Valkey environments

### Changed
- Refactored client architecture to use `UniversalClient` interface for backend abstraction
- Provider maintains full backward compatibility - existing Redis configurations work unchanged

### Documentation
- Added Valkey configuration examples for standalone, cluster, and TLS setups
- Documented Valkey backend limitations (Sentinel mode not currently supported)
- Updated provider schema documentation with `use_valkey` attribute

### Dependencies
- Added `github.com/valkey-io/valkey-glide/go/v2` for Valkey client support

### Important Notes
- **No Breaking Changes**: Existing Redis configurations continue to work without modification
- **Default Behavior**: Provider uses Redis backend by default when `use_valkey` is omitted or set to `false`
- **Valkey Limitation**: Sentinel configuration is not supported with Valkey backend - use standalone or cluster modes

## [1.0.2] - 2025-11-07

### Fixed
- Fixed Terraform state drift when commands attribute doesn't include `-@all` prefix
  - Provider now intelligently handles `-@all` prefix to prevent unnecessary updates
  - State comparison logic updated to recognize equivalent command configurations
- Improved ACL command rule building to avoid duplicate `-@all` prefixes

### Added
- Comprehensive test suite with HIGH and MEDIUM priority tests
  - **Unit Tests**: 13 test cases for helper functions covering:
    - ACL rule building with various parameter combinations
    - Commands with/without `-@all` prefix handling
    - Multiple passwords, keys, and channel patterns
    - Selectors and complex command combinations
    - Edge cases (null values, empty strings)
  - **Acceptance Tests**: 10 new integration tests covering:
    - Commands drift detection (with/without `-@all` prefix)
    - Multiple password management and rotation
    - Disabled user state management
    - Nopass user handling
    - Complex command combinations (@read, @write, etc.)
    - Multiple key and channel patterns
    - State drift detection and correction
- Helper function `ModifyUserInRedis()` for drift testing

### Changed
- Updated version in GNUmakefile to 1.0.2
- Enhanced test coverage for critical provider functionality

### Documentation
- Added important limitation notice about Redis ACL replication in Sentinel setups
- Clarified that ACL users are not automatically replicated to replica nodes during failover
- Recommended using Redis Cluster for high-availability scenarios requiring ACL persistence

## [1.0.1] - 2025-11-06

### Added
- Generated comprehensive documentation for Terraform Registry
- Provider, resource, and data source documentation with examples
- Auto-generated schema documentation using terraform-plugin-docs

### Fixed
- GoReleaser checksum file naming to follow Terraform Registry requirements
- GPG signing configuration for release artifacts
- Workflow improvements for better release reliability

### Changed
- Removed post-release automation tasks for cleaner release process
- Updated release workflow to handle proper checksum verification

## [1.0.0] - 2025-11-04

### Added

**Core Provider Features:**
- Initial release of the Redis ACL Terraform Provider
- Support for managing Redis ACL users with comprehensive configuration options
- Data sources for reading Redis ACL user and users information
- Full Terraform lifecycle management (Create, Read, Update, Delete, Import)

**Resources:**
- `redisacl_user` - Manage Redis ACL users with support for:
  - User enable/disable state
  - Key patterns and access controls
  - Channel patterns for pub/sub access
  - Command restrictions and permissions
  - Password management (single and multiple passwords)
  - Selector-based permissions (where supported)

**Data Sources:**
- `redisacl_user` - Read information about a specific Redis ACL user
- `redisacl_users` - Read information about all Redis ACL users

**Provider Configuration:**
- Flexible Redis connection options (address, username, password, database)
- TLS support with certificate validation options
- Connection pooling and timeout configuration
- Support for Redis 6.0+ ACL features

**Testing & Quality:**
- Comprehensive unit test suite with 18.3% coverage
- Full acceptance test suite with 59.1% coverage using testcontainers
- Automated integration testing with Redis 6.2, 7.0, and 7.2
- golangci-lint integration with minimal, high-value linter set
- Cross-platform builds (Linux, macOS, Windows on amd64/arm64)

**Documentation:**
- Complete provider documentation with examples
- Resource and data source reference documentation
- Usage examples for common scenarios
- Terraform Registry integration

**CI/CD & Release:**
- GitHub Actions workflows for continuous integration
- Automated testing across multiple Go and Terraform versions
- GoReleaser configuration for multi-platform releases
- GPG signing for security and Terraform Registry compliance
- Automated Terraform Registry publishing

### Technical Details

**Supported Platforms:**
- Linux (amd64, 386, arm, arm64)
- macOS (amd64, arm64)
- Windows (amd64, 386, arm64)
- FreeBSD (amd64, 386, arm, arm64)

**Compatibility:**
- Terraform >= 1.0
- Go 1.23+
- Redis 6.0+ (ACL support required)

**Dependencies:**
- terraform-plugin-framework v1.x
- go-redis/v9 for Redis connectivity
- testcontainers-go for integration testing

### Examples

Basic usage:
```hcl
terraform {
  required_providers {
    redisacl = {
      source  = "B3ns44d/redisacl"
      version = "~> 1.0.0"
    }
  }
}

provider "redisacl" {
  address = "localhost:6379"
}

resource "redisacl_user" "app_user" {
  name     = "app_user"
  enabled  = true
  keys     = "~app:*"
  channels = "&notifications:*"
  commands = "-@all +get +set +del"
  passwords = ["secure_password"]
}
```

## [0.1.0] - Development

### Added
- Initial project structure and development setup
