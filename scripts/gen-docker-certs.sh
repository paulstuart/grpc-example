#!/bin/bash
set -e

# Certificate generation script for Docker environments
# Creates self-signed certificates with Subject Alternative Names (SANs)
# to support multiple hostnames: localhost, Docker service names, and custom DNS

CERT_DIR="certs"
CERT_CONF="cert.conf"
SERVER_KEY="$CERT_DIR/server.key"
SERVER_CRT="$CERT_DIR/server.crt"
DAYS_VALID=365

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Create OpenSSL configuration with SANs
cat > "$CERT_CONF" <<EOF
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
C = US
ST = California
L = San Francisco
O = gRPC Example
OU = Development
CN = grpc-example

[v3_req]
subjectAltName = @alt_names
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth

[alt_names]
# Localhost variants
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1

# Docker service name (from docker-compose.yml)
DNS.2 = api

# Docker network wildcards (for scaled instances: api-1, api-2, etc.)
DNS.3 = api-1
DNS.4 = api-2
DNS.5 = api-3
DNS.6 = api-4
DNS.7 = api-5

# Custom DNS entry (from /etc/hosts setup)
DNS.8 = grpc.example

# Docker network IP range (rsyslog static IP and potential API IPs)
IP.3 = 172.21.0.10
IP.4 = 172.21.0.11
IP.5 = 172.21.0.12
IP.6 = 172.21.0.13
IP.7 = 172.21.0.14
EOF

echo "Generating self-signed certificate with Subject Alternative Names..."
echo "Certificate will be valid for $DAYS_VALID days"
echo ""
echo "SANs included:"
echo "  - localhost (127.0.0.1, ::1)"
echo "  - api, api-1, api-2, api-3, api-4, api-5 (Docker service names)"
echo "  - grpc.example (custom DNS)"
echo "  - 172.21.0.10-14 (Docker network IPs)"
echo ""

# Generate the certificate
openssl req -x509 -newkey rsa:2048 \
  -keyout "$SERVER_KEY" \
  -out "$SERVER_CRT" \
  -days "$DAYS_VALID" \
  -nodes \
  -config "$CERT_CONF" \
  -extensions v3_req

# Clean up config file
rm "$CERT_CONF"

echo ""
echo "Certificate generated successfully!"
echo "  Private Key: $SERVER_KEY"
echo "  Certificate: $SERVER_CRT"
echo ""

# Show certificate details
echo "Certificate details:"
openssl x509 -in "$SERVER_CRT" -noout -text | grep -A 1 "Subject Alternative Name"

echo ""
echo "Verify certificate:"
echo "  openssl x509 -in $SERVER_CRT -noout -text"
