// Package client provides GRPC interceptors for client.
package client

import (
	"context"

	"github.com/max-marek-projects/custodia/internal/requests"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// authInterceptor creates UnaryClientInterceptor function for automatic authentication
func (c *client) authInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		access, err := c.storage.Read()
		if err != nil {
			return err
		}
		if access.AccessToken != "" {
			ctx, err = requests.SetAccessTokenToMetadata(ctx, access.AccessToken)
			if err != nil {
				return err
			}
		}
		err = invoker(ctx, method, req, reply, cc, opts...)
		if err != nil && status.Code(err) == codes.Unauthenticated {
			if refreshErr := c.RefreshAccess(ctx); refreshErr != nil {
				return err
			}
			access, err := c.storage.Read()
			if err != nil {
				return err
			}
			if access.AccessToken != "" {
				ctx, err = requests.SetAccessTokenToMetadata(ctx, access.AccessToken)
				if err != nil {
					return err
				}
			}
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		return err
	}
}
