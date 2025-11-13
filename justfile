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