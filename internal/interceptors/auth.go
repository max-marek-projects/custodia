// Package interceptors provides GRPC interceptors for request processing.
package interceptors

import (
	"context"
	"slices"

	"github.com/max-marek-projects/custodia/internal/requests"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthMiddleware returns a middleware that validates the JWT cookie.
// Parameters:
//   - secretKey: key used to verify the JWT signature.
//
// Returns a middleware function that rejects requests without a valid cookie.
func GRPCAuthInterceptor(
	secretKey string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if slices.Contains([]string{
			"/custodia.Custodia/LoginUser",
			"/custodia.Custodia/RegisterUser",
			"/custodia.Custodia/Refresh",
		}, info.FullMethod) {
			return handler(ctx, req)
		}
		userID, err := requests.GetUserIDFromMetadata(ctx, secretKey)
		if err != nil {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid authorization",
			)
		}
		ctx = requests.SetUserIDToContext(ctx, userID)
		return handler(ctx, req)
	}
}
