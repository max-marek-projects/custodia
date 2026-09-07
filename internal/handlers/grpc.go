package handlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/requests"
	"github.com/max-marek-projects/custodia/internal/service"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler handles grpc endpoints for URL shortening and redirection.
type GRPCHandler struct {
	proto.UnimplementedCustodiaServer
	service service.Service
}

// NewGRPCHandler creates a new GRPCHandler.
func NewGRPCHandler(service service.Service) *GRPCHandler {
	return &GRPCHandler{service: service}
}

// ========== AUTH ==========

// RegisterUser handles user registration.
// Reads login/password from request, creates a user, and generates access tokens.
func (h *GRPCHandler) RegisterUser(
	ctx context.Context,
	req *proto.LoginRequest,
) (*proto.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is nil",
		)
	}
	if req.GetLogin() == "" || req.GetPassword() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"Empty login or password",
		)
	}
	registrationData, err := h.service.RegisterUser(ctx, &models.LoginRequest{Login: req.GetLogin(), Password: req.GetPassword(), DeviceName: req.GetDeviceName()})
	if err != nil {
		if errors.Is(err, service.ErrLoginAlreadyTaken) {
			return nil, status.Error(
				codes.AlreadyExists,
				err.Error(),
			)
		}
		logger.Log.Error("Failed to register user", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	response := &proto.LoginResponse{}
	response.SetUserId(registrationData.UserID)
	response.SetAccessToken(registrationData.AccessToken)
	response.SetRefreshToken(registrationData.RefreshToken)
	return response, nil
}

// LoginUser handles user login.
// Reads login/password from request, checks if user is registered and generates access tokens.
func (h *GRPCHandler) LoginUser(
	ctx context.Context,
	req *proto.LoginRequest,
) (*proto.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is nil",
		)
	}
	if req.GetLogin() == "" || req.GetPassword() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"Empty login or password",
		)
	}
	loginData, err := h.service.LoginUser(ctx, &models.LoginRequest{Login: req.GetLogin(), Password: req.GetPassword(), DeviceName: req.GetDeviceName()})
	if err != nil {
		if errors.Is(err, service.ErrWrongUsernamePassword) {
			return nil, status.Error(
				codes.PermissionDenied,
				"Wrong username or password",
			)
		}
		logger.Log.Error("Failed to login user", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	response := &proto.LoginResponse{}
	response.SetUserId(loginData.UserID)
	response.SetAccessToken(loginData.AccessToken)
	response.SetRefreshToken(loginData.RefreshToken)
	return response, nil
}

// Refresh creates new access token based on refresh token.
func (h *GRPCHandler) Refresh(
	ctx context.Context,
	req *proto.RefreshRequest,
) (*proto.RefreshResponse, error) {
	if req == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"request is nil",
		)
	}
	if req.GetRefreshToken() == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"Empty refresh token",
		)
	}
	accessToken, err := h.service.RefreshAccess(ctx, req.GetUserId(), req.GetRefreshToken(), req.GetDeviceName())
	if err != nil {
		if errors.Is(err, service.ErrRefreshTokenExpiredOrInvalid) {
			return nil, status.Error(
				codes.PermissionDenied,
				"Refresh token expired or invalid",
			)
		}
		logger.Log.Error("Failed to login user", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	response := &proto.RefreshResponse{}
	response.SetAccessToken(accessToken)
	return response, nil
}

// Logout logs current user out.
func (h *GRPCHandler) LogoutDevice(
	ctx context.Context,
	req *proto.LogoutDeviceRequest,
) (*proto.LogoutDeviceResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		logger.Log.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	err := h.service.Logout(ctx, userID, req.GetDeviceName())
	if err != nil {
		logger.Log.Error("Failed to logout user", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	return &proto.LogoutDeviceResponse{}, nil
}

// Logout logs current user out.
func (h *GRPCHandler) LogoutAllDevices(
	ctx context.Context,
	req *proto.LogoutAllDevicesRequest,
) (*proto.LogoutAllDevicesResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		logger.Log.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	err := h.service.LogoutAllDevices(ctx, userID)
	if err != nil {
		logger.Log.Error("Failed to logout user from all devices", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	return &proto.LogoutAllDevicesResponse{}, nil
}

// ========== SECRETS ==========

func (h *GRPCHandler) CreateSecret(
	ctx context.Context,
	req *proto.CreateSecretRequest,
) (*proto.CreateSecretResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		logger.Log.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	dataType, found := models.ProtoDataTypeToString[req.GetType()]
	if !found {
		logger.Log.Error("Unknown data type", slog.Int("data type", int(req.GetType())))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	err := h.service.CreateSecret(ctx, userID, dataType, req.GetName(), req.GetData(), req.GetSalt(), req.GetIv(), req.GetMetadata())
	if err != nil {
		logger.Log.Error("Failed to create secret", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	return &proto.CreateSecretResponse{}, nil
}

func (h *GRPCHandler) GetSecret(
	ctx context.Context,
	req *proto.GetSecretRequest,
) (*proto.GetSecretResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		logger.Log.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	dataType, found := models.ProtoDataTypeToString[req.GetType()]
	if !found {
		logger.Log.Error("Unknown data type", slog.Int("data type", int(req.GetType())))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	data, salt, iv, metadata, err := h.service.GetSecret(ctx, userID, dataType, req.GetName(), req.GetVersion())
	if err != nil {
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(
				codes.NotFound,
				"secret not found",
			)
		}
		logger.Log.Error("Failed to create secret", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	response := &proto.GetSecretResponse{}
	response.SetData(data)
	response.SetSalt(salt)
	response.SetIv(iv)
	response.SetMetadata(metadata)
	return response, nil
}

func (h *GRPCHandler) RollbackSecret(
	ctx context.Context,
	req *proto.RollbackSecretRequest,
) (*proto.RollbackSecretResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		logger.Log.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	err := h.service.RollbackSecret(ctx, userID, req.GetName())
	if err != nil {
		if errors.Is(err, service.ErrRollbackNotPossible) {
			return nil, status.Error(
				codes.PermissionDenied,
				"no version to roll back",
			)
		}
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(
				codes.NotFound,
				"secret not found",
			)
		}
		logger.Log.Error("Failed to rollback secret", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	response := &proto.RollbackSecretResponse{}
	return response, nil
}

func (h *GRPCHandler) DeleteSecret(
	ctx context.Context,
	req *proto.DeleteSecretRequest,
) (*proto.DeleteSecretResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		logger.Log.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	err := h.service.DeleteSecret(ctx, userID, req.GetName())
	if err != nil {
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(
				codes.NotFound,
				"cred not found",
			)
		}
		logger.Log.Error("Failed to delete secret", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	response := &proto.DeleteSecretResponse{}
	return response, nil
}

func (h *GRPCHandler) UpdateSecretData(
	ctx context.Context,
	req *proto.UpdateSecretDataRequest,
) (*proto.UpdateSecretDataResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		logger.Log.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	err := h.service.UpdateSecretData(ctx, userID, req.GetName(), req.GetData(), req.GetSalt(), req.GetIv())
	if err != nil {
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(
				codes.NotFound,
				"cred not found",
			)
		}
		logger.Log.Error("Failed to update secret", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	return &proto.UpdateSecretDataResponse{}, nil
}

func (h *GRPCHandler) UpdateSecretMetadata(
	ctx context.Context,
	req *proto.UpdateSecretMetadataRequest,
) (*proto.UpdateSecretMetadataResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		logger.Log.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	err := h.service.UpdateSecretMetadata(ctx, userID, req.GetName(), req.GetMetadata())
	if err != nil {
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(
				codes.NotFound,
				"cred not found",
			)
		}
		logger.Log.Error("Failed to update secret", slog.Any("error", err))
		return nil, status.Error(
			codes.Internal,
			"Internal error",
		)
	}
	return &proto.UpdateSecretMetadataResponse{}, nil
}
