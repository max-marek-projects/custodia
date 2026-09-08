package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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
		return nil, fmt.Errorf("failed to run migrations: %w", err)
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

// Close closes the database connection.
// It implements the io.Closer interface.
// Returns an error if closing fails.
func (dbs *dbStorage) Close(ctx context.Context) error {
	return dbs.storage.Close()
}

// ========== USERS MANAGEMENT ===========

// RegisterUser inserts a new user into the database.
// Parameters:
//   - userData: login and hashed password.
//
// Returns the new user ID or error (ErrAlreadyInStorage if login exists).
func (dbs *dbStorage) RegisterUser(ctx context.Context, userData models.UserData) (int64, error) {
	if userData.Login == "" {
		return 0, fmt.Errorf("empty login received: %w", ErrInvalidArgument)
	}
	if len(userData.PasswordHash) == 0 {
		return 0, fmt.Errorf("empty password hash received: %w", ErrInvalidArgument)
	}
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
func (dbs *dbStorage) CheckUser(ctx context.Context, username string) (int64, []byte, error) {
	if username == "" {
		return 0, nil, fmt.Errorf("empty username received: %w", ErrInvalidArgument)
	}
	var userID int64
	var hashedPassword []byte
	query := `--sql
        SELECT id, password_hash FROM users
		WHERE username = $1;
	`
	err := dbs.storage.QueryRowContext(ctx, query, username).Scan(&userID, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil, ErrUserNotFound
		}
		return 0, nil, fmt.Errorf("failed to get user from storage by id: %w", err)
	}
	return userID, hashedPassword, nil
}

// ========== TOKENS MANAGEMENT ===========

// SaveRefreshToken saves refresh token to storage
func (dbs *dbStorage) SaveRefreshToken(ctx context.Context, userID int64, tokenHash []byte, deviceName string, ttl time.Duration) error {
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if len(tokenHash) == 0 {
		return fmt.Errorf("empty token hash: %w", ErrInvalidArgument)
	}
	if deviceName == "" {
		return fmt.Errorf("empty device name: %w", ErrInvalidArgument)
	}
	if ttl == 0 {
		return fmt.Errorf("empty duration: %w", ErrInvalidArgument)
	}
	// create transaction
	tx, err := dbs.storage.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if deviceName == "" {
		return fmt.Errorf("empty device name: %w", ErrInvalidArgument)
	}
	// mark token as revoked
	_, err = tx.ExecContext(ctx, `
        UPDATE tokens
        SET revoked_at = NOW()
        WHERE user_id = $1 AND device_name = $2 AND revoked_at IS NULL
    `, userID, deviceName)
	if err != nil {
		return fmt.Errorf("failed to mark token as revoked: %w", err)
	}
	// create new token
	expiresAt := time.Now().Add(ttl)
	_, err = tx.ExecContext(ctx, `
        INSERT INTO tokens (user_id, token_hash, expires_at, device_name)
        VALUES ($1, $2, $3, $4)
    `, userID, tokenHash, expiresAt, deviceName)
	if err != nil {
		return fmt.Errorf("failed to add new refresh token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// CheckRefreshToken checks refresh token
func (dbs *dbStorage) CheckRefreshToken(ctx context.Context, userID int64, tokenHash []byte, deviceName string) error {
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if len(tokenHash) == 0 {
		return fmt.Errorf("empty token hash: %w", ErrInvalidArgument)
	}
	if deviceName == "" {
		return fmt.Errorf("empty device name: %w", ErrInvalidArgument)
	}
	// check refresh token
	var unusedVar int
	err := dbs.storage.QueryRowContext(ctx, `
        SELECT 1
        FROM tokens
        WHERE user_id = $1 AND token_hash = $2 AND device_name = $3 AND revoked_at IS NULL AND expires_at > NOW() 
    `, userID, tokenHash, deviceName).Scan(&unusedVar)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRefreshTokenExpiredOrInvalid
		}
		return fmt.Errorf("failed to get info about refresh token: %w", err)
	}
	return nil
}

// RevokeToken revokes refresh token in storage based on user id and device name
func (dbs *dbStorage) RevokeToken(ctx context.Context, userID int64, deviceName string) error {
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if deviceName == "" {
		return fmt.Errorf("empty device name: %w", ErrInvalidArgument)
	}
	// mark token as revoked
	result, err := dbs.storage.ExecContext(ctx, `
        UPDATE tokens
        SET revoked_at = NOW()
        WHERE user_id = $1 AND device_name = $2 AND revoked_at IS NULL
    `, userID, deviceName)
	if err != nil {
		return fmt.Errorf("failed to mark token as revoked: %w", err)
	}
	revokedTokensAmount, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get revoked tokens amount: %w", err)
	}
	if revokedTokensAmount == 0 {
		return ErrNoChanges
	}
	return nil
}

// RevokeToken revokes all refresh tokens in storage based on user id
func (dbs *dbStorage) RevokeAllTokens(ctx context.Context, userID int64) error {
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	// revoke all tokens
	result, err := dbs.storage.ExecContext(ctx, `
        UPDATE tokens
        SET revoked_at = NOW()
        WHERE user_id = $1 AND revoked_at IS NULL
    `, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all tokens as revoked: %w", err)
	}
	revokedTokensAmount, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get revoked tokens amount: %w", err)
	}
	if revokedTokensAmount == 0 {
		return ErrNoChanges
	}
	return nil
}

// ========== SECRETS ==========

func (dbs *dbStorage) CreateSecret(ctx context.Context, userID int64, dataType models.DataType, name string, data, salt, iv []byte, metadata map[string]string) error {
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if name == "" {
		return fmt.Errorf("empty secret name: %w", ErrInvalidArgument)
	}
	if len(data) == 0 {
		return fmt.Errorf("empty data: %w", ErrInvalidArgument)
	}
	if len(salt) == 0 {
		return fmt.Errorf("empty salt: %w", ErrInvalidArgument)
	}
	if len(iv) == 0 {
		return fmt.Errorf("empty iv: %w", ErrInvalidArgument)
	}
	// create secret
	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to convert metadata to bytes: %w", err)
	}
	query := `
        INSERT INTO secrets (user_id, type, name, data, salt, iv, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, name, version) DO NOTHING
    `
	result, err := dbs.storage.ExecContext(ctx, query,
		userID, dataType, name, data, salt, iv, rawMetadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return ErrAlreadyInStorage
	}
	return nil
}

// GetSecret retrieves a secret by userID, type, name, and optionally a specific version.
// If version is 0, it returns the latest active version. If version > 0, it returns
// that exact version. Returns sql.ErrNoRows if no matching secret exists.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: owner of the secret.
//   - dataType: type of the secret.
//   - name: name of the secret.
//   - version: specific version to fetch (0 = latest, >0 = exact version).
//
// Returns:
//   - data: the encrypted data.
//   - salt: the encryption salt.
//   - iv: the encryption IV.
//   - metadata: the metadata map (parsed from JSON).
//   - err: nil on success, or an error if the secret is not found or parsing fails.
func (dbs *dbStorage) GetSecret(
	ctx context.Context,
	userID int64,
	dataType models.DataType,
	name string,
	version uint64,
) (data, salt, iv []byte, metadata map[string]string, err error) {
	// check arguments
	if userID == 0 {
		return nil, nil, nil, nil, fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if name == "" {
		return nil, nil, nil, nil, fmt.Errorf("empty secret name: %w", ErrInvalidArgument)
	}
	// get secret
	var rawMetadata []byte
	query := `
		SELECT data, salt, iv, metadata FROM secrets
		WHERE user_id = $1 AND type = $2 AND name = $3
		  AND (version = $4 OR $4 = 0)
		  AND deleted_at IS NULL
		ORDER BY version DESC
		LIMIT 1
	`
	err = dbs.storage.QueryRowContext(ctx, query,
		userID, dataType, name, version,
	).Scan(&data, &salt, &iv, &rawMetadata)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil, nil, ErrSecretNotFound
		}
		return nil, nil, nil, nil, fmt.Errorf("failed to get secret: %w", err)
	}
	metadata = make(map[string]string)
	if len(rawMetadata) > 0 {
		err = json.Unmarshal(rawMetadata, &metadata)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}
	return data, salt, iv, metadata, nil
}

// RollbackSecret rolls back the secret to the previous version by soft-deleting the latest version.
// This makes the second-latest version the current active one. If there is only one active version,
// an error is returned.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: owner of the secret.
//   - dataType: type of the secret.
//   - name: name of the secret.
//
// Returns:
//   - error: nil on success, or an error if fewer than 2 active versions exist
//     or the update fails.
func (dbs *dbStorage) RollbackSecret(
	ctx context.Context,
	userID int64,
	name string,
) error {
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if name == "" {
		return fmt.Errorf("empty secret name: %w", ErrInvalidArgument)
	}
	// Find the maximum version among active secrets.
	tx, err := dbs.storage.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create database transaction: %w", err)
	}
	defer tx.Rollback()
	var maxVersion int64
	queryMax := `
		SELECT MAX(version)
		FROM secrets
		WHERE user_id = $1 AND name = $2 AND deleted_at IS NULL
	`
	err = tx.QueryRowContext(ctx, queryMax, userID, name).Scan(&maxVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to get max version: %w", err)
	}
	if maxVersion < 2 {
		return ErrSecretRollbackNotPossible
	}

	// Soft-delete the latest version.
	queryUpdate := `
		UPDATE secrets
		SET deleted_at = NOW()
		WHERE user_id = $1 AND name = $2 AND version = $3 AND deleted_at IS NULL
	`
	result, err := tx.ExecContext(ctx, queryUpdate, userID, name, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to rollback secret: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSecretNotFound
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (dbs *dbStorage) DeleteSecret(
	ctx context.Context,
	userID int64,
	name string,
) error {
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if name == "" {
		return fmt.Errorf("empty secret name: %w", ErrInvalidArgument)
	}
	// Soft-delete all versions version.
	queryUpdate := `
		UPDATE secrets
		SET deleted_at = NOW()
		WHERE user_id = $1 AND name = $2 AND deleted_at IS NULL
	`
	result, err := dbs.storage.ExecContext(ctx, queryUpdate, userID, name)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSecretNotFound
	}
	return nil
}

func (dbs *dbStorage) UpdateSecret(ctx context.Context, userID int64, name string, data, salt, iv []byte, metadata map[string]string) error {
	// check arguments
	if userID == 0 {
		return fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}
	if name == "" {
		return fmt.Errorf("empty secret name: %w", ErrInvalidArgument)
	}
	if data == nil && metadata == nil {
		return fmt.Errorf("missing data to update: %w", ErrInvalidArgument)
	}
	if data != nil {
		if salt == nil || iv == nil {
			return fmt.Errorf("to update data, both salt and iv must be provided: %w", ErrInvalidArgument)
		}
	}
	// start transaction
	tx, err := dbs.storage.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// get old data from database
	var oldData, oldSalt, oldIV []byte
	var oldRawMetadata []byte
	var oldVersion int64
	var dataType models.DataType

	selectQuery := `
        SELECT data, type, salt, iv, metadata, version
        FROM secrets
        WHERE user_id = $1 AND name = $2 AND deleted_at IS NULL
        ORDER BY version DESC
        LIMIT 1
        FOR UPDATE
    `
	err = tx.QueryRowContext(ctx, selectQuery, userID, name).
		Scan(&oldData, &dataType, &oldSalt, &oldIV, &oldRawMetadata, &oldVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to fetch current secret: %w", err)
	}

	oldMetadata := make(map[string]string)
	if len(oldRawMetadata) > 0 {
		err = json.Unmarshal(oldRawMetadata, &oldMetadata)
		if err != nil {
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	// set new field values
	newData := oldData
	newSalt := oldSalt
	newIV := oldIV
	newMetadata := oldMetadata
	if data != nil {
		newData = data
		newSalt = salt
		newIV = iv
	}
	if metadata != nil {
		newMetadata = metadata
	}

	newRawMetadata, err := json.Marshal(newMetadata)
	if err != nil {
		return fmt.Errorf("failed to convert metadata to bytes: %w", err)
	}

	// insert new version
	newVersion := oldVersion + 1
	insertQuery := `
        INSERT INTO secrets (user_id, name, type, data, salt, iv, metadata, version, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
    `
	_, err = tx.ExecContext(ctx, insertQuery,
		userID, name, dataType, newData, newSalt, newIV, newRawMetadata, newVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to insert new version: %w", err)
	}

	// commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
