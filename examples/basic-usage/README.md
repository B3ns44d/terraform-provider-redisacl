# Basic Usage Example

This example demonstrates basic Redis ACL user management with the terraform-provider-redisacl.

## What This Example Creates

1. **Read-Only Application User** (`readonly-app`)
   - Access to `app:*` and `cache:*` keys
   - Only read operations allowed
   - Suitable for applications that only need to read data

2. **Write Application User** (`write-app`)
   - Access to `data:*` and `temp:*` keys
   - Read and write operations (excluding dangerous commands)
   - Suitable for data ingestion services

3. **Admin User** (`admin-user`)
   - Full access to all keys and operations
   - Self-mutation allowed for administrative tasks

4. **Monitoring User** (`monitoring-user`)
   - Limited to monitoring and info commands only
   - No key or channel access
   - Suitable for monitoring tools and health checks

## Prerequisites

- Redis server running on `localhost:6379`
- Redis password configured (update in `main.tf`)
- Terraform installed

## Usage

1. **Update Configuration**
   ```bash
   # Edit main.tf and update the Redis password
   vim main.tf
   ```

2. **Initialize Terraform**
   ```bash
   terraform init
   ```

3. **Plan the Changes**
   ```bash
   terraform plan
   ```

4. **Apply the Configuration**
   ```bash
   terraform apply
   ```

5. **View Outputs**
   ```bash
   terraform output
   ```

## Testing the Users

After applying, you can test the created users:

```bash
# Test readonly user
redis-cli -u redis://readonly-app:app-readonly-password@localhost:6379
> SET app:test "value"  # Should fail
> GET app:test          # Should work

# Test write user  
redis-cli -u redis://write-app:app-write-password@localhost:6379
> SET data:test "value" # Should work
> GET data:test         # Should work
> FLUSHALL              # Should fail (dangerous command)

# Test monitoring user
redis-cli -u redis://monitoring-user:monitoring-password@localhost:6379
> PING                  # Should work
> INFO                  # Should work
> GET somekey           # Should fail
```

## Cleanup

To remove all created users:

```bash
terraform destroy
```

## Next Steps

- Check out the [multi-environment example](../multi-environment/) for production setups
- See the [advanced-permissions example](../advanced-permissions/) for complex ACL rules
- Review the [tls-example](../tls-example/) for secure connections