// Package interceptors provides gRPC interceptors for request processing.
package interceptors

import (
	"context"
	"fmt"
	"slices"

	"github.com/max-marek-projects/custodia/internal/requests"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCAuthInterceptor returns a unary server interceptor that validates JWT tokens
// from the incoming gRPC metadata. It skips authentication for specified whitelisted
// methods (LoginUser, RegisterUser, Refresh) and extracts the user ID from the token
// for all other methods. On success, it adds the user ID to the request context.
//
// Parameters:
//   - secretKey: the secret key used to verify the JWT signature.
//
// Returns:
//   - grpc.UnaryServerInterceptor: the interceptor function.
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
				fmt.Sprintf("invalid authorization: %s", err),
			)
		}
		ctx = requests.SetUserIDToContext(ctx, userID)
		return handler(ctx, req)
	}
}
