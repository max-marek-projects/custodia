// Package interceptors provides GRPC interceptors for request processing.
package interceptors

import (
	"context"
	"slices"
	"strings"

	"github.com/max-marek-projects/custodia/internal/auth"
	"github.com/max-marek-projects/custodia/internal/requests"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if slices.Contains([]string{
			"/custodia.Custodia/LoginUser",
			"/custodia.Custodia/RegisterUser",
			"/custodia.Custodia/Refresh",
		}, info.FullMethod) {
			return handler(ctx, req)
		}
		userID, err := getUserIDFromMetadata(ctx, secretKey)
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

func getUserIDFromMetadata(
	ctx context.Context,
	secretKey string,
) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Error(
			codes.Unauthenticated,
			"metadata is missing",
		)
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return 0, status.Error(
			codes.Unauthenticated,
			"authorization is missing",
		)
	}
	value := strings.TrimSpace(values[0])
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(value, bearerPrefix) {
		return 0, status.Error(
			codes.Unauthenticated,
			"invalid authorization scheme",
		)
	}
	token := strings.TrimSpace(
		strings.TrimPrefix(value, bearerPrefix),
	)
	if token == "" {
		return 0, status.Error(
			codes.Unauthenticated,
			"token is empty",
		)
	}
	return auth.GetUserIDFromToken(token, secretKey)
}
