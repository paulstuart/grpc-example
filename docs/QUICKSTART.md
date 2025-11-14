# Quick Start Guide

## Step 1: Build & Start the Stack

```bash
# Ensure you have certificates (needed for TLS)
just gen-certs

# Build and start all services
just docker-up-build

# This will start:
# - PostgreSQL database
# - OpenTelemetry Collector
# - Grafana Tempo
# - Grafana UI
# - gRPC API server
```

**Wait ~30 seconds** for all services to initialize, especially PostgreSQL.

## Step 2: Verify Everything is Running

```bash
# Check all containers are up
docker compose ps

# Should show 5 services all "Up"
# NAME                    STATUS
# grpc-postgres           Up (healthy)
# grpc-otel-collector     Up
# grpc-tempo              Up
# grpc-grafana            Up
# grpc-example-api-1      Up
```

## Step 3: Generate Some Traces

### Option A: Via HTTP Gateway (Easiest)
```bash
# Make some API calls
curl -k https://localhost:11000/v1/users
curl -k https://localhost:11000/v1/users
curl -k https://localhost:11000/v1/users
```

### Option B: Add a test user
```bash
# Create a test user via REST API
curl -k -X POST https://localhost:11000/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "username": "testuser",
    "role": "USER",
    "email": "test@example.com"
  }'

# List users
curl -k https://localhost:11000/v1/users
```

### Option C: Use the test client
```bash
# Build the client
just build-client

# Run it (may need -insecure flag)
./client -server localhost:10000 -insecure
```

## Step 4: Verify Traces Are Being Collected

```bash
# Run the diagnostic script
./check-otel.sh
```

Look for:
- ✅ "Found services: grpc-example-api"
- ✅ "Found N traces"

### Manual Verification

```bash
# Check if Tempo received any traces
curl 'http://localhost:3200/api/search/tag/service.name/values' | jq .

# Should return:
# {
#   "tagValues": [
#     "grpc-example-api",
#     "grpc-example-api-gateway"
#   ]
# }
```

## Step 5: View Traces in Grafana

1. **Open Grafana:** http://localhost:3000
   - No login required (anonymous auth enabled)

2. **Navigate to Explore:**
   - Click the compass icon (🧭) in the left sidebar
   - Or go directly to: http://localhost:3000/explore

3. **Select Tempo datasource:**
   - In the dropdown at top, select "Tempo"

4. **Search for traces:**

   **Method 1: Service Name Search**
   - Click "Search" tab
   - In "Service Name" dropdown, select "grpc-example-api"
   - Click "Run Query"

   **Method 2: TraceQL**
   - Click "TraceQL" tab
   - Enter: `{ service.name = "grpc-example-api" }`
   - Click "Run Query"

5. **Click on a trace** to see the full span details:
   - Waterfall view showing span hierarchy
   - HTTP request → gRPC call → Database query
   - User attributes if authenticated
   - Duration, status codes, errors

## Troubleshooting

### No Traces Appearing?

**1. Check API container logs:**
```bash
docker compose logs api | grep -i otel

# You should see:
# "OpenTelemetry initialized: service=grpc-example-api..."
# "Trace provider initialized"
# "Metric provider initialized"
```

**2. Check collector is receiving data:**
```bash
docker compose logs otel-collector | grep -i span

# Look for lines like:
# Traces {"#spans": 5}
```

**3. Check for errors:**
```bash
# API errors
docker compose logs api | grep -i error

# Collector errors
docker compose logs otel-collector | grep -i error

# Tempo errors
docker compose logs tempo | grep -i error
```

**4. Verify environment variables:**
```bash
docker compose exec api env | grep OTEL

# Should show:
# OTEL_ENABLED=true
# OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
```

**5. Test network connectivity:**
```bash
# From API container, ping collector
docker compose exec api sh -c "nc -zv otel-collector 4317"

# Should show: "otel-collector (172.x.x.x:4317) open"
```

### Container Won't Start?

```bash
# View startup logs
docker compose logs <service-name>

# Common issues:
# - PostgreSQL: Port 5432 already in use
# - Grafana: Port 3000 already in use
# - API: Port 10000/11000 already in use

# Solution: Stop conflicting services or change ports in docker compose.yml
```

### Rebuild After Changes

```bash
# Stop everything
docker compose down

# Rebuild and restart
docker compose up -d --build

# Watch logs
docker compose logs -f
```

## Viewing Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f api
docker compose logs -f otel-collector
docker compose logs -f tempo
docker compose logs -f grafana

# Last 50 lines
docker compose logs --tail=50 api
```

## Testing the Full Flow

```bash
# 1. Make a request
curl -k https://localhost:11000/v1/users

# 2. Check API logged it
docker compose logs --tail=5 api
# Should see: [Unary] Started /proto.UserService/ListUsers
#             [Unary] Completed /proto.UserService/ListUsers successfully

# 3. Check collector received it
docker compose logs --tail=5 otel-collector | grep -i trace

# 4. Query Tempo
curl 'http://localhost:3200/api/search/tag/service.name/values'

# 5. Open Grafana and search for the trace
open http://localhost:3000/explore
```

## Advanced: Enable Debug Logging

If you still don't see traces, enable debug mode:

**1. Edit `otel-collector-config.yaml`:**
```yaml
exporters:
  logging:
    loglevel: debug  # Change from 'info'
    sampling_initial: 1
    sampling_thereafter: 1
```

**2. Restart collector:**
```bash
docker compose restart otel-collector
docker compose logs -f otel-collector
```

Now you'll see every span being exported in the logs.

## Success Indicators

When everything is working, you'll see:

✅ **API Logs:**
```
OpenTelemetry initialized: service=grpc-example-api, version=1.0.0, endpoint=otel-collector:4317
Trace provider initialized
Metric provider initialized
[Unary] Started /proto.UserService/ListUsers
[Unary] Completed /proto.UserService/ListUsers successfully (duration: 15ms)
```

✅ **Collector Logs:**
```
Traces {"kind": "exporter", "data_type": "traces", "name": "otlp/tempo", "#spans": 3}
```

✅ **Tempo Query:**
```bash
$ curl 'http://localhost:3200/api/search/tag/service.name/values' | jq .
{
  "tagValues": [
    "grpc-example-api",
    "grpc-example-api-gateway"
  ]
}
```

✅ **Grafana:** Shows traces with full span hierarchy

## Scaling Test

Once traces are working:

```bash
# Scale to 3 API servers
just docker-scale 3

# Make requests
for i in {1..20}; do curl -k https://localhost:11000/v1/users; done

# Check traces in Grafana - you'll see different server instances
```

## Next Steps

Once you see traces in Grafana:
- Explore different gRPC methods (GetUser, AddUser, streaming methods)
- Enable authentication (`-enable-auth`) and see user attributes in traces
- Test database operations and see SQL query spans
- Create custom Grafana dashboards
- Set up alerts on error rates
