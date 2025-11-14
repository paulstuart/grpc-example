# Docker Compose Command Reference

## Quick Start

```bash
# 1. Generate certificates
just gen-certs

# 2. Start everything
just docker-up-build

# 3. Check health
just docker-diagnose

# 4. Generate test traffic
just docker-generate-traces

# 5. Open Grafana
just docker-open-grafana
```

## Essential Commands

### Start/Stop
```bash
# Start all services
docker compose up -d

# Start with rebuild
docker compose up -d --build

# Stop all services
docker compose down

# Stop and remove volumes (deletes data!)
docker compose down -v
```

### Scale API Servers
```bash
# Scale to 3 instances
docker compose up -d --scale api=3

# Or use just
just docker-scale 3
```

### View Logs
```bash
# All services, follow mode
docker compose logs -f

# Specific service
docker compose logs -f api
docker compose logs -f otel-collector
docker compose logs -f tempo
docker compose logs -f grafana

# Last N lines
docker compose logs --tail=50 api
```

### Service Management
```bash
# Check status
docker compose ps

# Restart a service
docker compose restart api

# Execute command in container
docker compose exec api sh

# Access PostgreSQL
docker compose exec postgres psql -U grpc_user -d grpc_example
```

## Debugging Commands

### Check OpenTelemetry
```bash
# Comprehensive check
just docker-check

# Check specific components
just docker-check-tempo       # Query Tempo for services
just docker-check-api-otel    # View API Otel logs
just docker-check-collector   # Check collector activity
```

### Manual Queries
```bash
# Query Tempo for services
curl 'http://localhost:3200/api/search/tag/service.name/values' | jq .

# Check Tempo health
curl http://localhost:3200/ready

# Check collector health
curl http://localhost:13133

# Search for traces (last 10 minutes)
START=$(($(date +%s) - 600))
END=$(date +%s)
curl -G "http://localhost:3200/api/search" \
  --data-urlencode "start=$START" \
  --data-urlencode "end=$END" | jq .
```

### Generate Test Traffic
```bash
# Use just command
just docker-generate-traces

# Or manually
for i in {1..10}; do
  curl -k https://localhost:11000/v1/users
  sleep 1
done
```

### View Environment Variables
```bash
# Check API container env vars
docker compose exec api env | grep OTEL
```

## Common Issues

### Services Won't Start
```bash
# View startup logs
docker compose logs <service-name>

# Rebuild and restart
docker compose down
docker compose up -d --build
```

### Port Conflicts
Edit `docker-compose.yml` to change ports:
- PostgreSQL: `5432:5432`
- Grafana: `3000:3000`
- API gRPC: `10000:10000`
- API HTTP: `11000:11000`

### No Traces Appearing
```bash
# 1. Check API initialized Otel
docker compose logs api | grep "OpenTelemetry initialized"

# 2. Generate traffic
curl -k https://localhost:11000/v1/users

# 3. Check collector received data
docker compose logs otel-collector | grep -i span

# 4. Query Tempo
curl 'http://localhost:3200/api/search/tag/service.name/values'
```

### Database Issues
```bash
# Check PostgreSQL is ready
docker compose exec postgres pg_isready -U grpc_user

# View database logs
docker compose logs postgres

# Connect to database
docker compose exec postgres psql -U grpc_user -d grpc_example
```

## Useful Justfile Commands

```bash
# Docker management
just docker-up              # Start stack
just docker-up-build        # Start with build
just docker-down            # Stop stack
just docker-logs            # View all logs
just docker-logs-api        # View API logs
just docker-ps              # Check status
just docker-psql            # PostgreSQL CLI

# Diagnostics
just docker-check           # Full health check
just docker-diagnose        # Complete diagnostic
just docker-generate-traces # Generate test traffic
just docker-open-grafana    # Open browser

# Scaling
just docker-scale 3         # Scale to 3 API servers
```

## Access URLs

| Service | URL | Description |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | Trace visualization UI |
| gRPC API | https://localhost:10000 | gRPC endpoint (TLS) |
| HTTP Gateway | https://localhost:11000 | REST API (TLS) |
| OpenAPI UI | https://localhost:11000/openapi-ui/ | Swagger UI |
| Tempo | http://localhost:3200 | Tempo API |
| Otel Collector | http://localhost:4317 | OTLP gRPC |
| Collector Health | http://localhost:13133 | Health check |
| PostgreSQL | localhost:5432 | Database (user: grpc_user, db: grpc_example) |
| Rsyslog | tcp://localhost:514 | Centralized log collection |

## Complete Workflow Example

```bash
# 1. Start the stack
just gen-certs
just docker-up-build

# 2. Wait for initialization
sleep 30

# 3. Check everything is healthy
just docker-check

# 4. Generate some traces
just docker-generate-traces

# 5. Verify traces in Tempo
curl 'http://localhost:3200/api/search/tag/service.name/values' | jq .

# 6. Open Grafana and explore
just docker-open-grafana
# Navigate to Explore → Tempo → Search for "grpc-example-api"

# 7. Scale up
just docker-scale 3

# 8. Generate more traffic
for i in {1..50}; do curl -k https://localhost:11000/v1/users; done

# 9. Watch traces from multiple instances in Grafana
```

## Cleanup

```bash
# Stop services (keep data)
docker compose down

# Remove everything including volumes
docker compose down -v

# Remove Docker images
docker rmi grpc-example:latest

# Full cleanup
docker compose down -v
docker system prune -a --volumes
```
