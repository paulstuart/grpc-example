# Docker Networking Architecture

This document explains the networking design decisions for this project.

## Network Configuration

### Custom Bridge Network

```yaml
networks:
  grpc-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.21.0.0/16
```

- **Subnet:** 172.21.0.0/16 (65,534 available IP addresses)
- **Driver:** bridge (default, provides container-to-container communication)
- **DNS:** Automatic service name resolution within the network

## Service Communication

### DNS-Based Communication (Preferred)

Services use DNS names for communication **inside the Docker network**:

```yaml
# API connects to other services by name
OTEL_EXPORTER_OTLP_ENDPOINT: otel-collector:4317  # ✅
DATABASE_URL: postgresql://postgres:5432/...      # ✅

# Otel collector connects to Tempo by name
endpoint: tempo:4317                              # ✅
```

**Why DNS names?**
- Easy to read and maintain
- Automatic resolution within Docker network
- Works with service scaling
- No hardcoded IPs to update

### Static IP for Rsyslog (Exception)

Rsyslog has a **static IP assignment**:

```yaml
rsyslog:
  networks:
    grpc-network:
      ipv4_address: 172.21.0.10  # Static IP
```

And all services use this **hardcoded IP** in their logging configuration:

```yaml
logging:
  driver: syslog
  options:
    syslog-address: "udp://172.21.0.10:514"  # Hardcoded IP
```

**Why hardcoded IP instead of DNS name?**

This is a **Docker limitation**, not a design choice:

1. **Timing Issue:** Docker's logging driver is initialized **before** the container starts
2. **No DNS Available:** Container DNS resolution isn't available until the container's network namespace is created
3. **Chicken-and-Egg:** The logging driver needs to know where to send logs before DNS exists

```
Docker daemon starts container:
1. Initialize logging driver  ← DNS not available yet!
2. Create network namespace
3. Attach to network
4. Enable DNS resolution     ← DNS available now
5. Start container process
```

If we tried to use `syslog-address: "udp://rsyslog:514"`, we'd get:
```
Error: dial udp: lookup rsyslog on 192.168.65.7:53: no such host
```

## IP Address Allocation

### Static Assignments

| Service | IP | Reason |
|---------|-----|---------|
| rsyslog | 172.21.0.10 | Required for Docker logging driver |

### Dynamic Assignments (DHCP)

All other services get dynamic IPs from the 172.21.0.0/16 pool:
- postgres (typically 172.21.0.2)
- tempo (typically 172.21.0.3)
- otel-collector (typically 172.21.0.4)
- grafana (typically 172.21.0.5)
- api (172.21.0.6+, increases with scaling)

## Why Not All Static IPs?

We could assign static IPs to all services:

```yaml
# We DON'T do this
postgres:
  networks:
    grpc-network:
      ipv4_address: 172.21.0.11
tempo:
  networks:
    grpc-network:
      ipv4_address: 172.21.0.12
```

**Reasons we don't:**

1. **DNS works fine** - Services can already communicate by name
2. **Easier scaling** - `docker compose up --scale api=5` works automatically
3. **Less maintenance** - No IP conflicts to manage
4. **Standard practice** - Only use static IPs when technically required

## Port Mappings

### Container Ports (Internal)

Services use standard ports inside the Docker network:

```yaml
# Inside Docker network
postgres:5432
grafana:3000
tempo:3200
otel-collector:4317
api:10000
api:11000  # HTTP Gateway
rsyslog:514
```

### Host Ports (External)

Services are exposed to the host with port mappings:

```yaml
ports:
  - "3000:3000"      # Grafana: host:3000 → container:3000
  - "5432:5432"      # PostgreSQL: host:5432 → container:5432
  - "10000:10000"    # API gRPC: host:10000 → container:10000
  - "11000:11000"    # API HTTP: host:11000 → container:11000
```

**Note:** When scaling API servers, Docker auto-assigns different host ports:
```bash
docker compose up --scale api=3

# Result:
api-1: host:10000 → container:10000, host:11000 → container:11000
api-2: host:10001 → container:10000, host:11001 → container:11000
api-3: host:10002 → container:10000, host:11002 → container:11000
```

## Service Discovery

### Inside Docker Network

Services discover each other automatically via DNS:

```bash
# From inside any container
ping postgres        # Works!
ping tempo           # Works!
ping otel-collector  # Works!
curl http://grafana:3000
```

### From Host Machine

Without DNS setup, use `localhost` with mapped ports:

```bash
curl http://localhost:3000              # Grafana
curl http://localhost:3200              # Tempo
psql postgresql://localhost:5432/...    # PostgreSQL
```

With DNS setup (using `just dns-setup`):

```bash
curl http://grafana:3000                # Grafana (via /etc/hosts)
curl http://tempo:3200                  # Tempo (via /etc/hosts)
psql postgresql://postgres:5432/...     # PostgreSQL (via /etc/hosts)
```

See [DNS_SETUP.md](DNS_SETUP.md) for details.

## Network Isolation

### Pros of Custom Network

- ✅ Services are isolated from other Docker containers
- ✅ Automatic service name resolution
- ✅ Can control subnet to avoid conflicts
- ✅ Supports network-level monitoring

### Cons

- Services can't directly communicate with containers on other networks
- Need explicit port publishing for host access

## Troubleshooting

### Check service IP addresses

```bash
# Inspect the network
docker network inspect grpc-example_grpc-network

# Or use jq to extract IPs
docker network inspect grpc-example_grpc-network | \
  jq -r '.[] | .Containers | to_entries[] | "\(.value.Name): \(.value.IPv4Address)"'
```

### Verify DNS resolution

```bash
# From inside a container
docker compose exec api ping -c 1 postgres
docker compose exec api ping -c 1 tempo
docker compose exec api nslookup rsyslog
```

### Test connectivity

```bash
# From host to containers
curl http://localhost:3000
curl http://localhost:3200

# From container to container
docker compose exec api curl http://grafana:3000
docker compose exec api curl http://tempo:3200
```

## Best Practices

### DO:
- ✅ Use DNS names for service-to-service communication
- ✅ Use static IPs only when technically required
- ✅ Document any static IP assignments
- ✅ Use descriptive service names
- ✅ Define explicit port mappings

### DON'T:
- ❌ Hardcode IPs in application code
- ❌ Use `host` network mode (breaks container isolation)
- ❌ Assign static IPs without documentation
- ❌ Forget to update rsyslog IP if you change it

## Summary

| Aspect | Approach | Reason |
|--------|----------|--------|
| **Network Type** | Custom bridge | Isolation + DNS |
| **Service Discovery** | DNS names | Standard Docker practice |
| **Rsyslog IP** | Static (172.21.0.10) | Docker logging driver limitation |
| **Other Services** | Dynamic DHCP | Flexibility + simplicity |
| **Logging Addresses** | Hardcoded IP | No DNS during driver init |
| **App Connections** | DNS names | Best practice |

The **only** hardcoded IP in the system is for rsyslog's syslog logging driver, and this is due to a Docker technical limitation, not a design preference.
