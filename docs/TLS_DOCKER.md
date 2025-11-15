# TLS Configuration for Docker Environments

This document explains how TLS certificates are configured to work across Docker containers.

## The Problem

When using self-signed TLS certificates in Docker environments, you face a chicken-and-egg problem:

1. **Local Development**: Self-signed certificates work fine when you control the client and can install the CA certificate
2. **Docker Containers**: Each container needs to trust the server's certificate, but self-signed certs aren't in the system trust store
3. **Service Names**: Docker containers use DNS service names (like `api`, `postgres`) instead of `localhost`, and certificates must include these names

Without proper configuration, you get certificate validation errors when containers try to communicate over TLS.

## The Solution: Subject Alternative Names (SANs)

We use **Subject Alternative Names (SANs)** in our self-signed certificates to support multiple hostnames and IP addresses.

### What are SANs?

SANs allow a single certificate to be valid for multiple hostnames and IP addresses. Modern TLS implementations require SANs (they reject certificates that only use the Common Name field).

### Our Certificate Includes

Our Docker-compatible certificate includes SANs for:

- **Localhost access**: `localhost`, `127.0.0.1`, `::1`
- **Docker service names**: `api`, `api-1`, `api-2`, `api-3`, `api-4`, `api-5`
- **Custom DNS**: `grpc.example` (from `/etc/hosts` setup)
- **Docker network IPs**: `172.21.0.10` - `172.21.0.14`

This covers:
- Host machine connections (`localhost`)
- Docker Compose service discovery (`api`)
- Scaled instances (`api-1`, `api-2`, etc.)
- Custom DNS entries
- Direct IP connections within the Docker network

## Generating Certificates

### For Docker Environments

Use the Docker-specific certificate generator:

```bash
just gen-docker-certs
```

This runs `scripts/gen-docker-certs.sh` which creates a certificate with all necessary SANs for Docker deployment.

### For Local Development Only

If you only need basic localhost support:

```bash
just gen-certs
```

This creates a simpler certificate without Docker-specific SANs.

## How It Works

### 1. Certificate Generation

The `scripts/gen-docker-certs.sh` script:
- Creates an OpenSSL configuration with comprehensive SANs
- Generates a 2048-bit RSA private key
- Creates a self-signed certificate valid for 365 days
- Includes proper key usage extensions for both server and client auth

### 2. Docker Image Build

The Dockerfile copies certificates into the container:

```dockerfile
# Copy certificates from build context
COPY --from=builder /build/certs ./certs

# Run with TLS enabled
CMD ["./grpc-example", "-host", "0.0.0.0", "-cert", "certs/server.crt", "-key", "certs/server.key"]
```

### 3. Client Configuration

Clients must use the certificate as their CA:

**Go client example:**
```go
certPEM, _ := os.ReadFile("certs/server.crt")
certPool := x509.NewCertPool()
certPool.AppendCertsFromPEM(certPEM)

tlsConfig := &tls.Config{
    RootCAs:    certPool,
    MinVersion: tls.VersionTLS12,
}
```

**Curl example:**
```bash
curl --cacert certs/server.crt https://api:11000/v1/users
```

## Testing

### Test Certificate SANs

Verify the certificate includes all necessary SANs:

```bash
openssl x509 -in certs/server.crt -noout -text | grep -A 2 "Subject Alternative Name"
```

Expected output:
```
X509v3 Subject Alternative Name:
    DNS:localhost, IP Address:127.0.0.1, IP Address:0:0:0:0:0:0:0:1,
    DNS:api, DNS:api-1, DNS:api-2, DNS:api-3, DNS:api-4, DNS:api-5,
    DNS:grpc.example, IP Address:172.21.0.10, IP Address:172.21.0.11, ...
```

### Test from Host

```bash
# Using localhost (in SAN)
curl --cacert certs/server.crt https://localhost:11005/v1/users

# Skip verification (not recommended for production)
curl -k https://localhost:11005/v1/users
```

### Test Container-to-Container

From within a container:

```bash
# Install curl in a container
docker compose exec postgres sh -c "apk add curl"

# Copy certificate
docker cp certs/server.crt grpc-postgres:/tmp/server.crt

# Test connection using Docker service name
docker compose exec postgres curl --cacert /tmp/server.crt https://api:11000/v1/users
```

Expected: `{"code":5,"message":"Not Found","details":[]}` (empty user list)

### Run Full Test Suite

```bash
./scripts/test-docker-tls.sh
```

## Production Considerations

For production deployments, use one of these approaches instead:

### Option 1: Proper Certificate Authority

Create a CA and issue signed certificates:

```bash
# Create CA
openssl genrsa -out ca.key 4096
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1024 -out ca.crt

# Create server cert signed by CA
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -out server.crt

# Distribute ca.crt to all clients
```

### Option 2: Let's Encrypt

For public-facing services:

```bash
just certs  # Uses lego for Let's Encrypt
```

### Option 3: Cloud Provider Certificates

Use managed certificates from your cloud provider:
- AWS Certificate Manager
- Google Cloud Certificate Manager
- Azure Key Vault Certificates

## Troubleshooting

### Certificate Validation Errors

If you see `x509: certificate signed by unknown authority`:
- Ensure the client is using the certificate as its CA (RootCAs)
- Verify the certificate file is readable
- Check that you're using the correct certificate file

### Hostname Mismatch Errors

If you see `x509: certificate is valid for X, not Y`:
- The hostname you're connecting to is not in the certificate's SANs
- Regenerate the certificate with the missing hostname
- Use a hostname that's already in the SANs

### Docker DNS Resolution

If `api` doesn't resolve:
- Ensure containers are on the same Docker network
- Check `docker network inspect grpc-example_grpc-network`
- Verify service name matches `docker-compose.yml`

## Files

- `scripts/gen-docker-certs.sh` - Certificate generation script
- `certs/server.crt` - Server certificate (generated)
- `certs/server.key` - Private key (generated)
- `scripts/test-docker-tls.sh` - Test script

## References

- [RFC 5280 - Subject Alternative Name](https://tools.ietf.org/html/rfc5280#section-4.2.1.6)
- [Docker Networking](https://docs.docker.com/network/)
- [gRPC Security](https://grpc.io/docs/guides/auth/)
