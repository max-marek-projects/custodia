// Package server provides interface for service logic level.
package server

import (
	"context"

	"github.com/max-marek-projects/custodia/internal/models"
)

// Service defines the business logic interface.
//
//go:generate mockery --name=Service --output=. --outpkg=server --filename=mock_service.gen_test.go --with-expecter --structname=MockService
type Service interface {
	// RegisterUser creates a new user with the provided credentials.
	// Returns the user ID, access token, and refresh token.
	RegisterUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error)

	// LoginUser authenticates a user and returns tokens.
	LoginUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error)

	// RefreshAccess generates a new access token using a valid refresh token.
	RefreshAccess(ctx context.Context, userID int64, refreshToken []byte, deviceName string) (string, error)

	// Logout revokes the refresh token for the given device.
	Logout(ctx context.Context, userID int64, deviceName string) error

	// LogoutAllDevices revokes all refresh tokens for the user.
	LogoutAllDevices(ctx context.Context, userID int64) error

	// CreateSecret stores a new secret for the user.
	CreateSecret(ctx context.Context, userID int64, dataType models.DataType, name string, data, salt, iv []byte, metadata map[string]string) error

	// GetSecret retrieves a secret by type, name, and optional version.
	GetSecret(ctx context.Context, userID int64, dataType models.DataType, name string, version uint64) (data, salt, iv []byte, metadata map[string]string, err error)

	// RollbackSecret reverts the secret to the previous version.
	RollbackSecret(ctx context.Context, userID int64, name string) error

	// DeleteSecret permanently soft-deletes the secret.
	DeleteSecret(ctx context.Context, userID int64, name string) error

	// UpdateSecretData updates only the encrypted data (requires new salt and iv).
	UpdateSecretData(ctx context.Context, userID int64, name string, data, salt, iv []byte) error

	// UpdateSecretMetadata updates only the metadata of the secret.
	UpdateSecretMetadata(ctx context.Context, userID int64, name string, metadata map[string]string) error

	// ListSecrets returns a list of the user's secrets, optionally filtered by metadata.
	// Only the latest version of each secret is included.
	ListSecrets(ctx context.Context, userID int64, filter map[string]string) ([]models.SecretInfo, error)

	// Close closes all open connections
	Close(ctx context.Context) error
}
