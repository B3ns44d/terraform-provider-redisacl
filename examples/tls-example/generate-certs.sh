#!/bin/bash
# Generate self-signed certificates for testing TLS connections
# WARNING: These certificates are for testing only, not for production use!

set -e

CERT_DIR="certs"
DAYS=365

echo "🔐 Generating self-signed certificates for Redis TLS testing..."

# Create certificate directory
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# Generate CA private key
echo "📝 Generating CA private key..."
openssl genrsa -out ca.key 4096

# Generate CA certificate
echo "📝 Generating CA certificate..."
openssl req -new -x509 -days $DAYS -key ca.key -out ca.pem -subj "/C=US/ST=Test/L=Test/O=Redis-ACL-Test/CN=Test-CA"

# Generate server private key
echo "📝 Generating server private key..."
openssl genrsa -out server.key 4096

# Generate server certificate signing request
echo "📝 Generating server certificate signing request..."
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=Test/L=Test/O=Redis-ACL-Test/CN=localhost"

# Create server certificate extensions
cat > server.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = redis.local
DNS.3 = secure-redis.example.com
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Generate server certificate
echo "📝 Generating server certificate..."
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out server.crt -days $DAYS -extensions v3_req -extfile server.ext

# Generate client private key
echo "📝 Generating client private key..."
openssl genrsa -out client.key 4096

# Generate client certificate signing request
echo "📝 Generating client certificate signing request..."
openssl req -new -key client.key -out client.csr -subj "/C=US/ST=Test/L=Test/O=Redis-ACL-Test/CN=redis-client"

# Generate client certificate
echo "📝 Generating client certificate..."
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out client.crt -days $DAYS

# Set appropriate permissions
chmod 600 *.key
chmod 644 *.crt *.pem

# Clean up temporary files
rm -f *.csr *.ext *.srl

echo "✅ Certificate generation complete!"
echo ""
echo "📁 Generated files in $CERT_DIR/:"
echo "   ca.pem       - Certificate Authority (for server verification)"
echo "   ca.key       - CA private key"
echo "   server.crt   - Server certificate"
echo "   server.key   - Server private key"
echo "   client.crt   - Client certificate (for mutual TLS)"
echo "   client.key   - Client private key (for mutual TLS)"
echo ""
echo "🔧 To use with Redis server, add to redis.conf:"
echo "   tls-port 6380"
echo "   port 0"
echo "   tls-cert-file $(pwd)/server.crt"
echo "   tls-key-file $(pwd)/server.key"
echo "   tls-ca-cert-file $(pwd)/ca.pem"
echo "   tls-auth-clients yes"
echo ""
echo "⚠️  WARNING: These certificates are for testing only!"
echo "   Do not use in production environments."