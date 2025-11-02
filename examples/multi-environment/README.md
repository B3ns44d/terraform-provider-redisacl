# Multi-Environment Example

This example demonstrates how to manage Redis ACL users across multiple environments (development, staging, and production) with environment-specific configurations and security policies.

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Development │    │   Staging   │    │ Production  │
│   Redis     │    │    Redis    │    │    Redis    │
├─────────────┤    ├─────────────┤    ├─────────────┤
│ app-user    │    │ app-user    │    │ app-user    │
│ monitoring  │    │ monitoring  │    │ monitoring  │
│             │    │ job-proc    │    │ job-proc    │
└─────────────┘    └─────────────┘    └─────────────┘
     Local              TLS              TLS
   Permissive        Restrictive    Most Restrictive
```

## Features Demonstrated

- **Environment-Specific Providers**: Different Redis instances for each environment
- **Progressive Security**: More restrictive permissions in higher environments
- **Variable-Driven Configuration**: Easy to manage across environments
- **TLS Configuration**: Secure connections for staging and production
- **Role-Based Access**: Different users for different purposes

## Environment Configurations

### Development
- **Security**: Permissive (easier debugging)
- **Connection**: Local, no TLS
- **Users**: app-user, monitoring
- **Permissions**: Full access to most operations

### Staging
- **Security**: Production-like restrictions
- **Connection**: Remote with TLS
- **Users**: app-user, job-processor, monitoring
- **Permissions**: Limited to application-specific keys and operations

### Production
- **Security**: Most restrictive
- **Connection**: Remote with TLS
- **Users**: app-user, job-processor, monitoring
- **Permissions**: Minimal required access only

## User Roles

### Application User (`app-user`)
- **Purpose**: Main application database access
- **Dev**: Full access for debugging
- **Staging/Prod**: Limited to application keys only

### Job Processor (`job-processor`)
- **Purpose**: Background job processing
- **Environments**: Staging and Production only
- **Access**: Job queues and temporary data

### Monitoring (`monitoring`)
- **Purpose**: Health checks and metrics collection
- **Access**: Info commands only, no data access
- **Security**: Progressively more restricted in higher environments

## Setup

1. **Copy Configuration File**
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. **Update Variables**
   Edit `terraform.tfvars` with your actual Redis connection details:
   ```hcl
   environments = {
     dev = {
       redis_address  = "localhost:6379"
       redis_password = "your-dev-password"
       use_tls        = false
     }
     # ... update staging and prod
   }
   
   app_passwords = {
     dev     = "secure-dev-password"
     staging = "secure-staging-password"
     prod    = "very-secure-prod-password"
   }
   ```

3. **Initialize and Apply**
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Verification

After applying, verify the setup:

```bash
# Check created users in each environment
terraform output environment_summary

# View permission differences
terraform output app_user_permissions

# Test development environment
redis-cli -u redis://app-user:dev-app-password@localhost:6379
> SET app:test "value"  # Should work
> SET admin:test "value"  # Should fail in staging/prod

# Test monitoring user
redis-cli -u redis://monitoring:dev-monitoring-password@localhost:6379
> PING  # Should work
> INFO  # Should work
> GET app:test  # Should fail
```

## Security Considerations

### Development
- ✅ Permissive for debugging
- ⚠️ Not suitable for sensitive data
- 🔓 No TLS (local only)

### Staging
- ✅ Production-like security
- ✅ TLS encryption
- ✅ Limited key access
- ✅ No dangerous commands

### Production
- ✅ Maximum security
- ✅ TLS encryption
- ✅ Minimal required access
- ✅ No admin/scripting commands
- ✅ Restricted monitoring access

## Best Practices Demonstrated

1. **Environment Isolation**: Separate providers and configurations
2. **Progressive Security**: Tighter controls in higher environments
3. **Principle of Least Privilege**: Minimal required access
4. **Secure Secrets Management**: Sensitive variables marked appropriately
5. **Role-Based Access**: Different users for different purposes
6. **Infrastructure as Code**: Consistent, repeatable deployments

## Extending This Example

- Add more environments (QA, UAT, etc.)
- Implement certificate-based TLS authentication
- Add Redis Sentinel or Cluster configurations
- Integrate with secret management systems (Vault, AWS Secrets Manager)
- Add automated testing for each environment

## Cleanup

To remove all users from all environments:

```bash
terraform destroy
```

**Note**: This will remove users from all environments. Use targeted destroys for specific environments:

```bash
# Remove only development users
terraform destroy -target=redisacl_user.app_user_dev -target=redisacl_user.monitoring_dev
```