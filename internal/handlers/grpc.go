// Package handlers provides gRPC handlers for authentication and secret management.
package handlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/requests"
	"github.com/max-marek-projects/custodia/internal/service"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler implements the Custodia gRPC service.
type GRPCHandler struct {
	proto.UnimplementedCustodiaServer
	service Service
	logger  *slog.Logger
}

// NewGRPCHandler creates a new GRPCHandler instance.
//
// Parameters:
//   - srv: the business logic service implementation.
//
// Returns:
//   - *GRPCHandler: the initialized handler.
func NewGRPCHandler(srv Service, logger *slog.Logger) *GRPCHandler {
	return &GRPCHandler{service: srv, logger: logger}
}

// Close closes the underlying service.
// It is called during server shutdown.
func (h *GRPCHandler) Close(ctx context.Context) error {
	return h.service.Close(ctx)
}

// ---------- Auth ----------

// RegisterUser handles user registration.
// It validates the request, calls the service to create a user, and returns tokens.
//
// Parameters:
//   - ctx: the request context.
//   - req: the registration request containing login, password, and device name.
//
// Returns:
//   - *proto.LoginResponse: contains user ID and tokens.
//   - error: gRPC status error if validation fails, login already exists, or internal error.
func (h *GRPCHandler) RegisterUser(
	ctx context.Context,
	req *proto.LoginRequest,
) (*proto.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if req.GetLogin() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty login or password")
	}
	registrationData, err := h.service.RegisterUser(ctx, &models.LoginRequest{
		Login:      req.GetLogin(),
		Password:   req.GetPassword(),
		DeviceName: req.GetDeviceName(),
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrLoginAlreadyTaken) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		h.logger.Error("Failed to register user", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	response := &proto.LoginResponse{}
	response.SetUserId(registrationData.UserID)
	response.SetAccessToken(registrationData.AccessToken)
	response.SetRefreshToken(registrationData.RefreshToken)
	return response, nil
}

// LoginUser handles user authentication.
// It validates credentials and returns tokens upon success.
//
// Parameters:
//   - ctx: the request context.
//   - req: the login request with credentials and device name.
//
// Returns:
//   - *proto.LoginResponse: contains user ID and tokens.
//   - error: gRPC status error if validation fails, wrong credentials, or internal error.
func (h *GRPCHandler) LoginUser(
	ctx context.Context,
	req *proto.LoginRequest,
) (*proto.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if req.GetLogin() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty login or password")
	}
	loginData, err := h.service.LoginUser(ctx, &models.LoginRequest{
		Login:      req.GetLogin(),
		Password:   req.GetPassword(),
		DeviceName: req.GetDeviceName(),
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrWrongUsernamePassword) {
			return nil, status.Error(codes.PermissionDenied, "Wrong username or password")
		}
		h.logger.Error("Failed to login user", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	response := &proto.LoginResponse{}
	response.SetUserId(loginData.UserID)
	response.SetAccessToken(loginData.AccessToken)
	response.SetRefreshToken(loginData.RefreshToken)
	return response, nil
}

// Refresh generates a new access token using a valid refresh token.
//
// Parameters:
//   - ctx: the request context.
//   - req: contains user ID, refresh token, and device name.
//
// Returns:
//   - *proto.RefreshResponse: contains the new access token.
//   - error: gRPC status error if validation fails, token invalid, or internal error.
func (h *GRPCHandler) Refresh(
	ctx context.Context,
	req *proto.RefreshRequest,
) (*proto.RefreshResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if req.GetRefreshToken() == nil {
		return nil, status.Error(codes.InvalidArgument, "Empty refresh token")
	}
	accessToken, err := h.service.RefreshAccess(ctx, req.GetUserId(), req.GetRefreshToken(), req.GetDeviceName())
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrRefreshTokenExpiredOrInvalid) {
			return nil, status.Error(codes.PermissionDenied, "Refresh token expired or invalid")
		}
		h.logger.Error("Failed to refresh token", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	response := &proto.RefreshResponse{}
	response.SetAccessToken(accessToken)
	return response, nil
}

// LogoutDevice revokes the refresh token for the specified device.
// It expects the user ID in the context (set by auth interceptor).
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: contains the device name to revoke.
//
// Returns:
//   - *proto.LogoutDeviceResponse: empty on success.
//   - error: gRPC status error if user ID missing, validation fails, or internal error.
func (h *GRPCHandler) LogoutDevice(
	ctx context.Context,
	req *proto.LogoutDeviceRequest,
) (*proto.LogoutDeviceResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	err := h.service.Logout(ctx, userID, req.GetDeviceName())
	if err != nil && !errors.Is(err, service.ErrNoChanges) {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		h.logger.Error("Failed to logout user", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &proto.LogoutDeviceResponse{}, nil
}

// LogoutAllDevices revokes all refresh tokens for the user.
// It expects the user ID in the context.
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: empty request.
//
// Returns:
//   - *proto.LogoutAllDevicesResponse: empty on success.
//   - error: gRPC status error if user ID missing, validation fails, or internal error.
func (h *GRPCHandler) LogoutAllDevices(
	ctx context.Context,
	req *proto.LogoutAllDevicesRequest,
) (*proto.LogoutAllDevicesResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	err := h.service.LogoutAllDevices(ctx, userID)
	if err != nil && !errors.Is(err, service.ErrNoChanges) {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		h.logger.Error("Failed to logout user from all devices", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &proto.LogoutAllDevicesResponse{}, nil
}

// ---------- Secrets ----------

// CreateSecret stores a new secret for the authenticated user.
// It expects the user ID in the context.
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: contains secret type, name, encrypted data, salt, IV, and metadata.
//
// Returns:
//   - *proto.CreateSecretResponse: empty on success.
//   - error: gRPC status error if user ID missing, validation fails, secret exists, or internal error.
func (h *GRPCHandler) CreateSecret(
	ctx context.Context,
	req *proto.CreateSecretRequest,
) (*proto.CreateSecretResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	dataType, found := models.ProtoDataTypeToString[req.GetType()]
	if !found {
		h.logger.Error("Unknown data type", slog.Int("data type", int(req.GetType())))
		return nil, status.Error(codes.InvalidArgument, "Invalid data type")
	}
	err := h.service.CreateSecret(ctx, userID, dataType, req.GetName(), req.GetData(), req.GetSalt(), req.GetIv(), req.GetMetadata())
	if err != nil {
		if errors.Is(err, service.ErrSecretConflict) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		h.logger.Error("Failed to create secret", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &proto.CreateSecretResponse{}, nil
}

// GetSecret retrieves a secret by type, name, and version.
// It expects the user ID in the context.
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: contains secret type, name, and version (0 for latest).
//
// Returns:
//   - *proto.GetSecretResponse: contains the secret data, salt, IV, and metadata.
//   - error: gRPC status error if user ID missing, validation fails, secret not found, or internal error.
func (h *GRPCHandler) GetSecret(
	ctx context.Context,
	req *proto.GetSecretRequest,
) (*proto.GetSecretResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	dataType, found := models.ProtoDataTypeToString[req.GetType()]
	if !found {
		h.logger.Error("Unknown data type", slog.Int("data type", int(req.GetType())))
		return nil, status.Error(codes.InvalidArgument, "Invalid data type")
	}
	data, salt, iv, metadata, err := h.service.GetSecret(ctx, userID, dataType, req.GetName(), req.GetVersion())
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "secret not found")
		}
		h.logger.Error("Failed to get secret", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	response := &proto.GetSecretResponse{}
	response.SetData(data)
	response.SetSalt(salt)
	response.SetIv(iv)
	response.SetMetadata(metadata)
	return response, nil
}

// RollbackSecret reverts the secret to the previous version.
// It expects the user ID in the context.
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: contains the secret name.
//
// Returns:
//   - *proto.RollbackSecretResponse: empty on success.
//   - error: gRPC status error if user ID missing, validation fails, not enough versions, secret not found, or internal error.
func (h *GRPCHandler) RollbackSecret(
	ctx context.Context,
	req *proto.RollbackSecretRequest,
) (*proto.RollbackSecretResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	err := h.service.RollbackSecret(ctx, userID, req.GetName())
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrRollbackNotPossible) {
			return nil, status.Error(codes.FailedPrecondition, "no version to roll back")
		}
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "secret not found")
		}
		h.logger.Error("Failed to rollback secret", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &proto.RollbackSecretResponse{}, nil
}

// DeleteSecret permanently soft-deletes all versions of the secret.
// It expects the user ID in the context.
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: contains the secret name.
//
// Returns:
//   - *proto.DeleteSecretResponse: empty on success.
//   - error: gRPC status error if user ID missing, validation fails, secret not found, or internal error.
func (h *GRPCHandler) DeleteSecret(
	ctx context.Context,
	req *proto.DeleteSecretRequest,
) (*proto.DeleteSecretResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	err := h.service.DeleteSecret(ctx, userID, req.GetName())
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "cred not found")
		}
		h.logger.Error("Failed to delete secret", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &proto.DeleteSecretResponse{}, nil
}

// UpdateSecretData updates only the encrypted data of a secret (requires new salt/IV).
// It expects the user ID in the context.
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: contains the secret name, new data, salt, and IV.
//
// Returns:
//   - *proto.UpdateSecretDataResponse: empty on success.
//   - error: gRPC status error if user ID missing, validation fails, secret not found, or internal error.
func (h *GRPCHandler) UpdateSecretData(
	ctx context.Context,
	req *proto.UpdateSecretDataRequest,
) (*proto.UpdateSecretDataResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	err := h.service.UpdateSecretData(ctx, userID, req.GetName(), req.GetData(), req.GetSalt(), req.GetIv())
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "cred not found")
		}
		h.logger.Error("Failed to update secret", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &proto.UpdateSecretDataResponse{}, nil
}

// UpdateSecretMetadata updates only the metadata of a secret.
// It expects the user ID in the context.
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: contains the secret name and new metadata.
//
// Returns:
//   - *proto.UpdateSecretMetadataResponse: empty on success.
//   - error: gRPC status error if user ID missing, validation fails, secret not found, or internal error.
func (h *GRPCHandler) UpdateSecretMetadata(
	ctx context.Context,
	req *proto.UpdateSecretMetadataRequest,
) (*proto.UpdateSecretMetadataResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	err := h.service.UpdateSecretMetadata(ctx, userID, req.GetName(), req.GetMetadata())
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, service.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "cred not found")
		}
		h.logger.Error("Failed to update secret", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &proto.UpdateSecretMetadataResponse{}, nil
}

// ListSecrets returns all secrets of the authenticated user, optionally filtered by metadata.
// It expects the user ID in the context.
//
// Parameters:
//   - ctx: the request context (must contain user ID).
//   - req: contains optional metadata filter (key-value pairs).
//
// Returns:
//   - *proto.ListSecretsResponse: contains the list of matching secrets.
//   - error: gRPC status error if user ID missing, validation fails, or internal error.
func (h *GRPCHandler) ListSecrets(
	ctx context.Context,
	req *proto.ListSecretsRequest,
) (*proto.ListSecretsResponse, error) {
	userID, found := requests.GetUserIDFromContext(ctx)
	if !found {
		h.logger.Error("Failed to get user id from context. Interceptor error")
		return nil, status.Error(codes.Internal, "Internal error")
	}
	filter := req.GetMetadata()
	secrets, err := h.service.ListSecrets(ctx, userID, filter)
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		h.logger.Error("Failed to list secrets", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	response := &proto.ListSecretsResponse{}
	var secretsItems []*proto.SecretInfo
	for _, s := range secrets {
		item := &proto.SecretInfo{}
		item.SetName(s.Name)
		item.SetType(models.StringDataTypeToProto[s.Type])
		item.SetMetadata(s.Metadata)
		item.SetLatestVersion(s.LatestVersion)
		secretsItems = append(secretsItems, item)
	}
	response.SetSecrets(secretsItems)
	return response, nil
}
