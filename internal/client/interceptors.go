// Package client provides GRPC interceptors for client.
package client

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/requests"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// authInterceptor returns a unary client interceptor that automatically adds
// the access token to outgoing requests and handles token refresh on
// Unauthenticated errors.
//
// It reads the access token from storage and attaches it to the context metadata
// via SetAccessTokenToMetadata. If the request fails with codes.Unauthenticated,
// it attempts to refresh the access token using RefreshAccess and retries the
// request once with the new token.
//
// Returns:
//   - grpc.UnaryClientInterceptor: the interceptor function.
func (c *client) authInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		access, err := c.storage.Read()
		if err != nil {
			return fmt.Errorf("failed to read storage: %w", err)
		}
		if access.AccessToken != "" {
			ctx, err = requests.SetAccessTokenToMetadata(ctx, access.AccessToken)
			if err != nil {
				return fmt.Errorf("failed to set auth metadata to context: %w", err)
			}
		}
		err = invoker(ctx, method, req, reply, cc, opts...)
		if err != nil && status.Code(err) == codes.Unauthenticated {
			if refreshErr := c.RefreshAccess(ctx); refreshErr != nil {
				return fmt.Errorf("failed to refresh access token: %w", err)
			}
			access, err := c.storage.Read()
			if err != nil {
				return fmt.Errorf("failed to read storage: %w", err)
			}
			if access.AccessToken != "" {
				ctx, err = requests.SetAccessTokenToMetadata(ctx, access.AccessToken)
				if err != nil {
					return fmt.Errorf("failed to set auth metadata to context: %w", err)
				}
			}
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		return err
	}
}
