# DNS Setup for Docker Services

This guide explains how to access Docker services from your host machine using their Docker DNS names.

## Problem

By default, Docker service names (like `grafana`, `tempo`, `postgres`) only work inside the Docker network. From your host machine, you have to use `localhost` with mapped ports.

## Solution

We provide a helper script that adds DNS entries to your `/etc/hosts` file, allowing you to use Docker service names from your host.

## Quick Start

### Setup DNS (one-time)

```bash
# Add Docker service DNS entries to /etc/hosts
just dns-setup
```

This adds:
- `rsyslog` / `grpc-rsyslog` → 127.0.0.1
- `postgres` / `grpc-postgres` → 127.0.0.1
- `tempo` / `grpc-tempo` → 127.0.0.1
- `grafana` / `grpc-grafana` → 127.0.0.1
- `otel-collector` / `grpc-otel-collector` → 127.0.0.1
- `api` / `grpc-example-api` → 127.0.0.1

### Check Status

```bash
# Check if DNS entries are installed
just dns-status
```

### Remove DNS (cleanup)

```bash
# Remove Docker service DNS entries from /etc/hosts
just dns-remove
```

## Usage Examples

### Before DNS Setup

```bash
# Had to use localhost with port mappings
curl http://localhost:3000              # Grafana
curl http://localhost:3200              # Tempo
curl https://localhost:11004/v1/users   # API (note: port 11004)
psql postgresql://localhost:5432/grpc_example
```

### After DNS Setup

```bash
# Can use service names (but still need correct ports)
curl http://grafana:3000                # Grafana
curl http://tempo:3200                  # Tempo
curl https://api:11004/v1/users         # API (port still 11004, not 11000)
psql postgresql://postgres:5432/grpc_example
```

## Important Notes

### Port Mappings Still Apply

DNS setup only resolves names to `127.0.0.1`. You still need to use the **host-side** port mappings:

| Service | Docker Internal | Host Port | Use From Host |
|---------|----------------|-----------|---------------|
| Grafana | `grafana:3000` | 3000 | `http://grafana:3000` |
| Tempo | `tempo:3200` | 3200 | `http://tempo:3200` |
| API gRPC | `api:10000` | 10004* | `https://api:10004` |
| API HTTP | `api:11000` | 11004* | `https://api:11004` |
| PostgreSQL | `postgres:5432` | 5432 | `postgresql://postgres:5432` |
| Otel Collector | `otel-collector:4317` | 4317 | `otel-collector:4317` |
| Rsyslog TCP | `rsyslog:514` | 514 | `tcp://rsyslog:514` |

\* Port numbers may vary when scaling API instances

### Check Current Port Mappings

```bash
docker compose ps

# Or use:
just docker-ps
```

### Why This Works

Docker Compose publishes container ports to the host at `127.0.0.1`. By adding service names to `/etc/hosts` pointing to `127.0.0.1`, we can use the service names instead of `localhost`.

## Manual Setup (Alternative)

If you prefer not to use the script, manually edit `/etc/hosts`:

```bash
sudo nano /etc/hosts
```

Add these lines:

```
# BEGIN grpc-example Docker services
127.0.0.1 rsyslog grpc-rsyslog
127.0.0.1 postgres grpc-postgres
127.0.0.1 tempo grpc-tempo
127.0.0.1 grafana grpc-grafana
127.0.0.1 otel-collector grpc-otel-collector
127.0.0.1 api grpc-example-api
# END grpc-example Docker services
```

## Limitations

1. **Only works on the host machine** - Other machines on your network can't use these names
2. **Requires sudo** - Editing `/etc/hosts` requires root privileges
3. **Port mappings still apply** - You still use host-side ports, not container ports
4. **No wildcard support** - Each service must be explicitly listed

## Advanced: True DNS Resolution

For more advanced setups (multiple developers, production-like environments), consider:

1. **dnsmasq** - Local DNS server that can handle wildcards (e.g., `*.docker`)
2. **Traefik** - Reverse proxy with automatic service discovery
3. **Kubernetes** - Full DNS and service mesh capabilities

For this project, the `/etc/hosts` approach is simple and sufficient.

## Troubleshooting

### DNS entries not working

```bash
# Check if entries exist
just dns-status

# If not, add them
just dns-setup

# Verify they're in the file
grep grpc-example /etc/hosts
```

### Port conflicts

```bash
# Check what ports are actually mapped
docker compose ps

# You may need to use the mapped port, not the internal port
```

### Permission denied

The script requires `sudo` to modify `/etc/hosts`. Make sure you run:

```bash
just dns-setup    # This will prompt for sudo password
```

Not:
```bash
sudo just dns-setup  # This won't work correctly
```

## Justfile Commands

```bash
just dns-setup      # Add DNS entries (requires sudo)
just dns-remove     # Remove DNS entries (requires sudo)
just dns-status     # Check if DNS is configured
```

## Direct Script Usage

```bash
# Add entries
sudo ./scripts/setup-dns.sh add

# Remove entries
sudo ./scripts/setup-dns.sh remove

# Check status
sudo ./scripts/setup-dns.sh status
```
