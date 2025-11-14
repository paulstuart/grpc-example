package interceptors

import (
	"context"
	"log"
	"log/slog"
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

type JWTApprover struct {
	*auth.JWTManager
	auth.ClaimsApprover
}

// NewApprover creates a new JWT Approver with the given JWT manager and ClaimsApprover
func NewApprover(jwtManager *auth.JWTManager, appr auth.ClaimsApprover) auth.Approver {
	ap := JWTApprover{
		jwtManager,
		appr,
	}
	return ap
}

// FakeClaimsApprover is a stub implementation of ClaimsApprover for demonstration purposes
type FakeClaimsApprover struct{}

// ValidMethod should be passed in from the RBAC package
func (my FakeClaimsApprover) ValidMethod(fullMethod string, claim *auth.Claims) error {
	// Implement method-specific validation if needed
	switch {
	case fullMethod == "/proto.UserService/ListUsers" && claim.Email == "hello@example.com":
		return auth.ErrInvalidClaims
	case fullMethod == "/proto.UserService/ListUsers" && claim.Username == "mynameismud":
		return auth.ErrNoPermission // just so we can see different errors
	default:

	}
	slog.Info("====> FakeClaimsApprover automatically validating method", "method", fullMethod, "user", claim.Username)
	return nil
}

const (
	// ClaimsContextKey is the key used to store JWT claims in context
	// TODO: make this dynamic?
	ClaimsContextKey contextKey = "jwt_claims"
)

func NewJWTManager(secretKey string, tokenDuration time.Duration, issuer string) *auth.JWTManager {
	if tokenDuration.Nanoseconds() == 0 {
		tokenDuration = time.Hour
	}
	return auth.NewJWTManager(secretKey, tokenDuration, issuer)
}

// JWTAuthUnaryInterceptor provides JWT authentication for unary RPCs
func JWTAuthUnaryInterceptor(vapid auth.Approver) grpc.UnaryServerInterceptor {
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
		claims, err := validateJWT(ctx, vapid)
		if err != nil {
			log.Printf("[JWT Auth] Unauthorized access attempt to %s: %v", info.FullMethod, err)
			return nil, err
		}
		// TODO: any call for special handling of errors here? Extend auth.Approver?
		if err := vapid.ValidMethod(info.FullMethod, claims); err != nil {
			log.Printf("[JWT Auth] Forbidden stream access attempt to %s by user %s: %v",
				info.FullMethod, claims.Username, err)
			return nil, status.Error(codes.PermissionDenied, "insufficient permissions for method")
		}

		// Add claims to context for downstream use
		ctx = context.WithValue(ctx, ClaimsContextKey, claims)
		log.Printf("[JWT Auth] Authorized access to %s by user %s (roles: %v)",
			info.FullMethod, claims.Username, claims.Roles)
		return handler(ctx, req)
	}
}

// JWTAuthStreamInterceptor provides JWT authentication for streaming RPCs
func JWTAuthStreamInterceptor(jwtManager auth.Approver) grpc.StreamServerInterceptor {
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

		if err := jwtManager.ValidMethod(info.FullMethod, claims); err != nil {
			log.Printf("[JWT Auth] Forbidden stream access attempt to %s by user %s: %v",
				info.FullMethod, claims.Username, err)
			return status.Error(codes.PermissionDenied, "insufficient permissions for method")
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
func validateJWT(ctx context.Context, jwtManager auth.Approver) (*auth.Claims, error) {
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

	// fmt.Printf("Validated JWT for user: %s\n", claims.Username)

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
