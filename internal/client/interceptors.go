// Package client provides GRPC interceptors for client.
package client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+access.AccessToken)
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
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+access.AccessToken)
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		return err
	}
}
