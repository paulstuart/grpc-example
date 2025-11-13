package main

import (
	"context"
	"crypto/tls"
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
	pb "github.com/paulstuart/grpc-example/proto"
	"github.com/paulstuart/grpc-example/server"
)

var (
	gRPCPort     = flag.Int("grpc-port", 10000, "The gRPC server port")
	gatewayPort  = flag.Int("gateway-port", 11000, "The gRPC-Gateway server port")
	nocheck      = flag.Bool("insecure", false, "don't complain about self-signed certs")
	enableAuth   = flag.Bool("enable-auth", false, "enable authentication interceptor")
	printMetrics = flag.Bool("print-metrics", false, "print metrics on shutdown")
	hostname     = flag.String("bind to host address", "localhost", "control access based on host address")
)

//go:embed third_party/OpenAPI/*
var content embed.FS

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// serveOpenAPI serves an OpenAPI UI on /openapi-ui/
func serveOpenAPI(mux *http.ServeMux) error {
	mime.AddExtensionType(".svg", "image/svg+xml")

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

	log.Println("Starting gRPC Example Server...")
	log.Printf("gRPC Port: %d", *gRPCPort)
	log.Printf("Gateway Port: %d", *gatewayPort)
	log.Printf("Auth Enabled: %v", *enableAuth)
	log.Printf("Host address: %s", *hostname)

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
		unaryInterceptors = append(unaryInterceptors, interceptors.AuthUnaryInterceptor())
		streamInterceptors = append(streamInterceptors, interceptors.AuthStreamInterceptor())
		log.Println("Authentication interceptor enabled - use 'authorization: demo-api-key-12345' in metadata")
	}

	// Chain interceptors
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewServerTLSFromCert(&insecure.Cert)),
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
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")))

	conn, err := grpc.Dial(dialAddr, dialOpts...)
	if err != nil {
		log.Fatalf("Failed to dial server: %v", err)
	}
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

	gatewayAddr := fmt.Sprintf("localhost:%d", *gatewayPort)
	log.Printf("Serving gRPC-Gateway on https://%s", gatewayAddr)
	log.Printf("Serving OpenAPI Documentation on https://%s/openapi-ui/", gatewayAddr)

	gwServer := &http.Server{
		Addr: gatewayAddr,
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{insecure.Cert},
			InsecureSkipVerify: *nocheck,
		},
		Handler: mux,
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
