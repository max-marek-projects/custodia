package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/max-marek-projects/custodia/internal/auth"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// Service defines the business logic interface for the loyalty system.
type Service interface {
	// users management
	RegisterUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error)
	LoginUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error)
	RefreshAccess(ctx context.Context, userID int64, refreshToken []byte, deviceName string) (string, error)
	Logout(ctx context.Context, userID int64, deviceName string) error
	LogoutAllDevices(ctx context.Context, userID int64) error
	// secrets management
	CreateSecret(ctx context.Context, userID int64, dataType models.DataType, name string, data, salt, iv []byte, metadata map[string]string) error
	GetSecret(ctx context.Context, userID int64, dataType models.DataType, name string, version uint64) (data, salt, iv []byte, metadata map[string]string, err error)
	RollbackSecret(
		ctx context.Context,
		userID int64,
		name string,
	) error
	DeleteSecret(
		ctx context.Context,
		userID int64,
		name string,
	) error
	UpdateSecretData(ctx context.Context, userID int64, name string, data, salt, iv []byte) error
	UpdateSecretMetadata(ctx context.Context, userID int64, name string, metadata map[string]string) error
}

// NewService creates a service instance with the given storage and accrual system address.
// Parameters:
//   - storage: repository implementation.
//   - accrualSystemAddress: base URL of the external accrual system.
//
// Returns the service or an error if resetting statuses fails.
func NewService(storage repository.Storage, secretKey string, accessTokenLifespan, refreshTokenLifespan time.Duration) Service {
	return &service{storage: storage, secretKey: secretKey, accessTokenLifespan: accessTokenLifespan, refreshTokenLifespan: refreshTokenLifespan}
}

type service struct {
	storage              repository.Storage
	secretKey            string
	accessTokenLifespan  time.Duration
	refreshTokenLifespan time.Duration
}

// ========== AUTH ==========

// RegisterUser creates a new user with hashed password.
// Returns ErrorLoginAlreadyTaken if login already exists.
func (service *service) RegisterUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	hashedUserData := models.UserData{Login: userData.Login, PasswordHash: hashed}
	userID, err := service.storage.RegisterUser(ctx, hashedUserData)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return nil, ErrLoginAlreadyTaken
		}
		return nil, fmt.Errorf("failed to register user in storage: %w", err)
	}
	accessToken, err := auth.GenerateToken(userID, service.secretKey, service.accessTokenLifespan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, err := service.createNewRefreshToken(ctx, userID, userData.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}
	return &models.LoginResponse{UserID: userID, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// LoginUser validates credentials and returns user ID.
// Returns ErrorWrongUsernamePassword if login or password is incorrect.
func (service *service) LoginUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error) {
	userID, hashedPassword, err := service.storage.CheckUser(ctx, userData.Login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrWrongUsernamePassword
		}
		return nil, fmt.Errorf("failed to check user in storage: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(userData.Password))
	if err != nil {
		return nil, ErrWrongUsernamePassword
	}
	accessToken, err := auth.GenerateToken(userID, service.secretKey, service.accessTokenLifespan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, err := service.createNewRefreshToken(ctx, userID, userData.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %w", err)
	}
	return &models.LoginResponse{UserID: userID, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// createNewRefreshToken creates new refresh token.
func (service *service) createNewRefreshToken(ctx context.Context, userID int64, deviceName string) ([]byte, error) {
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	err = service.storage.SaveRefreshToken(ctx, userID, refreshToken, deviceName, service.refreshTokenLifespan)
	if err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}
	return refreshToken, nil
}

// RefreshAccess creates new access token.
func (service *service) RefreshAccess(ctx context.Context, userID int64, refreshToken []byte, deviceName string) (string, error) {
	if len(refreshToken) == 0 {
		return "", fmt.Errorf("empty refresh token")
	}
	err := service.storage.CheckRefreshToken(ctx, userID, refreshToken, deviceName)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenExpiredOrInvalid) {
			return "", ErrRefreshTokenExpiredOrInvalid
		}
		return "", fmt.Errorf("failed to validate refresh token: %w", err)
	}
	accessToken, err := auth.GenerateToken(userID, service.secretKey, service.accessTokenLifespan)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}
	return accessToken, nil
}

// Logout logs current user out.
func (service *service) Logout(ctx context.Context, userID int64, deviceName string) error {
	err := service.storage.RevokeToken(ctx, userID, deviceName)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

// Logout logs current user out from all devices
func (service *service) LogoutAllDevices(ctx context.Context, userID int64) error {
	err := service.storage.RevokeAllTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all tokens: %w", err)
	}
	return nil
}

// ========== SECRETS ==========

func (service *service) CreateSecret(ctx context.Context, userID int64, dataType models.DataType, name string, data, salt, iv []byte, metadata map[string]string) error {
	err := service.storage.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}

func (service *service) GetSecret(ctx context.Context, userID int64, dataType models.DataType, name string, version uint64) (data, salt, iv []byte, metadata map[string]string, err error) {
	data, salt, iv, metadata, err = service.storage.GetSecret(ctx, userID, dataType, name, version)
	if err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			return nil, nil, nil, nil, ErrSecretNotFound
		}
		return nil, nil, nil, nil, fmt.Errorf("failed to get secret from storage: %w", err)
	}
	return data, salt, iv, metadata, nil
}

func (service *service) RollbackSecret(
	ctx context.Context,
	userID int64,
	name string,
) error {
	err := service.storage.RollbackSecret(ctx, userID, name)
	if err != nil {
		if errors.Is(err, repository.ErrSecretRollbackNotPossible) {
			return ErrRollbackNotPossible
		}
		return fmt.Errorf("failed to rollback secret in storage: %w", err)
	}
	return nil
}

func (service *service) DeleteSecret(
	ctx context.Context,
	userID int64,
	name string,
) error {
	err := service.storage.DeleteSecret(ctx, userID, name)
	if err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to delete secret in storage: %w", err)
	}
	return nil
}

func (service *service) UpdateSecretData(ctx context.Context, userID int64, name string, data, salt, iv []byte) error {
	err := service.storage.UpdateSecret(ctx, userID, name, data, salt, iv, nil)
	if err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}

func (service *service) UpdateSecretMetadata(ctx context.Context, userID int64, name string, metadata map[string]string) error {
	err := service.storage.UpdateSecret(ctx, userID, name, nil, nil, nil, metadata)
	if err != nil {
		if errors.Is(err, repository.ErrSecretNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}
