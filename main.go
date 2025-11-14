package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/paulstuart/grpc-example/insecure"
	"github.com/paulstuart/grpc-example/interceptors"
	pb "github.com/paulstuart/grpc-example/proto/pkg"
	"github.com/paulstuart/grpc-example/server"
)

var (
	defaultPort = DefaultEnv("GRPC_PORT", 10000)
	defaultRest = DefaultEnv("GRPC_GATEWAY_PORT", 11000)
	defaultHost = DefaultEnv("GRPC_HOST", "localhost")
	// Use JWT_SECRET if set, otherwise fall back to GRPC_SECRET_KEY
	secretKey = getJWTSecret()
	jwtIssuer = DefaultEnv("GRPC_ISSUER", "grpc-example")

	gRPCPort      = flag.Int("grpc-port", defaultPort, "The gRPC server port")
	gatewayPort   = flag.Int("gateway-port", defaultRest, "The gRPC-Gateway server port")
	nocheck       = flag.Bool("insecure", false, "don't complain about self-signed certs")
	enableAuth    = flag.Bool("enable-auth", false, "enable authentication interceptor")
	printMetrics  = flag.Bool("print-metrics", false, "print metrics on shutdown")
	hostname      = flag.String("host", defaultHost, "bind to host address")
	validateToken = flag.String("validate", "", "validate this JWT token and exit")
	certFile      = flag.String("cert", "certs/server.crt", "TLS certificate file")
	keyFile       = flag.String("key", "certs/server.key", "TLS key file")
)

// getJWTSecret returns the JWT secret key from environment variables
// Priority: JWT_SECRET > GRPC_SECRET_KEY > default
func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	if secret := os.Getenv("GRPC_SECRET_KEY"); secret != "" {
		return secret
	}
	return "our little secret"
}

// loadTLSCredentials loads TLS certificate and key from files
// Returns the certificate, a TLS config, and a cert pool for client use
func loadTLSCredentials(certFile, keyFile string) (*tls.Certificate, *tls.Config, *x509.CertPool, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load TLS key pair: %w", err)
	}

	// Parse the certificate to create a cert pool for clients
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		return nil, nil, nil, fmt.Errorf("failed to parse certificate")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return &cert, tlsConfig, certPool, nil
}

func DefaultEnv[T any](name string, def T) T {
	if val, ok := os.LookupEnv(name); ok {
		var ret T
		switch any(def).(type) {
		case string:
			ret = any(val).(T)
		case int:
			// i, err := strconv.ParseInt(val, 10, strconv.IntSize)
			// if err != nil{
			// 	log.Fatalf("failed to parse env var %s as int: %v", name, err)
			// }
			// return int(i)
			// return int(T(i))
			var v int
			_, err := fmt.Sscanf(val, "%d", &v)
			if err == nil {
				ret = any(v).(T)
			} else {
				ret = def
			}
		case bool:
			var v bool
			_, err := fmt.Sscanf(val, "%t", &v)
			if err == nil {
				ret = any(v).(T)
			} else {
				ret = def
			}
		default:
			log.Fatalf("unsupported env var type for %s - %T", name, def)
		}
		return ret
	}
	return def
}

//go:embed third_party/OpenAPI/*
var content embed.FS

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// serveOpenAPI serves an OpenAPI UI on /openapi-ui/
func serveOpenAPI(mux *http.ServeMux) error {
	if err := mime.AddExtensionType(".svg", "image/svg+xml"); err != nil {
		return fmt.Errorf("mime add ext fail: %w", err)
	}

	prefix := "/openapi-ui/"
	dirname := "third_party/OpenAPI"
	sub, err := fs.Sub(content, dirname)
	if err != nil {
		return fmt.Errorf("sub dir fail: %w", err)
	}
	dir := http.FS(sub)

	mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(dir)))
	return nil
}

func main() {
	flag.Parse()

	if *validateToken != "" {
		secretKey := secretKey
		jwtMgr := interceptors.NewJWTManager(secretKey, time.Hour*24, jwtIssuer)
		claims, err := jwtMgr.ValidateToken(*validateToken)
		if err != nil {
			log.Fatalf("Token validation failed: %v", err)
		}
		log.Printf("Token is valid. Claims: Username=%s, Roles=%v, IssuedAt=%v, ExpiresAt=%v",
			claims.Username, claims.Roles, claims.IssuedAt, claims.ExpiresAt)
		return
	}

	log.Println("Starting gRPC Example Server...")
	log.Printf("gRPC Port: %d", *gRPCPort)
	log.Printf("Gateway Port: %d", *gatewayPort)
	log.Printf("Auth Enabled: %v", *enableAuth)
	log.Printf("Host address: %s", *hostname)

	// Load TLS credentials
	tlsCert, tlsConfig, certPool, err := loadTLSCredentials(*certFile, *keyFile)
	if err != nil {
		// Fall back to insecure embedded credentials
		log.Printf("Warning: Failed to load TLS credentials (%v), falling back to embedded self-signed cert", err)
		tlsCert = &insecure.Cert
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{insecure.Cert},
			MinVersion:   tls.VersionTLS12,
		}
		certPool = insecure.CertPool
	} else {
		log.Printf("TLS enabled: cert=%s, key=%s", *certFile, *keyFile)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create gRPC server with interceptors
	addr := fmt.Sprintf("%s:%d", *hostname, *gRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Build interceptor chain
	var unaryInterceptors []grpc.UnaryServerInterceptor
	var streamInterceptors []grpc.StreamServerInterceptor

	// Always add logging
	unaryInterceptors = append(unaryInterceptors, interceptors.LoggingUnaryInterceptor())
	streamInterceptors = append(streamInterceptors, interceptors.LoggingStreamInterceptor())

	// Add metrics
	unaryInterceptors = append(unaryInterceptors, interceptors.MetricsUnaryInterceptor())
	streamInterceptors = append(streamInterceptors, interceptors.MetricsStreamInterceptor())

	// Optionally add auth
	if *enableAuth {
		jwtMgr := interceptors.NewJWTManager(secretKey, time.Hour*24, jwtIssuer)
		var approver interceptors.FakeClaimsApprover     // TODO: replace with real RBAC approver
		jm := interceptors.NewApprover(jwtMgr, approver) //auth.MyApprover{jwtManager: jwtMgr}
		unaryInterceptors = append(unaryInterceptors, interceptors.JWTAuthUnaryInterceptor(jm))
		streamInterceptors = append(streamInterceptors, interceptors.JWTAuthStreamInterceptor(jm))
		// log.Println("Authentication interceptor enabled - use 'authorization: demo-api-key-12345' in metadata")
		log.Println("Authentication interceptor enabled - using JWT tokens for Bear")
	}

	// Chain interceptors
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewServerTLSFromCert(tlsCert)),
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	}

	grpcServer := grpc.NewServer(opts...)

	// Register the UserService with default in-memory storage
	pb.RegisterUserServiceServer(grpcServer, server.NewWithDefaultStorage())

	// Serve gRPC Server in background
	log.Printf("Serving gRPC on https://%s", addr)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Setup gRPC-Gateway
	dialAddr := fmt.Sprintf("%s:%d", *hostname, *gRPCPort)

	var dialOpts []grpc.DialOption
	// Use the cert pool from loaded credentials (or embedded if fallback occurred)
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certPool, "")))

	conn, err := grpc.NewClient(dialAddr, dialOpts...)
	if err != nil {
		log.Fatalf("Failed to dial server: %v", err)
	}
	//nolint:errcheck // close doesn't matter here
	defer conn.Close()

	mux := http.NewServeMux()
	gwmux := runtime.NewServeMux()

	err = pb.RegisterUserServiceHandler(ctx, gwmux, conn)
	if err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	mux.Handle("/", gwmux)

	// Try to serve OpenAPI UI if files exist
	if err := serveOpenAPI(mux); err != nil {
		log.Printf("Warning: Failed to serve OpenAPI UI: %v", err)
	}

	gatewayAddr := fmt.Sprintf("%s:%d", *hostname, *gatewayPort)
	log.Printf("Serving gRPC-Gateway on https://%s", gatewayAddr)
	log.Printf("Serving OpenAPI Documentation on https://%s/openapi-ui/", gatewayAddr)

	// Update TLS config for gateway with InsecureSkipVerify if needed
	gatewayTLSConfig := tlsConfig
	if *nocheck {
		gatewayTLSConfig = &tls.Config{
			Certificates:       tlsConfig.Certificates,
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
	}

	gwServer := &http.Server{
		Addr:      gatewayAddr,
		TLSConfig: gatewayTLSConfig,
		Handler:   mux,
	}

	// Serve HTTP Gateway in background
	go func() {
		if err := gwServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP gateway: %v", err)
		}
	}()

	log.Println("Server started successfully!")
	log.Println("Press Ctrl+C to shutdown...")

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nShutdown signal received, gracefully shutting down...")

	// Print metrics if requested
	if *printMetrics {
		log.Println("\n=== Final Metrics ===")
		interceptors.GetMetrics().PrintStats()
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP gateway
	if err := gwServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP gateway shutdown error: %v", err)
	}

	// Gracefully stop gRPC server
	grpcServer.GracefulStop()

	log.Println("Server shutdown complete")
}
