package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key"

func TestGenerateToken(t *testing.T) {
	t.Run("returns non-empty token with valid claims", func(t *testing.T) {
		userID := int64(123)
		lifespan := 15 * time.Minute
		tokenStr, err := GenerateToken(userID, testSecret, lifespan)
		require.NoError(t, err, "token generation failed")
		assert.NotEmpty(t, tokenStr, "token string is empty")

		// Parse to verify contents
		claims := &Claims{}
		parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			return []byte(testSecret), nil
		})
		require.NoError(t, err, "failed to parse generated token")
		assert.True(t, parsed.Valid, "parsed token is not valid")
		assert.Equal(t, userID, claims.UserID, "user ID mismatch")
		assert.NotNil(t, claims.ExpiresAt, "expiration not set")
	})

	t.Run("different user IDs produce different tokens", func(t *testing.T) {
		token1, _ := GenerateToken(1, testSecret, time.Hour)
		token2, _ := GenerateToken(2, testSecret, time.Hour)
		assert.NotEqual(t, token1, token2, "tokens for different users should differ")
	})
}

func TestGetUserIDFromToken(t *testing.T) {
	var validUserID int64 = 100
	// var validToken string
	// var tokenWithZeroUserID string

	// Token signed with wrong algorithm (RS256)
	_ = jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UserID: validUserID,
	})
	t.Run("returns user ID for valid token", func(t *testing.T) {
		// Setup valid token
		tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			UserID: validUserID,
		})
		validToken, err := tokenObj.SignedString([]byte(testSecret))
		require.NoError(t, err, "failed to generate valid token for test")
		// get user ID from valid token
		uid, err := GetUserIDFromToken(validToken, testSecret)
		require.NoError(t, err)
		assert.Equal(t, validUserID, uid)
	})

	t.Run("returns error for expired token", func(t *testing.T) {
		// Setup expired token
		expiredObj := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
			UserID: validUserID,
		})
		expiredToken, err := expiredObj.SignedString([]byte(testSecret))
		require.NoError(t, err)
		// token should be expired
		_, err = GetUserIDFromToken(expiredToken, testSecret)
		assert.Error(t, err, "expired token should produce error")
		assert.Contains(t, err.Error(), "token is expired", "error should indicate expiration")
	})

	t.Run("returns error for token with zero user ID", func(t *testing.T) {
		// Setup token with zero user ID
		zeroObj := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			UserID: 0,
		})
		tokenWithZeroUserID, err := zeroObj.SignedString([]byte(testSecret))
		require.NoError(t, err)
		// should be rejected
		_, err = GetUserIDFromToken(tokenWithZeroUserID, testSecret)
		assert.Error(t, err, "token with empty user ID should be rejected")
		assert.Equal(t, "userID is empty", err.Error())
	})

	t.Run("returns error for malformed token", func(t *testing.T) {
		_, err := GetUserIDFromToken("this-is-not-a-jwt", testSecret)
		assert.Error(t, err)
	})
}
