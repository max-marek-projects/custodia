package repository

import (
	"context"
	"time"

	"github.com/max-marek-projects/custodia/internal/models"
)

// Storage defines the interface for data persistence operations.
//
//go:generate mockery --name=Storage --output=../service  --outpkg=service --filename=mock_storage.gen_test.go --with-expecter --structname=MockStorage
type Storage interface {
	// User management
	RegisterUser(ctx context.Context, userData models.UserData) (int64, error)
	CheckUser(ctx context.Context, username string) (int64, []byte, error)

	// Tokens management
	SaveRefreshToken(ctx context.Context, userID int64, tokenHash []byte, deviceName string, ttl time.Duration) error
	CheckRefreshToken(ctx context.Context, userID int64, tokenHash []byte, deviceName string) error
	RevokeToken(ctx context.Context, userID int64, deviceName string) error
	RevokeAllTokens(ctx context.Context, userID int64) error

	// Secrets management
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
	UpdateSecret(ctx context.Context, userID int64, name string, data, salt, iv []byte, metadata map[string]string) error
	ListSecrets(ctx context.Context, userID int64, filter map[string]string) ([]models.SecretInfo, error)
	Close(ctx context.Context) error
}
