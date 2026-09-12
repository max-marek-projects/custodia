// Package service provides business logic for user authentication, session management,
// and secret storage operations.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/max-marek-projects/custodia/internal/auth"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/repository"
	"github.com/max-marek-projects/custodia/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// Service defines the business logic interface.
//
//go:generate mockery --name=Service --output=../handlers --outpkg=handlers --filename=mock_service.gen_test.go --with-expecter --structname=MockService
//go:generate mockery --name=Service --output=../server --outpkg=server --filename=mock_service.gen_test.go --with-expecter --structname=MockService
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

// NewService creates a new service instance with the given dependencies.
//
// Parameters:
//   - storage: repository implementation for data persistence.
//   - secretKey: key used for JWT signing.
//   - accessTokenLifespan: validity duration for access tokens.
//   - refreshTokenLifespan: validity duration for refresh tokens.
//
// Returns:
//   - Service: the service instance.
func NewService(
	storage repository.Storage,
	secretKey string,
	accessTokenLifespan, refreshTokenLifespan time.Duration,
) (Service, error) {
	if err := utils.ValidateCookieSecret(secretKey); err != nil {
		return nil, fmt.Errorf("not valid cookie secret: %w", err)
	}
	return &service{
		storage:              storage,
		secretKey:            secretKey,
		accessTokenLifespan:  accessTokenLifespan,
		refreshTokenLifespan: refreshTokenLifespan,
	}, nil
}

type service struct {
	storage              repository.Storage
	secretKey            string
	accessTokenLifespan  time.Duration
	refreshTokenLifespan time.Duration
}

// ---------- Auth ----------

// RegisterUser creates a new user with a hashed password.
// If the login already exists, returns ErrLoginAlreadyTaken.
// On success, returns access and refresh tokens.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userData: login credentials and device name.
//
// Returns:
//   - *models.LoginResponse: tokens and user ID.
//   - error: non‑nil if validation fails, password hashing fails,
//     storage fails, or token generation fails.
func (s *service) RegisterUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error) {
	if userData == nil {
		return nil, ErrInvalidArgument
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	hashedUserData := models.UserData{Login: userData.Login, PasswordHash: hashed}
	userID, err := s.storage.RegisterUser(ctx, hashedUserData)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidArgument) {
			return nil, ErrInvalidArgument
		}
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return nil, ErrLoginAlreadyTaken
		}
		return nil, fmt.Errorf("failed to register user in storage: %w", err)
	}
	accessToken, err := auth.GenerateToken(userID, s.secretKey, s.accessTokenLifespan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, err := s.createNewRefreshToken(ctx, userID, userData.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}
	return &models.LoginResponse{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// LoginUser validates user credentials and returns tokens.
// If login or password is incorrect, returns ErrWrongUsernamePassword.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userData: login credentials and device name.
//
// Returns:
//   - *models.LoginResponse: tokens and user ID.
//   - error: non‑nil if validation fails, user not found,
//     password mismatch, or token generation fails.
func (s *service) LoginUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error) {
	if userData == nil {
		return nil, ErrInvalidArgument
	}
	userID, hashedPassword, err := s.storage.CheckUser(ctx, userData.Login)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidArgument) {
			return nil, ErrInvalidArgument
		}
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrWrongUsernamePassword
		}
		return nil, fmt.Errorf("failed to check user in storage: %w", err)
	}
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(userData.Password))
	if err != nil {
		return nil, ErrWrongUsernamePassword
	}
	accessToken, err := auth.GenerateToken(userID, s.secretKey, s.accessTokenLifespan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, err := s.createNewRefreshToken(ctx, userID, userData.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}
	return &models.LoginResponse{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// createNewRefreshToken generates a new refresh token and persists it.
// Returns the raw token bytes or an error.
func (s *service) createNewRefreshToken(ctx context.Context, userID int64, deviceName string) ([]byte, error) {
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	err = s.storage.SaveRefreshToken(ctx, userID, refreshToken, deviceName, s.refreshTokenLifespan)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidArgument) {
			return nil, ErrInvalidArgument
		}
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}
	return refreshToken, nil
}

// RefreshAccess generates a new access token using a valid refresh token.
// If the refresh token is invalid or expired, returns ErrRefreshTokenExpiredOrInvalid.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - refreshToken: raw refresh token bytes.
//   - deviceName: device identifier.
//
// Returns:
//   - string: new access token.
//   - error: non‑nil if validation fails or token generation fails.
func (s *service) RefreshAccess(ctx context.Context, userID int64, refreshToken []byte, deviceName string) (string, error) {
	if len(refreshToken) == 0 {
		return "", fmt.Errorf("empty refresh token")
	}
	err := s.storage.CheckRefreshToken(ctx, userID, refreshToken, deviceName)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidArgument) {
			return "", ErrInvalidArgument
		}
		if errors.Is(err, repository.ErrRefreshTokenExpiredOrInvalid) {
			return "", ErrRefreshTokenExpiredOrInvalid
		}
		return "", fmt.Errorf("failed to validate refresh token: %w", err)
	}
	accessToken, err := auth.GenerateToken(userID, s.secretKey, s.accessTokenLifespan)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}
	return accessToken, nil
}

// Logout revokes the refresh token for the specified device.
// If no active token exists, returns ErrNoChanges.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - deviceName: device identifier.
//
// Returns:
//   - error: nil on success, or an error if revocation fails.
func (s *service) Logout(ctx context.Context, userID int64, deviceName string) error {
	err := s.storage.RevokeToken(ctx, userID, deviceName)
	if err != nil {
		if errors.Is(err, repository.ErrNoChanges) {
			return ErrNoChanges
		}
		if errors.Is(err, repository.ErrInvalidArgument) {
			return ErrInvalidArgument
		}
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

// LogoutAllDevices revokes all refresh tokens for the user.
// If no active tokens exist, returns ErrNoChanges.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//
// Returns:
//   - error: nil on success, or an error if revocation fails.
func (s *service) LogoutAllDevices(ctx context.Context, userID int64) error {
	err := s.storage.RevokeAllTokens(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNoChanges) {
			return ErrNoChanges
		}
		if errors.Is(err, repository.ErrInvalidArgument) {
			return ErrInvalidArgument
		}
		return fmt.Errorf("failed to revoke all tokens: %w", err)
	}
	return nil
}

// ---------- Secrets ----------

// CreateSecret stores a new secret for the user.
// If a secret with the same (user_id, name) already exists, returns ErrSecretConflict.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - dataType: type of the secret (e.g., credentials, text, binary, card).
//   - name: unique name for the secret.
//   - data: encrypted data.
//   - salt: encryption salt.
//   - iv: encryption IV.
//   - metadata: additional key‑value pairs (may be empty).
//
// Returns:
//   - error: nil on success, or an error if validation fails or storage fails.
func (s *service) CreateSecret(ctx context.Context, userID int64, dataType models.DataType, name string, data, salt, iv []byte, metadata map[string]string) error {
	err := s.storage.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return ErrSecretConflict
		}
		if errors.Is(err, repository.ErrInvalidArgument) {
			return ErrInvalidArgument
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}

// GetSecret retrieves a secret by type, name, and optional version.
// If version is 0, returns the latest active version.
// If the secret does not exist, returns ErrSecretNotFound.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - dataType: type of the secret.
//   - name: name of the secret.
//   - version: specific version (0 for latest).
//
// Returns:
//   - data: encrypted data.
//   - salt: encryption salt.
//   - iv: encryption IV.
//   - metadata: metadata map.
//   - error: nil on success, or an error if not found or storage fails.
func (s *service) GetSecret(
	ctx context.Context,
	userID int64,
	dataType models.DataType,
	name string,
	version uint64,
) (data, salt, iv []byte, metadata map[string]string, err error) {
	data, salt, iv, metadata, err = s.storage.GetSecret(ctx, userID, dataType, name, version)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidArgument) {
			return nil, nil, nil, nil, ErrInvalidArgument
		}
		if errors.Is(err, repository.ErrSecretNotFound) {
			return nil, nil, nil, nil, ErrSecretNotFound
		}
		return nil, nil, nil, nil, fmt.Errorf("failed to get secret from storage: %w", err)
	}
	return data, salt, iv, metadata, nil
}

// RollbackSecret reverts the secret to the previous version.
// Requires at least two active versions; otherwise returns ErrRollbackNotPossible.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - name: name of the secret.
//
// Returns:
//   - error: nil on success, or an error if rollback is not possible or storage fails.
func (s *service) RollbackSecret(ctx context.Context, userID int64, name string) error {
	err := s.storage.RollbackSecret(ctx, userID, name)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidArgument) {
			return ErrInvalidArgument
		}
		if errors.Is(err, repository.ErrSecretNotFound) {
			return ErrSecretNotFound
		}
		if errors.Is(err, repository.ErrSecretRollbackNotPossible) {
			return ErrRollbackNotPossible
		}
		return fmt.Errorf("failed to rollback secret in storage: %w", err)
	}
	return nil
}

// DeleteSecret soft‑deletes all versions of the secret.
// If no active secret exists, returns ErrSecretNotFound.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - name: name of the secret.
//
// Returns:
//   - error: nil on success, or an error if not found or storage fails.
func (s *service) DeleteSecret(ctx context.Context, userID int64, name string) error {
	err := s.storage.DeleteSecret(ctx, userID, name)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidArgument) {
			return ErrInvalidArgument
		}
		if errors.Is(err, repository.ErrSecretNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to delete secret in storage: %w", err)
	}
	return nil
}

// UpdateSecretData updates only the encrypted data of the secret.
// Requires new salt and iv.
// If the secret does not exist, returns ErrSecretNotFound.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - name: name of the secret.
//   - data: new encrypted data.
//   - salt: new salt.
//   - iv: new IV.
//
// Returns:
//   - error: nil on success, or an error if storage fails.
func (s *service) UpdateSecretData(ctx context.Context, userID int64, name string, data, salt, iv []byte) error {
	err := s.storage.UpdateSecret(ctx, userID, name, data, salt, iv, nil)
	if err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to update secret data: %w", err)
	}
	return nil
}

// UpdateSecretMetadata updates only the metadata of the secret.
// If the secret does not exist, returns ErrSecretNotFound.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - name: name of the secret.
//   - metadata: new metadata map.
//
// Returns:
//   - error: nil on success, or an error if storage fails.
func (s *service) UpdateSecretMetadata(ctx context.Context, userID int64, name string, metadata map[string]string) error {
	err := s.storage.UpdateSecret(ctx, userID, name, nil, nil, nil, metadata)
	if err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to update secret metadata: %w", err)
	}
	return nil
}

// ListSecrets returns a list of the user's secrets, optionally filtered by metadata.
// The filter uses AND semantics: only secrets containing all key-value pairs are returned.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: user identifier.
//   - filter: metadata key-value pairs to match; nil or empty returns all secrets.
//
// Returns:
//   - []repository.SecretInfo: matching secrets.
//   - error: non-nil if validation or storage fails.
func (s *service) ListSecrets(
	ctx context.Context,
	userID int64,
	filter map[string]string,
) ([]models.SecretInfo, error) {
	if userID == 0 {
		return nil, ErrInvalidArgument
	}
	secrets, err := s.storage.ListSecrets(ctx, userID, filter)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidArgument) {
			return nil, ErrInvalidArgument
		}
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	return secrets, nil
}

// Close closes the underlying storage connection.
// It should be called when the service is no longer needed.
func (s *service) Close(ctx context.Context) error {
	return s.storage.Close(ctx)
}
