SERVER_NAME := "pauleyphonic"

# Default recipe to display help information
default:
    @just --list

# Install required development tools
install:
    @echo "Installing development tools..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
    go install github.com/bufbuild/buf/cmd/buf@latest
    @echo "All tools installed successfully!"

# Generate protobuf and gRPC code using Buf
buf:
    @echo "Generating protobuf code with buf..."
    cd proto/protos && buf generate
    @echo "Code generation complete!"

# Generate protobuf and gRPC code (legacy protoc method)
generate:
    @echo "Generating protobuf code..."
    protoc -I proto/protos \
      -I ~/.cache/buf/v1/module/data/buf.build/googleapis/googleapis/d1263fe26f8e430a967dc22a4d0cad18 \
      --go_out=proto/pkg --go_opt=paths=source_relative \
      --go-grpc_out=proto/pkg --go-grpc_opt=paths=source_relative \
      --grpc-gateway_out=proto/pkg --grpc-gateway_opt=paths=source_relative,generate_unbound_methods=true \
      proto/protos/example.proto
    @echo "Code generation complete!"

# Lint protobuf files using Buf
protolint:
    @echo "Linting protobuf files..."
    cd proto/protos && buf lint

# Format protobuf files using Buf
format:
    @echo "Formatting protobuf files..."
    cd proto/protos && buf format -w

# Check for breaking changes in protobuf files
breaking:
    @echo "Checking for breaking changes..."
    cd proto/protos && buf breaking --against '.git#branch=master'

# Build the server binary
build:
    @echo "Building server..."
    go build -o grpc-example .
    @echo "Build complete!"

# Run the server
run: build
    @echo "Starting gRPC server..."
    ./grpc-example

# Run the server with insecure flag
run-insecure: build
    @echo "Starting gRPC server (insecure mode)..."
    ./grpc-example --insecure

# Run the server with custom TLS certificates
# Example: just run-with-tls cert=certs/server.crt key=certs/server.key
run-with-tls cert="certs/server.crt" key="certs/server.key": build
    @echo "Starting gRPC server with TLS..."
    ./grpc-example -cert={{cert}} -key={{key}}

# Run the server with authentication enabled
run-server-with-auth: build
    @echo "Starting gRPC server with authentication..."
    ./grpc-example -enable-auth

# Run tests
test:
    @echo "Running tests..."
    go test -v ./...

# Run tests with coverage
test-coverage:
    @echo "Running tests with coverage..."
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Clean generated files and binaries
clean:
    @echo "Cleaning generated files..."
    rm -f grpc-example
    rm -f coverage.out coverage.html
    rm -f proto/pkg/*.pb.go
    rm -f proto/pkg/*.pb.gw.go
    rm -rf third_party/OpenAPI/*
    @echo "Clean complete!"

# Update dependencies
update-deps:
    @echo "Updating Go dependencies..."
    go get -u ./...
    go mod tidy
    @echo "Dependencies updated!"

# Tidy Go modules
tidy:
    @echo "Tidying Go modules..."
    go mod tidy

# Run the client example
build-client:
    @echo "Building client example..."
    go build -o client cmd/client/main.go

run-client: build-client
    @echo "Running client example..."
    @./client

# Full regeneration and build
rebuild: clean generate build
    @echo "Full rebuild complete!"

# Check code quality (lint + test)
check: lint test
    @echo "All checks passed!"

# Newly added

lint:
    golangci-lint run ./...

# JWT Authentication targets

# Build the JWT token generator
build-tokengen:
    @#echo >&2 "Building token generator..."
    go build -o tokengen cmd/tokengen/main.go

# Generate a JWT token (use with JWT_SECRET env var or pass -secret flag)
# Example: just gen-token user-id=123 username=john email=john@example.com
# Or with JWT_SECRET: JWT_SECRET=my-secret just gen-token user-id=123 username=john email=john@example.com
gen-token user-id username email roles="user" duration="24h": build-tokengen
    @./tokengen -user-id={{user-id}} -username={{username}} -email={{email}} -roles={{roles}} -duration={{duration}}

# Run unit tests for auth package
test-auth:
    @echo "Running auth package tests..."
    go test -v ./auth/...

# Run integration tests for JWT auth interceptors
test-auth-integration:
    @echo "Running auth integration tests..."
    go test -v ./interceptors/...

# Run all auth tests
test-auth-all: test-auth test-auth-integration
    @echo "All auth tests completed!"

# Validate a JWT token
# Example: JWT_SECRET=my-secret just validate-token "eyJ..."
validate-token token:
    @./grpc-example -validate="{{token}}"

# Generate a token and run the client with it
# Example: JWT_SECRET=my-secret just run-client-with-auth user-id=123 username=john email=john@example.com
run-client-with-auth user-id username email roles="user": build-client build-tokengen
    #!/usr/bin/env bash
    set -euo pipefail
    TOKEN=$(./tokengen -user-id={{user-id}} -username={{username}} -email={{email}} -roles={{roles}} 2>/dev/null)
    echo "Using token for user: {{username}}" >&2
    ./client -token="$TOKEN"

CERTS_DIR := "certs"
SERVER_CERT := CERTS_DIR + "/server.crt"
SERVER_KEY := CERTS_DIR + "/server.key"

# Generate self-signed TLS certificates with proper SANs (Subject Alternative Names)
# This is required for modern TLS implementations that reject certificates with only CN
gen-certs:
    @echo "Generating TLS certificates with SANs..."
    @mkdir -p {{CERTS_DIR}}
    @openssl genrsa -out {{SERVER_KEY}} 2048 2>/dev/null
    @openssl req -x509 -new -nodes -key {{SERVER_KEY}} -sha256 -days 365 \
        -out {{SERVER_CERT}} \
        -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,DNS:*.localhost,DNS:{{SERVER_NAME}},IP:127.0.0.1,IP:0.0.0.0,IP:192.168.1.6" 2>/dev/null
    @echo "✓ Generated certificates in {{CERTS_DIR}}/"
    @echo "  Certificate: {{SERVER_CERT}}"
    @echo "  Private Key: {{SERVER_KEY}}"
    @echo "  SANs: localhost, *.localhost, {{SERVER_NAME}}, 127.0.0.1, 0.0.0.0, 192.168.1.6"

# Generate Let's Encrypt certificates (for production use)
certs:
    lego --email="pauleyphonic@gmail.com" --domains="internal.paulstuart.org" --http run

# Legacy targets (use gen-certs instead)
serverkey:
    @mkdir -p {{CERTS_DIR}}
    openssl genrsa -out {{SERVER_KEY}} 2048

signtls: serverkey
    @echo "Generating self-signed certificate with SANs..."
    openssl req -x509 -new -nodes -key {{SERVER_KEY}} -sha256 -days 365 \
        -out {{SERVER_CERT}} \
        -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,DNS:*.localhost,DNS:{{SERVER_NAME}},IP:127.0.0.1,IP:0.0.0.0,IP:192.168.1.6"

sample-server:
    ./grpc-example -host {{SERVER_NAME}} -enable-auth |& tee run5.log

sample-client:
   ./client -server {{SERVER_NAME}}:10000 -token $(cat testtoken) |& tee client01.log

# Docker / Docker Compose Commands

# Build Docker image for the API server
docker-build:
    @echo "Building Docker image..."
    docker build -t grpc-example:latest .
    @echo "Docker image built successfully!"

# Start the full stack (database, otel-collector, tempo, grafana, API)
docker-up:
    @echo "Starting Docker Compose stack..."
    docker compose up -d
    @echo "Stack is running!"
    @echo "  - Grafana:      http://localhost:3000"
    @echo "  - gRPC API:     https://localhost:10000"
    @echo "  - HTTP Gateway: https://localhost:11000"
    @echo "  - PostgreSQL:   localhost:5432"

# Start the stack with build
docker-up-build:
    @echo "Building and starting Docker Compose stack..."
    docker compose up -d --build
    @echo "Stack is running!"

# Scale API servers
# Example: just docker-scale 3
docker-scale count="2":
    @echo "Scaling API servers to {{count}} instances..."
    docker compose up -d --scale api={{count}}
    @echo "API servers scaled to {{count}} instances!"

# Stop the Docker Compose stack
docker-down:
    @echo "Stopping Docker Compose stack..."
    docker compose down
    @echo "Stack stopped!"

# Stop and remove volumes (destructive - deletes database data)
docker-down-volumes:
    @echo "Stopping Docker Compose stack and removing volumes..."
    docker compose down -v
    @echo "Stack stopped and volumes removed!"

# View logs from all services
docker-logs:
    docker compose logs -f

# View logs from API servers only
docker-logs-api:
    docker compose logs -f api

# View logs from a specific service
# Example: just docker-logs-service postgres
docker-logs-service service:
    docker compose logs -f {{service}}

# Restart a specific service
# Example: just docker-restart api
docker-restart service:
    @echo "Restarting {{service}}..."
    docker compose restart {{service}}

# Execute a command in the running API container
# Example: just docker-exec "ls -la"
docker-exec cmd:
    docker compose exec api {{cmd}}

# Access PostgreSQL CLI
docker-psql:
    docker compose exec postgres psql -U grpc_user -d grpc_example

# Access PostgreSQL container shell
docker-postgres-shell:
    docker compose exec postgres sh

# Access Rsyslog container shell
docker-rsyslog-shell:
    docker compose exec rsyslog sh

# View centralized logs from rsyslog
docker-rsyslog-logs:
    docker compose exec rsyslog tail -f /var/log/remote/all.log

# View logs for a specific service from rsyslog
# Example: just docker-rsyslog-service-logs postgres
docker-rsyslog-service-logs service:
    docker compose exec rsyslog find /var/lib/remote -name "{{service}}.log" -exec tail -f {} \;

# List all log files in rsyslog
docker-rsyslog-list:
    docker compose exec rsyslog find /var/log/remote -type f -name "*.log" | sort

# Check status of all services
docker-ps:
    docker compose ps

# Run with OpenTelemetry enabled (local development)
run-with-otel: build
    @echo "Starting server with OpenTelemetry..."
    OTEL_ENABLED=true OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 ./grpc-example

# Run with PostgreSQL database (requires PostgreSQL running)
run-with-postgres: build
    @echo "Starting server with PostgreSQL..."
    DATABASE_URL="postgresql://grpc_user:grpc_password@localhost:5432/grpc_example?sslmode=disable" ./grpc-example

# Full stack test: start everything and run client tests
docker-test: docker-up
    @echo "Waiting for services to be ready..."
    @sleep 10
    @echo "Running client tests against Docker stack..."
    @# Add client test commands here
    @echo "Tests complete!"

# Diagnostic and debugging commands

# Run comprehensive health check
docker-check:
    @./check-otel.sh

# Check if Tempo has any traces
docker-check-tempo:
    @echo "Checking Tempo for traces..."
    @curl -s 'http://localhost:3200/api/search/tag/service.name/values' | jq . || echo "Error querying Tempo"

# View Otel-related logs from API
docker-check-api-otel:
    @echo "API OpenTelemetry initialization logs:"
    @docker compose logs api | grep -i "otel\|trace.*initialized\|metric.*initialized" || echo "No Otel logs found"

# Check collector activity
docker-check-collector:
    @echo "Recent collector activity:"
    @docker compose logs --tail=20 otel-collector | grep -i "trace\|span\|export" || echo "No trace activity"

# Make test requests to generate traces
docker-generate-traces:
    @echo "Generating test traces..."
    @for i in 1 2 3 4 5; do \
        echo "Request $$i..."; \
        curl -sk https://localhost:11000/v1/users > /dev/null 2>&1; \
        sleep 1; \
    done
    @echo "✓ Sent 5 test requests"
    @echo "Check Grafana: http://localhost:3000/explore"

# Open Grafana in browser
docker-open-grafana:
    @echo "Opening Grafana..."
    @open http://localhost:3000 || xdg-open http://localhost:3000 || echo "Open http://localhost:3000 in your browser"

# Test network connectivity (host → containers and container → container)
docker-test-connectivity:
    @echo "Testing network connectivity..."
    @echo ""
    @echo "=== Host → Container Connectivity ==="
    @curl -s http://localhost:3000 > /dev/null && echo "✅ Grafana (localhost:3000)" || echo "❌ Grafana unreachable"
    @curl -s http://localhost:3200/ready > /dev/null && echo "✅ Tempo (localhost:3200)" || echo "❌ Tempo unreachable"
    @curl -s http://localhost:13133 > /dev/null && echo "✅ Otel Collector (localhost:13133)" || echo "❌ Collector unreachable"
    @curl -sk https://localhost:11000/v1/users > /dev/null 2>&1 && echo "✅ API HTTP Gateway (localhost:11000)" || echo "❌ API Gateway unreachable"
    @echo ""
    @echo "=== Container → Container Connectivity (from API) ==="
    @docker compose exec -T api sh -c "nc -zv -w2 rsyslog 514 2>&1" | grep -q "open" && echo "✅ API → Rsyslog (514)" || echo "❌ API → Rsyslog failed"
    @docker compose exec -T api sh -c "nc -zv -w2 otel-collector 4317 2>&1" | grep -q "open" && echo "✅ API → Otel Collector (4317)" || echo "❌ API → Collector failed"
    @docker compose exec -T api sh -c "nc -zv -w2 postgres 5432 2>&1" | grep -q "open" && echo "✅ API → PostgreSQL (5432)" || echo "❌ API → PostgreSQL failed"
    @docker compose exec -T api sh -c "nc -zv -w2 tempo 3200 2>&1" | grep -q "open" && echo "✅ API → Tempo (3200)" || echo "❌ API → Tempo failed"
    @docker compose exec -T api sh -c "nc -zv -w2 grafana 3000 2>&1" | grep -q "open" && echo "✅ API → Grafana (3000)" || echo "❌ API → Grafana failed"
    @echo ""
    @echo "Connectivity test complete!"

# Inspect Docker network details
docker-network-inspect:
    @echo "=== grpc-network Details ==="
    @docker network inspect grpc-network | jq '.[0].Containers | to_entries[] | {Name: .value.Name, IP: .value.IPv4Address, Container: .key[0:12]}'

# Complete diagnostic workflow
docker-diagnose: docker-check docker-check-tempo docker-check-api-otel docker-check-collector docker-test-connectivity
    @echo ""
    @echo "Diagnostic complete. If no traces found, try:"
    @echo "  1. just docker-generate-traces"
    @echo "  2. just docker-check-tempo"
    @echo "  3. just docker-open-grafana"
