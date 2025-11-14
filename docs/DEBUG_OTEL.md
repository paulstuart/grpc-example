# Debugging OpenTelemetry Traces

## Quick Diagnostic Steps

### 1. Check if Otel Collector is Receiving Data

```bash
# View Otel Collector logs - should show trace exports
docker compose logs otel-collector | grep -i trace

# Look for lines like:
# "Traces" -> "ResourceSpans"
```

### 2. Query Tempo Directly

```bash
# Check Tempo health
curl http://localhost:3200/ready

# Query Tempo for traces (last hour)
curl -G http://localhost:3200/api/search \
  --data-urlencode 'start='$(($(date +%s)-3600)) \
  --data-urlencode 'end='$(date +%s) \
  | jq .

# List all services seen by Tempo
curl http://localhost:3200/api/search/tags | jq .

# List all service names
curl 'http://localhost:3200/api/search/tag/service.name/values' | jq .
```

### 3. Check API Server Logs

```bash
# View API server logs - look for "OpenTelemetry initialized"
docker compose logs api | grep -i otel

# Should see:
# "OpenTelemetry initialized: service=grpc-example-api, version=1.0.0, endpoint=otel-collector:4317"
# "Trace provider initialized"
# "Metric provider initialized"
# "OpenTelemetry metrics initialized"
```

### 4. Enable Debug Logging in Otel Collector

Edit `otel-collector-config.yaml` to increase verbosity:

```yaml
exporters:
  logging:
    loglevel: debug  # Changed from 'info'
    sampling_initial: 1  # Log every trace initially
    sampling_thereafter: 1
```

Then restart:
```bash
docker compose restart otel-collector
docker compose logs -f otel-collector
```

### 5. Test with a Simple Request

```bash
# Make a gRPC call via the HTTP Gateway
curl -k https://localhost:11000/v1/users

# Check logs immediately
docker compose logs --tail=50 api
docker compose logs --tail=50 otel-collector
```

### 6. Verify Network Connectivity

```bash
# From API container, test connection to collector
docker compose exec api sh -c "nc -zv otel-collector 4317"

# Should output: "otel-collector (172.x.x.x:4317) open"
```

## Common Issues & Fixes

### Issue 1: Otel Not Enabled in Docker

Check environment variables in docker compose.yml:
```yaml
api:
  environment:
    OTEL_ENABLED: "true"  # Make sure this is set!
```

### Issue 2: Collector Endpoint Wrong

API should use internal Docker network name:
```yaml
OTEL_EXPORTER_OTLP_ENDPOINT: otel-collector:4317  # NOT localhost:4317
```

### Issue 3: TLS Issues

The collector config uses `insecure: true`:
```yaml
exporters:
  otlp/tempo:
    endpoint: tempo:4317
    tls:
      insecure: true
```

### Issue 4: Grafana Not Connected to Tempo

Check datasource:
```bash
# Get datasources from Grafana
curl http://localhost:3000/api/datasources | jq .
```

Should show Tempo datasource with URL: `http://tempo:3200`

## Detailed Verification Script

Create this script to check everything:

```bash
#!/bin/bash
# save as check-otel.sh

echo "=== Checking OpenTelemetry Setup ==="

echo -e "\n1. Container Status:"
docker compose ps

echo -e "\n2. API Otel Initialization:"
docker compose logs api | grep -i "opentelemetry\|trace\|metric" | tail -10

echo -e "\n3. Collector Receiving Data:"
docker compose logs otel-collector | grep -i "trace\|span" | tail -10

echo -e "\n4. Tempo Health:"
curl -s http://localhost:3200/ready && echo " ✓ Tempo is ready" || echo " ✗ Tempo not ready"

echo -e "\n5. Services in Tempo:"
curl -s 'http://localhost:3200/api/search/tag/service.name/values' | jq -r '.tagValues[]' 2>/dev/null || echo "No services found"

echo -e "\n6. Recent Traces (last 10 min):"
START=$(($(date +%s) - 600))
END=$(date +%s)
curl -s -G "http://localhost:3200/api/search" \
  --data-urlencode "start=$START" \
  --data-urlencode "end=$END" \
  | jq '.traces | length' 2>/dev/null || echo "0"

echo -e "\n7. Collector Health:"
curl -s http://localhost:13133 && echo " ✓ Collector healthy" || echo " ✗ Collector not healthy"
```

Run it:
```bash
chmod +x check-otel.sh
./check-otel.sh
```

## Force Trace Generation

Use the test client to generate guaranteed traffic:

```bash
# Build client
just build-client

# Run client against Docker stack (may need to adjust for TLS)
./client -server localhost:10000 -insecure

# Or via HTTP Gateway
for i in {1..10}; do
  curl -k https://localhost:11000/v1/users
  sleep 1
done
```

## Check Grafana Explore

1. Open http://localhost:3000
2. Click "Explore" (compass icon)
3. Select "Tempo" datasource from dropdown
4. Try these queries:

**Query 1: Search by service name**
```
service.name="grpc-example-api"
```

**Query 2: Search by span name**
```
name="/proto.UserService/ListUsers"
```

**Query 3: TraceQL query**
```
{ service.name = "grpc-example-api" }
```

## Manual Trace Verification

Send a trace manually to verify the pipeline:

```bash
# Install grpcurl if needed
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Send OTLP trace directly to collector
# (This is complex, easier to just trigger via API calls)

# Instead, use the API with verbose logging
docker compose logs -f api &
curl -k https://localhost:11000/v1/users
```

Watch for log lines like:
- `[Unary] Started /proto.UserService/ListUsers`
- `[Unary] Completed /proto.UserService/ListUsers successfully`

## Environment Variable Check

Verify Otel is actually enabled:

```bash
# Check API container environment
docker compose exec api env | grep OTEL

# Should show:
# OTEL_ENABLED=true
# OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
```

## If Still No Traces

1. **Check API is actually using Otel interceptors:**
```bash
docker compose logs api | grep "OpenTelemetry.*initialized"
```

2. **Verify collector is starting correctly:**
```bash
docker compose logs otel-collector | grep -i error
```

3. **Check Tempo logs for errors:**
```bash
docker compose logs tempo | grep -i error
```

4. **Restart everything cleanly:**
```bash
docker compose down
docker compose up -d
sleep 10  # Wait for everything to start
./check-otel.sh
```

## Expected Output When Working

### API Logs
```
OpenTelemetry initialized: service=grpc-example-api, version=1.0.0, endpoint=otel-collector:4317
Trace provider initialized
Metric provider initialized
OpenTelemetry metrics initialized
HTTP Gateway instrumented with OpenTelemetry
[Unary] Started /proto.UserService/ListUsers
[Unary] Completed /proto.UserService/ListUsers successfully (duration: 2.5ms)
```

### Collector Logs
```
Traces  {"kind": "exporter", "data_type": "traces", "name": "otlp/tempo", "#spans": 5}
```

### Tempo Query Result
```json
{
  "traces": [
    {
      "traceID": "abc123...",
      "rootServiceName": "grpc-example-api",
      "rootTraceName": "GET /v1/users"
    }
  ]
}
```
