package interceptors

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// captureLogs captures slog output into a buffer and returns the buffer and a cleanup function.
func captureLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	testLogger := slog.New(handler)
	oldLogger := logger.Log
	logger.Log = testLogger
	return &buf, func() {
		logger.Log = oldLogger
	}
}

func TestGRPCLoggerInterceptor(t *testing.T) {
	interceptor := GRPCLoggerInterceptor()

	// Mock handler that returns a response and no error.
	okHandler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	// Mock handler that returns a gRPC error.
	grpcErrorHandler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Mock handler that returns a plain error.
	plainErrorHandler := func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("some internal error")
	}

	t.Run("logs successful request", func(t *testing.T) {
		buf, cleanup := captureLogs(t)
		defer cleanup()

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Method",
		}
		ctx := context.Background()

		resp, err := interceptor(ctx, nil, info, okHandler)
		require.NoError(t, err)
		assert.Equal(t, "response", resp)

		// Check log output
		logOutput := buf.String()
		assert.Contains(t, logOutput, `method=/test.Service/Method`)
		assert.Contains(t, logOutput, `status=OK`)
		assert.Contains(t, logOutput, `duration=`)
	})

	t.Run("logs error with gRPC status", func(t *testing.T) {
		buf, cleanup := captureLogs(t)
		defer cleanup()

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Protected",
		}
		ctx := context.Background()

		_, err := interceptor(ctx, nil, info, grpcErrorHandler)
		assert.Error(t, err)

		logOutput := buf.String()
		assert.Contains(t, logOutput, `method=/test.Service/Protected`)
		assert.Contains(t, logOutput, `status=PermissionDenied`)
		assert.Contains(t, logOutput, `duration=`)
	})

	t.Run("logs error without gRPC status", func(t *testing.T) {
		buf, cleanup := captureLogs(t)
		defer cleanup()

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Fail",
		}
		ctx := context.Background()

		_, err := interceptor(ctx, nil, info, plainErrorHandler)
		assert.Error(t, err)

		logOutput := buf.String()
		assert.Contains(t, logOutput, `method=/test.Service/Fail`)
		assert.Contains(t, logOutput, `status=Unknown`)
		assert.Contains(t, logOutput, `duration=`)
	})

	t.Run("logger is called with correct fields", func(t *testing.T) {
		buf, cleanup := captureLogs(t)
		defer cleanup()

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Detailed",
		}
		ctx := context.Background()

		_, err := interceptor(ctx, nil, info, okHandler)
		require.NoError(t, err)

		logOutput := buf.String()
		// Check that all three fields are present and appear once.
		fields := []string{"method=", "duration=", "status="}
		for _, field := range fields {
			count := strings.Count(logOutput, field)
			assert.Equal(t, 1, count, "field %q appears %d times", field, count)
		}
		// Ensure the log level is "INFO" – TextHandler includes level.
		assert.Contains(t, logOutput, `level=INFO`)
	})
}
