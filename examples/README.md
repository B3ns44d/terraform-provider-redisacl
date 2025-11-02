# Terraform Redis ACL Provider Examples

This directory contains comprehensive examples demonstrating various use cases and configurations for the Redis ACL Terraform provider.

## 📁 Available Examples

### 🚀 [Basic Usage](./basic-usage/)
**Perfect for getting started**
- Simple user creation and management
- Different user roles (readonly, write, admin, monitoring)
- Basic ACL permissions and patterns
- Local Redis setup

**What you'll learn:**
- Creating users with different permission levels
- Using data sources to read user information
- Basic key and command patterns
- Output usage for verification

### 🌍 [Multi-Environment](./multi-environment/)
**Production-ready environment management**
- Development, staging, and production configurations
- Environment-specific security policies
- Variable-driven configuration
- Progressive security hardening

**What you'll learn:**
- Managing multiple Redis environments
- Environment-specific user permissions
- Using Terraform variables effectively
- Security best practices across environments

### 🔐 [TLS Example](./tls-example/)
**Secure connections and certificate management**
- Basic TLS (server authentication)
- Mutual TLS (client + server authentication)
- Redis Sentinel with TLS
- Redis Cluster with TLS
- Certificate generation for testing

**What you'll learn:**
- Configuring TLS connections
- Certificate management
- Mutual TLS authentication
- Troubleshooting TLS issues

## 🎯 Quick Start

1. **Choose an example** based on your use case
2. **Navigate to the example directory**
3. **Follow the README** in each example
4. **Copy and customize** for your needs

```bash
# Start with basic usage
cd basic-usage/
terraform init
terraform apply

# Or jump to multi-environment
cd multi-environment/
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your settings
terraform init
terraform apply
```

## 🔧 Common Patterns

### User Role Patterns

```hcl
# Read-only application user
resource "redisacl_user" "readonly" {
  name     = "app-readonly"
  keys     = "~app:* ~cache:*"
  commands = "+@read -@write -@dangerous"
}

# Write application user
resource "redisacl_user" "readwrite" {
  name     = "app-readwrite"
  keys     = "~app:* ~temp:*"
  commands = "+@read +@write -@dangerous"
}

# Admin user
resource "redisacl_user" "admin" {
  name     = "admin"
  keys     = "~*"
  commands = "+@all"
  allow_self_mutation = true
}

# Monitoring user
resource "redisacl_user" "monitoring" {
  name     = "monitoring"
  keys     = "~"  # No key access
  commands = "+ping +info +client +memory"
}
```

### Connection Patterns

```hcl
# Local development
provider "redisacl" {
  address = "localhost:6379"
}

# Production with authentication
provider "redisacl" {
  address  = "prod-redis.example.com:6379"
  password = var.redis_password
}

# Secure with TLS
provider "redisacl" {
  address  = "secure-redis.example.com:6380"
  password = var.redis_password
  use_tls  = true
}

# High availability with Sentinel
provider "redisacl" {
  password = var.redis_password
  sentinel {
    master_name = "mymaster"
    addresses   = ["sentinel1:26379", "sentinel2:26379"]
  }
}
```

## 📋 Prerequisites

All examples assume you have:

- **Terraform** 1.0+ installed
- **Redis server** accessible (local or remote)
- **Docker** (for TLS certificate generation)
- **Basic Redis knowledge** (helpful but not required)

## 🛠️ Customization Tips

### Environment Variables
```bash
# Set Redis connection via environment
export REDIS_URL="redis://user:pass@host:port/db"

# Use with provider
provider "redisacl" {
  # Will use REDIS_URL if no other config provided
}
```

### Variable Files
```hcl
# terraform.tfvars
redis_configs = {
  dev = {
    address = "localhost:6379"
    password = "dev-pass"
  }
  prod = {
    address = "prod-redis.com:6379"
    password = "prod-pass"
  }
}
```

### Conditional Resources
```hcl
# Create admin user only in development
resource "redisacl_user" "debug_admin" {
  count = var.environment == "dev" ? 1 : 0
  
  name     = "debug-admin"
  commands = "+@all"
}
```

## 🧪 Testing Your Configuration

### Validate Configuration
```bash
terraform validate
terraform plan
```

### Test User Access
```bash
# Test with redis-cli
redis-cli -u redis://username:password@host:port
> PING
> SET test:key "value"
> GET test:key
```

### Verify Permissions
```bash
# Should work for read user
redis-cli --user readonly-user -a password GET app:data

# Should fail for read user
redis-cli --user readonly-user -a password SET app:data "value"
```

## 🔍 Troubleshooting

### Common Issues

1. **Connection Failed**
   ```
   Error: Unable to connect to Redis
   ```
   - Check Redis server is running
   - Verify address and port
   - Check authentication credentials

2. **Permission Denied**
   ```
   Error: NOPERM this user has no permissions
   ```
   - Verify user exists and is enabled
   - Check ACL rules match your expectations
   - Use `ACL GETUSER username` to debug

3. **TLS Issues**
   ```
   Error: x509: certificate signed by unknown authority
   ```
   - Check certificate paths
   - Verify CA certificate
   - See [TLS example](./tls-example/) for detailed troubleshooting

### Debug Commands

```bash
# Check Terraform state
terraform show
terraform state list

# Verify Redis ACL
redis-cli ACL LIST
redis-cli ACL GETUSER username

# Test connection
redis-cli -u redis://user:pass@host:port ping
```

## 📚 Additional Resources

- [Redis ACL Documentation](https://redis.io/docs/management/security/acl/)
- [Terraform Provider Documentation](../README.md)
- [Redis Security Best Practices](https://redis.io/docs/management/security/)

## 🤝 Contributing Examples

Have a useful example? We'd love to include it!

1. Create a new directory with a descriptive name
2. Include a complete `main.tf` with comments
3. Add a detailed `README.md`
4. Include any necessary helper files
5. Test thoroughly before submitting

Example structure:
```
new-example/
├── main.tf                    # Main Terraform configuration
├── README.md                  # Detailed documentation
├── terraform.tfvars.example   # Example variables
└── outputs.tf                 # Useful outputs (optional)
```

---

**Need help?** Open an issue or start a discussion in the main repository!