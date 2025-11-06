# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
