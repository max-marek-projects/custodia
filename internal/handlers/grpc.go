package handlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
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
