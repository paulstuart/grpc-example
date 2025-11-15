# Control Plane Container

The Control Plane is a full-featured Linux development and troubleshooting environment that provides:

1. **Web Dashboard** - Single pane of glass with links to all services
2. **Development Environment** - Full Go compiler and build tools
3. **Troubleshooting Tools** - Network utilities, database clients, and debugging tools
4. **Workspace Access** - Entire repository mounted for running tests and building code

## Quick Start

### Access the Dashboard

```bash
# Open in browser
just dashboard

# Or manually:
open http://localhost:8080
```

The dashboard provides quick links to:
- API endpoints (gRPC, HTTP Gateway, OpenAPI docs)
- Observability (Grafana, Tempo, OTEL Collector)
- Infrastructure (PostgreSQL, Rsyslog)
- Health checks

### Access the Shell

```bash
# Interactive shell
just shell

# Or manually:
docker compose exec control-plane bash
```

Once inside, you have access to:
- The entire repository at `/workspace`
- All Docker services via DNS (`api`, `postgres`, `grafana`, etc.)
- Full development toolchain (Go, Just, Buf, protoc tools)
- Network and debugging utilities

## Features

### 1. Development Tools

The control-plane includes a complete development environment:

**Go Environment:**
- Go 1.24 compiler
- All protobuf generators (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `protoc-gen-openapiv2`)
- Buf (modern protobuf tooling)
- Just (command runner)

**Build and Test:**
```bash
# Access shell
just shell

# Inside container:
cd /workspace

# Build the project
just build

# Run tests
go test ./...

# Generate protobuf code
just buf

# Build and run client
just build-client
./client -server api:10000
```

### 2. Network Troubleshooting

**DNS Tools:**
```bash
# Check DNS resolution
dig api
dig postgres
nslookup grafana

# Trace route
traceroute tempo
```

**Connectivity Testing:**
```bash
# Check if port is open
nc -zv api 10000
nc -zv postgres 5432

# Test HTTP endpoints
curl http://grafana:3000
curl http://tempo:3200

# Test with TLS certificate validation
curl --cacert /workspace/certs/server.crt https://api:11000/v1/users
```

**Packet Analysis:**
```bash
# Capture traffic to/from API
tcpdump -i any host api -n

# Capture specific port
tcpdump -i any port 10000 -n

# Save to file for later analysis
tcpdump -i any port 10000 -w /tmp/capture.pcap
```

### 3. Database Access

**PostgreSQL Client:**
```bash
# Connect to database
psql postgresql://grpc_user:grpc_password@postgres:5432/grpc_example

# Run quick queries
psql postgresql://grpc_user:grpc_password@postgres:5432/grpc_example -c "SELECT * FROM users;"

# Check connection
pg_isready -h postgres -p 5432
```

### 4. TLS/Certificate Testing

**OpenSSL Tools:**
```bash
# View certificate details
openssl x509 -in /workspace/certs/server.crt -text -noout

# Check certificate SANs
openssl x509 -in /workspace/certs/server.crt -text | grep -A 2 "Subject Alternative Name"

# Test TLS connection to API
openssl s_client -connect api:11000 -servername api -CAfile /workspace/certs/server.crt

# Verify certificate chain
openssl verify -CAfile /workspace/certs/server.crt /workspace/certs/server.crt
```

### 5. Service Health Monitoring

**Check All Services:**
```bash
# From host
just docker-test-from-control

# Or manually from inside container:
curl --cacert /workspace/certs/server.crt https://api:11000/health
curl http://tempo:3200
curl http://localhost:13133  # OTEL collector health
curl http://localhost:8889/metrics  # OTEL Prometheus metrics
pg_isready -h postgres -p 5432
```

## Just Commands

Convenient commands for working with the control-plane:

```bash
# Open dashboard in browser
just dashboard

# Access interactive shell
just shell

# Run tests inside container
just docker-test-inside

# Run any justfile command inside container
just docker-just build
just docker-just buf

# Test connectivity from control-plane
just docker-test-from-control

# View control-plane logs
just docker-logs-control
```

## Common Workflows

### Debugging API Issues

```bash
# 1. Access shell
just shell

# 2. Check DNS resolution
dig api

# 3. Test connectivity
nc -zv api 10000
nc -zv api 11000

# 4. Test with curl
curl --cacert /workspace/certs/server.crt https://api:11000/v1/users

# 5. Capture traffic
tcpdump -i any host api -n -A
```

### Running Tests in the Cluster

```bash
# Access shell
just shell

# Run all tests
cd /workspace
go test ./...

# Run specific package tests
go test ./server/...
go test ./auth/...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Building and Testing Client

```bash
# Access shell
just shell

cd /workspace

# Build client
go build -o client cmd/client/main.go

# Test against API service (with TLS)
./client -server api:10000 -cert certs/server.crt

# Test with JWT token
./client -server api:10000 -cert certs/server.crt -token "$(cat testtoken)"
```

### Database Debugging

```bash
# Access shell
just shell

# Interactive PostgreSQL session
psql postgresql://grpc_user:grpc_password@postgres:5432/grpc_example

# Inside psql:
\dt              -- List tables
\d users         -- Describe users table
SELECT * FROM users LIMIT 10;
\q              -- Quit

# One-liner queries
psql postgresql://grpc_user:grpc_password@postgres:5432/grpc_example \
  -c "SELECT COUNT(*) FROM users;"
```

### Network Analysis

```bash
# Access shell
just shell

# Check routing
ip route

# Check network interfaces
ip addr

# Monitor real-time connections
netstat -an | grep ESTABLISHED

# Check what's listening
netstat -tuln

# DNS lookups for all services
for svc in api postgres tempo grafana otel-collector rsyslog; do
  echo "$svc: $(dig +short $svc)"
done
```

## Installed Tools

### Development
- **go** - Go 1.24 compiler and tools
- **git** - Version control
- **make** - Build automation
- **just** - Modern command runner
- **buf** - Protobuf tooling
- **vim/nano** - Text editors

### Network Utilities
- **curl/wget** - HTTP clients
- **netcat** - Network utility (nc)
- **dig/nslookup** - DNS tools
- **traceroute** - Route tracing
- **tcpdump** - Packet capture and analysis
- **netstat/ss** - Network statistics
- **ip/ifconfig** - Network configuration

### Database Clients
- **psql** - PostgreSQL client
- **pg_isready** - PostgreSQL connection check

### TLS/Security
- **openssl** - TLS/SSL toolkit
- **ca-certificates** - System certificates

### Utilities
- **jq** - JSON processor
- **htop** - Process viewer
- **tmux** - Terminal multiplexer
- **procps** - Process utilities

## Environment Variables

The control-plane container has access to:

```bash
PORT=8080                    # Dashboard port
POSTGRES_URL=postgresql://grpc_user:grpc_password@postgres:5432/grpc_example
```

## Volume Mounts

The control-plane has the following volumes mounted:

- **Repository**: `.` → `/workspace` (read-write)
  - Full access to source code, tests, and build artifacts
  - Changes made inside container persist on host

- **Certificates**: `./certs` → `/workspace/certs` (read-only)
  - Access to TLS certificates for testing

## Networking

The control-plane is connected to the `grpc-network` Docker network and can access all services by name:

| Service | Address | Description |
|---------|---------|-------------|
| `api:10000` | gRPC API | TLS-enabled gRPC |
| `api:11000` | HTTP Gateway | TLS-enabled HTTP |
| `postgres:5432` | PostgreSQL | Database |
| `tempo:3200` | Tempo | Distributed tracing |
| `grafana:3000` | Grafana | Visualization |
| `otel-collector:4317` | OTEL gRPC | OpenTelemetry |
| `rsyslog:514` | Rsyslog | Centralized logging |

## Docker Compose Configuration

```yaml
control-plane:
  build:
    context: .
    dockerfile: Dockerfile.control-plane
  ports:
    - "8080:8080"  # Dashboard
  volumes:
    - .:/workspace              # Entire repo
    - ./certs:/workspace/certs  # Certificates
  networks:
    - grpc-network
```

## Tips and Tricks

### Keep Tools Running in Background

Use `tmux` for persistent sessions:

```bash
# Start tmux
tmux

# Create multiple panes
Ctrl-b %    # Split vertically
Ctrl-b "    # Split horizontally

# Example: monitor logs in one pane, run commands in another
# Pane 1: tcpdump -i any host api -n
# Pane 2: curl --cacert /workspace/certs/server.crt https://api:11000/v1/users

# Detach: Ctrl-b d
# Reattach: tmux attach
```

### Quick Health Check Script

Create a health check script inside the container:

```bash
# Inside container
cat > /tmp/healthcheck.sh <<'EOF'
#!/bin/bash
echo "=== Service Health Check ==="
echo "API: $(curl -sk --cacert /workspace/certs/server.crt https://api:11000/health || echo "FAIL")"
echo "Grafana: $(curl -s http://grafana:3000 >/dev/null && echo "OK" || echo "FAIL")"
echo "Tempo: $(curl -s http://tempo:3200 >/dev/null && echo "OK" || echo "FAIL")"
echo "OTEL Health: $(curl -s http://localhost:13133 >/dev/null && echo "OK" || echo "FAIL")"
echo "PostgreSQL: $(pg_isready -h postgres -p 5432 -q && echo "OK" || echo "FAIL")"
EOF

chmod +x /tmp/healthcheck.sh
/tmp/healthcheck.sh
```

### Save Debugging Session

```bash
# Start a script session to record everything
script /tmp/debug-session.log

# Do your debugging...

# Exit
exit

# Copy log to host (from host machine)
docker cp grpc-control-plane:/tmp/debug-session.log ./debug-session.log
```

## Troubleshooting

### Container Won't Start

Check logs:
```bash
docker compose logs control-plane
```

Common issues:
- Port 8080 already in use
- Volume mount permissions
- Docker network issues

### Can't Access Services by DNS

Test DNS resolution:
```bash
docker compose exec control-plane dig api
```

If failing:
- Ensure all containers are on `grpc-network`
- Restart Docker daemon
- Check `docker network inspect grpc-network`

### Workspace Changes Not Persisting

The workspace is mounted as read-write, so changes should persist. If not:
- Check volume mount in `docker-compose.yml`
- Ensure you're in `/workspace` directory
- Check file permissions

## See Also

- [Docker Networking](DOCKER_NETWORKING.md) - Network architecture
- [TLS in Docker](TLS_DOCKER.md) - Certificate configuration
- [Docker Commands](DOCKER_COMMANDS.md) - All Docker commands
- [Quick Start Guide](QUICKSTART.md) - Getting started
