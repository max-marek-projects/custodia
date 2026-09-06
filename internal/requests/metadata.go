package requests

import (
	"context"
	"fmt"
	"strings"

	"github.com/max-marek-projects/custodia/internal/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func GetUserIDFromMetadata(
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

func SetAccessTokenToMetadata(
	ctx context.Context,
	accessToken string,
) (context.Context, error) {
	if accessToken == "" {
		return ctx, fmt.Errorf("empty access token")
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+accessToken), nil
}
