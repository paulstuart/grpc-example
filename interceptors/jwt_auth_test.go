package interceptors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/paulstuart/grpc-example/auth"
)

const (
	testSecret = "test-jwt-secret-key-12345"
	testIssuer = "grpc-example-test"
)

func setupTestJWTManager() auth.Approver {
	jm := auth.NewJWTManager(testSecret, 1*time.Hour, testIssuer)
	return NewApprover(jm, FakeClaimsApprover{})
}

func TestJWTAuthUnaryInterceptor(t *testing.T) {
	jwtManager := setupTestJWTManager()
	interceptor := JWTAuthUnaryInterceptor(jwtManager)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Check if claims are in context
		claims := GetClaimsFromContext(ctx)
		assert.NotNil(t, claims)
		return "success", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/proto.UserService/GetUser",
	}

	t.Run("valid token", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		md := metadata.Pairs("authorization", "Bearer "+token)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		resp, err := interceptor(ctx, nil, info, handler)
		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		ctx := context.Background()

		_, err := interceptor(ctx, nil, info, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.Contains(t, err.Error(), "missing metadata")
	})

	t.Run("missing bearer prefix", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		md := metadata.Pairs("authorization", token) // Missing "Bearer " prefix
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err = interceptor(ctx, nil, info, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.Contains(t, err.Error(), "invalid authorization format")
	})

	t.Run("invalid token", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer invalid-token")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err := interceptor(ctx, nil, info, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("expired token", func(t *testing.T) {
		shortManager := auth.NewJWTManager(testSecret, 1*time.Millisecond, testIssuer)
		token, err := shortManager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		md := metadata.Pairs("authorization", "Bearer "+token)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err = interceptor(ctx, nil, info, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("wrong secret key", func(t *testing.T) {
		wrongManager := auth.NewJWTManager("wrong-secret", 1*time.Hour, testIssuer)
		token, err := wrongManager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		md := metadata.Pairs("authorization", "Bearer "+token)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err = interceptor(ctx, nil, info, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func TestJWTAuthStreamInterceptor(t *testing.T) {
	jwtManager := setupTestJWTManager()
	interceptor := JWTAuthStreamInterceptor(jwtManager)

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// Check if claims are in context
		claims := GetClaimsFromContext(stream.Context())
		assert.NotNil(t, claims)
		return nil
	}

	info := &grpc.StreamServerInfo{
		FullMethod: "/proto.UserService/ListUsers",
	}

	t.Run("valid token", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		md := metadata.Pairs("authorization", "Bearer "+token)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		stream := &mockServerStream{ctx: ctx}
		err = interceptor(nil, stream, info, handler)
		assert.NoError(t, err)
	})

	t.Run("missing authorization", func(t *testing.T) {
		stream := &mockServerStream{ctx: context.Background()}
		err := interceptor(nil, stream, info, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func TestGetClaimsFromContext(t *testing.T) {
	t.Run("claims present", func(t *testing.T) {
		expectedClaims := &auth.Claims{
			UserID:   "user-123",
			Username: "john",
			Email:    "john@example.com",
			Roles:    []string{"user"},
		}

		ctx := context.WithValue(context.Background(), ClaimsContextKey, expectedClaims)
		claims := GetClaimsFromContext(ctx)

		assert.NotNil(t, claims)
		assert.Equal(t, expectedClaims.UserID, claims.UserID)
		assert.Equal(t, expectedClaims.Username, claims.Username)
	})

	t.Run("claims not present", func(t *testing.T) {
		ctx := context.Background()
		claims := GetClaimsFromContext(ctx)
		assert.Nil(t, claims)
	})

	t.Run("invalid claims type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ClaimsContextKey, "invalid")
		claims := GetClaimsFromContext(ctx)
		assert.Nil(t, claims)
	})
}

func TestRequireRole(t *testing.T) {
	t.Run("has required role", func(t *testing.T) {
		claims := &auth.Claims{
			Roles: []string{"admin", "user"},
		}
		ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)

		requireAdmin := RequireRole("admin")
		err := requireAdmin(ctx)
		assert.NoError(t, err)
	})

	t.Run("missing required role", func(t *testing.T) {
		claims := &auth.Claims{
			Roles: []string{"user"},
		}
		ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)

		requireAdmin := RequireRole("admin")
		err := requireAdmin(ctx)
		assert.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("has one of multiple required roles", func(t *testing.T) {
		claims := &auth.Claims{
			Roles: []string{"moderator"},
		}
		ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)

		requireAdminOrMod := RequireRole("admin", "moderator")
		err := requireAdminOrMod(ctx)
		assert.NoError(t, err)
	})

	t.Run("no claims in context", func(t *testing.T) {
		ctx := context.Background()

		requireAdmin := RequireRole("admin")
		err := requireAdmin(ctx)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func TestValidateJWT(t *testing.T) {
	jwtManager := setupTestJWTManager()

	t.Run("valid token with proper format", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		md := metadata.Pairs("authorization", "Bearer "+token)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		claims, err := validateJWT(ctx, jwtManager)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "john", claims.Username)
	})

	t.Run("empty token after Bearer", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer ")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err := validateJWT(ctx, jwtManager)
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.Contains(t, err.Error(), "empty token")
	})
}

// mockServerStream implements grpc.ServerStream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockServerStream) RecvMsg(msg interface{}) error {
	return nil
}
