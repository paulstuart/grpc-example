# JWT Authentication

This package provides JWT (JSON Web Token) authentication for the gRPC example application.

## Features

- **Token Generation**: Create JWT tokens with user information and roles
- **Token Validation**: Validate and verify JWT tokens
- **Token Refresh**: Refresh expired tokens (useful for seamless user experience)
- **Role-Based Access Control**: Support for multiple user roles
- **Configurable Expiration**: Set custom token duration
- **gRPC Interceptors**: Ready-to-use interceptors for both unary and streaming RPCs

## Components

### 1. JWT Manager (`auth/jwt.go`)

Core JWT functionality:

```go
manager := auth.NewJWTManager(secretKey, duration, issuer)

// Generate a token
token, err := manager.GenerateToken(userID, username, email, roles)

// Validate a token
claims, err := manager.ValidateToken(token)

// Refresh a token
newToken, err := manager.RefreshToken(oldToken)
```

### 2. JWT Claims

JWT claims include:
- `user_id`: Unique user identifier
- `username`: Username
- `email`: User email address
- `roles`: Array of user roles (e.g., "admin", "user", "moderator")
- Standard JWT claims (issuer, subject, expiry, etc.)

### 3. gRPC Interceptors (`interceptors/jwt_auth.go`)

Interceptors for protecting gRPC endpoints:

```go
// For unary RPCs
jwtManager := auth.NewJWTManager(secret, duration, issuer)
unaryInterceptor := interceptors.JWTAuthUnaryInterceptor(jwtManager)

// For streaming RPCs
streamInterceptor := interceptors.JWTAuthStreamInterceptor(jwtManager)
```

### 4. Token Generator CLI (`cmd/tokengen/main.go`)

Command-line tool for generating tokens:

```bash
# Set JWT secret
export JWT_SECRET="your-secret-key"

# Generate a token
just gen-token user-id=123 username=john email=john@example.com

# Generate with custom roles and duration
just gen-token user-id=1 username=admin email=admin@example.com roles="admin,user" duration=7h
```

## Usage

### Server Setup

Update your server to use JWT authentication:

```go
import (
    "github.com/paulstuart/grpc-example/auth"
    "github.com/paulstuart/grpc-example/interceptors"
)

// Create JWT manager
jwtManager := auth.NewJWTManager(
    os.Getenv("JWT_SECRET"),
    24*time.Hour, // token duration
    "grpc-example", // issuer
)

// Add interceptors
grpcServer := grpc.NewServer(
    grpc.UnaryInterceptor(interceptors.JWTAuthUnaryInterceptor(jwtManager)),
    grpc.StreamInterceptor(interceptors.JWTAuthStreamInterceptor(jwtManager)),
)
```

### Client Usage

The client automatically includes JWT tokens when provided:

```bash
# Generate a token
export TOKEN=$(JWT_SECRET=my-secret just gen-token \
    user-id=123 \
    username=john \
    email=john@example.com | \
    grep "^eyJ" | head -1)

# Use token with client
./client -token "$TOKEN"
```

### Accessing Claims in Handlers

Extract JWT claims from the context in your gRPC handlers:

```go
import "github.com/paulstuart/grpc-example/interceptors"

func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    claims := interceptors.GetClaimsFromContext(ctx)
    if claims == nil {
        return nil, status.Error(codes.Unauthenticated, "no authentication")
    }

    log.Printf("Request from user: %s (roles: %v)", claims.Username, claims.Roles)

    // Check roles
    if !claims.HasRole("admin") {
        return nil, status.Error(codes.PermissionDenied, "admin access required")
    }

    // Your handler logic...
}
```

### Role-Based Authorization

Use the `RequireRole` helper for role checking:

```go
func (s *server) AdminOnlyMethod(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // Require admin role
    if err := interceptors.RequireRole("admin")(ctx); err != nil {
        return nil, err
    }

    // Your admin logic...
}
```

## Testing

### Unit Tests

Run JWT package tests:

```bash
just test-auth
```

### Integration Tests

Run interceptor integration tests:

```bash
just test-auth-integration
```

### All Tests

Run all authentication tests:

```bash
just test-auth-all
```

## Configuration

### Environment Variables

- `JWT_SECRET`: Secret key for signing tokens (recommended, used by both server and tokengen)
- `GRPC_SECRET_KEY`: Alternative secret key (deprecated, use JWT_SECRET instead)

**Important**: The server will check for `JWT_SECRET` first, then fall back to `GRPC_SECRET_KEY`. Always use `JWT_SECRET` for consistency between the server and token generator. TODO: pick one. `GRPC_` may be a good common them to stick to

### Server Flags

Enable JWT authentication on the server:

```bash
./grpc-example -enable-auth
```

### Token Duration

Customize token duration when generating:

```bash
just gen-token user-id=123 username=john email=john@example.com duration=168h  # 7 days
```

## Security Considerations

1. **Secret Key**: Use a strong, random secret key in production. Store it securely (e.g., environment variable, secrets manager)

2. **Token Duration**: Balance security and user experience:
   - Shorter durations (1-24 hours) are more secure
   - Use refresh tokens for longer sessions

3. **HTTPS**: Always use TLS/HTTPS in production to prevent token interception

4. **Token Storage**: On the client side, store tokens securely:
   - Never store in localStorage if possible
   - Use secure, HttpOnly cookies when applicable
   - Clear tokens on logout

5. **Public Methods**: Configure public methods that don't require authentication in `interceptors/auth.go:isPublicMethod()`

## Examples

### Generate and Use Token

```bash
# Terminal 1: Start server with authentication
export JWT_SECRET="your-secret-key-here"
./grpc-example -enable-auth

# Terminal 2: Generate token and call API
export JWT_SECRET="your-secret-key-here"
export TOKEN=$(just gen-token user-id=1 username=admin email=admin@example.com | grep "^eyJ" | head -1)
./client -token "$TOKEN"
```

### Token Claims Structure

```json
{
  "user_id": "123",
  "username": "john",
  "email": "john@example.com",
  "roles": ["user"],
  "iss": "grpc-example",
  "sub": "123",
  "exp": 1234567890,
  "iat": 1234567890,
  "nbf": 1234567890
}
```

## Troubleshooting

### "missing authorization header"
- Ensure the client is sending the `authorization` metadata
- Use format: `Bearer <token>`

### "invalid token"
- Check that JWT_SECRET matches between token generation and validation
- Verify token hasn't been tampered with

### "token has expired"
- Generate a new token
- Consider using token refresh functionality
- Increase token duration if appropriate

### Tests failing with timing issues
- Some tests use time.Sleep() for expiration testing
- If tests fail intermittently, increase sleep durations in test files
