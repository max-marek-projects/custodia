package requests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/max-marek-projects/custodia/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const testSecret = "test-secret-key"

// generateValidToken creates a valid JWT for testing.
func generateValidToken(userID int64, secret string) string {
	token, err := auth.GenerateToken(userID, secret, time.Hour)
	if err != nil {
		panic(err)
	}
	return token
}

func TestGetUserIDFromMetadata(t *testing.T) {
	t.Run("successful extraction", func(t *testing.T) {
		userID := int64(123)
		token := generateValidToken(userID, testSecret)
		md := metadata.Pairs(authMetadataName, fmt.Sprintf("%s %s", bearerPrefix, token))
		ctx := metadata.NewIncomingContext(context.Background(), md)

		gotUserID, err := GetUserIDFromMetadata(ctx, testSecret)
		require.NoError(t, err)
		assert.Equal(t, userID, gotUserID)
	})

	t.Run("missing metadata", func(t *testing.T) {
		ctx := context.Background() // no metadata
		_, err := GetUserIDFromMetadata(ctx, testSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing metadata")
	})

	t.Run("missing authorization header", func(t *testing.T) {
		md := metadata.Pairs("some-other", "value")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		_, err := GetUserIDFromMetadata(ctx, testSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authorization is missing")
	})

	t.Run("invalid scheme", func(t *testing.T) {
		md := metadata.Pairs(authMetadataName, "Basic dXNlcjpwYXNz")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		_, err := GetUserIDFromMetadata(ctx, testSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid authorization scheme")
	})

	t.Run("empty token", func(t *testing.T) {
		md := metadata.Pairs(authMetadataName, fmt.Sprintf("%s ", bearerPrefix))
		ctx := metadata.NewIncomingContext(context.Background(), md)
		_, err := GetUserIDFromMetadata(ctx, testSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is empty")
	})

	t.Run("invalid token (wrong secret)", func(t *testing.T) {
		userID := int64(456)
		token := generateValidToken(userID, "different-secret")
		md := metadata.Pairs(authMetadataName, fmt.Sprintf("%s %s", bearerPrefix, token))
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err := GetUserIDFromMetadata(ctx, testSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse token")
	})

	t.Run("invalid token (expired)", func(t *testing.T) {
		// Generate token with negative lifespan (already expired)
		token, err := auth.GenerateToken(789, testSecret, -time.Hour)
		require.NoError(t, err)
		md := metadata.Pairs(authMetadataName, fmt.Sprintf("%s %s", bearerPrefix, token))
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err = GetUserIDFromMetadata(ctx, testSecret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is expired")
	})
}

func TestSetAccessTokenToMetadata(t *testing.T) {
	t.Run("successful addition", func(t *testing.T) {
		ctx := context.Background()
		token := "my-access-token"
		newCtx, err := SetAccessTokenToMetadata(ctx, token)
		require.NoError(t, err)

		md, ok := metadata.FromOutgoingContext(newCtx)
		assert.True(t, ok, "outgoing metadata should exist")
		values := md.Get(authMetadataName)
		assert.Len(t, values, 1, "should have one authorization header")
		assert.Equal(t, fmt.Sprintf("%s %s", bearerPrefix, token), values[0])
	})

	t.Run("empty token returns error", func(t *testing.T) {
		ctx := context.Background()
		_, err := SetAccessTokenToMetadata(ctx, "")
		assert.Error(t, err)
		assert.Equal(t, "token is empty", err.Error())
	})

	t.Run("appends to existing metadata", func(t *testing.T) {
		ctx := context.Background()
		// Pre-populate with some other metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", "123")
		token := "another-token"
		newCtx, err := SetAccessTokenToMetadata(ctx, token)
		require.NoError(t, err)

		md, ok := metadata.FromOutgoingContext(newCtx)
		assert.True(t, ok)
		assert.Equal(t, []string{"123"}, md.Get("x-request-id"))
		assert.Equal(t, []string{fmt.Sprintf("%s %s", bearerPrefix, token)}, md.Get(authMetadataName))
	})
}
