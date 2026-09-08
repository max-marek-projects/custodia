package client

import (
	"context"
	"testing"

	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestClient_authInterceptor(t *testing.T) {
	// Setup: create client with mock gRPC client and temporary storage
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("APPDATA", tmpDir)

	storage := newTokenStorage("", "")
	session := &Session{}
	mockGrpc := NewMockClient(t)

	cl := &client{
		client:  mockGrpc,
		storage: storage,
		session: session,
	}

	ctx := context.Background()
	method := "/test.Service/Method"
	req := struct{}{}
	reply := struct{}{}
	var opts []grpc.CallOption

	// Helper to check if context has authorization header
	hasAuthHeader := func(ctx context.Context, token string) bool {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			return false
		}
		vals := md.Get("authorization")
		return len(vals) > 0 && vals[0] == "Bearer "+token
	}

	t.Run("no token in storage, invoker called without auth", func(t *testing.T) {
		// Storage is empty (no tokens)
		invokerCalled := false
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			invokerCalled = true
			// Check that no authorization header is present
			md, ok := metadata.FromOutgoingContext(ctx)
			if ok {
				assert.Empty(t, md.Get("authorization"))
			}
			return nil
		}

		interceptor := cl.authInterceptor()
		err := interceptor(ctx, method, req, reply, nil, invoker, opts...)
		require.NoError(t, err)
		assert.True(t, invokerCalled)
	})

	t.Run("valid token present, invoker called with auth header", func(t *testing.T) {
		token := "valid_access"
		err := storage.Save(&models.Tokens{AccessToken: token})
		require.NoError(t, err)

		invokerCalled := false
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			invokerCalled = true
			assert.True(t, hasAuthHeader(ctx, token))
			return nil
		}

		interceptor := cl.authInterceptor()
		err = interceptor(ctx, method, req, reply, nil, invoker, opts...)
		require.NoError(t, err)
		assert.True(t, invokerCalled)
	})

	t.Run("unauthenticated error triggers token refresh and retry", func(t *testing.T) {
		initialToken := "expired_token"
		newToken := "new_access"
		refreshToken := []byte("refresh")

		err := storage.Save(&models.Tokens{
			AccessToken:  initialToken,
			RefreshToken: refreshToken,
		})
		require.NoError(t, err)

		readTokens, err := storage.Read()
		require.NoError(t, err)
		require.Equal(t, initialToken, readTokens.AccessToken)
		require.Equal(t, refreshToken, readTokens.RefreshToken)

		mockGrpc.EXPECT().
			Refresh(mock.Anything, mock.MatchedBy(func(req *proto.RefreshRequest) bool {
				return req.GetRefreshToken() != nil
			}), mock.Anything).
			Return(func() *proto.RefreshResponse {
				resp := &proto.RefreshResponse{}
				resp.SetAccessToken(newToken)
				return resp
			}(), nil).
			Once()

		invokerCallCount := 0
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			invokerCallCount++
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				return status.Error(codes.Internal, "no metadata")
			}
			authVals := md.Get("authorization")
			if len(authVals) == 0 {
				return status.Error(codes.Internal, "no authorization header")
			}

			if invokerCallCount == 1 {
				if authVals[0] != "Bearer "+initialToken {
					return status.Error(codes.Internal, "wrong token in first call")
				}
				return status.Error(codes.Unauthenticated, "token expired")
			} else if invokerCallCount == 2 {
				if authVals[0] != "Bearer "+newToken {
					return status.Error(codes.Internal, "wrong token after refresh")
				}
				return nil
			}
			return nil
		}

		interceptor := cl.authInterceptor()
		err = interceptor(ctx, method, req, reply, nil, invoker, opts...)
		require.NoError(t, err)
		assert.Equal(t, 2, invokerCallCount)
	})

	t.Run("refresh fails, original error returned", func(t *testing.T) {
		initialToken := "expired_token"
		err := storage.Save(&models.Tokens{AccessToken: initialToken, RefreshToken: []byte("refresh")})
		require.NoError(t, err)

		// Mock Refresh to return error
		mockGrpc.EXPECT().
			Refresh(mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.PermissionDenied, "refresh denied")).
			Once()

		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return status.Error(codes.Unauthenticated, "token expired")
		}

		interceptor := cl.authInterceptor()
		err = interceptor(ctx, method, req, reply, nil, invoker, opts...)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.Contains(t, err.Error(), "token expired") // original error
	})

}
