# CrudBox - User Management Interface

A simple, idiomatic Go web application using HTMX for complete CRUD operations on users and roles.

## Features

**Complete CRUD Operations:**
- **Create**: Add new users with profile information
- **Read**: View individual user details and list all users
- **Update**: Edit existing user information
- **Delete**: Remove users from the system

**Additional Features:**
- Filter users by role (Guest, Member, Admin, Moderator)
- Real-time updates using HTMX (no page refreshes)
- Clean, responsive UI with proper styling
- JWT authentication support
- All data sourced from the JSON API server

## Prerequisites

- Go 1.22 or later
- Running gRPC-Gateway API server (default: https://localhost:11000)
- JWT token for authentication (if API has auth enabled)

## Running the Application

### Basic Usage

```bash
cd ux
go run main.go
```

The server will start on `http://localhost:8080` by default.

### With Custom Configuration

```bash
go run main.go -port 3000 -api-url https://localhost:11000 -token "your-jwt-token"
```

### Using Environment Variables

```bash
export JWT_TOKEN="your-jwt-token"
go run main.go
```

## Command Line Flags

- `-port`: HTTP server port (default: 8080)
- `-api-url`: gRPC Gateway API URL (default: https://localhost:11000)
- `-token`: JWT authentication token (can also use JWT_TOKEN environment variable)

## Building

```bash
go build -o crudbox
./crudbox
```

## Project Structure

```
ux/
├── main.go              # Application entry point
├── go.mod               # Go module definition
├── README.md            # This file
├── client/
│   └── client.go        # API client for gRPC Gateway
├── handlers/
│   └── handlers.go      # HTTP request handlers
└── templates/
    ├── base.html        # Base HTML template
    ├── index.html       # Home page
    ├── users.html       # Users listing page
    ├── users-list.html  # Users table (HTMX partial)
    ├── user-form.html   # Create user form (HTMX partial)
    ├── user-edit.html   # Edit user form (HTMX partial)
    └── user-detail.html # User details page
```

## Usage

1. Start the gRPC server first (from the project root):
   ```bash
   ./grpc-example --enable-auth
   ```

2. Generate a JWT token (if auth is enabled):
   ```bash
   JWT_SECRET="test-secret-key-12345" just gen-token user-id=1 username=admin email=admin@example.com roles=admin
   ```

3. Start the UX server:
   ```bash
   cd ux
   JWT_TOKEN="your-generated-token" go run main.go
   ```

4. Open your browser to `http://localhost:8080`

## API Integration

This application connects to the gRPC-Gateway JSON API. The default endpoints are:

- `GET /api/v1/users` - List all users
- `POST /api/v1/users` - Create a user
- `GET /api/v1/users/{id}` - Get user by ID
- `PATCH /api/v1/users/{id}` - Update user
- `DELETE /api/v1/users/{id}` - Delete user
- `GET /api/v1/users/role/{role}` - List users by role

## Role Values

- 0: GUEST
- 1: MEMBER
- 2: ADMIN
- 3: MODERATOR

## Status Values

- 0: INACTIVE
- 1: ACTIVE
- 2: SUSPENDED
- 3: DELETED

## Notes

- The application uses self-signed certificates by default (InsecureSkipVerify is enabled)
- For production use, ensure proper TLS certificate validation
- JWT token authentication is required if the API server has auth enabled
- The application follows idiomatic Go patterns and best practices
