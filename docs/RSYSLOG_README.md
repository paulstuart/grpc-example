# Centralized Logging with Rsyslog

## Overview

The stack now includes a centralized rsyslog server that collects logs from all services, providing a single point for log aggregation and analysis.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  PostgreSQL │────▶│              │     │             │
└─────────────┘     │              │     │             │
                    │              │     │             │
┌─────────────┐     │              │     │  Rsyslog    │
│ Otel Collect│────▶│   Syslog     │────▶│  Logs       │
└─────────────┘     │   Network    │     │  Storage    │
                    │              │     │             │
┌─────────────┐     │              │     │ /var/log/   │
│    Tempo    │────▶│              │     │  remote/    │
└─────────────┘     │              │     │             │
                    │              │     └─────────────┘
┌─────────────┐     │              │
│   Grafana   │────▶│              │
└─────────────┘     │              │
                    │              │
┌─────────────┐     │              │
│  gRPC API   │────▶│              │
└─────────────┘     └──────────────┘
```

## Features

### Centralized Collection
- All services forward their logs to rsyslog via TCP/UDP on port 514
- Uses Docker's native syslog logging driver
- Logs organized by hostname and program name

### Log Organization
Logs are stored in `/var/log/remote/` organized as:
```
/var/log/remote/
├── all.log                          # Combined log from all services
├── grpc-postgres/
│   └── postgres.log                 # PostgreSQL logs
├── grpc-otel-collector/
│   └── otel-collector.log          # Collector logs
├── grpc-tempo/
│   └── tempo.log                    # Tempo logs
├── grpc-grafana/
│   └── grafana.log                  # Grafana logs
└── grpc-example-api-1/
    └── grpc-api.log                 # API server logs
```

### Tag-based Filtering
Each service has a unique tag for easy identification:
- `postgres` - PostgreSQL database
- `otel-collector` - OpenTelemetry Collector
- `tempo` - Grafana Tempo
- `grafana` - Grafana UI
- `grpc-api` - gRPC API servers

## Usage

### View All Logs (Combined)

```bash
# View all logs in real-time
just docker-rsyslog-logs

# Or manually
docker compose exec rsyslog tail -f /var/log/remote/all.log
```

### View Service-Specific Logs

```bash
# Access rsyslog shell
just docker-rsyslog-shell

# Inside the container, navigate to logs
cd /var/log/remote
ls -la

# View specific service logs
tail -f grpc-postgres/postgres.log
tail -f grpc-otel-collector/otel-collector.log
tail -f grpc-example-api-1/grpc-api.log
```

### List All Available Logs

```bash
# List all log files
just docker-rsyslog-list

# Output shows:
# /var/log/remote/all.log
# /var/log/remote/grpc-grafana/grafana.log
# /var/log/remote/grpc-otel-collector/otel-collector.log
# /var/log/remote/grpc-postgres/postgres.log
# /var/log/remote/grpc-tempo/tempo.log
# /var/log/remote/grpc-example-api-1/grpc-api.log
```

### Search Logs

```bash
# Access rsyslog container
docker compose exec rsyslog sh

# Search all logs for errors
grep -i error /var/log/remote/all.log

# Search specific service logs
grep -i "connection refused" /var/log/remote/grpc-postgres/postgres.log

# Count log entries by service
wc -l /var/log/remote/*//*.log
```

## Configuration

### Rsyslog Server Config

Located in `rsyslog.conf`:

```conf
# Accept logs via UDP and TCP on port 514
module(load="imudp")
module(load="imtcp")
input(type="imudp" port="514")
input(type="imtcp" port="514")

# Template for organizing logs by hostname/program
template(name="DynFile" type="string"
         string="/var/log/remote/%HOSTNAME%/%PROGRAMNAME%.log")

# Log format with timestamp, hostname, and message
template(name="DetailedFormat" type="string"
         string="%TIMESTAMP% %HOSTNAME% %syslogtag%%msg%\n")
```

### Docker Logging Configuration

Each service in `docker-compose.yml` uses:

```yaml
logging:
  driver: syslog
  options:
    syslog-address: "tcp://rsyslog:514"
    tag: "service-name"
```

## Justfile Commands

```bash
# Access rsyslog shell
just docker-rsyslog-shell

# View all combined logs
just docker-rsyslog-logs

# View specific service logs
just docker-rsyslog-service-logs postgres
just docker-rsyslog-service-logs grpc-api

# List all available log files
just docker-rsyslog-list
```

## Log Persistence

Logs are stored in a Docker volume: `rsyslog_data`

```bash
# Logs persist across container restarts
docker compose restart rsyslog

# To clear logs, remove the volume (destructive!)
docker compose down -v  # Removes ALL volumes including logs
```

## Viewing Logs During Development

### Option 1: Centralized (Rsyslog)
```bash
# View all services in one place
just docker-rsyslog-logs

# Advantage: Single stream, organized by hostname
# Disadvantage: Slight delay due to network transmission
```

### Option 2: Per-Service (Docker Logs)
```bash
# View docker logs (still works, but goes to rsyslog)
docker compose logs -f api

# Note: This shows local docker buffering before rsyslog
```

### Option 3: Combined View
```bash
# Terminal 1: Watch rsyslog aggregated logs
just docker-rsyslog-logs

# Terminal 2: Watch specific service
docker compose logs -f postgres
```

## Log Rotation

To prevent logs from growing indefinitely, you can implement rotation:

### Manual Rotation
```bash
# Access rsyslog container
docker compose exec rsyslog sh

# Truncate old logs
truncate -s 0 /var/log/remote/all.log

# Or rotate logs
mv /var/log/remote/all.log /var/log/remote/all.log.old
```

### Automated Rotation (TODO)
Add logrotate to the rsyslog container:
```dockerfile
RUN apk add --no-cache logrotate
COPY logrotate.conf /etc/logrotate.d/rsyslog
```

## Troubleshooting

### Logs Not Appearing

1. **Check rsyslog is running:**
```bash
docker compose ps rsyslog
# Should show "Up (healthy)"
```

2. **Verify rsyslog is listening:**
```bash
docker compose exec rsyslog netstat -tuln | grep 514
# Should show:
# tcp        0      0 0.0.0.0:514             0.0.0.0:*               LISTEN
# udp        0      0 0.0.0.0:514             0.0.0.0:*
```

3. **Check service can reach rsyslog:**
```bash
just docker-test-connectivity
# Should show: ✅ API → Rsyslog (514)
```

4. **Verify log directory:**
```bash
docker compose exec rsyslog ls -la /var/log/remote/
```

### Permission Issues

```bash
# Check log directory permissions
docker compose exec rsyslog ls -la /var/log/

# Fix if needed
docker compose exec rsyslog chmod 755 /var/log/remote
```

### High Volume / Performance

For high-traffic scenarios:

1. **Use UDP instead of TCP** (faster, but less reliable):
```yaml
logging:
  driver: syslog
  options:
    syslog-address: "udp://rsyslog:514"
```

2. **Reduce verbosity** in application logs

3. **Enable log sampling** in rsyslog config

## Integration with Observability Stack

### Correlation with Traces

Logs include timestamps that can be correlated with:
- OpenTelemetry traces in Tempo
- Metrics in Prometheus/Grafana

Example workflow:
1. View trace in Grafana showing an error
2. Note the timestamp
3. Search rsyslog for that timestamp:
```bash
docker compose exec rsyslog grep "2025-01-14T10:30:45" /var/log/remote/all.log
```

### Future: Loki Integration

For advanced log querying, consider adding Grafana Loki:
- Loki can ingest from rsyslog
- Provides powerful log queries in Grafana UI
- Correlates logs with traces automatically

## Best Practices

1. **Tag Services Appropriately**: Use descriptive tags for easy filtering
2. **Monitor Disk Usage**: Logs can grow large; monitor `/var/log/remote`
3. **Implement Rotation**: Set up logrotate for production
4. **Use Structured Logging**: JSON logs are easier to parse and search
5. **Set Log Levels**: Use appropriate log levels (DEBUG, INFO, WARN, ERROR)

## Example Queries

```bash
# Find all errors in the last hour
docker compose exec rsyslog sh -c "
  tail -10000 /var/log/remote/all.log | \
  grep -i error | \
  tail -50
"

# Count log entries per service
docker compose exec rsyslog sh -c "
  for log in /var/log/remote/*/*.log; do
    echo \"\$log: \$(wc -l < \$log) lines\"
  done
"

# Find slow queries (if logged)
docker compose exec rsyslog grep -i "slow query" /var/log/remote/grpc-postgres/postgres.log
```

## Scaling Considerations

When scaling API servers (`docker compose up --scale api=3`):
- Each instance gets its own log directory
- Example:
  - `/var/log/remote/grpc-example-api-1/grpc-api.log`
  - `/var/log/remote/grpc-example-api-2/grpc-api.log`
  - `/var/log/remote/grpc-example-api-3/grpc-api.log`

View all API logs combined:
```bash
docker compose exec rsyslog tail -f /var/log/remote/grpc-example-api-*/grpc-api.log
```
