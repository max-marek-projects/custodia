package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/max-marek-projects/custodia/internal/config"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// connect establishes a connection to the database using the provided DBConf.
// Parameters:
//   - cfg: configuration containing the database URL and pool settings.
//
// Returns a ready-to-use sql.DB connection or an error if connection or ping fails.
func connect(cfg *config.DBConf) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open a connection to the DB: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connection to the DB is not ready: %w", err)
	}

	return db, nil
}

// dbStorage implements Storage interface using PostgreSQL.
type dbStorage struct {
	storage *sql.DB
	config  *config.DBConf
}

// NewDBStorage creates a new database storage instance and runs migrations.
// Parameters:
//   - dbURL: PostgreSQL connection string.
//
// Returns the storage instance or an error if connection or migration fails.
func NewDBStorage(dbURL string, forceMigrations bool) (*dbStorage, error) {
	config := config.NewDBConf(dbURL, forceMigrations)
	storage, err := connect(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB storage: %w", err)
	}
	dbs := &dbStorage{
		storage: storage,
		config:  config,
	}
	err = dbs.runMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to create DB storage: %w", err)
	}
	return dbs, nil
}

// runMigrations applies database migrations from the configured path.
func (dbs *dbStorage) runMigrations() error {
	logger.Log.Info("Running migrations", slog.String("path", dbs.config.MigrationsPath))
	m, err := migrate.New(
		"file://"+dbs.config.MigrationsPath,
		dbs.config.URL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrations: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		if errDirty, ok := errors.AsType[migrate.ErrDirty](err); ok && dbs.config.ForceMigrations {
			logger.Log.Warn("Database is dirty, forcing to previous version", slog.Int("dirty_version", errDirty.Version))
			if err := m.Force(max(errDirty.Version-1, 1)); err != nil {
				return fmt.Errorf("failed to force version: %w", err)
			}
			if err := dbs.runMigrations(); err != nil {
				return fmt.Errorf("failed to run migrations after force: %w", err)
			}
			logger.Log.Info("Migrations applied successfully after force")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// RegisterUser inserts a new user into the database.
// Parameters:
//   - userData: login and hashed password.
//
// Returns the new user ID or error (ErrAlreadyInStorage if login exists).
func (dbs *dbStorage) RegisterUser(ctx context.Context, userData models.UserData) (int64, error) {
	var userID int64
	query := `--sql
        INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (username) DO NOTHING
		RETURNING id;
	`
	err := dbs.storage.QueryRowContext(ctx, query, userData.Login, userData.PasswordHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrAlreadyInStorage
		}
		return 0, fmt.Errorf("failed to add user to storage: %w", err)
	}
	return userID, nil
}

// CheckUser retrieves user ID and hashed password by login.
// Returns ErrUserNotFound if the login does not exist.
func (dbs *dbStorage) CheckUser(ctx context.Context, username string) (int64, string, error) {
	var userID int64
	var hashedPassword string
	query := `--sql
        SELECT id, password_hash FROM users
		WHERE username = $1;
	`
	err := dbs.storage.QueryRowContext(ctx, query, username).Scan(&userID, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", ErrUserNotFound
		}
		return 0, "", fmt.Errorf("failed to get user from storage by id: %w", err)
	}
	return userID, hashedPassword, nil
}
