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
# Example: JWT_SECRET=my-secret just run-with-auth user-id=123 username=john email=john@example.com
run-with-auth user-id username email roles="user": build-client build-tokengen
    #!/usr/bin/env bash
    set -euo pipefail
    TOKEN=$(./tokengen -user-id={{user-id}} -username={{username}} -email={{email}} -roles={{roles}} 2>/dev/null)
    echo "Using token for user: {{username}}" >&2
    ./client -token="$TOKEN"

CERTS_DIR := "certs"
SERVER_CERT := CERTS_DIR + "/server.crt"
SERVER_KEY := CERTS_DIR + "/server.key"

certs:
    lego --email="pauleyphonic@gmail.com" --domains="internal.paulstuart.org" --http run

serverkey:
    @mkdir -p {{CERTS_DIR}}
    openssl genrsa -out {{SERVER_KEY}} 2048

signtls: serverkey
    openssl req -x509 -new -nodes -key {{SERVER_KEY}} -sha256 -days 365 -out {{SERVER_CERT}}