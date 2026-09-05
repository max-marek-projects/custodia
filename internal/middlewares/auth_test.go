package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/max-marek-projects/custodia/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secretKey string = "123"

func TestAuthMiddlewareMissingCookie(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	authHandler := AuthMiddleware(secretKey)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	authHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddlewareInvalidCookie(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	authHandler := AuthMiddleware(secretKey)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "InvalidToken",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	authHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddlewareValid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	authHandler := AuthMiddleware(secretKey)(handler)

	cookieRec := httptest.NewRecorder()
	err := auth.SetUserCookie(cookieRec, 123, secretKey)
	require.NoError(t, err)
	response := cookieRec.Result()
	defer response.Body.Close()
	cookie := response.Cookies()[0]

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(cookie)

	rec := httptest.NewRecorder()
	authHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}
