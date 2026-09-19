package requests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetAndGetUserID(t *testing.T) {
	t.Run("set and retrieve user ID", func(t *testing.T) {
		ctx := context.Background()
		userID := int64(42)

		newCtx := SetUserIDToContext(ctx, userID)
		got, ok := GetUserIDFromContext(newCtx)

		assert.True(t, ok, "should retrieve user ID successfully")
		assert.Equal(t, userID, got, "retrieved user ID does not match")
	})

	t.Run("retrieve from context without value", func(t *testing.T) {
		ctx := context.Background()
		got, ok := GetUserIDFromContext(ctx)

		assert.False(t, ok, "should not find user ID")
		assert.Equal(t, int64(0), got, "should return zero value for int64")
	})

	t.Run("retrieve from context with wrong type", func(t *testing.T) {
		ctx := context.Background()
		// Put a value of wrong type under the same key
		ctx = context.WithValue(ctx, userIDKey, "not-an-int")

		got, ok := GetUserIDFromContext(ctx)

		assert.False(t, ok, "type assertion should fail")
		assert.Equal(t, int64(0), got, "should return zero value")
	})

	t.Run("multiple contexts do not interfere", func(t *testing.T) {
		ctx1 := context.Background()
		ctx2 := context.Background()

		ctx1 = SetUserIDToContext(ctx1, 100)
		ctx2 = SetUserIDToContext(ctx2, 200)

		got1, ok1 := GetUserIDFromContext(ctx1)
		got2, ok2 := GetUserIDFromContext(ctx2)

		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, int64(100), got1)
		assert.Equal(t, int64(200), got2)
	})
}
