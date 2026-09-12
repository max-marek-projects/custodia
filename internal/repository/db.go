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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/max-marek-projects/custodia/internal/config"
	"github.com/max-marek-projects/custodia/internal/models"

	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// connect establishes a connection to the database using the provided DBConf.
// Parameters:
//   - cfg: configuration containing the database URL and pool settings.
//
// Returns a ready-to-use sql.DB connection or an error if connection or ping fails.
// connect establishes a connection to the database using the provided DBConf.
// It returns a pgxpool.Pool, which is a concurrency-safe connection pool.
// Parameters:
//   - ctx: context for the connection setup.
//   - cfg: configuration containing the database URL and pool settings.
//
// Returns a ready-to-use pgxpool.Pool or an error if connection or ping fails.
func connect(ctx context.Context, cfg *config.DBConf) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}
	poolConfig.MaxConns = cfg.MaxOpenConns
	poolConfig.MinConns = cfg.MaxIdleConns
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	// Ping to verify the connection.
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("connection to the DB is not ready: %w", err)
	}
	return pool, nil
}

type DBPool interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Close()
}

// dbStorage implements Storage interface using PostgreSQL.
type dbStorage struct {
	storage DBPool
	config  *config.DBConf
	logger  *slog.Logger
}

// NewDBStorage creates a new database storage instance and runs migrations.
// Parameters:
//   - dbURL: PostgreSQL connection string.
//
// Returns the storage instance or an error if connection or migration fails.
func NewDBStorage(dbURL string, forceMigrations bool, logger *slog.Logger) (*dbStorage, error) {
	ctx := context.Background()
	config := config.NewDBConf(dbURL, forceMigrations)
	storage, err := connect(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB storage: %w", err)
	}
	dbs := &dbStorage{
		storage: storage,
		config:  config,
		logger:  logger,
	}
	err = dbs.runMigrations(storage)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	return dbs, nil
}

// runMigrations applies database migrations from the configured path.
func (dbs *dbStorage) runMigrations(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	dbs.logger.Info("Running migrations", slog.String("path", dbs.config.MigrationsPath))
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres migration driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+dbs.config.MigrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrations: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		if errDirty, ok := errors.AsType[migrate.ErrDirty](err); ok && dbs.config.ForceMigrations {
			dbs.logger.Warn("Database is dirty, forcing to previous version", slog.Int("dirty_version", errDirty.Version))
			if err := m.Force(max(errDirty.Version-1, 1)); err != nil {
				return fmt.Errorf("failed to force version: %w", err)
			}
			if err := dbs.runMigrations(pool); err != nil {
				return fmt.Errorf("failed to run migrations after force: %w", err)
			}
			dbs.logger.Info("Migrations applied successfully after force")
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
	dbs.storage.Close()
	return nil
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
	err := dbs.storage.QueryRow(ctx, query, userData.Login, userData.PasswordHash).Scan(&userID)
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
	err := dbs.storage.QueryRow(ctx, query, username).Scan(&userID, &hashedPassword)
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
	tx, err := dbs.storage.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	// mark token as revoked
	_, err = tx.Exec(ctx, `
        UPDATE tokens
        SET revoked_at = NOW()
        WHERE user_id = $1 AND device_name = $2 AND revoked_at IS NULL
    `, userID, deviceName)
	if err != nil {
		return fmt.Errorf("failed to mark token as revoked: %w", err)
	}
	// create new token
	expiresAt := time.Now().Add(ttl)
	_, err = tx.Exec(ctx, `
        INSERT INTO tokens (user_id, token_hash, expires_at, device_name)
        VALUES ($1, $2, $3, $4)
    `, userID, tokenHash, expiresAt, deviceName)
	if err != nil {
		return fmt.Errorf("failed to add new refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
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
	err := dbs.storage.QueryRow(ctx, `
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
	result, err := dbs.storage.Exec(ctx, `
        UPDATE tokens
        SET revoked_at = NOW()
        WHERE user_id = $1 AND device_name = $2 AND revoked_at IS NULL
    `, userID, deviceName)
	if err != nil {
		return fmt.Errorf("failed to mark token as revoked: %w", err)
	}
	revokedTokensAmount := result.RowsAffected()
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
	result, err := dbs.storage.Exec(ctx, `
        UPDATE tokens
        SET revoked_at = NOW()
        WHERE user_id = $1 AND revoked_at IS NULL
    `, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all tokens as revoked: %w", err)
	}
	revokedTokensAmount := result.RowsAffected()
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
		ON CONFLICT (user_id, name, version) WHERE deleted_at IS NULL DO NOTHING
    `
	result, err := dbs.storage.Exec(ctx, query,
		userID, dataType, name, data, salt, iv, rawMetadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	rowsAffected := result.RowsAffected()
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
	err = dbs.storage.QueryRow(ctx, query,
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

// RollbackSecret rolls back the secret to the previous version by soft-deleting
// the latest active version. It requires at least two active versions; if fewer
// exist, returns ErrSecretRollbackNotPossible. If the secret does not exist,
// returns ErrSecretNotFound.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: owner of the secret.
//   - name: name of the secret.
//
// Returns:
//   - error: nil on success; ErrSecretNotFound if the secret has no active
//     versions; ErrSecretRollbackNotPossible if only one version exists.
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

	tx, err := dbs.storage.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to create database transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// MAX() on an empty result set returns a single row with NULL, not
	// sql.ErrNoRows. Scan into *int64 to distinguish "no rows" from "zero".
	var maxVersion int64
	queryMax := `
		SELECT COALESCE(MAX(version), 0)
		FROM secrets
		WHERE user_id = $1 AND name = $2 AND deleted_at IS NULL
	`
	if err := tx.QueryRow(ctx, queryMax, userID, name).Scan(&maxVersion); err != nil {
		return fmt.Errorf("failed to get max version: %w", err)
	}
	if maxVersion == 0 {
		return ErrSecretNotFound
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
	result, err := tx.Exec(ctx, queryUpdate, userID, name, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to rollback secret: %w", err)
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrSecretNotFound
	}
	if err := tx.Commit(ctx); err != nil {
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
	result, err := dbs.storage.Exec(ctx, queryUpdate, userID, name)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	rowsAffected := result.RowsAffected()
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
	tx, err := dbs.storage.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

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
	err = tx.QueryRow(ctx, selectQuery, userID, name).
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
	_, err = tx.Exec(ctx, insertQuery,
		userID, name, dataType, newData, newSalt, newIV, newRawMetadata, newVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to insert new version: %w", err)
	}

	// commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListSecrets returns all active secrets of the user, optionally filtered by metadata.
// The filter uses JSONB containment (@>): only secrets containing ALL key-value pairs
// from the filter are returned. If filter is empty or nil, all active secrets are returned.
// For each secret, only the latest (highest) active version is included.
//
// Parameters:
//   - ctx: context for cancellation.
//   - userID: owner of the secrets.
//   - filter: map of metadata key-value pairs to match.
//
// Returns:
//   - []SecretInfo: slice of matching secrets, sorted by name.
//   - error: non-nil if the query fails.
func (dbs *dbStorage) ListSecrets(
	ctx context.Context,
	userID int64,
	filter map[string]string,
) ([]models.SecretInfo, error) {
	if userID == 0 {
		return nil, fmt.Errorf("empty user id: %w", ErrInvalidArgument)
	}

	// Build the metadata filter.
	var (
		query string
		args  []any
	)
	if len(filter) > 0 {
		rawFilter, err := json.Marshal(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata filter: %w", err)
		}
		query = `
			SELECT DISTINCT ON (name, type) name, type, metadata, version
			FROM secrets
			WHERE user_id = $1
			  AND deleted_at IS NULL
			  AND metadata @> $2::jsonb
			ORDER BY name, type, version DESC
		`
		args = []any{userID, rawFilter}
	} else {
		query = `
			SELECT DISTINCT ON (name, type) name, type, metadata, version
			FROM secrets
			WHERE user_id = $1
			  AND deleted_at IS NULL
			ORDER BY name, type, version DESC
		`
		args = []any{userID}
	}

	rows, err := dbs.storage.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer rows.Close()

	var result []models.SecretInfo
	for rows.Next() {
		var (
			info        models.SecretInfo
			rawMetadata []byte
		)
		if err := rows.Scan(&info.Name, &info.Type, &rawMetadata, &info.LatestVersion); err != nil {
			return nil, fmt.Errorf("failed to scan secret row: %w", err)
		}
		info.Metadata = make(map[string]string)
		if len(rawMetadata) > 0 {
			if err := json.Unmarshal(rawMetadata, &info.Metadata); err != nil {
				return nil, fmt.Errorf("failed to parse metadata: %w", err)
			}
		}
		result = append(result, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return result, nil
}
