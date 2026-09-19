package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errConnDone is a generic connection error used in tests instead of sql.ErrConnDone,
// because pgx does not have an exact equivalent of database/sql's ErrConnDone.
var errConnDone = errors.New("connection is done")

func newMockStorage(t *testing.T) (*dbStorage, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return &dbStorage{storage: mock}, mock
}

// ========== USERS ==========

func TestDBStorage_RegisterUser(t *testing.T) {
	storage, mock := newMockStorage(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userData := models.UserData{Login: "testuser", PasswordHash: []byte("hash")}
		expectedID := int64(42)

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(userData.Login, userData.PasswordHash).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(expectedID))

		id, err := storage.RegisterUser(ctx, userData)
		assert.NoError(t, err)
		assert.Equal(t, expectedID, id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("conflict", func(t *testing.T) {
		userData := models.UserData{Login: "existing", PasswordHash: []byte("hash")}

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(userData.Login, userData.PasswordHash).
			WillReturnError(pgx.ErrNoRows)

		_, err := storage.RegisterUser(ctx, userData)
		assert.ErrorIs(t, err, ErrAlreadyInStorage)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		userData := models.UserData{Login: "error", PasswordHash: []byte("hash")}

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(userData.Login, userData.PasswordHash).
			WillReturnError(errConnDone)

		_, err := storage.RegisterUser(ctx, userData)
		assert.ErrorIs(t, err, errConnDone)
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
	storage, mock := newMockStorage(t)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		username := "testuser"
		expectedID := int64(42)
		expectedHash := []byte("hashedpass")

		mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE username = \$1`).
			WithArgs(username).
			WillReturnRows(pgxmock.NewRows([]string{"id", "password_hash"}).
				AddRow(expectedID, expectedHash))

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
			WillReturnError(pgx.ErrNoRows)

		_, _, err := storage.CheckUser(ctx, username)
		assert.ErrorIs(t, err, ErrUserNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		username := "error"

		mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE username = \$1`).
			WithArgs(username).
			WillReturnError(errConnDone)

		_, _, err := storage.CheckUser(ctx, username)
		assert.ErrorIs(t, err, errConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty username", func(t *testing.T) {
		_, _, err := storage.CheckUser(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidArgument)
		assert.Contains(t, err.Error(), "empty username")
	})
}

// ========== TOKENS ==========

func TestDBStorage_SaveRefreshToken(t *testing.T) {
	storage, mock := newMockStorage(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		tokenHash := []byte("hash")
		device := "device"
		ttl := 10 * time.Minute

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO tokens \(user_id, token_hash, expires_at, device_name\) VALUES \(\$1, \$2, \$3, \$4\)`).
			WithArgs(userID, tokenHash, pgxmock.AnyArg(), device).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
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
			WillReturnError(errConnDone)
		mock.ExpectRollback()

		err := storage.SaveRefreshToken(ctx, userID, tokenHash, device, ttl)
		assert.ErrorIs(t, err, errConnDone)
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
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO tokens \(user_id, token_hash, expires_at, device_name\) VALUES \(\$1, \$2, \$3, \$4\)`).
			WithArgs(userID, tokenHash, pgxmock.AnyArg(), device).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit().WillReturnError(pgx.ErrTxClosed)

		err := storage.SaveRefreshToken(ctx, userID, tokenHash, device, ttl)
		assert.ErrorIs(t, err, pgx.ErrTxClosed)
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
	storage, mock := newMockStorage(t)
	ctx := context.Background()

	t.Run("valid token", func(t *testing.T) {
		userID := int64(1)
		tokenHash := []byte("hash")
		device := "device"

		mock.ExpectQuery(`SELECT 1 FROM tokens WHERE user_id = \$1 AND token_hash = \$2 AND device_name = \$3 AND revoked_at IS NULL AND expires_at > NOW\(\)`).
			WithArgs(userID, tokenHash, device).
			WillReturnRows(pgxmock.NewRows([]string{"1"}).AddRow(1))

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
			WillReturnError(pgx.ErrNoRows)

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
	storage, mock := newMockStorage(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		device := "device"

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := storage.RevokeToken(ctx, userID, device)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		userID := int64(1)
		device := "device"

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnError(errConnDone)

		err := storage.RevokeToken(ctx, userID, device)
		assert.ErrorIs(t, err, errConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows affected", func(t *testing.T) {
		userID := int64(1)
		device := "device"

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND device_name = \$2 AND revoked_at IS NULL`).
			WithArgs(userID, device).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

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
	storage, mock := newMockStorage(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND revoked_at IS NULL`).
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 3))

		err := storage.RevokeAllTokens(ctx, userID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND revoked_at IS NULL`).
			WithArgs(userID).
			WillReturnError(errConnDone)

		err := storage.RevokeAllTokens(ctx, userID)
		assert.ErrorIs(t, err, errConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows affected", func(t *testing.T) {
		userID := int64(1)

		mock.ExpectExec(`UPDATE tokens SET revoked_at = NOW\(\) WHERE user_id = \$1 AND revoked_at IS NULL`).
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := storage.RevokeAllTokens(ctx, userID)
		assert.ErrorIs(t, err, ErrNoChanges)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		err := storage.RevokeAllTokens(ctx, 0)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

// ========== SECRETS ==========

func TestDBStorage_CreateSecret(t *testing.T) {
	ctx := context.Background()

	userID := int64(1)
	dataType := models.DataTypeCredentials
	name := "mysecret"
	data := []byte("data")
	salt := []byte("salt")
	iv := []byte("iv")
	metadata := map[string]string{"key": "value"}
	rawMetadata, _ := json.Marshal(metadata)

	t.Run("success", func(t *testing.T) {
		storage, mock := newMockStorage(t)
		mock.ExpectExec(`INSERT INTO secrets \(user_id, type, name, data, salt, iv, metadata\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)`).
			WithArgs(userID, dataType, name, data, salt, iv, rawMetadata).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := storage.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		storage, mock := newMockStorage(t)
		mock.ExpectExec(`INSERT INTO secrets`).
			WithArgs(userID, dataType, name, data, salt, iv, rawMetadata).
			WillReturnError(errConnDone)

		err := storage.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.ErrorIs(t, err, errConnDone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("conflict", func(t *testing.T) {
		storage, mock := newMockStorage(t)
		mock.ExpectExec(`INSERT INTO secrets \(user_id, type, name, data, salt, iv, metadata\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)`).
			WithArgs(userID, dataType, name, data, salt, iv, rawMetadata).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		err := storage.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.ErrorIs(t, err, ErrAlreadyInStorage)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.CreateSecret(ctx, 0, models.DataTypeCredentials, "name", []byte("d"), []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.CreateSecret(ctx, 1, models.DataTypeCredentials, "", []byte("d"), []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty data", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.CreateSecret(ctx, 1, models.DataTypeCredentials, "name", []byte{}, []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty salt", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.CreateSecret(ctx, 1, models.DataTypeCredentials, "name", []byte("d"), []byte{}, []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty iv", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.CreateSecret(ctx, 1, models.DataTypeCredentials, "name", []byte("d"), []byte("s"), []byte{}, nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_GetSecret(t *testing.T) {
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
		storage, mock := newMockStorage(t)
		mock.ExpectQuery(`SELECT data, salt, iv, metadata FROM secrets WHERE user_id = \$1 AND type = \$2 AND name = \$3 AND \(version = \$4 OR \$4 = 0\) AND deleted_at IS NULL ORDER BY version DESC LIMIT 1`).
			WithArgs(userID, dataType, name, version).
			WillReturnRows(pgxmock.NewRows([]string{"data", "salt", "iv", "metadata"}).
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
		storage, mock := newMockStorage(t)
		mock.ExpectQuery(`SELECT data, salt, iv, metadata FROM secrets WHERE user_id = \$1 AND type = \$2 AND name = \$3 AND \(version = \$4 OR \$4 = 0\) AND deleted_at IS NULL ORDER BY version DESC LIMIT 1`).
			WithArgs(userID, dataType, name, version).
			WillReturnError(pgx.ErrNoRows)

		_, _, _, _, err := storage.GetSecret(ctx, userID, dataType, name, version)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		_, _, _, _, err := storage.GetSecret(ctx, 0, models.DataTypeCredentials, "name", 0)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		_, _, _, _, err := storage.GetSecret(ctx, 1, models.DataTypeCredentials, "", 0)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_RollbackSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"
		storage, mock := newMockStorage(t)

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT COALESCE\(MAX\(version\), 0\)`).
			WithArgs(userID, name).
			WillReturnRows(pgxmock.NewRows([]string{"coalesce"}).AddRow(int64(3)))
		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\)`).
			WithArgs(userID, name, int64(3)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		err := storage.RollbackSecret(ctx, userID, name)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not enough versions", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"
		storage, mock := newMockStorage(t)

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT COALESCE\(MAX\(version\), 0\)`).
			WithArgs(userID, name).
			WillReturnRows(pgxmock.NewRows([]string{"coalesce"}).AddRow(int64(1)))
		mock.ExpectRollback()

		err := storage.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretRollbackNotPossible)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no active versions returns ErrSecretNotFound", func(t *testing.T) {
		userID := int64(1)
		name := "missing"
		storage, mock := newMockStorage(t)

		mock.ExpectBegin()
		// With COALESCE, an empty result set yields 0, not NULL.
		mock.ExpectQuery(`SELECT COALESCE\(MAX\(version\), 0\)`).
			WithArgs(userID, name).
			WillReturnRows(pgxmock.NewRows([]string{"coalesce"}).AddRow(int64(0)))
		mock.ExpectRollback()

		err := storage.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update affects zero rows returns ErrSecretNotFound", func(t *testing.T) {
		userID := int64(1)
		name := "race"
		storage, mock := newMockStorage(t)

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT COALESCE\(MAX\(version\), 0\)`).
			WithArgs(userID, name).
			WillReturnRows(pgxmock.NewRows([]string{"coalesce"}).AddRow(int64(2)))
		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\)`).
			WithArgs(userID, name, int64(2)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectRollback()

		err := storage.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDBStorage_DeleteSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		name := "mysecret"
		storage, mock := newMockStorage(t)
		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\) WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := storage.DeleteSecret(ctx, userID, name)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		userID := int64(1)
		name := "missing"
		storage, mock := newMockStorage(t)
		mock.ExpectExec(`UPDATE secrets SET deleted_at = NOW\(\) WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL`).
			WithArgs(userID, name).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := storage.DeleteSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty user id", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.DeleteSecret(ctx, 0, "name")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.DeleteSecret(ctx, 1, "")
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_UpdateSecret(t *testing.T) {
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
		storage, mock := newMockStorage(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT data, type, salt, iv, metadata, version FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL ORDER BY version DESC LIMIT 1 FOR UPDATE`).
			WithArgs(userID, name).
			WillReturnRows(pgxmock.NewRows([]string{"data", "type", "salt", "iv", "metadata", "version"}).
				AddRow(oldData, models.DataTypeCredentials, oldSalt, oldIV, oldRaw, 1))
		mock.ExpectExec(`INSERT INTO secrets \(user_id, name, type, data, salt, iv, metadata, version, created_at\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, NOW\(\)\)`).
			WithArgs(userID, name, models.DataTypeCredentials, newData, newSalt, newIV, newRaw, int64(2)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
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
		storage, mock := newMockStorage(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT data, type, salt, iv, metadata, version FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL ORDER BY version DESC LIMIT 1 FOR UPDATE`).
			WithArgs(userID, name).
			WillReturnRows(pgxmock.NewRows([]string{"data", "type", "salt", "iv", "metadata", "version"}).
				AddRow(oldData, models.DataTypeCredentials, oldSalt, oldIV, oldRaw, 1))
		mock.ExpectExec(`INSERT INTO secrets \(user_id, name, type, data, salt, iv, metadata, version, created_at\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, NOW\(\)\)`).
			WithArgs(userID, name, models.DataTypeCredentials, oldData, oldSalt, oldIV, newRaw, int64(2)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
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
		storage, mock := newMockStorage(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT data, type, salt, iv, metadata, version FROM secrets WHERE user_id = \$1 AND name = \$2 AND deleted_at IS NULL ORDER BY version DESC LIMIT 1 FOR UPDATE`).
			WithArgs(userID, name).
			WillReturnError(pgx.ErrNoRows)
		mock.ExpectRollback()

		err := storage.UpdateSecret(ctx, userID, name, newData, newSalt, newIV, nil)
		assert.ErrorIs(t, err, ErrSecretNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("missing salt or iv when updating data", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.UpdateSecret(ctx, 1, "mysecret", []byte("newdata"), nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "both salt and iv must be provided")
	})

	t.Run("empty user id", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.UpdateSecret(ctx, 0, "name", []byte("d"), []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("empty name", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.UpdateSecret(ctx, 1, "", []byte("d"), []byte("s"), []byte("i"), nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
	t.Run("no data to update", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		err := storage.UpdateSecret(ctx, 1, "name", nil, nil, nil, nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
		assert.Contains(t, err.Error(), "missing data to update")
	})
}

func TestDBStorage_ListSecrets(t *testing.T) {
	ctx := context.Background()

	t.Run("no filter returns all", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "type", "metadata", "version"}).
			AddRow("login1", "credentials", []byte(`{"env":"prod"}`), uint64(3)).
			AddRow("note1", "text", []byte(`{}`), uint64(1))
		storage, mock := newMockStorage(t)
		mock.ExpectQuery(`SELECT DISTINCT ON \(name, type\) name, type, metadata, version`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		result, err := storage.ListSecrets(ctx, 1, nil)
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "login1", result[0].Name)
		assert.Equal(t, models.DataTypeCredentials, result[0].Type)
		assert.Equal(t, uint64(3), result[0].LatestVersion)
		assert.Equal(t, map[string]string{"env": "prod"}, result[0].Metadata)
	})

	t.Run("with filter uses @> operator", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"name", "type", "metadata", "version"}).
			AddRow("login1", "credentials", []byte(`{"env":"prod"}`), 2)
		storage, mock := newMockStorage(t)
		mock.ExpectQuery(`metadata @> \$2::jsonb`).
			WithArgs(int64(1), []byte(`{"env":"prod"}`)).
			WillReturnRows(rows)

		result, err := storage.ListSecrets(ctx, 1, map[string]string{"env": "prod"})
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "login1", result[0].Name)
	})

	t.Run("empty user id", func(t *testing.T) {
		storage, _ := newMockStorage(t)
		_, err := storage.ListSecrets(ctx, 0, nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestDBStorage_Close(t *testing.T) {
	storage, mock := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectClose()
	err := storage.Close(ctx)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
