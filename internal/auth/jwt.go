// Package auth provides JWT-based authentication using HTTP cookies.
package auth

import (
	"errors"
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

func GenerateToken(userID int64, secretKey string, lifespan time.Duration) (string, error) {
	logger.Log.Info("Signing token with key", slog.String("key", secretKey))
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(lifespan)),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// SetUserCookie creates a signed JWT for the given user ID and sets it as an HTTP cookie.
// Parameters:
//   - w: ResponseWriter to write the cookie.
//   - userID: ID to embed in the token.
//   - secretKey: key used for signing.
//
// Returns an error if token signing fails.
func SetUserCookie(w http.ResponseWriter, userID int64, secretKey string, lifespan time.Duration) error {
	tokenString, err := GenerateToken(userID, secretKey, lifespan)
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

// GetUserIDFromToken parses and validates a JWT token string.
// It returns the user ID from the claims or an error if the token is invalid.
func GetUserIDFromToken(token string, secretKey string) (int64, error) {
	claims := &Claims{}
	tokenData, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return 0, err
	}
	if !tokenData.Valid {
		return 0, errors.New("invalid token")
	}
	if claims.UserID == 0 {
		return 0, errors.New("userID is empty")
	}
	return claims.UserID, nil
}
