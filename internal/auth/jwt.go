// Package auth provides JWT-based authentication utilities.
package auth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims represents the JWT claims containing a user ID.
type Claims struct {
	jwt.RegisteredClaims
	UserID int64 // Unique user identifier.
}

// GenerateToken creates a new JWT string for the given user ID.
//
// Parameters:
//   - userID: The unique identifier of the user to embed in the token.
//   - secretKey: The secret key used for HMAC-SHA256 signing.
//   - lifespan: The duration after which the token expires.
//
// Returns:
//   - string: The signed JWT token string.
//   - error: Non-nil if token signing fails (e.g., invalid key).
func GenerateToken(userID int64, secretKey string, lifespan time.Duration) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(lifespan)),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to create signed string: %w", err)
	}
	return tokenString, nil
}

// GetUserIDFromToken parses and validates a JWT token string.
//
// Parameters:
//   - token: The JWT token string to validate.
//   - secretKey: The secret key used for signature verification.
//
// Returns:
//   - int64: The user ID extracted from the token.
//   - error: Non-nil if the token is invalid, expired, malformed,
//     signed with a different method, or contains an empty user ID.
func GetUserIDFromToken(token string, secretKey string, logger *slog.Logger) (int64, error) {
	claims := &Claims{}
	tokenData, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrUnexpectedSigningMethod
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}
	if !tokenData.Valid {
		return 0, ErrInvalidToken
	}
	if claims.UserID == 0 {
		return 0, ErrEmptyUserID
	}
	logger.Debug("Extracted user id from token", slog.Int64("user_id", claims.UserID))
	return claims.UserID, nil
}
