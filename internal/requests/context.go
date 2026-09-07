// Package requests provides helpers for storing and retrieving context values.
package requests

import "context"

type contextKey string

const userIDKey contextKey = "userID"

// SetUserIDToContext adds the user ID to the context and returns the new context.
// This is used to propagate the authenticated user ID to downstream handlers.
//
// Parameters:
//   - ctx: the parent context.
//   - userID: the authenticated user ID to store.
//
// Returns:
//   - context.Context: the new context with the user ID value set.
func SetUserIDToContext(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext retrieves the user ID from the context.
//
// Parameters:
//   - ctx: the context from which to extract the user ID.
//
// Returns:
//   - int64: the stored user ID, or 0 if not present.
//   - bool: true if the value was present and of the correct type, false otherwise.
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}
