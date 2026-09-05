package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// Service defines the business logic interface for the loyalty system.
type Service interface {
	RegisterUser(ctx context.Context, userData models.LoginRequest) (int64, error)
	LoginUser(ctx context.Context, userData models.LoginRequest) (int64, error)
}

// NewService creates a service instance with the given storage and accrual system address.
// Parameters:
//   - storage: repository implementation.
//   - accrualSystemAddress: base URL of the external accrual system.
//
// Returns the service or an error if resetting statuses fails.
func NewService(storage repository.Storage) Service {
	return &service{storage: storage}
}

type service struct {
	storage repository.Storage
}

// RegisterUser creates a new user with hashed password.
// Returns ErrorLoginAlreadyTaken if login already exists.
func (service *service) RegisterUser(ctx context.Context, userData models.LoginRequest) (int64, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %v", err)
	}
	hashedUserData := models.UserData{Login: userData.Login, PasswordHash: string(hashed)}
	userID, err := service.storage.RegisterUser(ctx, hashedUserData)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyInStorage) {
			return 0, ErrLoginAlreadyTaken
		}
		return 0, fmt.Errorf("failed to register user in storage: %v", err)
	}
	return userID, nil
}

// LoginUser validates credentials and returns user ID.
// Returns ErrorWrongUsernamePassword if login or password is incorrect.
func (service *service) LoginUser(ctx context.Context, userData models.LoginRequest) (int64, error) {
	userID, hashedPassword, err := service.storage.CheckUser(ctx, userData.Login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return 0, ErrWrongUsernamePassword
		}
		return 0, fmt.Errorf("failed to check user in storage: %v", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(userData.Password))
	if err != nil {
		return 0, ErrWrongUsernamePassword
	}
	return userID, nil
}
