package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecretKey = "test-secret-key-for-jwt-testing-12345"
	testIssuer    = "grpc-example-test"
)

func TestNewJWTManager(t *testing.T) {
	duration := 1 * time.Hour
	manager := NewJWTManager(testSecretKey, duration, testIssuer)

	assert.NotNil(t, manager)
	assert.Equal(t, []byte(testSecretKey), manager.secretKey)
	assert.Equal(t, duration, manager.tokenDuration)
	assert.Equal(t, testIssuer, manager.issuer)
}

func TestGenerateToken(t *testing.T) {
	manager := NewJWTManager(testSecretKey, 1*time.Hour, testIssuer)

	tests := []struct {
		name     string
		userID   string
		username string
		email    string
		roles    []string
	}{
		{
			name:     "admin user",
			userID:   "user-123",
			username: "admin",
			email:    "admin@example.com",
			roles:    []string{"admin", "user"},
		},
		{
			name:     "regular user",
			userID:   "user-456",
			username: "john",
			email:    "john@example.com",
			roles:    []string{"user"},
		},
		{
			name:     "no roles",
			userID:   "user-789",
			username: "guest",
			email:    "guest@example.com",
			roles:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.GenerateToken(tt.userID, tt.username, tt.email, tt.roles)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Verify token can be parsed
			parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(testSecretKey), nil
			})
			require.NoError(t, err)
			assert.True(t, parsedToken.Valid)

			claims, ok := parsedToken.Claims.(*Claims)
			require.True(t, ok)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, tt.username, claims.Username)
			assert.Equal(t, tt.email, claims.Email)
			assert.Equal(t, tt.roles, claims.Roles)
			assert.Equal(t, testIssuer, claims.Issuer)
			assert.Equal(t, tt.userID, claims.Subject)
		})
	}
}

func TestValidateToken(t *testing.T) {
	manager := NewJWTManager(testSecretKey, 1*time.Hour, testIssuer)

	t.Run("valid token", func(t *testing.T) {
		token, err := manager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		claims, err := manager.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "john", claims.Username)
		assert.Equal(t, "john@example.com", claims.Email)
		assert.Equal(t, []string{"user"}, claims.Roles)
	})

	t.Run("expired token", func(t *testing.T) {
		shortManager := NewJWTManager(testSecretKey, 1*time.Millisecond, testIssuer)
		token, err := shortManager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		_, err = shortManager.ValidateToken(token)
		assert.ErrorIs(t, err, ErrExpiredToken)
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := manager.ValidateToken("invalid-token")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("tampered token", func(t *testing.T) {
		token, err := manager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		// Tamper with the token
		tamperedToken := token + "tampered"

		_, err = manager.ValidateToken(tamperedToken)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("wrong secret key", func(t *testing.T) {
		token, err := manager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		wrongManager := NewJWTManager("wrong-secret-key", 1*time.Hour, testIssuer)
		_, err = wrongManager.ValidateToken(token)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestRefreshToken(t *testing.T) {
	manager := NewJWTManager(testSecretKey, 1*time.Hour, testIssuer)

	t.Run("refresh valid token", func(t *testing.T) {
		originalToken, err := manager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		time.Sleep(1 * time.Second) // Wait long enough to get different timestamps

		refreshedToken, err := manager.RefreshToken(originalToken)
		require.NoError(t, err)

		// Tokens should be different due to different IssuedAt times
		if originalToken == refreshedToken {
			t.Log("Warning: tokens are identical, timestamps may be too close")
		}

		// Validate refreshed token
		claims, err := manager.ValidateToken(refreshedToken)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "john", claims.Username)
		assert.Equal(t, "john@example.com", claims.Email)
	})

	t.Run("refresh expired token", func(t *testing.T) {
		// Create short-lived token
		shortManager := NewJWTManager(testSecretKey, 100*time.Millisecond, testIssuer)
		originalToken, err := shortManager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Verify original token is expired
		_, err = shortManager.ValidateToken(originalToken)
		assert.ErrorIs(t, err, ErrExpiredToken)

		// Create a new manager with longer duration for refresh
		longManager := NewJWTManager(testSecretKey, 1*time.Hour, testIssuer)

		// Should still be able to refresh expired token
		refreshedToken, err := longManager.RefreshToken(originalToken)
		require.NoError(t, err)

		// New token should be valid with long manager
		claims, err := longManager.ValidateToken(refreshedToken)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("refresh invalid token", func(t *testing.T) {
		_, err := manager.RefreshToken("invalid-token")
		assert.Error(t, err)
	})
}

func TestClaimsHasRole(t *testing.T) {
	tests := []struct {
		name     string
		roles    []string
		checkRole string
		expected bool
	}{
		{
			name:     "has role",
			roles:    []string{"admin", "user"},
			checkRole: "admin",
			expected: true,
		},
		{
			name:     "does not have role",
			roles:    []string{"user"},
			checkRole: "admin",
			expected: false,
		},
		{
			name:     "empty roles",
			roles:    []string{},
			checkRole: "admin",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &Claims{
				Roles: tt.roles,
			}
			assert.Equal(t, tt.expected, claims.HasRole(tt.checkRole))
		})
	}
}

func TestClaimsHasAnyRole(t *testing.T) {
	tests := []struct {
		name       string
		userRoles  []string
		checkRoles []string
		expected   bool
	}{
		{
			name:       "has one of multiple roles",
			userRoles:  []string{"user", "moderator"},
			checkRoles: []string{"admin", "moderator"},
			expected:   true,
		},
		{
			name:       "has none of the roles",
			userRoles:  []string{"user"},
			checkRoles: []string{"admin", "moderator"},
			expected:   false,
		},
		{
			name:       "empty user roles",
			userRoles:  []string{},
			checkRoles: []string{"admin"},
			expected:   false,
		},
		{
			name:       "empty check roles",
			userRoles:  []string{"admin"},
			checkRoles: []string{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &Claims{
				Roles: tt.userRoles,
			}
			assert.Equal(t, tt.expected, claims.HasAnyRole(tt.checkRoles...))
		})
	}
}

func TestTokenExpiry(t *testing.T) {
	t.Run("token expires after duration", func(t *testing.T) {
		manager := NewJWTManager(testSecretKey, 500*time.Millisecond, testIssuer)
		token, err := manager.GenerateToken("user-123", "john", "john@example.com", []string{"user"})
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(600 * time.Millisecond)

		// Token should now be expired
		_, err = manager.ValidateToken(token)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrExpiredToken)
	})
}

func TestConcurrentTokenGeneration(t *testing.T) {
	manager := NewJWTManager(testSecretKey, 1*time.Hour, testIssuer)

	const numGoroutines = 100
	tokens := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			token, err := manager.GenerateToken(
				string(rune(id)),
				"user",
				"user@example.com",
				[]string{"user"},
			)
			if err != nil {
				errors <- err
				return
			}
			tokens <- token
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case token := <-tokens:
			assert.NotEmpty(t, token)
		case err := <-errors:
			t.Errorf("unexpected error: %v", err)
		}
	}
}
