// Package interceptors provides gRPC interceptors for request processing.
package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCLoggerInterceptor returns a unary server interceptor that logs each gRPC request.
// It logs the method name, duration, and final gRPC status code.
// If the handler returns an error with a gRPC status, the code is extracted;
// otherwise, the code is codes.OK.
//
// Parameters:
//   - none (uses global logger.Log)
//
// Returns:
//   - grpc.UnaryServerInterceptor: the interceptor function.
func GRPCLoggerInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		// Determine gRPC status code
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			} else {
				// If error is not a gRPC status, treat as Unknown
				code = codes.Unknown
			}
		}

		logger.Info(
			"Processed gRPC request",
			slog.String("method", info.FullMethod),
			slog.Duration("duration", duration),
			slog.String("status", code.String()),
		)

		return resp, err
	}
}
