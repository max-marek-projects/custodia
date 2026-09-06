package repository

import (
	"context"
	"time"

	"github.com/max-marek-projects/custodia/internal/models"
)

// Storage defines the interface for data persistence operations.
type Storage interface {
	// User management
	RegisterUser(ctx context.Context, userData models.UserData) (int64, error)
	CheckUser(ctx context.Context, username string) (int64, []byte, error)

	// Tokens management
	SaveRefreshToken(ctx context.Context, userID int64, tokenHash []byte, deviceName string, ttl time.Duration) error
	CheckRefreshToken(ctx context.Context, userID int64, tokenHash []byte, deviceName string) error
	RevokeToken(ctx context.Context, userID int64, deviceName string) error
	RevokeAllTokens(ctx context.Context, userID int64) error
}
