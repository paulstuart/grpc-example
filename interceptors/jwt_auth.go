package interceptors

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/paulstuart/grpc-example/auth"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// ClaimsContextKey is the key used to store JWT claims in context
	ClaimsContextKey contextKey = "jwt_claims"
)

func NewJWTManager(secretKey string, tokenDuration time.Duration, issuer string) *auth.JWTManager {
	if tokenDuration.Nanoseconds() == 0 {
		tokenDuration = time.Hour
	}
	return auth.NewJWTManager(secretKey, tokenDuration, issuer)
}

// JWTAuthUnaryInterceptor provides JWT authentication for unary RPCs
func JWTAuthUnaryInterceptor(jwtManager *auth.JWTManager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Skip auth for certain methods if needed
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		claims, err := validateJWT(ctx, jwtManager)
		if err != nil {
			log.Printf("[JWT Auth] Unauthorized access attempt to %s: %v", info.FullMethod, err)
			return nil, err
		}

		// Add claims to context for downstream use
		ctx = context.WithValue(ctx, ClaimsContextKey, claims)
		log.Printf("[JWT Auth] Authorized access to %s by user %s (roles: %v)",
			info.FullMethod, claims.Username, claims.Roles)
		return handler(ctx, req)
	}
}

// JWTAuthStreamInterceptor provides JWT authentication for streaming RPCs
func JWTAuthStreamInterceptor(jwtManager *auth.JWTManager) grpc.StreamServerInterceptor {
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

		claims, err := validateJWT(ss.Context(), jwtManager)
		if err != nil {
			log.Printf("[JWT Auth] Unauthorized stream access attempt to %s: %v", info.FullMethod, err)
			return err
		}

		// Create wrapped stream with claims in context
		wrappedStream := &serverStreamWithContext{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), ClaimsContextKey, claims),
		}

		log.Printf("[JWT Auth] Authorized stream access to %s by user %s (roles: %v)",
			info.FullMethod, claims.Username, claims.Roles)
		return handler(srv, wrappedStream)
	}
}

// validateJWT extracts and validates the JWT token from context
func validateJWT(ctx context.Context, jwtManager *auth.JWTManager) (*auth.Claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Check for authorization header
	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	// Extract token from "Bearer <token>" format
	authHeader := values[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization format, expected 'Bearer <token>'")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "empty token")
	}

	// Validate token
	claims, err := jwtManager.ValidateToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	fmt.Printf("Validated JWT for user: %s\n", claims.Username)

	return claims, nil
}

// serverStreamWithContext wraps a ServerStream with a custom context
type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the custom context
func (s *serverStreamWithContext) Context() context.Context {
	return s.ctx
}

// GetClaimsFromContext extracts JWT claims from context
// Returns nil if no claims are present
func GetClaimsFromContext(ctx context.Context) *auth.Claims {
	claims, ok := ctx.Value(ClaimsContextKey).(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

// RequireRole creates a middleware that requires specific roles
func RequireRole(roles ...string) func(context.Context) error {
	return func(ctx context.Context) error {
		claims := GetClaimsFromContext(ctx)
		if claims == nil {
			return status.Error(codes.Unauthenticated, "no authentication claims found")
		}

		if !claims.HasAnyRole(roles...) {
			return status.Errorf(codes.PermissionDenied,
				"insufficient permissions, required roles: %v", roles)
		}

		return nil
	}
}
