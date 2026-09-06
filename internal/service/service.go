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
	RegisterUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error)
	LoginUser(ctx context.Context, userData *models.LoginRequest) (*models.LoginResponse, error)
	RefreshAccess(ctx context.Context, userID int64, refreshToken []byte, deviceName string) (string, error)
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
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}
	hashedUserData := models.UserData{Login: userData.Login, PasswordHash: hashed}
	userID, err := service.storage.RegisterUser(ctx, hashedUserData)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return nil, ErrLoginAlreadyTaken
		}
		return nil, fmt.Errorf("failed to register user in storage: %v", err)
	}
	accessToken, err := auth.GenerateToken(userID, service.secretKey, service.accessTokenLifespan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}
	refreshToken, err := service.createNewRefreshToken(ctx, userID, userData.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %v", err)
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
		return nil, fmt.Errorf("failed to check user in storage: %v", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(userData.Password))
	if err != nil {
		return nil, ErrWrongUsernamePassword
	}
	accessToken, err := auth.GenerateToken(userID, service.secretKey, service.accessTokenLifespan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}
	refreshToken, err := service.createNewRefreshToken(ctx, userID, userData.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create new refresh token: %v", err)
	}
	return &models.LoginResponse{UserID: userID, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// createNewRefreshToken creates new refresh token.
func (service *service) createNewRefreshToken(ctx context.Context, userID int64, deviceName string) ([]byte, error) {
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %v", err)
	}
	err = service.storage.SaveRefreshToken(ctx, userID, refreshToken, deviceName, service.refreshTokenLifespan)
	if err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %v", err)
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
		return "", fmt.Errorf("failed to validate refresh token: %v", err)
	}
	accessToken, err := auth.GenerateToken(userID, service.secretKey, service.accessTokenLifespan)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %v", err)
	}
	return accessToken, nil
}
