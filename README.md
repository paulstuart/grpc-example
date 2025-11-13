# gRPC-Example

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A comprehensive example demonstrating gRPC capabilities with Go, showcasing all RPC patterns, modern tooling, and best practices.

## Features

### gRPC Patterns
This project demonstrates **all four gRPC communication patterns**:

- **Unary RPC** - Simple request/response (AddUser, GetUser, UpdateUser, DeleteUser)
- **Server Streaming RPC** - Server streams multiple responses (ListUsers, ListUsersByRole)
- **Client Streaming RPC** - Client streams multiple requests (BatchAddUsers)
- **Bidirectional Streaming RPC** - Both client and server stream (UserActivityStream, SyncUsers)

### Protocol Buffers Features
Comprehensive demonstration of protobuf data modeling:

- **Enums** - Role, UserStatus, ActivityType
- **Nested Messages** - Profile, Address
- **Oneofs** - Contact info (email or phone)
- **Maps** - Metadata, preferences
- **Repeated Fields** - Tags, addresses
- **Well-Known Types** - Timestamp, Duration, FieldMask, Empty

### Modern Tooling

- **Buf** - Modern protobuf tooling instead of protoc directly
- **Justfile** - Modern command runner instead of Makefile
- **Standard Protobuf** - Uses google.golang.org/protobuf (no deprecated gogo/protobuf)
- **gRPC-Gateway** - Automatic REST API generation

### Architecture

- **Storage Interface** - Pluggable backend abstraction
- **In-Memory Storage** - Default implementation with proper locking
- **Interceptors** - Logging, authentication, and metrics
- **Graceful Shutdown** - Proper cleanup and resource management

## Quick Start

### Prerequisites

- Go 1.20 or later
- Buf (install via `go install github.com/bufbuild/buf/cmd/buf@latest`)
- Just (optional, install from https://github.com/casey/just)

### Installation

```bash
git clone https://github.com/paulstuart/grpc-example
cd grpc-example
```

### Install Development Tools

Using Just (recommended):
```bash
just install
```

Or using Go directly:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
go install github.com/bufbuild/buf/cmd/buf@latest
```

### Generate Protobuf Code

Using Just:
```bash
just generate
```

Or using protoc directly (requires buf mod update to cache googleapis):
```bash
protoc -I proto \
  -I ~/.cache/buf/v1/module/data/buf.build/googleapis/googleapis/$(ls ~/.cache/buf/v1/module/data/buf.build/googleapis/googleapis/ | head -1) \
  --go_out=proto --go_opt=paths=source_relative \
  --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=proto --grpc-gateway_opt=paths=source_relative \
  proto/example.proto
```

Note: The generated files are already included in the repository, so you only need to regenerate if you modify the proto files.

### Build

Using Just:
```bash
just build
```

Or using Go:
```bash
go build .
```

### Run the Server

Using Just:
```bash
just run
```

Or directly:
```bash
./grpc-example
```

**Server Options:**
- `--grpc-port` - gRPC server port (default: 10000)
- `--gateway-port` - HTTP gateway port (default: 11000)
- `--insecure` - Skip TLS verification
- `--enable-auth` - Enable authentication interceptor
- `--print-metrics` - Print metrics on shutdown

Example with auth and metrics:
```bash
./grpc-example --enable-auth --print-metrics
```

## Running the Client

The client demonstrates all RPC patterns:

```bash
go run cmd/client/main.go
```

**Client Options:**
- `--server` - gRPC server address (default: localhost:10000)
- `--insecure` - Use insecure connection

The client will execute demonstrations of:
1. Unary RPC: AddUser
2. Unary RPC: GetUser
3. Unary RPC: UpdateUser (with Field Mask)
4. Server Streaming RPC: ListUsers
5. Server Streaming RPC: ListUsersByRole
6. Client Streaming RPC: BatchAddUsers
7. Bidirectional Streaming RPC: UserActivityStream
8. Bidirectional Streaming RPC: SyncUsers
9. Unary RPC: DeleteUser

## API Endpoints

The server exposes both gRPC and REST endpoints via gRPC-Gateway:

### gRPC Endpoints
- `UserService/AddUser` - Add a new user
- `UserService/GetUser` - Get user by ID
- `UserService/UpdateUser` - Update user with field mask
- `UserService/DeleteUser` - Delete user
- `UserService/ListUsers` - List users with filters (server streaming)
- `UserService/ListUsersByRole` - List users by role (server streaming)
- `UserService/BatchAddUsers` - Batch add users (client streaming)
- `UserService/UserActivityStream` - Track user activity (bidirectional streaming)
- `UserService/SyncUsers` - Sync user data (bidirectional streaming)

### REST Endpoints
- `POST /api/v1/users` - Add user
- `GET /api/v1/users/{id}` - Get user
- `PATCH /api/v1/users/{id}` - Update user
- `DELETE /api/v1/users/{id}` - Delete user
- `GET /api/v1/users` - List users
- `GET /api/v1/users/role/{role}` - List users by role

### OpenAPI Documentation
- `https://localhost:11000/openapi-ui/` - Interactive API documentation

## Project Structure

```
.
├── buf.gen.yaml              # Buf code generation config
├── buf.yaml                  # Buf linting and breaking change detection
├── justfile                  # Modern command runner (replaces Makefile)
├── CLAUDE.md                 # Project requirements and goals
├── cmd/
│   └── client/
│       └── main.go           # Comprehensive client demonstrating all RPC patterns
├── interceptors/
│   ├── logging.go            # Request/response logging
│   ├── auth.go               # Authentication (demo implementation)
│   └── metrics.go            # Request metrics collection
├── proto/
│   ├── example.proto         # Comprehensive protobuf definitions
│   ├── example.pb.go         # Generated protobuf code
│   ├── example_grpc.pb.go    # Generated gRPC code
│   └── example.pb.gw.go      # Generated gRPC-Gateway code
├── server/
│   ├── storage.go            # Storage interface definition
│   ├── memory_storage.go     # In-memory storage implementation
│   └── server.go             # gRPC server implementation
├── insecure/
│   └── insecure.go           # Development TLS certificates
└── main.go                   # Server entry point

```

## Development

### Regenerate Code

After modifying `proto/example.proto`:

```bash
just generate
```

### Lint Protobuf Files

```bash
just lint
```

### Format Protobuf Files

```bash
just format
```

### Run Tests

```bash
just test
```

### Clean Generated Files

```bash
just clean
```

### Full Rebuild

```bash
just rebuild
```

## Interceptors

### Logging Interceptor
Automatically logs all RPC calls with timing information for both unary and streaming RPCs.

### Authentication Interceptor
Simple token-based authentication (demo implementation). Enable with `--enable-auth` flag.

**API Key:** `demo-api-key-12345`

Include in gRPC metadata:
```go
md := metadata.Pairs("authorization", "demo-api-key-12345")
ctx := metadata.NewOutgoingContext(context.Background(), md)
```

### Metrics Interceptor
Collects request counts, error rates, and timing information. View with `--print-metrics` flag on shutdown.

## Storage Abstraction

The server uses a pluggable storage interface:

```go
type Storage interface {
    AddUser(ctx context.Context, user *pb.User) error
    GetUser(ctx context.Context, id uint32) (*pb.User, error)
    UpdateUser(ctx context.Context, user *pb.User) error
    DeleteUser(ctx context.Context, id uint32) error
    ListUsers(ctx context.Context, filter *ListFilter) ([]*pb.User, error)
    ListUsersByRole(ctx context.Context, role pb.Role) ([]*pb.User, error)
    UserExists(ctx context.Context, id uint32) (bool, error)
    Count(ctx context.Context) (int, error)
}
```

This makes it easy to add database backends (PostgreSQL, MySQL, MongoDB, etc.) without changing the server code.

## Security Notes

This project uses self-signed certificates for development purposes. For production use:

1. Replace the certificates in `insecure/` with proper TLS certificates
2. Implement proper authentication (JWT, OAuth2, etc.)
3. Enable authorization checks
4. Use secure credential storage
5. Enable rate limiting and request validation

## Contributing

This is a demonstration project showcasing gRPC capabilities. Feel free to use it as a reference for your own projects.

## License

Apache 2.0 - See [LICENSE](LICENSE) file for details.

## References

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers](https://protobuf.dev/)
- [Buf Documentation](https://buf.build/docs/)
- [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway)
- [Just Command Runner](https://github.com/casey/just)

## Acknowledgments

This project demonstrates modern gRPC development practices with comprehensive coverage of:
- All four gRPC communication patterns
- Extensive Protocol Buffers features
- Modern tooling (Buf, Just)
- Production-ready patterns (interceptors, storage abstraction, graceful shutdown)
- Migrated from deprecated gogo/protobuf to standard google.golang.org/protobuf
