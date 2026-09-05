// Package middlewares provides HTTP middleware components for request processing.
package middlewares

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/max-marek-projects/custodia/internal/auth"
	"github.com/max-marek-projects/custodia/internal/logger"
)

// AuthMiddleware returns a middleware that validates the JWT cookie.
// Parameters:
//   - secretKey: key used to verify the JWT signature.
//
// Returns a middleware function that rejects requests without a valid cookie.
func AuthMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check if userID is exists and is valid
			_, err := auth.GetUserIDFromRequest(r, secretKey)
			if err != nil {
				if !errors.Is(err, http.ErrNoCookie) {
					// invalid cookie -> 401 Unauthorized
					logger.Log.Error("Received invalid cookie", slog.Any("error", err))
				}
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
