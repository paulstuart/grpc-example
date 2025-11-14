# Docker Compose Service Names Reference

## Important: Service Name vs Container Name

When using `docker compose` commands, always use the **service name**, not the container name!

| Service Name | Container Name | Use This With `docker compose` |
|--------------|----------------|----------------------------------|
| `postgres` | `grpc-postgres` | ✅ `docker compose exec postgres sh` |
| `otel-collector` | `grpc-otel-collector` | ✅ `docker compose exec otel-collector sh` |
| `tempo` | `grpc-tempo` | ✅ `docker compose exec tempo sh` |
| `grafana` | `grpc-grafana` | ✅ `docker compose exec grafana sh` |
| `api` | `grpc-example-api-1` (or `-2`, `-3` when scaled) | ✅ `docker compose exec api sh` |

## Common Commands

### PostgreSQL

```bash
# ✅ Correct - Access PostgreSQL CLI
docker compose exec postgres psql -U grpc_user -d grpc_example

# ✅ Correct - Access container shell
docker compose exec postgres sh

# ✅ Using justfile
just docker-psql              # PostgreSQL CLI
just docker-postgres-shell    # Container shell

# ❌ Wrong - will fail
docker compose exec grpc-postgres sh
```

### API Server

```bash
# ✅ Correct - Access API container
docker compose exec api sh

# ✅ Check environment variables
docker compose exec api env | grep OTEL

# ✅ Test connectivity
docker compose exec api nc -zv otel-collector 4317
```

### Otel Collector

```bash
# ✅ Access collector container
docker compose exec otel-collector sh

# ✅ View logs
docker compose logs otel-collector
```

### Tempo

```bash
# ✅ Access Tempo container
docker compose exec tempo sh

# ✅ View logs
docker compose logs tempo
```

### Grafana

```bash
# ✅ Access Grafana container
docker compose exec grafana sh

# ✅ View logs
docker compose logs grafana
```

## Why This Matters

- `docker compose` commands work with **service names** (defined in docker-compose.yml)
- `docker` commands work with **container names** (set by `container_name:`)

### Examples

```bash
# Docker Compose (use service names)
docker compose exec postgres psql -U grpc_user -d grpc_example  ✅
docker compose logs postgres                                     ✅
docker compose restart postgres                                  ✅

# Standalone Docker (can use container names)
docker exec -it grpc-postgres psql -U grpc_user -d grpc_example ✅
docker logs grpc-postgres                                        ✅
docker restart grpc-postgres                                     ✅
```

## Quick Command Reference

```bash
# List all running services (shows service names)
docker compose ps

# List all running containers (shows container names)
docker ps

# Access any service shell
docker compose exec <service-name> sh

# View logs for a service
docker compose logs <service-name>

# Restart a service
docker compose restart <service-name>

# Scale a service (only works with service names!)
docker compose up -d --scale api=3
```

## Troubleshooting

If you get an error like:
```
Error: No such service: grpc-postgres
```

You're using the **container name** instead of the **service name**. Change it:
- `grpc-postgres` → `postgres`
- `grpc-otel-collector` → `otel-collector`
- `grpc-tempo` → `tempo`
- `grpc-grafana` → `grafana`
- `grpc-example-api-1` → `api`

## When Scaling

When you scale the API service:
```bash
docker compose up -d --scale api=3
```

Docker creates multiple containers:
- `grpc-example-api-1`
- `grpc-example-api-2`
- `grpc-example-api-3`

But you still use the service name `api`:
```bash
# This accesses the first available API container
docker compose exec api sh

# To access a specific instance, use standalone docker:
docker exec -it grpc-example-api-2 sh
```
