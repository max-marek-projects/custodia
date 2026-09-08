package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/requests"
	"github.com/max-marek-projects/custodia/internal/service"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCHandler_RegisterUser(t *testing.T) {
	ctx := context.Background()
	type want struct {
		code    codes.Code
		message string
		hasData bool
	}
	tests := []struct {
		name    string
		req     *proto.LoginRequest
		svcErr  error
		svcResp *models.LoginResponse
		want    want
	}{
		{
			name: "success",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			svcResp: &models.LoginResponse{UserID: 1, AccessToken: "access", RefreshToken: []byte("refresh")},
			want:    want{code: codes.OK, hasData: true},
		},
		{
			name: "nil request",
			req:  nil,
			want: want{code: codes.InvalidArgument, message: "request is nil"},
		},
		{
			name: "empty login",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			want: want{code: codes.InvalidArgument, message: "Empty login or password"},
		},
		{
			name: "empty password",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("")
				r.SetDeviceName("dev")
				return r
			}(),
			want: want{code: codes.InvalidArgument, message: "Empty login or password"},
		},
		{
			name: "login already taken",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: service.ErrLoginAlreadyTaken,
			want:   want{code: codes.AlreadyExists, message: service.ErrLoginAlreadyTaken.Error()},
		},
		{
			name: "invalid argument from service",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name: "internal error",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.svcErr != nil || tt.svcResp != nil {
				mockSvc.EXPECT().
					RegisterUser(ctx, mock.MatchedBy(func(req *models.LoginRequest) bool {
						return req.Login == tt.req.GetLogin() && req.Password == tt.req.GetPassword() && req.DeviceName == tt.req.GetDeviceName()
					})).
					Return(tt.svcResp, tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.RegisterUser(ctx, tt.req)

			if tt.want.code == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.svcResp.UserID, resp.GetUserId())
				assert.Equal(t, tt.svcResp.AccessToken, resp.GetAccessToken())
				assert.Equal(t, tt.svcResp.RefreshToken, resp.GetRefreshToken())
			} else {
				assert.Nil(t, resp)
				assert.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_LoginUser(t *testing.T) {
	ctx := context.Background()
	type want struct {
		code    codes.Code
		message string
		hasData bool
	}
	tests := []struct {
		name    string
		req     *proto.LoginRequest
		svcErr  error
		svcResp *models.LoginResponse
		want    want
	}{
		{
			name: "success",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			svcResp: &models.LoginResponse{UserID: 1, AccessToken: "access", RefreshToken: []byte("refresh")},
			want:    want{code: codes.OK, hasData: true},
		},
		{
			name: "nil request",
			req:  nil,
			want: want{code: codes.InvalidArgument, message: "request is nil"},
		},
		{
			name: "empty login",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			want: want{code: codes.InvalidArgument, message: "Empty login or password"},
		},
		{
			name: "empty password",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("")
				r.SetDeviceName("dev")
				return r
			}(),
			want: want{code: codes.InvalidArgument, message: "Empty login or password"},
		},
		{
			name: "wrong credentials",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("wrong")
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: service.ErrWrongUsernamePassword,
			want:   want{code: codes.PermissionDenied, message: "Wrong username or password"},
		},
		{
			name: "invalid argument from service",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name: "internal error",
			req: func() *proto.LoginRequest {
				r := &proto.LoginRequest{}
				r.SetLogin("user")
				r.SetPassword("pass")
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.svcErr != nil || tt.svcResp != nil {
				mockSvc.EXPECT().
					LoginUser(ctx, mock.MatchedBy(func(req *models.LoginRequest) bool {
						return req.Login == tt.req.GetLogin() && req.Password == tt.req.GetPassword() && req.DeviceName == tt.req.GetDeviceName()
					})).
					Return(tt.svcResp, tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.LoginUser(ctx, tt.req)

			if tt.want.code == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, tt.svcResp.UserID, resp.GetUserId())
				assert.Equal(t, tt.svcResp.AccessToken, resp.GetAccessToken())
				assert.Equal(t, tt.svcResp.RefreshToken, resp.GetRefreshToken())
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_Refresh(t *testing.T) {
	ctx := context.Background()
	type want struct {
		code    codes.Code
		message string
		hasData bool
	}
	tests := []struct {
		name     string
		req      *proto.RefreshRequest
		svcErr   error
		svcToken string
		want     want
	}{
		{
			name: "success",
			req: func() *proto.RefreshRequest {
				r := &proto.RefreshRequest{}
				r.SetUserId(1)
				r.SetRefreshToken([]byte("refresh"))
				r.SetDeviceName("dev")
				return r
			}(),
			svcToken: "new_access",
			want:     want{code: codes.OK, hasData: true},
		},
		{
			name: "nil request",
			req:  nil,
			want: want{code: codes.InvalidArgument, message: "request is nil"},
		},
		{
			name: "empty refresh token",
			req: func() *proto.RefreshRequest {
				r := &proto.RefreshRequest{}
				r.SetUserId(1)
				r.SetDeviceName("dev")
				return r
			}(),
			want: want{code: codes.InvalidArgument, message: "Empty refresh token"},
		},
		{
			name: "invalid token",
			req: func() *proto.RefreshRequest {
				r := &proto.RefreshRequest{}
				r.SetUserId(1)
				r.SetRefreshToken([]byte("invalid"))
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: service.ErrRefreshTokenExpiredOrInvalid,
			want:   want{code: codes.PermissionDenied, message: "Refresh token expired or invalid"},
		},
		{
			name: "invalid argument from service",
			req: func() *proto.RefreshRequest {
				r := &proto.RefreshRequest{}
				r.SetUserId(1)
				r.SetRefreshToken([]byte("token"))
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name: "internal error",
			req: func() *proto.RefreshRequest {
				r := &proto.RefreshRequest{}
				r.SetUserId(1)
				r.SetRefreshToken([]byte("token"))
				r.SetDeviceName("dev")
				return r
			}(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.svcErr != nil || tt.svcToken != "" {
				mockSvc.EXPECT().
					RefreshAccess(ctx, tt.req.GetUserId(), tt.req.GetRefreshToken(), tt.req.GetDeviceName()).
					Return(tt.svcToken, tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.Refresh(ctx, tt.req)

			if tt.want.code == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, tt.svcToken, resp.GetAccessToken())
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_LogoutDevice(t *testing.T) {
	ctx := context.Background()
	userID := int64(123)
	ctxWithUser := requests.SetUserIDToContext(ctx, userID)

	type want struct {
		code    codes.Code
		message string
	}
	tests := []struct {
		name   string
		ctx    context.Context
		req    *proto.LogoutDeviceRequest
		svcErr error
		want   want
	}{
		{
			name: "success",
			ctx:  ctxWithUser,
			req:  func() *proto.LogoutDeviceRequest { r := &proto.LogoutDeviceRequest{}; r.SetDeviceName("dev"); return r }(),
			want: want{code: codes.OK},
		},
		{
			name: "user ID missing in context",
			ctx:  context.Background(),
			req:  func() *proto.LogoutDeviceRequest { r := &proto.LogoutDeviceRequest{}; r.SetDeviceName("dev"); return r }(),
			want: want{code: codes.Internal, message: "Internal error"},
		},
		{
			name:   "no changes (idempotent)",
			ctx:    ctxWithUser,
			req:    func() *proto.LogoutDeviceRequest { r := &proto.LogoutDeviceRequest{}; r.SetDeviceName("dev"); return r }(),
			svcErr: service.ErrNoChanges,
			want:   want{code: codes.OK},
		},
		{
			name:   "invalid argument from service",
			ctx:    ctxWithUser,
			req:    func() *proto.LogoutDeviceRequest { r := &proto.LogoutDeviceRequest{}; r.SetDeviceName(""); return r }(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name:   "internal error",
			ctx:    ctxWithUser,
			req:    func() *proto.LogoutDeviceRequest { r := &proto.LogoutDeviceRequest{}; r.SetDeviceName("dev"); return r }(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.want.code == codes.OK && tt.ctx != context.Background() {
				mockSvc.EXPECT().
					Logout(tt.ctx, userID, tt.req.GetDeviceName()).
					Return(tt.svcErr)
			} else if tt.svcErr != nil && !errors.Is(tt.svcErr, service.ErrNoChanges) {
				mockSvc.EXPECT().
					Logout(tt.ctx, userID, tt.req.GetDeviceName()).
					Return(tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.LogoutDevice(tt.ctx, tt.req)

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_LogoutAllDevices(t *testing.T) {
	ctx := context.Background()
	userID := int64(123)
	ctxWithUser := requests.SetUserIDToContext(ctx, userID)

	type want struct {
		code    codes.Code
		message string
	}
	tests := []struct {
		name   string
		ctx    context.Context
		svcErr error
		want   want
	}{
		{"success", ctxWithUser, nil, want{code: codes.OK}},
		{"user ID missing", context.Background(), nil, want{code: codes.Internal, message: "Internal error"}},
		{"no changes", ctxWithUser, service.ErrNoChanges, want{code: codes.OK}},
		{"invalid argument from service", ctxWithUser, service.ErrInvalidArgument, want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()}},
		{"internal error", ctxWithUser, errors.New("db down"), want{code: codes.Internal, message: "Internal error"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.want.code == codes.OK && tt.ctx != context.Background() {
				mockSvc.EXPECT().
					LogoutAllDevices(tt.ctx, userID).
					Return(tt.svcErr)
			} else if tt.svcErr != nil && !errors.Is(tt.svcErr, service.ErrNoChanges) {
				mockSvc.EXPECT().
					LogoutAllDevices(tt.ctx, userID).
					Return(tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.LogoutAllDevices(tt.ctx, &proto.LogoutAllDevicesRequest{})

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

// ----- Secrets handlers tests -----

func TestGRPCHandler_CreateSecret(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	ctxWithUser := requests.SetUserIDToContext(ctx, userID)

	type want struct {
		code    codes.Code
		message string
	}
	tests := []struct {
		name   string
		ctx    context.Context
		req    *proto.CreateSecretRequest
		svcErr error
		want   want
	}{
		{
			name: "success",
			ctx:  ctxWithUser,
			req: func() *proto.CreateSecretRequest {
				r := &proto.CreateSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("secret")
				r.SetData([]byte("data"))
				r.SetSalt([]byte("salt"))
				r.SetIv([]byte("iv"))
				r.SetMetadata(map[string]string{"key": "value"})
				return r
			}(),
			want: want{code: codes.OK},
		},
		{
			name: "user ID missing",
			ctx:  context.Background(),
			req: func() *proto.CreateSecretRequest {
				r := &proto.CreateSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("secret")
				return r
			}(),
			want: want{code: codes.Internal, message: "Internal error"},
		},
		{
			name: "invalid data type",
			ctx:  ctxWithUser,
			req: func() *proto.CreateSecretRequest {
				r := &proto.CreateSecretRequest{}
				r.SetType(999)
				r.SetName("secret")
				return r
			}(),
			want: want{code: codes.InvalidArgument, message: "Invalid data type"},
		},
		{
			name: "conflict",
			ctx:  ctxWithUser,
			req: func() *proto.CreateSecretRequest {
				r := &proto.CreateSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("secret")
				return r
			}(),
			svcErr: service.ErrSecretConflict,
			want:   want{code: codes.AlreadyExists, message: service.ErrSecretConflict.Error()},
		},
		{
			name: "invalid argument",
			ctx:  ctxWithUser,
			req: func() *proto.CreateSecretRequest {
				r := &proto.CreateSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("")
				return r
			}(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name: "internal error",
			ctx:  ctxWithUser,
			req: func() *proto.CreateSecretRequest {
				r := &proto.CreateSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("secret")
				return r
			}(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.want.code == codes.OK || tt.svcErr != nil {
				dataType := models.ProtoDataTypeToString[tt.req.GetType()]
				mockSvc.EXPECT().
					CreateSecret(tt.ctx, userID, dataType, tt.req.GetName(), tt.req.GetData(), tt.req.GetSalt(), tt.req.GetIv(), tt.req.GetMetadata()).
					Return(tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.CreateSecret(tt.ctx, tt.req)

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_GetSecret(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	ctxWithUser := requests.SetUserIDToContext(ctx, userID)

	type want struct {
		code    codes.Code
		message string
		hasData bool
	}
	tests := []struct {
		name    string
		ctx     context.Context
		req     *proto.GetSecretRequest
		svcData []byte
		svcSalt []byte
		svcIv   []byte
		svcMeta map[string]string
		svcErr  error
		want    want
	}{
		{
			name: "success",
			ctx:  ctxWithUser,
			req: func() *proto.GetSecretRequest {
				r := &proto.GetSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("secret")
				r.SetVersion(0)
				return r
			}(),
			svcData: []byte("data"),
			svcSalt: []byte("salt"),
			svcIv:   []byte("iv"),
			svcMeta: map[string]string{"key": "value"},
			want:    want{code: codes.OK, hasData: true},
		},
		{
			name: "user ID missing",
			ctx:  context.Background(),
			req: func() *proto.GetSecretRequest {
				r := &proto.GetSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("secret")
				return r
			}(),
			want: want{code: codes.Internal, message: "Internal error"},
		},
		{
			name: "invalid data type",
			ctx:  ctxWithUser,
			req: func() *proto.GetSecretRequest {
				r := &proto.GetSecretRequest{}
				r.SetType(999)
				r.SetName("secret")
				return r
			}(),
			want: want{code: codes.InvalidArgument, message: "Invalid data type"},
		},
		{
			name: "not found",
			ctx:  ctxWithUser,
			req: func() *proto.GetSecretRequest {
				r := &proto.GetSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("missing")
				r.SetVersion(0)
				return r
			}(),
			svcErr: service.ErrSecretNotFound,
			want:   want{code: codes.NotFound, message: "secret not found"},
		},
		{
			name: "invalid argument",
			ctx:  ctxWithUser,
			req: func() *proto.GetSecretRequest {
				r := &proto.GetSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("")
				r.SetVersion(0)
				return r
			}(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name: "internal error",
			ctx:  ctxWithUser,
			req: func() *proto.GetSecretRequest {
				r := &proto.GetSecretRequest{}
				r.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
				r.SetName("secret")
				r.SetVersion(0)
				return r
			}(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.want.code == codes.OK || tt.svcErr != nil {
				dataType := models.ProtoDataTypeToString[tt.req.GetType()]
				mockSvc.EXPECT().
					GetSecret(tt.ctx, userID, dataType, tt.req.GetName(), tt.req.GetVersion()).
					Return(tt.svcData, tt.svcSalt, tt.svcIv, tt.svcMeta, tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.GetSecret(tt.ctx, tt.req)

			if tt.want.code == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, tt.svcData, resp.GetData())
				assert.Equal(t, tt.svcSalt, resp.GetSalt())
				assert.Equal(t, tt.svcIv, resp.GetIv())
				assert.Equal(t, tt.svcMeta, resp.GetMetadata())
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_RollbackSecret(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	ctxWithUser := requests.SetUserIDToContext(ctx, userID)

	type want struct {
		code    codes.Code
		message string
	}
	tests := []struct {
		name   string
		ctx    context.Context
		req    *proto.RollbackSecretRequest
		svcErr error
		want   want
	}{
		{
			name: "success",
			ctx:  ctxWithUser,
			req: func() *proto.RollbackSecretRequest {
				r := &proto.RollbackSecretRequest{}
				r.SetName("secret")
				return r
			}(),
			want: want{code: codes.OK},
		},
		{
			name: "user ID missing",
			ctx:  context.Background(),
			req: func() *proto.RollbackSecretRequest {
				r := &proto.RollbackSecretRequest{}
				r.SetName("secret")
				return r
			}(),
			want: want{code: codes.Internal, message: "Internal error"},
		},
		{
			name:   "invalid argument",
			ctx:    ctxWithUser,
			req:    func() *proto.RollbackSecretRequest { r := &proto.RollbackSecretRequest{}; r.SetName(""); return r }(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name: "rollback not possible (no previous version)",
			ctx:  ctxWithUser,
			req: func() *proto.RollbackSecretRequest {
				r := &proto.RollbackSecretRequest{}
				r.SetName("secret")
				return r
			}(),
			svcErr: service.ErrRollbackNotPossible,
			want:   want{code: codes.NotFound, message: "no version to roll back"},
		},
		{
			name: "secret not found",
			ctx:  ctxWithUser,
			req: func() *proto.RollbackSecretRequest {
				r := &proto.RollbackSecretRequest{}
				r.SetName("missing")
				return r
			}(),
			svcErr: service.ErrSecretNotFound,
			want:   want{code: codes.NotFound, message: "secret not found"},
		},
		{
			name: "internal error",
			ctx:  ctxWithUser,
			req: func() *proto.RollbackSecretRequest {
				r := &proto.RollbackSecretRequest{}
				r.SetName("secret")
				return r
			}(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.want.code == codes.OK || tt.svcErr != nil {
				mockSvc.EXPECT().
					RollbackSecret(tt.ctx, userID, tt.req.GetName()).
					Return(tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.RollbackSecret(tt.ctx, tt.req)

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_DeleteSecret(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	ctxWithUser := requests.SetUserIDToContext(ctx, userID)

	type want struct {
		code    codes.Code
		message string
	}
	tests := []struct {
		name   string
		ctx    context.Context
		req    *proto.DeleteSecretRequest
		svcErr error
		want   want
	}{
		{
			name: "success",
			ctx:  ctxWithUser,
			req:  func() *proto.DeleteSecretRequest { r := &proto.DeleteSecretRequest{}; r.SetName("secret"); return r }(),
			want: want{code: codes.OK},
		},
		{
			name: "user ID missing",
			ctx:  context.Background(),
			req:  func() *proto.DeleteSecretRequest { r := &proto.DeleteSecretRequest{}; r.SetName("secret"); return r }(),
			want: want{code: codes.Internal, message: "Internal error"},
		},
		{
			name:   "invalid argument",
			ctx:    ctxWithUser,
			req:    func() *proto.DeleteSecretRequest { r := &proto.DeleteSecretRequest{}; r.SetName(""); return r }(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name:   "secret not found",
			ctx:    ctxWithUser,
			req:    func() *proto.DeleteSecretRequest { r := &proto.DeleteSecretRequest{}; r.SetName("missing"); return r }(),
			svcErr: service.ErrSecretNotFound,
			want:   want{code: codes.NotFound, message: "cred not found"},
		},
		{
			name:   "internal error",
			ctx:    ctxWithUser,
			req:    func() *proto.DeleteSecretRequest { r := &proto.DeleteSecretRequest{}; r.SetName("secret"); return r }(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.want.code == codes.OK || tt.svcErr != nil {
				mockSvc.EXPECT().
					DeleteSecret(tt.ctx, userID, tt.req.GetName()).
					Return(tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.DeleteSecret(tt.ctx, tt.req)

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_UpdateSecretData(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	ctxWithUser := requests.SetUserIDToContext(ctx, userID)

	type want struct {
		code    codes.Code
		message string
	}
	tests := []struct {
		name   string
		ctx    context.Context
		req    *proto.UpdateSecretDataRequest
		svcErr error
		want   want
	}{
		{
			name: "success",
			ctx:  ctxWithUser,
			req: func() *proto.UpdateSecretDataRequest {
				r := &proto.UpdateSecretDataRequest{}
				r.SetName("secret")
				r.SetData([]byte("newdata"))
				r.SetSalt([]byte("newsalt"))
				r.SetIv([]byte("newiv"))
				return r
			}(),
			want: want{code: codes.OK},
		},
		{
			name: "user ID missing",
			ctx:  context.Background(),
			req: func() *proto.UpdateSecretDataRequest {
				r := &proto.UpdateSecretDataRequest{}
				r.SetName("secret")
				return r
			}(),
			want: want{code: codes.Internal, message: "Internal error"},
		},
		{
			name:   "invalid argument",
			ctx:    ctxWithUser,
			req:    func() *proto.UpdateSecretDataRequest { r := &proto.UpdateSecretDataRequest{}; r.SetName(""); return r }(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name: "secret not found",
			ctx:  ctxWithUser,
			req: func() *proto.UpdateSecretDataRequest {
				r := &proto.UpdateSecretDataRequest{}
				r.SetName("missing")
				return r
			}(),
			svcErr: service.ErrSecretNotFound,
			want:   want{code: codes.NotFound, message: "cred not found"},
		},
		{
			name: "internal error",
			ctx:  ctxWithUser,
			req: func() *proto.UpdateSecretDataRequest {
				r := &proto.UpdateSecretDataRequest{}
				r.SetName("secret")
				return r
			}(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.want.code == codes.OK || tt.svcErr != nil {
				mockSvc.EXPECT().
					UpdateSecretData(tt.ctx, userID, tt.req.GetName(), tt.req.GetData(), tt.req.GetSalt(), tt.req.GetIv()).
					Return(tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.UpdateSecretData(tt.ctx, tt.req)

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_UpdateSecretMetadata(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	ctxWithUser := requests.SetUserIDToContext(ctx, userID)

	type want struct {
		code    codes.Code
		message string
	}
	tests := []struct {
		name   string
		ctx    context.Context
		req    *proto.UpdateSecretMetadataRequest
		svcErr error
		want   want
	}{
		{
			name: "success",
			ctx:  ctxWithUser,
			req: func() *proto.UpdateSecretMetadataRequest {
				r := &proto.UpdateSecretMetadataRequest{}
				r.SetName("secret")
				r.SetMetadata(map[string]string{"new": "meta"})
				return r
			}(),
			want: want{code: codes.OK},
		},
		{
			name: "user ID missing",
			ctx:  context.Background(),
			req: func() *proto.UpdateSecretMetadataRequest {
				r := &proto.UpdateSecretMetadataRequest{}
				r.SetName("secret")
				return r
			}(),
			want: want{code: codes.Internal, message: "Internal error"},
		},
		{
			name: "invalid argument",
			ctx:  ctxWithUser,
			req: func() *proto.UpdateSecretMetadataRequest {
				r := &proto.UpdateSecretMetadataRequest{}
				r.SetName("")
				return r
			}(),
			svcErr: service.ErrInvalidArgument,
			want:   want{code: codes.InvalidArgument, message: service.ErrInvalidArgument.Error()},
		},
		{
			name: "secret not found",
			ctx:  ctxWithUser,
			req: func() *proto.UpdateSecretMetadataRequest {
				r := &proto.UpdateSecretMetadataRequest{}
				r.SetName("missing")
				return r
			}(),
			svcErr: service.ErrSecretNotFound,
			want:   want{code: codes.NotFound, message: "cred not found"},
		},
		{
			name: "internal error",
			ctx:  ctxWithUser,
			req: func() *proto.UpdateSecretMetadataRequest {
				r := &proto.UpdateSecretMetadataRequest{}
				r.SetName("secret")
				return r
			}(),
			svcErr: errors.New("db down"),
			want:   want{code: codes.Internal, message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := NewMockService(t)
			if tt.want.code == codes.OK || tt.svcErr != nil {
				mockSvc.EXPECT().
					UpdateSecretMetadata(tt.ctx, userID, tt.req.GetName(), tt.req.GetMetadata()).
					Return(tt.svcErr)
			}
			h := NewGRPCHandler(mockSvc)
			resp, err := h.UpdateSecretMetadata(tt.ctx, tt.req)

			if tt.want.code == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			} else {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.want.code, st.Code())
				assert.Contains(t, st.Message(), tt.want.message)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPCHandler_Close(t *testing.T) {
	mockService := NewMockService(t)
	mockService.EXPECT().Close(mock.Anything).Return(nil)

	h := NewGRPCHandler(mockService)
	err := h.Close(context.Background())
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}
