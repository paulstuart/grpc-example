package interceptors

import (
	"context"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Simple auth interceptor for demonstration purposes
// In a real application, this would validate JWT tokens, API keys, etc.

const (
	// Example API key for demonstration
	validAPIKey = "demo-api-key-12345"
)

// AuthUnaryInterceptor provides simple authentication for unary RPCs
func AuthUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for certain methods if needed
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		if err := authorize(ctx); err != nil {
			log.Printf("[Auth] Unauthorized access attempt to %s", info.FullMethod)
			return nil, err
		}

		log.Printf("[Auth] Authorized access to %s", info.FullMethod)
		return handler(ctx, req)
	}
}

// AuthStreamInterceptor provides simple authentication for streaming RPCs
func AuthStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Skip auth for certain methods if needed
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		if err := authorize(ss.Context()); err != nil {
			log.Printf("[Auth] Unauthorized stream access attempt to %s", info.FullMethod)
			return err
		}

		log.Printf("[Auth] Authorized stream access to %s", info.FullMethod)
		return handler(srv, ss)
	}
}

// authorize checks if the request has valid authentication
func authorize(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Check for API key in metadata
	values := md.Get("authorization")
	if len(values) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := values[0]
	// Support both "Bearer <token>" and raw token formats
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}

	// Simple validation - in production, validate JWT or check against database
	if token != validAPIKey {
		return status.Error(codes.Unauthenticated, "invalid authorization token")
	}

	return nil
}

// isPublicMethod determines if a method should skip authentication
func isPublicMethod(method string) bool {
	// Add methods that should be publicly accessible
	// For this demo, we'll make all methods require auth
	// In a real app, you might have:
	// publicMethods := []string{
	//     "/proto.UserService/Login",
	//     "/proto.UserService/Register",
	// }
	return false
}
