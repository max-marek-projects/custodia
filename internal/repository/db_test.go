package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBStorage_RegisterUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userData := models.UserData{Login: "testuser", PasswordHash: []byte("hash")}
		expectedID := int64(42)

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(userData.Login, userData.PasswordHash).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedID))

		id, err := storage.RegisterUser(ctx, userData)
		assert.NoError(t, err)
		assert.Equal(t, expectedID, id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("conflict", func(t *testing.T) {
		userData := models.UserData{Login: "existing", PasswordHash: []byte("hash")}

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(userData.Login, userData.PasswordHash).
			WillReturnError(sql.ErrNoRows)

		_, err := storage.RegisterUser(ctx, userData)
		assert.ErrorIs(t, err, ErrAlreadyInStorage)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		userData := models.UserData{Login: "error", PasswordHash: []byte("hash")}

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(userData.Login, userData.PasswordHash).
			WillReturnError(sql.ErrConnDone)

		_, err := storage.RegisterUser(ctx, userData)
		assert.ErrorIs(t, err, sql.ErrConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty login", func(t *testing.T) {
		userData := models.UserData{Login: "", PasswordHash: []byte("hash")}
		_, err := storage.RegisterUser(ctx, userData)
		assert.ErrorIs(t, err, ErrInvalidArgument)
		assert.Contains(t, err.Error(), "empty login")
	})

	t.Run("empty password hash", func(t *testing.T) {
		userData := models.UserData{Login: "user", PasswordHash: []byte{}}
		_, err := storage.RegisterUser(ctx, userData)
		assert.ErrorIs(t, err, ErrInvalidArgument)
		assert.Contains(t, err.Error(), "empty password hash")
	})
}

func TestDBStorage_CheckUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		username := "testuser"
		expectedID := int64(42)
		expectedHash := []byte("hashedpass")

		mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE username = \$1`).
			WithArgs(username).
			WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(expectedID, expectedHash))

		id, hash, err := storage.CheckUser(ctx, username)
		assert.NoError(t, err)
		assert.Equal(t, expectedID, id)
		assert.Equal(t, expectedHash, hash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		username := "missing"

		mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE username = \$1`).
			WithArgs(username).
			WillReturnError(sql.ErrNoRows)

		_, _, err := storage.CheckUser(ctx, username)
		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		username := "error"

		mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE username = \$1`).
			WithArgs(username).
			WillReturnError(sql.ErrConnDone)

		_, _, err := storage.CheckUser(ctx, username)
		assert.ErrorIs(t, err, sql.ErrConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty username", func(t *testing.T) {
		_, _, err := storage.CheckUser(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidArgument)
		assert.Contains(t, err.Error(), "empty username")
	})
}

func TestDBStorage_SaveRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		tokenHash := []byte("hash")
		device := "device"
		ttl := 10 * time.Minute

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO tokens \(user_id, token_hash, expires_at, device_name\) VALUES \(\$1, \$2, \$3, \$4\)`).
			WithArgs(userID, tokenHash, sqlmock.AnyArg(), device).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := storage.SaveRefreshToken(ctx, userID, tokenHash, device, ttl)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("revoke fails", func(t *testing.T) {
		userID := int64(1)
		tokenHash := []byte("hash")
		device := "device"
		ttl := 10 * time.Minute

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := storage.SaveRefreshToken(ctx, userID, tokenHash, device, ttl)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit fails", func(t *testing.T) {
		userID := int64(1)
		tokenHash := []byte("hash")
		device := "device"
		ttl := 10 * time.Minute

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO tokens \(user_id, token_hash, expires_at, device_name\) VALUES \(\$1, \$2, \$3, \$4\)`).
			WithArgs(userID, tokenHash, sqlmock.AnyArg(), device).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(sql.ErrTxDone)

		err := storage.SaveRefreshToken(ctx, userID, tokenHash, device, ttl)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrTxDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.SaveRefreshToken(ctx, 0, []byte("hash"), "device", time.Minute)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty token hash", func(t *testing.T) {
		err := storage.SaveRefreshToken(ctx, 1, []byte{}, "device", time.Minute)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty device name", func(t *testing.T) {
		err := storage.SaveRefreshToken(ctx, 1, []byte("hash"), "", time.Minute)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("zero ttl", func(t *testing.T) {
		err := storage.SaveRefreshToken(ctx, 1, []byte("hash"), "device", 0)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_CheckRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("valid token", func(t *testing.T) {
		userID := int64(1)
		tokenHash := []byte("hash")
		device := "device"

		mock.ExpectQuery(`SELECT 1 FROM tokens WHERE user_id = \$1 AND token_hash = \$2 AND device_name = \$3 AND revoked_at IS NULL AND expires_at > NOW\(\)`).
			WithArgs(userID, tokenHash, device).
			WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

		err := storage.CheckRefreshToken(ctx, userID, tokenHash, device)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid/expired", func(t *testing.T) {
		userID := int64(1)
		tokenHash := []byte("hash")
		device := "device"

		mock.ExpectQuery(`SELECT 1 FROM tokens WHERE user_id = \$1 AND token_hash = \$2 AND device_name = \$3 AND revoked_at IS NULL AND expires_at > NOW\(\)`).
			WithArgs(userID, tokenHash, device).
			WillReturnError(sql.ErrNoRows)

		err := storage.CheckRefreshToken(ctx, userID, tokenHash, device)
		assert.ErrorIs(t, err, ErrRefreshTokenExpiredOrInvalid)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.CheckRefreshToken(ctx, 0, []byte("hash"), "device")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty token hash", func(t *testing.T) {
		err := storage.CheckRefreshToken(ctx, 1, []byte{}, "device")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty device name", func(t *testing.T) {
		err := storage.CheckRefreshToken(ctx, 1, []byte("hash"), "")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_RevokeToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		device := "device"

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := storage.RevokeToken(ctx, userID, device)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		userID := int64(1)
		device := "device"

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnError(sql.ErrConnDone)

		err := storage.RevokeToken(ctx, userID, device)
		assert.ErrorIs(t, err, sql.ErrConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows affected", func(t *testing.T) {
		userID := int64(1)
		device := "device"

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := storage.RevokeToken(ctx, userID, device)
		assert.ErrorIs(t, err, ErrNoChanges)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.RevokeToken(ctx, 0, "device")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty device name", func(t *testing.T) {
		err := storage.RevokeToken(ctx, 1, "")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_RevokeAllTokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND revoked_at IS NULL`).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 3))

		err := storage.RevokeAllTokens(ctx, userID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND revoked_at IS NULL`).
			WithArgs(userID).
			WillReturnError(sql.ErrConnDone)

		err := storage.RevokeAllTokens(ctx, userID)
		assert.ErrorIs(t, err, sql.ErrConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows affected", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND revoked_at IS NULL`).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := storage.RevokeAllTokens(ctx, userID)
		assert.ErrorIs(t, err, ErrNoChanges)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.RevokeAllTokens(ctx, 0)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_CreateSecret(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		dataType := models.DataTypeCredentials
		name := "mysecret"
		data := []byte("data")
		salt := []byte("salt")
		iv := []byte("iv")
		metadata := map[string]string{"key": "value"}
		rawMetadata, _ := json.Marshal(metadata)

		mock.ExpectExec(`INSERT INTO secrets \(user_id, type, name, data, salt, iv, metadata\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)`).
			WithArgs(userID, dataType, name, data, salt, iv, rawMetadata).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := storage.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		userID := int64(1)
		dataType := models.DataTypeCredentials
		name := "mysecret"
		data := []byte("data")
		salt := []byte("salt")
		iv := []byte("iv")
		metadata := map[string]string{"key": "value"}
		rawMetadata, _ := json.Marshal(metadata)

		mock.ExpectExec(`INSERT INTO secrets`).
			WithArgs(userID, dataType, name, data, salt, iv, rawMetadata).
			WillReturnError(sql.ErrConnDone)

		err := storage.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.ErrorIs(t, err, sql.ErrConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("conflict", func(t *testing.T) {
		userID := int64(1)
		dataType := models.DataTypeCredentials
		name := "mysecret"
		data := []byte("data")
		salt := []byte("salt")
		iv := []byte("iv")
		metadata := map[string]string{"key": "value"}
		rawMetadata, _ := json.Marshal(metadata)

		mock.ExpectExec(`INSERT INTO secrets \(user_id, type, name, data, salt, iv, metadata\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)`).
			WithArgs(userID, dataType, name, data, salt, iv, rawMetadata).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := storage.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.ErrorIs(t, err, ErrAlreadyInStorage)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.CreateSecret(ctx, 0, models.DataTypeCredentials, "name", []byte("d"), []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		err := storage.CreateSecret(ctx, 1, models.DataTypeCredentials, "", []byte("d"), []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty data", func(t *testing.T) {
		err := storage.CreateSecret(ctx, 1, models.DataTypeCredentials, "name", []byte{}, []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty salt", func(t *testing.T) {
		err := storage.CreateSecret(ctx, 1, models.DataTypeCredentials, "name", []byte("d"), []byte{}, []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty iv", func(t *testing.T) {
		err := storage.CreateSecret(ctx, 1, models.DataTypeCredentials, "name", []byte("d"), []byte("s"), []byte{}, nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_GetSecret(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success latest version", func(t *testing.T) {
		userID := int64(1)
		dataType := models.DataTypeCredentials
		name := "mysecret"
		version := uint64(0)
		data := []byte("data")
		salt := []byte("salt")
		iv := []byte("iv")
		metadata := map[string]string{"key": "value"}
		rawMetadata, _ := json.Marshal(metadata)

		mock.ExpectQuery(`SELECT data, salt, iv, metadata FROM secrets WHERE user_id = \$1 AND type = \$2 AND name = \$3 AND \(version = \$4 OR \$4 = 0\) AND deleted_at IS NULL ORDER BY version DESC LIMIT 1`).
			WithArgs(userID, dataType, name, version).
			WillReturnRows(sqlmock.NewRows([]string{"data", "salt", "iv", "metadata"}).
				AddRow(data, salt, iv, rawMetadata))

		gotData, gotSalt, gotIv, gotMetadata, err := storage.GetSecret(ctx, userID, dataType, name, version)
		assert.NoError(t, err)
		assert.Equal(t, data, gotData)
		assert.Equal(t, salt, gotSalt)
		assert.Equal(t, iv, gotIv)
		assert.Equal(t, metadata, gotMetadata)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		userID := int64(1)
		dataType := models.DataTypeCredentials
		name := "missing"
		version := uint64(0)

		mock.ExpectQuery(`SELECT data, salt, iv, metadata FROM secrets WHERE user_id = \$1 AND type = \$2 AND name = \$3 AND \(version = \$4 OR \$4 = 0\) AND deleted_at IS NULL ORDER BY version DESC LIMIT 1`).
			WithArgs(userID, dataType, name, version).
			WillReturnError(sql.ErrNoRows)

		_, _, _, _, err := storage.GetSecret(ctx, userID, dataType, name, version)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		_, _, _, _, err := storage.GetSecret(ctx, 0, models.DataTypeCredentials, "name", 0)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		_, _, _, _, err := storage.GetSecret(ctx, 1, models.DataTypeCredentials, "", 0)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_RollbackSecret(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT MAX\(version\) FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(3))
		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\) WHERE user_id = \$1 AND name = \$2 AND version = \$3 AND deleted_at IS NULL`).
			WithArgs(userID, name, 3).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := storage.RollbackSecret(ctx, userID, name)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not enough versions", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT MAX\(version\) FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(1))
		mock.ExpectRollback()

		err := storage.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretRollbackNotPossible)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update fails", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT MAX\(version\) FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(2))
		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\) WHERE user_id = \$1 AND name = \$2 AND version = \$3 AND deleted_at IS NULL`).
			WithArgs(userID, name, 2).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := storage.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, sql.ErrConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("select max no rows", func(t *testing.T) {
		userID := int64(1)
		name := "nonexistent"

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT MAX\(version\) FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		err := storage.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit fails", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT MAX\(version\) FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(3))
		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\) WHERE user_id = \$1 AND name = \$2 AND version = \$3 AND deleted_at IS NULL`).
			WithArgs(userID, name, 3).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(sql.ErrTxDone)

		err := storage.RollbackSecret(ctx, userID, name)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrTxDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.RollbackSecret(ctx, 0, "name")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		err := storage.RollbackSecret(ctx, 1, "")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_DeleteSecret(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"

		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\) WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := storage.DeleteSecret(ctx, userID, name)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		userID := int64(1)
		name := "missing"

		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\) WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := storage.DeleteSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.DeleteSecret(ctx, 0, "name")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		err := storage.DeleteSecret(ctx, 1, "")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_UpdateSecret(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &dbStorage{storage: db}
	ctx := context.Background()

	t.Run("success update data and metadata", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"
		newData := []byte("newdata")
		newSalt := []byte("newsalt")
		newIV := []byte("newiv")
		newMetadata := map[string]string{"new": "meta"}

		oldData := []byte("olddata")
		oldSalt := []byte("oldsalt")
		oldIV := []byte("oldiv")
		oldMetadata := map[string]string{"old": "meta"}
		oldRaw, _ := json.Marshal(oldMetadata)
		newRaw, _ := json.Marshal(newMetadata)

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT data, type, salt, iv, metadata, version FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL ORDER BY version DESC LIMIT 1 FOR UPDATE`).
			WithArgs(userID, name).
			WillReturnRows(sqlmock.NewRows([]string{"data", "type", "salt", "iv", "metadata", "version"}).
				AddRow(oldData, models.DataTypeCredentials, oldSalt, oldIV, oldRaw, 1))
		mock.ExpectExec(`INSERT INTO secrets \(user_id, name, type, data, salt, iv, metadata, version, created_at\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, NOW\(\)\)`).
			WithArgs(userID, name, models.DataTypeCredentials, newData, newSalt, newIV, newRaw, 2).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := storage.UpdateSecret(ctx, userID, name, newData, newSalt, newIV, newMetadata)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update metadata only", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"
		newMetadata := map[string]string{"only": "meta"}

		oldData := []byte("olddata")
		oldSalt := []byte("oldsalt")
		oldIV := []byte("oldiv")
		oldMetadata := map[string]string{"old": "meta"}
		oldRaw, _ := json.Marshal(oldMetadata)
		newRaw, _ := json.Marshal(newMetadata)

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT data, type, salt, iv, metadata, version FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL ORDER BY version DESC LIMIT 1 FOR UPDATE`).
			WithArgs(userID, name).
			WillReturnRows(sqlmock.NewRows([]string{"data", "type", "salt", "iv", "metadata", "version"}).
				AddRow(oldData, models.DataTypeCredentials, oldSalt, oldIV, oldRaw, 1))
		mock.ExpectExec(`INSERT INTO secrets \(user_id, name, type, data, salt, iv, metadata, version, created_at\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, NOW\(\)\)`).
			WithArgs(userID, name, models.DataTypeCredentials, oldData, oldSalt, oldIV, newRaw, 2).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := storage.UpdateSecret(ctx, userID, name, nil, nil, nil, newMetadata)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("secret not found", func(t *testing.T) {
		userID := int64(1)
		name := "missing"
		newData := []byte("data")
		newSalt := []byte("salt")
		newIV := []byte("iv")

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT data, type, salt, iv, metadata, version FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL ORDER BY version DESC LIMIT 1 FOR UPDATE`).
			WithArgs(userID, name).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		err := storage.UpdateSecret(ctx, userID, name, newData, newSalt, newIV, nil)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("missing salt or iv when updating data", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"
		newData := []byte("newdata")

		// No expectations on mock because validation fails before any DB call.
		err := storage.UpdateSecret(ctx, userID, name, newData, nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "both salt and iv must be provided")
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.UpdateSecret(ctx, 0, "name", []byte("d"), []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		err := storage.UpdateSecret(ctx, 1, "", []byte("d"), []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("no data to update", func(t *testing.T) {
		err := storage.UpdateSecret(ctx, 1, "name", nil, nil, nil, nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
		assert.Contains(t, err.Error(), "missing data to update")
	})
}
