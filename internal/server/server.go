// Package server provides HTTP server setup with routing and middleware.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/max-marek-projects/custodia/internal/handlers"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/middlewares"
)

// Server wraps http.Server with custom configuration.
type Server struct {
	http.Server
}

// NewServer creates a new HTTP server with routes and middlewares.
// Parameters:
//   - addr: listening address (e.g., ":8080").
//   - h: handler instance with business logic.
//   - readTimeout, writeTimeout: timeouts for server.
//   - cookieSecret: secret key for auth middleware.
//
// Returns a configured Server pointer.
func NewServer(addr string, h *handlers.Handler, readTimeout, writeTimeout time.Duration, cookieSecret string) *Server {
	r := chi.NewRouter()

	//middlewares
	r.Use(middleware.Recoverer)
	r.Use(middlewares.RequestsLogger)

	r.Route("/api/user", func(api chi.Router) {
		// public endpoints
		api.Post("/register", h.RegisterUser)
		api.Post("/login", h.LoginUser)
		// protected endpoints
		api.Group(func(protected chi.Router) {
			protected.Use(middlewares.AuthMiddleware(cookieSecret))
		})
	})

	return &Server{
		Server: http.Server{
			Addr:         addr,
			Handler:      r,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
	}
}

// ListenAndServe starts the HTTP server and logs the address.
// Returns an error if the server cannot start.
func (s *Server) ListenAndServe() error {
	logger.Log.Info("Starting server", slog.String("address", s.Addr))
	return s.Server.ListenAndServe()
}
