// Package requests provides helpers for extracting authentication tokens
// from gRPC metadata and injecting them into outgoing contexts.
package requests

import (
	"context"
	"fmt"
	"strings"

	"github.com/max-marek-projects/custodia/internal/auth"
	"google.golang.org/grpc/metadata"
)

const authMetadataName = "authorization"
const bearerPrefix = "Bearer"

// GetUserIDFromMetadata extracts the JWT token from the incoming gRPC metadata,
// validates it, and returns the associated user ID.
// It expects the token to be in the "authorization" header with the "Bearer " scheme.
//
// Parameters:
//   - ctx: the incoming gRPC context containing metadata.
//   - secretKey: the secret key used for JWT signature verification.
//
// Returns:
//   - int64: the user ID extracted from the token.
//   - error: a gRPC status error (codes.Unauthenticated) if metadata is missing,
//     the authorization header is absent or malformed, or token validation fails.
func GetUserIDFromMetadata(
	ctx context.Context,
	secretKey string,
) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, ErrMissingMetadata
	}
	values := md.Get(authMetadataName)
	if len(values) == 0 {
		return 0, ErrAuthorizationDataMissing
	}
	value := strings.TrimSpace(values[0])
	if !strings.HasPrefix(value, bearerPrefix) {
		return 0, ErrInvalidAuthorizationData
	}
	token := strings.TrimSpace(
		strings.TrimPrefix(value, bearerPrefix),
	)
	if token == "" {
		return 0, ErrEmptyToken
	}
	return auth.GetUserIDFromToken(token, secretKey)
}

// SetAccessTokenToMetadata adds the Bearer access token to the outgoing gRPC
// metadata under the "authorization" key.
// It returns a new context with the metadata appended.
//
// Parameters:
//   - ctx: the parent context to which the metadata will be added.
//   - accessToken: the JWT access token (without the "Bearer " prefix).
//
// Returns:
//   - context.Context: the new context containing the metadata.
//   - error: non‑nil if the accessToken is empty.
func SetAccessTokenToMetadata(
	ctx context.Context,
	accessToken string,
) (context.Context, error) {
	if accessToken == "" {
		return ctx, ErrEmptyToken
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md.Set("authorization", fmt.Sprintf("%s %s", bearerPrefix, accessToken))
	return metadata.NewOutgoingContext(ctx, md), nil
}
