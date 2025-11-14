# OpenTelemetry Integration & Docker Deployment Guide

This document describes the OpenTelemetry (Otel) integration and Docker Compose deployment setup for the gRPC Example project.

## Overview

The project now includes:
- **Full OpenTelemetry instrumentation** (traces + metrics)
- **PostgreSQL storage backend** with distributed tracing
- **Docker Compose stack** with all observability components
- **Grafana + Tempo** for trace visualization
- **Scalable API servers**

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Clients   │────▶│  gRPC API    │────▶│  PostgreSQL │
│             │     │   Servers    │     │  Database   │
└─────────────┘     └──────┬───────┘     └─────────────┘
                           │
                           │ OTLP
                           ▼
                    ┌──────────────┐
                    │     Otel     │
                    │  Collector   │
                    └──────┬───────┘
                           │
                           │ Traces
                           ▼
                    ┌──────────────┐     ┌─────────────┐
                    │    Tempo     │────▶│   Grafana   │
                    │  (Storage)   │     │    (UI)     │
                    └──────────────┘     └─────────────┘
```

## Features

### 1. OpenTelemetry Tracing
- **Automatic span creation** for all gRPC calls (Unary, Server/Client/Bidi streaming)
- **Context propagation** from HTTP Gateway → gRPC → Database
- **User attribution** - JWT claims added as span attributes
- **Error tracking** - Errors recorded with full context
- **Database tracing** - All PostgreSQL queries traced with Otel

### 2. OpenTelemetry Metrics
- Request counts by method and status
- Request duration histograms
- Active request gauges
- Error counters
- All exported to Prometheus-compatible format

### 3. PostgreSQL Storage
- Full implementation of the Storage interface
- Supports all proto field types (oneof, maps, repeated, nested messages)
- Distributed tracing on all database operations
- Connection pooling with pgx v5

### 4. Docker Compose Stack
Services:
- **postgres**: PostgreSQL 16 database
- **otel-collector**: OpenTelemetry Collector (receives and exports telemetry)
- **tempo**: Grafana Tempo (trace storage backend)
- **grafana**: Grafana (visualization)
- **api**: gRPC API servers (scalable)

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Just command runner (optional but recommended)

### 1. Generate TLS Certificates
```bash
just gen-certs
```

### 2. Start the Full Stack
```bash
# Start all services
just docker-up-build

# Or manually with docker compose
docker compose up -d --build
```

### 3. Scale API Servers
```bash
# Scale to 3 API server instances
just docker-scale 3

# Or manually
docker compose up -d --scale api=3
```

### 4. Access Services

| Service | URL | Purpose |
|---------|-----|---------|
| Grafana | http://localhost:3000 | Trace visualization |
| gRPC API | https://localhost:10000 | gRPC endpoint |
| HTTP Gateway | https://localhost:11000 | REST API |
| PostgreSQL | localhost:5432 | Database |
| Otel Collector | localhost:4317 | OTLP receiver |

### 5. View Traces in Grafana

1. Open Grafana: http://localhost:3000 (auto-login enabled)
2. Navigate to "Explore" → Select "Tempo" datasource
3. Search for traces with: `service.name="grpc-example-api"`
4. Or use pre-configured dashboard: "gRPC Traces"

## Configuration

### Environment Variables

Server configuration via environment variables or flags:

#### gRPC Server
- `GRPC_PORT` - gRPC server port (default: 10000)
- `GRPC_GATEWAY_PORT` - HTTP gateway port (default: 11000)
- `GRPC_HOST` - Bind address (default: localhost)

#### Authentication
- `JWT_SECRET` - Secret key for JWT tokens
- `GRPC_ISSUER` - JWT issuer name

#### OpenTelemetry
- `OTEL_ENABLED` - Enable/disable Otel (default: true)
- `OTEL_EXPORTER_OTLP_ENDPOINT` - Collector endpoint (default: localhost:4317)
- `SERVICE_NAME` - Service name for traces (default: grpc-example)
- `ENVIRONMENT` - Deployment environment (default: development)

#### Database
- `DATABASE_URL` - PostgreSQL connection string
  - Format: `postgresql://user:password@host:port/database?sslmode=disable`
  - If empty, uses in-memory storage

### Command-Line Flags

All environment variables have corresponding flags:

```bash
./grpc-example \
  -grpc-port 10000 \
  -gateway-port 11000 \
  -host 0.0.0.0 \
  -otel-enabled \
  -otel-endpoint localhost:4317 \
  -service-name grpc-example \
  -environment production \
  -db "postgresql://user:pass@localhost:5432/dbname" \
  -enable-auth
```

## Justfile Commands

### Docker Commands

```bash
# Build Docker image
just docker-build

# Start full stack
just docker-up

# Start with rebuild
just docker-up-build

# Scale API servers
just docker-scale 5

# Stop stack
just docker-down

# Stop and remove volumes (destructive)
just docker-down-volumes

# View logs
just docker-logs              # All services
just docker-logs-api          # API servers only
just docker-logs-service postgres  # Specific service

# Restart service
just docker-restart api

# Access PostgreSQL CLI
just docker-psql

# Check service status
just docker-ps
```

### Local Development

```bash
# Build server
just build

# Run with OpenTelemetry (requires collector running)
just run-with-otel

# Run with PostgreSQL (requires DB running)
just run-with-postgres

# Run with authentication
just run-server-with-auth
```

## File Structure

### New Files Created

```
.
├── otel/
│   ├── setup.go              # Otel initialization
│   └── http_middleware.go    # HTTP instrumentation
│
├── interceptors/
│   └── otel_interceptors.go  # Otel-enhanced gRPC interceptors
│
├── server/
│   └── postgres_storage.go   # PostgreSQL storage implementation
│
├── Dockerfile                # Multi-stage API server build
├── docker compose.yml        # Full observability stack
├── otel-collector-config.yaml
├── tempo-config.yaml
├── grafana-datasources.yaml
├── grafana-dashboards.yaml
├── dashboards/
│   └── grpc-traces.json
└── .env.example              # Environment variable template
```

## OpenTelemetry Implementation Details

### Trace Spans

Every gRPC call generates spans with these attributes:
- `rpc.system` = "grpc"
- `rpc.service` = service name (e.g., "proto.UserService")
- `rpc.method` = method name (e.g., "GetUser")
- `rpc.grpc.kind` = call type (unary, server_stream, client_stream, bidi_stream)
- `rpc.grpc.status_code` = result status
- `rpc.duration_ms` = call duration

If authenticated:
- `user.id` = username from JWT
- `user.email` = email from JWT
- `user.roles` = user roles array

### Database Spans

PostgreSQL operations create child spans:
- `db.system` = "postgresql"
- `db.operation` = SQL operation (SELECT, INSERT, UPDATE, DELETE)
- `db.table` = table name
- Additional attributes based on operation

### Metrics

Collected metrics:
- `grpc.server.request.count` - Total requests by method and status
- `grpc.server.request.duration` - Request duration histogram
- `grpc.server.request.errors` - Error count by method
- `grpc.server.active_requests` - Current active requests gauge

## Scaling

### Horizontal Scaling

The API service is designed to be horizontally scalable:

```bash
# Scale to N instances
docker compose up -d --scale api=N
```

Each instance:
- Connects to shared PostgreSQL database
- Sends traces to shared Otel Collector
- Operates independently (stateless)

### Load Balancing

For production, add a load balancer (e.g., nginx, Traefik) in front of API servers.

Example nginx config:
```nginx
upstream grpc_backend {
    server api:10000;
    # Add more servers when scaling
}

server {
    listen 10000 http2;
    location / {
        grpc_pass grpc://grpc_backend;
    }
}
```

## Troubleshooting

### Traces Not Appearing in Grafana

1. Check Otel Collector is running:
   ```bash
   docker compose logs otel-collector
   ```

2. Verify collector health:
   ```bash
   curl http://localhost:13133
   ```

3. Check Tempo is receiving data:
   ```bash
   docker compose logs tempo
   ```

### Database Connection Errors

1. Verify PostgreSQL is ready:
   ```bash
   docker compose exec postgres pg_isready
   ```

2. Check connection string format:
   ```
   postgresql://user:password@host:port/database?sslmode=disable
   ```

### API Server Crashes

1. Check logs:
   ```bash
   just docker-logs-api
   ```

2. Verify all dependencies are healthy:
   ```bash
   just docker-ps
   ```

## Next Steps

### Kubernetes Deployment (Future)

Once Docker Compose is validated, the stack can be migrated to Kubernetes:

1. Convert docker compose.yml to K8s manifests
2. Use Helm charts for:
   - Tempo
   - Grafana
   - PostgreSQL (or use managed service)
3. Configure HPA (Horizontal Pod Autoscaler) for API servers
4. Set up Ingress for external access

### Additional Enhancements

- Add Prometheus for metrics storage
- Implement Loki for log aggregation
- Add service mesh (Istio/Linkerd) for advanced observability
- Implement exemplars (link metrics to traces)
- Add custom Grafana dashboards
- Implement SLO tracking

## Performance Considerations

### Otel Overhead

- Tracing adds ~1-5% overhead in most cases
- Batching reduces export overhead
- Sampling can be configured for high-traffic services

### Database Performance

- Connection pooling is enabled (default: 4 connections)
- Adjust pool size via pgx connection string:
  ```
  postgresql://...?pool_max_conns=20
  ```

### Collector Scaling

For high traffic, scale the Otel Collector:
```yaml
otel-collector:
  deploy:
    replicas: 3
```

## Security Notes

### Production Checklist

- [ ] Use TLS for gRPC (disable `-insecure` flag)
- [ ] Use TLS for Otel Collector connections
- [ ] Secure PostgreSQL with SSL
- [ ] Enable Grafana authentication (remove anonymous access)
- [ ] Use secrets management (not environment variables)
- [ ] Implement network policies
- [ ] Enable JWT authentication (`-enable-auth`)
- [ ] Rotate JWT secrets regularly

## Support

For issues or questions:
1. Check logs: `just docker-logs`
2. Review configuration files
3. Verify all services are healthy: `just docker-ps`

## References

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [Grafana Tempo Documentation](https://grafana.com/docs/tempo/latest/)
- [gRPC-Gateway Documentation](https://grpc-ecosystem.github.io/grpc-gateway/)
- [PostgreSQL pgx Driver](https://github.com/jackc/pgx)
