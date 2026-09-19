package interceptors

import (
	"context"
	"testing"
	"time"

	"github.com/max-marek-projects/custodia/internal/auth"
	"github.com/max-marek-projects/custodia/internal/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testSecret = "test-secret-key"

// generateValidToken creates a valid JWT for testing.
func generateValidToken(userID int64) string {
	token, err := auth.GenerateToken(userID, testSecret, time.Hour)
	if err != nil {
		panic(err)
	}
	return token
}

func TestGRPCAuthInterceptor(t *testing.T) {
	interceptor := GRPCAuthInterceptor(testSecret)

	// Mock handler that returns the user ID from context (if present) for verification.
	mockHandler := func(ctx context.Context, req any) (any, error) {
		// Try to get user ID from context
		userID, ok := requests.GetUserIDFromContext(ctx)
		if ok {
			return userID, nil
		}
		return int64(0), nil
	}

	t.Run("whitelisted method skips authentication", func(t *testing.T) {
		info := &grpc.UnaryServerInfo{
			FullMethod: "/custodia.Custodia/LoginUser",
		}
		ctx := context.Background()
		resp, err := interceptor(ctx, nil, info, mockHandler)
		require.NoError(t, err)
		// mockHandler returns 0 because no user ID in context (since skipped)
		userID, ok := resp.(int64)
		assert.True(t, ok)
		assert.Equal(t, int64(0), userID)
	})

	t.Run("non-whitelisted method with valid token", func(t *testing.T) {
		userID := int64(123)
		token := generateValidToken(userID)
		md := metadata.Pairs("authorization", "Bearer "+token)
		ctx := metadata.NewIncomingContext(context.Background(), md)
		info := &grpc.UnaryServerInfo{
			FullMethod: "/custodia.Custodia/SomeProtectedMethod",
		}
		resp, err := interceptor(ctx, nil, info, mockHandler)
		require.NoError(t, err)
		// mockHandler returns user ID from context
		gotUserID, ok := resp.(int64)
		assert.True(t, ok)
		assert.Equal(t, userID, gotUserID)
	})

	t.Run("non-whitelisted method with missing metadata", func(t *testing.T) {
		ctx := context.Background() // no metadata
		info := &grpc.UnaryServerInfo{
			FullMethod: "/custodia.Custodia/SomeProtectedMethod",
		}
		_, err := interceptor(ctx, nil, info, mockHandler)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Contains(t, st.Message(), "invalid authorization")
	})

	t.Run("non-whitelisted method with missing authorization header", func(t *testing.T) {
		md := metadata.Pairs("some-other", "value")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		info := &grpc.UnaryServerInfo{
			FullMethod: "/custodia.Custodia/SomeProtectedMethod",
		}
		_, err := interceptor(ctx, nil, info, mockHandler)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Contains(t, st.Message(), "authorization is missing")
	})

	t.Run("non-whitelisted method with invalid token", func(t *testing.T) {
		// Use wrong secret to generate token
		wrongToken, err := auth.GenerateToken(456, "wrong-secret", time.Hour)
		require.NoError(t, err)
		md := metadata.Pairs("authorization", "Bearer "+wrongToken)
		ctx := metadata.NewIncomingContext(context.Background(), md)
		info := &grpc.UnaryServerInfo{
			FullMethod: "/custodia.Custodia/SomeProtectedMethod",
		}
		_, err = interceptor(ctx, nil, info, mockHandler)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Contains(t, st.Message(), "invalid authorization")
	})

	t.Run("non-whitelisted method with expired token", func(t *testing.T) {
		expiredToken, err := auth.GenerateToken(789, testSecret, -time.Hour)
		require.NoError(t, err)
		md := metadata.Pairs("authorization", "Bearer "+expiredToken)
		ctx := metadata.NewIncomingContext(context.Background(), md)
		info := &grpc.UnaryServerInfo{
			FullMethod: "/custodia.Custodia/SomeProtectedMethod",
		}
		_, err = interceptor(ctx, nil, info, mockHandler)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Contains(t, st.Message(), "invalid authorization")
	})
}
