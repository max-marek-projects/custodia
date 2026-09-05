package repository

import (
	"context"

	"github.com/max-marek-projects/custodia/internal/models"
)

// Storage defines the interface for data persistence operations.
type Storage interface {
	// User management
	RegisterUser(ctx context.Context, userData models.UserData) (int64, error)
	CheckUser(ctx context.Context, username string) (int64, string, error)
}
