# Rsyslog Quick Start

## What Was Added

A centralized rsyslog server that collects logs from all services in the cluster.

## Architecture

```
All Services → Rsyslog Server → Organized Log Files
   (syslog)      (tcp:514)        (/var/log/remote/)
```

## Quick Commands

```bash
# View all logs (combined from all services)
just docker-rsyslog-logs

# View logs from a specific service
docker compose exec rsyslog tail -f /var/log/remote/grpc-postgres/postgres.log

# List all available log files
just docker-rsyslog-list

# Access rsyslog shell
just docker-rsyslog-shell

# Search all logs for errors
docker compose exec rsyslog grep -i error /var/log/remote/all.log
```

## Services Logging to Rsyslog

All services now forward their logs:

| Service | Tag | Log Location |
|---------|-----|--------------|
| PostgreSQL | `postgres` | `/var/log/remote/grpc-postgres/postgres.log` |
| Otel Collector | `otel-collector` | `/var/log/remote/grpc-otel-collector/otel-collector.log` |
| Tempo | `tempo` | `/var/log/remote/grpc-tempo/tempo.log` |
| Grafana | `grafana` | `/var/log/remote/grpc-grafana/grafana.log` |
| gRPC API | `grpc-api` | `/var/log/remote/grpc-example-api-1/grpc-api.log` |

## Files Created

1. **`Dockerfile.rsyslog`** - Alpine-based rsyslog container
2. **`rsyslog.conf`** - Rsyslog configuration (UDP/TCP on 514)
3. **Updated `docker-compose.yml`** - Added rsyslog service and logging drivers
4. **`RSYSLOG_README.md`** - Complete documentation

## How It Works

1. Each Docker service uses the `syslog` logging driver
2. Logs are sent to `rsyslog:514` (TCP)
3. Rsyslog organizes logs by hostname and program name
4. All logs are also written to `/var/log/remote/all.log`

## Testing

```bash
# 1. Start the stack
just docker-up-build

# 2. Generate some activity
curl -k https://localhost:11000/v1/users

# 3. Watch logs appear
just docker-rsyslog-logs

# 4. Check specific service
docker compose exec rsyslog tail -20 /var/log/remote/grpc-example-api-1/grpc-api.log
```

## Benefits

✅ **Single Source of Truth** - All logs in one place
✅ **Survives Restarts** - Logs persist in `rsyslog_data` volume
✅ **Organized Structure** - Logs organized by service and hostname
✅ **Easy Searching** - Grep across all services at once
✅ **Cluster Ready** - Perfect for multi-node deployments

## Comparison

### Before (Docker Logs)
```bash
# View each service separately
docker compose logs postgres
docker compose logs api
docker compose logs tempo
# ... repeat for each service
```

### After (Rsyslog)
```bash
# View everything in one place
just docker-rsyslog-logs

# Or search across all services
docker compose exec rsyslog grep "error" /var/log/remote/all.log
```

## Advanced Usage

### Tail Multiple Services

```bash
# Watch API and PostgreSQL logs together
docker compose exec rsyslog sh -c '
  tail -f /var/log/remote/grpc-example-api-1/grpc-api.log \
          /var/log/remote/grpc-postgres/postgres.log
'
```

### Search with Context

```bash
# Find errors with 5 lines of context
docker compose exec rsyslog grep -C 5 -i error /var/log/remote/all.log
```

### Count Logs by Service

```bash
docker compose exec rsyslog sh -c '
  for log in /var/log/remote/*/*.log; do
    echo "$log: $(wc -l < $log) lines"
  done
'
```

## Next Steps

See **`RSYSLOG_README.md`** for complete documentation including:
- Log rotation strategies
- Integration with Grafana Loki
- Performance tuning
- Troubleshooting guide
