# TLS and Mutual TLS Example

This example demonstrates how to configure the Redis ACL provider for secure connections using TLS and mutual TLS (mTLS) authentication.

## Security Levels Demonstrated

1. **Basic TLS**: Server authentication only (client verifies server)
2. **Mutual TLS**: Both client and server authentication
3. **Sentinel with TLS**: High availability with encryption
4. **Cluster with TLS**: Distributed Redis with encryption

## Prerequisites

### For Testing (Self-Signed Certificates)

1. **Generate Test Certificates**
   ```bash
   # Make the script executable
   chmod +x generate-certs.sh
   
   # Generate self-signed certificates
   ./generate-certs.sh
   ```

2. **Configure Redis with TLS**
   Add to your `redis.conf`:
   ```
   # Enable TLS
   tls-port 6380
   port 0  # Disable non-TLS port
   
   # Server certificates
   tls-cert-file /path/to/certs/server.crt
   tls-key-file /path/to/certs/server.key
   
   # CA for client verification (mutual TLS)
   tls-ca-cert-file /path/to/certs/ca.pem
   tls-auth-clients yes
   
   # Optional: TLS protocols
   tls-protocols "TLSv1.2 TLSv1.3"
   ```

3. **Start Redis with TLS**
   ```bash
   redis-server /path/to/redis.conf
   ```

### For Production

1. **Obtain Valid Certificates**
   - Use certificates from a trusted CA (Let's Encrypt, commercial CA)
   - Or use your organization's internal CA

2. **Configure Certificate Paths**
   Update `terraform.tfvars`:
   ```hcl
   redis_tls_config = {
     address              = "your-redis-server.com:6380"
     password             = "your-secure-password"
     ca_cert_path         = "/path/to/ca.pem"
     client_cert_path     = "/path/to/client.crt"
     client_key_path      = "/path/to/client.key"
     insecure_skip_verify = false  # Always false in production
   }
   ```

## Configuration Examples

### 1. Basic TLS (Server Authentication Only)

```hcl
provider "redisacl" {
  address  = "secure-redis.example.com:6380"
  password = "secure-password"
  use_tls  = true
  
  # Verify server certificate against system CA store
  # For custom CA, add: tls_ca_cert = file("ca.pem")
}
```

### 2. Mutual TLS (Client + Server Authentication)

```hcl
provider "redisacl" {
  address = "mtls-redis.example.com:6380"
  use_tls = true
  
  # Server verification
  tls_ca_cert = file("certs/ca.pem")
  
  # Client authentication
  tls_cert = file("certs/client.crt")
  tls_key  = file("certs/client.key")
}
```

### 3. Redis Sentinel with TLS

```hcl
provider "redisacl" {
  username = "default"
  password = "master-password"
  use_tls  = true
  
  sentinel {
    master_name = "mymaster"
    addresses   = [
      "sentinel1.example.com:26380",
      "sentinel2.example.com:26380"
    ]
  }
  
  tls_ca_cert = file("certs/ca.pem")
}
```

### 4. Redis Cluster with TLS

```hcl
provider "redisacl" {
  username = "cluster-user"
  password = "cluster-password"
  use_tls  = true
  
  cluster {
    addresses = [
      "node1.example.com:6380",
      "node2.example.com:6380",
      "node3.example.com:6380"
    ]
  }
  
  tls_ca_cert = file("certs/ca.pem")
  tls_cert    = file("certs/client.crt")
  tls_key     = file("certs/client.key")
}
```

## Usage

1. **Setup Certificates**
   ```bash
   # For testing
   ./generate-certs.sh
   
   # For production, place your certificates in the certs/ directory
   ```

2. **Configure Variables**
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your settings
   ```

3. **Deploy**
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Testing TLS Connections

### Test Basic TLS Connection
```bash
# Using redis-cli with TLS
redis-cli --tls \
  --cert certs/client.crt \
  --key certs/client.key \
  --cacert certs/ca.pem \
  -h localhost -p 6380 \
  -a your-password \
  ping
```

### Test User Authentication
```bash
# Test app user with TLS
redis-cli --tls \
  --cert certs/client.crt \
  --key certs/client.key \
  --cacert certs/ca.pem \
  -h localhost -p 6380 \
  --user app-tls-user \
  -a secure-app-password \
  ping

# Test permissions
redis-cli --tls \
  --cert certs/client.crt \
  --key certs/client.key \
  --cacert certs/ca.pem \
  -h localhost -p 6380 \
  --user app-tls-user \
  -a secure-app-password \
  set app:test "value"  # Should work

redis-cli --tls \
  --cert certs/client.crt \
  --key certs/client.key \
  --cacert certs/ca.pem \
  -h localhost -p 6380 \
  --user app-tls-user \
  -a secure-app-password \
  flushall  # Should fail (dangerous command)
```

## Security Best Practices

### Certificate Management
- ✅ Use certificates from trusted CAs in production
- ✅ Regularly rotate certificates before expiration
- ✅ Store private keys securely with restricted permissions (600)
- ✅ Use different certificates for different environments
- ❌ Never commit private keys to version control

### TLS Configuration
- ✅ Always use TLS 1.2 or higher
- ✅ Disable insecure protocols (SSLv3, TLS 1.0, TLS 1.1)
- ✅ Use strong cipher suites
- ✅ Enable certificate verification (`insecure_skip_verify = false`)
- ✅ Use mutual TLS for high-security environments

### Access Control
- ✅ Combine TLS with Redis ACL for defense in depth
- ✅ Use principle of least privilege for user permissions
- ✅ Regularly audit user access and permissions
- ✅ Monitor failed authentication attempts

## Troubleshooting

### Common TLS Issues

1. **Certificate Verification Failed**
   ```
   Error: x509: certificate signed by unknown authority
   ```
   - Solution: Ensure `tls_ca_cert` points to the correct CA certificate
   - For testing: Set `tls_insecure_skip_verify = true` (not for production)

2. **Certificate Name Mismatch**
   ```
   Error: x509: certificate is valid for localhost, not redis.example.com
   ```
   - Solution: Ensure certificate includes correct DNS names/IPs in SAN
   - Update certificate or use correct hostname

3. **Client Certificate Required**
   ```
   Error: tls: client didn't provide a certificate
   ```
   - Solution: Provide client certificate for mutual TLS
   - Add `tls_cert` and `tls_key` to provider configuration

4. **Permission Denied**
   ```
   Error: permission denied to read private key
   ```
   - Solution: Check file permissions on certificate files
   - Run: `chmod 600 certs/*.key`

### Debugging Commands

```bash
# Check certificate details
openssl x509 -in certs/server.crt -text -noout

# Verify certificate chain
openssl verify -CAfile certs/ca.pem certs/server.crt

# Test TLS connection
openssl s_client -connect localhost:6380 -cert certs/client.crt -key certs/client.key

# Check Redis TLS configuration
redis-cli --tls --cert certs/client.crt --key certs/client.key --cacert certs/ca.pem -p 6380 config get tls-*
```

## Files Generated

After running `generate-certs.sh`:

```
certs/
├── ca.pem       # Certificate Authority (public)
├── ca.key       # CA private key
├── server.crt   # Server certificate (public)
├── server.key   # Server private key
├── client.crt   # Client certificate (public)
└── client.key   # Client private key
```

## Cleanup

```bash
# Remove Terraform resources
terraform destroy

# Remove test certificates (optional)
rm -rf certs/
```

## Next Steps

- Integrate with HashiCorp Vault for certificate management
- Set up certificate rotation automation
- Configure monitoring for certificate expiration
- Implement certificate-based authentication for applications