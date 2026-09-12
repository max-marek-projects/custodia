// Package server provides gRPC server setup with TLS, interceptors, and lifecycle management.
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
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CreateTLSConf creates a TLS configuration for the server.
// It loads the X.509 key pair from the provided certificate and key files.
//
// Parameters:
//   - certificate: path to the PEM-encoded certificate file.
//   - key: path to the PEM-encoded private key file.
//
// Returns:
//   - *tls.Config: the TLS configuration (minimum version TLS 1.2).
//   - error: non‑nil if loading the key pair fails.
func CreateTLSConf(certificate, key string) (*tls.Config, error) {
	if certificate == "" || key == "" {
		return nil, fmt.Errorf("certificate or key path is empty")
	}
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate files: %w", err)
	}
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	logger.Log.Info("TLS configuration created")
	return tlsConf, nil
}

// Server wraps a gRPC server with its listening address.
type Server struct {
	*grpc.Server
	Addr    string
	handler *handlers.GRPCHandler
}

// NewServer creates a new gRPC server with the given address, handler, timeouts,
// cookie secret for auth, and optional TLS configuration.
// It registers the Custodia service and chains authentication and logging interceptors.
//
// Parameters:
//   - addr: the listening address (e.g., ":8080").
//   - h: the gRPC handler implementation.
//   - readTimeout: (unused, kept for compatibility) – can be removed.
//   - writeTimeout: (unused) – kept for compatibility.
//   - cookieSecret: secret key for JWT authentication.
//   - tlsConfig: optional TLS configuration; if nil, plaintext is used.
//
// Returns:
//   - *Server: the initialized server instance.
func NewServer(
	addr string,
	handler *handlers.GRPCHandler,
	readTimeout, writeTimeout time.Duration,
	cookieSecret string,
	tlsConfig *tls.Config,
) (*Server, error) {
	if err := utils.ValidateCookieSecret(cookieSecret); err != nil {
		return nil, fmt.Errorf("wrong cookie secret: %w", err)
	}
	var opts []grpc.ServerOption
	if tlsConfig != nil {
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(creds))
	}
	opts = append(opts,
		grpc.ChainUnaryInterceptor(
			interceptors.GRPCAuthInterceptor(cookieSecret),
			interceptors.GRPCLoggerInterceptor(),
		),
	)
	server := grpc.NewServer(opts...)
	proto.RegisterCustodiaServer(server, handler)
	return &Server{
		Server:  server,
		Addr:    addr,
		handler: handler,
	}, nil
}

// ListenAndServe starts the gRPC server on the configured address.
// It logs the address and returns an error if the listener cannot be created
// or the server fails to serve.
//
// Returns:
//   - error: nil on successful start (blocking) or an error if startup fails.
func (s *Server) ListenAndServe() error {
	logger.Log.Info("Starting GRPC server", slog.String("address", s.Addr))
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("create grpc listener: %w", err)
	}
	if err := s.Serve(listener); err != nil {
		logger.Log.Error("grpc server stopped", slog.Any("error", err))
		return fmt.Errorf("server run error: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the gRPC server and closes the handler.
// It calls GracefulStop() and then handler.Close().
// The context is only used for the handler.Close call.
//
// Parameters:
//   - ctx: context (ignored, kept for interface compatibility).
//
// Returns:
//   - error: always nil.
func (s *Server) Shutdown(ctx context.Context) error {
	s.GracefulStop()
	return s.handler.Close(ctx)
}
