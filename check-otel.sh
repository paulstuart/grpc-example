#!/bin/bash
# OpenTelemetry Stack Diagnostic Script

set -e

echo "======================================"
echo "OpenTelemetry Stack Health Check"
echo "======================================"

echo -e "\n📦 1. Container Status:"
docker compose ps 2>/dev/null || echo "❌ Docker Compose not running"

echo -e "\n🔧 2. API Otel Initialization:"
echo "Looking for Otel initialization logs..."
docker compose logs api 2>/dev/null | grep -i "opentelemetry\|trace.*initialized\|metric.*initialized" | tail -5 || echo "⚠️  No Otel logs found in API"

echo -e "\n📊 3. Collector Receiving Data:"
echo "Checking if collector is processing traces..."
docker compose logs otel-collector 2>/dev/null | grep -i "trace\|#spans" | tail -5 || echo "⚠️  No trace logs in collector"

echo -e "\n💓 4. Tempo Health Check:"
if curl -s http://localhost:3200/ready >/dev/null 2>&1; then
    echo "✅ Tempo is ready"
else
    echo "❌ Tempo not responding"
fi

echo -e "\n🏷️  5. Services Tracked by Tempo:"
SERVICES=$(curl -s 'http://localhost:3200/api/search/tag/service.name/values' 2>/dev/null | jq -r '.tagValues[]? // empty' 2>/dev/null)
if [ -n "$SERVICES" ]; then
    echo "✅ Found services:"
    echo "$SERVICES" | sed 's/^/   - /'
else
    echo "⚠️  No services found (this means no traces have been received)"
fi

echo -e "\n🔍 6. Recent Traces Count (last 10 minutes):"
START=$(($(date +%s) - 600))
END=$(date +%s)
TRACE_COUNT=$(curl -s -G "http://localhost:3200/api/search" \
  --data-urlencode "start=$START" \
  --data-urlencode "end=$END" 2>/dev/null \
  | jq '.traces | length' 2>/dev/null)
if [ -n "$TRACE_COUNT" ] && [ "$TRACE_COUNT" -gt 0 ]; then
    echo "✅ Found $TRACE_COUNT traces"
else
    echo "⚠️  No traces found in last 10 minutes"
fi

echo -e "\n🏥 7. Collector Health Endpoint:"
if curl -s http://localhost:13133 >/dev/null 2>&1; then
    echo "✅ Collector health endpoint responding"
else
    echo "❌ Collector health endpoint not responding"
fi

echo -e "\n🌐 8. API Environment Variables:"
docker compose exec -T api env 2>/dev/null | grep OTEL || echo "⚠️  No OTEL env vars found"

echo -e "\n📝 9. Recent API Activity (last 10 lines):"
docker compose logs --tail=10 api 2>/dev/null || echo "❌ Cannot read API logs"

echo -e "\n🔬 10. Collector Errors:"
ERRORS=$(docker compose logs otel-collector 2>/dev/null | grep -i error | tail -3)
if [ -n "$ERRORS" ]; then
    echo "⚠️  Found errors:"
    echo "$ERRORS"
else
    echo "✅ No errors in collector logs"
fi

echo -e "\n======================================"
echo "Health Check Complete"
echo "======================================"

# Summary
echo -e "\n📋 Quick Actions:"
echo "  • View API logs:       docker compose logs -f api"
echo "  • View collector logs: docker compose logs -f otel-collector"
echo "  • Test API:            curl -k https://localhost:11000/v1/users"
echo "  • Open Grafana:        http://localhost:3000"
echo "  • Query Tempo:         curl 'http://localhost:3200/api/search/tag/service.name/values'"
