// Package server provides HTTP server setup with routing and middleware.
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/max-marek-projects/custodia/internal/handlers"
	"github.com/max-marek-projects/custodia/internal/interceptors"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CreateTLSConf creates TLS configuration for server.
func CreateTLSConf(certificate, key string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		return nil, err
	}
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	logger.Log.Info("TLS configuration created")
	return tlsConf, nil
}

// Server wraps an http.Server with pre-configured middleware and routes.
type Server struct {
	*grpc.Server
	Addr string
}

// NewServer creates a new Server instance with the given address, handler, timeouts, auditor, and cookie secret.
// It sets up chi router with all necessary middleware and routes:
// - Recoverer, Gzip, Logger, Audit middleware for all routes.
// - Public routes: /ping, /{id}
// - Protected routes (with AuthMiddleware): POST /, /api/shorten, /api/shorten/batch, /api/user/urls (GET/DELETE).
func NewServer(
	addr string,
	h *handlers.GRPCHandler,
	readTimeout, writeTimeout time.Duration,
	cookieSecret string,
	tlsConfig *tls.Config,
) *Server {
	var opts []grpc.ServerOption
	if tlsConfig != nil {
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(creds))
	}
	opts = append(opts,
		grpc.ChainUnaryInterceptor(
			interceptors.GRPCAuthInterceptor(cookieSecret),
		),
	)
	server := grpc.NewServer(opts...)
	proto.RegisterCustodiaServer(server, h)
	return &Server{
		Server: server,
		Addr:   addr,
	}
}

// ListenAndServe starts the HTTP server and logs the address.
// Returns an error if the server cannot start.
func (s *Server) ListenAndServe() error {
	logger.Log.Info("Starting GRPC server", slog.String("address", s.Addr))
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("create grpc listener: %w", err)
	}
	if err := s.Serve(listener); err != nil {
		logger.Log.Error(
			"grpc server stopped",
			slog.Any("error", err),
		)
		return err
	}
	return nil
}

// Shutdown stops the HTTP server.
// Expects context.
// Returns an error if the server wasn't closed properly.
func (s *Server) Shutdown(ctx context.Context) error {
	s.GracefulStop()
	return nil
}
