package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/paulstuart/grpc-example/auth"
)

var (
	userID    = flag.String("user-id", "", "User ID (required)")
	username  = flag.String("username", "", "Username (required)")
	email     = flag.String("email", "", "User email (required)")
	roles     = flag.String("roles", "user", "Comma-separated list of roles")
	duration  = flag.Duration("duration", 24*time.Hour, "Token duration (e.g., 1h, 24h, 7d)")
	secretKey = flag.String("secret", "", "JWT secret key (env: JWT_SECRET)")
	issuer    = flag.String("issuer", "grpc-example", "Token issuer")
	showHelp  = flag.Bool("help", false, "Show help")
)

func main() {
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Validate required fields
	if *userID == "" {
		log.Fatal("Error: -user-id is required")
	}
	if *username == "" {
		log.Fatal("Error: -username is required")
	}
	if *email == "" {
		log.Fatal("Error: -email is required")
	}

	// Get secret key from flag or environment
	secret := *secretKey
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		log.Fatal("Error: JWT secret key must be provided via -secret flag or JWT_SECRET environment variable")
	}

	// Parse roles
	roleList := strings.Split(*roles, ",")
	for i, role := range roleList {
		roleList[i] = strings.TrimSpace(role)
	}

	// Create JWT manager and generate token
	manager := auth.NewJWTManager(secret, *duration, *issuer)
	token, err := manager.GenerateToken(*userID, *username, *email, roleList)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	// Output token details
	fmt.Fprintln(os.Stderr, "=== JWT Token Generated ===")
	fmt.Fprintf(os.Stderr, "User ID:   %s\n", *userID)
	fmt.Fprintf(os.Stderr, "Username:  %s\n", *username)
	fmt.Fprintf(os.Stderr, "Email:     %s\n", *email)
	fmt.Fprintf(os.Stderr, "Roles:     %v\n", roleList)
	fmt.Fprintf(os.Stderr, "Duration:  %v\n", *duration)
	fmt.Fprintf(os.Stderr, "Issuer:    %s\n", *issuer)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Token:")
	//nolint:errcheck // output to stdout, error doesn't matter
	fmt.Fprintln(os.Stdout, token) // NOTE: just token goes to stdout
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Use this token in the Authorization header:")
	fmt.Fprintf(os.Stderr, "Authorization: Bearer %s\n", token)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Example gRPC metadata:")
	fmt.Fprintf(os.Stderr, "authorization: Bearer %s\n", token)
}

func printHelp() {
	fmt.Fprintln(os.Stderr, "JWT Token Generator for gRPC Example")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  tokengen [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Required Flags:")
	fmt.Fprintln(os.Stderr, "  -user-id string")
	fmt.Fprintln(os.Stderr, "        User ID")
	fmt.Fprintln(os.Stderr, "  -username string")
	fmt.Fprintln(os.Stderr, "        Username")
	fmt.Fprintln(os.Stderr, "  -email string")
	fmt.Fprintln(os.Stderr, "        User email address")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Optional Flags:")
	fmt.Fprintln(os.Stderr, "  -roles string")
	fmt.Fprintln(os.Stderr, "        Comma-separated list of roles (default: user)")
	fmt.Fprintln(os.Stderr, "  -duration duration")
	fmt.Fprintln(os.Stderr, "        Token validity duration (default: 24h)")
	fmt.Fprintln(os.Stderr, "        Examples: 1h, 24h, 168h (7 days)")
	fmt.Fprintln(os.Stderr, "  -secret string")
	fmt.Fprintln(os.Stderr, "        JWT secret key (can also use JWT_SECRET env var)")
	fmt.Fprintln(os.Stderr, "  -issuer string")
	fmt.Fprintln(os.Stderr, "        Token issuer (default: grpc-example)")
	fmt.Fprintln(os.Stderr, "  -help")
	fmt.Fprintln(os.Stderr, "        Show this help message")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Environment Variables:")
	fmt.Fprintln(os.Stderr, "  JWT_SECRET")
	fmt.Fprintln(os.Stderr, "        JWT secret key (alternative to -secret flag)")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  # Generate token for regular user")
	fmt.Fprintln(os.Stderr, "  tokengen -user-id=123 -username=john -email=john@example.com")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  # Generate token for admin with multiple roles")
	fmt.Fprintln(os.Stderr, "  tokengen -user-id=1 -username=admin -email=admin@example.com -roles=admin,user")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  # Generate token with custom duration")
	fmt.Fprintln(os.Stderr, "  tokengen -user-id=123 -username=john -email=john@example.com -duration=7h")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  # Use environment variable for secret")
	fmt.Fprintln(os.Stderr, "  export JWT_SECRET=my-secret-key")
	fmt.Fprintln(os.Stderr, "  tokengen -user-id=123 -username=john -email=john@example.com")
}
