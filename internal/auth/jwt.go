// Package auth provides JWT-based authentication using HTTP cookies.
package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/max-marek-projects/custodia/internal/logger"
)

const cookieName = "token"

// Claims represents the JWT claims containing a user ID.
type Claims struct {
	jwt.RegisteredClaims
	UserID int64 // Unique user identifier.
}

// SetUserCookie creates a signed JWT for the given user ID and sets it as an HTTP cookie.
// Parameters:
//   - w: ResponseWriter to write the cookie.
//   - userID: ID to embed in the token.
//   - secretKey: key used for signing.
//
// Returns an error if token signing fails.
func SetUserCookie(w http.ResponseWriter, userID int64, secretKey string) error {
	logger.Log.Info("Signing token with key", slog.String("key", secretKey))
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// extractUserIDFromToken parses a JWT string, validates its signature, and returns the user ID.
// Parameters:
//   - token: raw JWT string.
//   - secretKey: key for signature verification.
//
// Returns the user ID on success or an error if parsing or validation fails.
func extractUserIDFromToken(token string, secretKey string) (int64, error) {
	claims := &Claims{}
	tokenData, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to parse jwt: %v", err)
	}
	if !tokenData.Valid {
		return 0, errors.New("invalid token")
	}
	return claims.UserID, nil
}

// GetUserIDFromRequest extracts the JWT from the request cookie and returns the user ID.
// Parameters:
//   - r: HTTP request containing the cookie.
//   - secretKey: key to verify the token.
//
// Returns the user ID or an error if the cookie is missing, token is invalid, or signature fails.
func GetUserIDFromRequest(r *http.Request, secretKey string) (int64, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return 0, err
	}
	return extractUserIDFromToken(cookie.Value, secretKey)
}
