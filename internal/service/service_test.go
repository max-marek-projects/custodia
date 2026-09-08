package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const testSecretKey = "test-secret-key"

func TestService_RegisterUser(t *testing.T) {
	ctx := context.Background()
	loginReq := &models.LoginRequest{
		Login:      "testuser",
		Password:   "password",
		DeviceName: "device",
	}

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)

		mockStorage.EXPECT().
			RegisterUser(ctx, mock.MatchedBy(func(data models.UserData) bool {
				return data.Login == loginReq.Login && len(data.PasswordHash) > 0
			})).
			Return(int64(1), nil)

		mockStorage.EXPECT().
			SaveRefreshToken(ctx, int64(1), mock.Anything, "device", mock.Anything).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		resp, err := svc.RegisterUser(ctx, loginReq)
		require.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("invalid argument from storage", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RegisterUser(ctx, mock.Anything).
			Return(int64(0), repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.RegisterUser(ctx, loginReq)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})

	t.Run("login already taken", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RegisterUser(ctx, mock.Anything).
			Return(int64(0), repository.ErrAlreadyInStorage)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.RegisterUser(ctx, loginReq)
		assert.ErrorIs(t, err, ErrLoginAlreadyTaken)
	})

	t.Run("storage error", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RegisterUser(ctx, mock.Anything).
			Return(int64(0), errors.New("db error"))

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.RegisterUser(ctx, loginReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register user")
	})

	t.Run("nil userData", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.RegisterUser(ctx, nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestService_LoginUser(t *testing.T) {
	ctx := context.Background()
	loginReq := &models.LoginRequest{
		Login:      "testuser",
		Password:   "password",
		DeviceName: "device",
	}
	hashed, _ := bcrypt.GenerateFromPassword([]byte(loginReq.Password), bcrypt.DefaultCost)

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CheckUser(ctx, loginReq.Login).
			Return(int64(1), hashed, nil)
		mockStorage.EXPECT().
			SaveRefreshToken(ctx, int64(1), mock.Anything, "device", mock.Anything).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		resp, err := svc.LoginUser(ctx, loginReq)
		require.NoError(t, err)
		assert.Equal(t, int64(1), resp.UserID)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("wrong password", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CheckUser(ctx, loginReq.Login).
			Return(int64(1), hashed, nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		wrongReq := &models.LoginRequest{Login: loginReq.Login, Password: "wrong", DeviceName: "device"}
		_, err := svc.LoginUser(ctx, wrongReq)
		assert.ErrorIs(t, err, ErrWrongUsernamePassword)
	})

	t.Run("user not found", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CheckUser(ctx, loginReq.Login).
			Return(int64(0), nil, repository.ErrUserNotFound)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.LoginUser(ctx, loginReq)
		assert.ErrorIs(t, err, ErrWrongUsernamePassword)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CheckUser(ctx, loginReq.Login).
			Return(int64(0), nil, repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.LoginUser(ctx, loginReq)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})

	t.Run("nil userData", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.LoginUser(ctx, nil)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestService_RefreshAccess(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	refreshToken := []byte("valid_refresh")
	device := "device"

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CheckRefreshToken(ctx, userID, refreshToken, device).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		token, err := svc.RefreshAccess(ctx, userID, refreshToken, device)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("invalid token", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CheckRefreshToken(ctx, userID, refreshToken, device).
			Return(repository.ErrRefreshTokenExpiredOrInvalid)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.RefreshAccess(ctx, userID, refreshToken, device)
		assert.ErrorIs(t, err, ErrRefreshTokenExpiredOrInvalid)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CheckRefreshToken(ctx, userID, refreshToken, device).
			Return(repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.RefreshAccess(ctx, userID, refreshToken, device)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})

	t.Run("empty refresh token", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, err := svc.RefreshAccess(ctx, userID, nil, device)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty refresh token")
	})
}

func TestService_Logout(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	device := "device"

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RevokeToken(ctx, userID, device).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.Logout(ctx, userID, device)
		assert.NoError(t, err)
	})

	t.Run("no changes", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RevokeToken(ctx, userID, device).
			Return(repository.ErrNoChanges)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.Logout(ctx, userID, device)
		assert.ErrorIs(t, err, ErrNoChanges)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RevokeToken(ctx, userID, device).
			Return(repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.Logout(ctx, userID, device)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestService_LogoutAllDevices(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RevokeAllTokens(ctx, userID).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.LogoutAllDevices(ctx, userID)
		assert.NoError(t, err)
	})

	t.Run("no changes", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RevokeAllTokens(ctx, userID).
			Return(repository.ErrNoChanges)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.LogoutAllDevices(ctx, userID)
		assert.ErrorIs(t, err, ErrNoChanges)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RevokeAllTokens(ctx, userID).
			Return(repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.LogoutAllDevices(ctx, userID)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

// ---------- Secrets Tests ----------

func TestService_CreateSecret(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	dataType := models.DataTypeCredentials
	name := "mysecret"
	data := []byte("data")
	salt := []byte("salt")
	iv := []byte("iv")
	metadata := map[string]string{"key": "value"}

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.NoError(t, err)
	})

	t.Run("conflict", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata).
			Return(repository.ErrAlreadyInStorage)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.ErrorIs(t, err, ErrSecretConflict)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata).
			Return(repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.CreateSecret(ctx, userID, dataType, name, data, salt, iv, metadata)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestService_GetSecret(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	dataType := models.DataTypeCredentials
	name := "mysecret"
	version := uint64(0)
	expectedData := []byte("data")
	expectedSalt := []byte("salt")
	expectedIv := []byte("iv")
	expectedMetadata := map[string]string{"key": "value"}

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			GetSecret(ctx, userID, dataType, name, version).
			Return(expectedData, expectedSalt, expectedIv, expectedMetadata, nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		data, salt, iv, metadata, err := svc.GetSecret(ctx, userID, dataType, name, version)
		require.NoError(t, err)
		assert.Equal(t, expectedData, data)
		assert.Equal(t, expectedSalt, salt)
		assert.Equal(t, expectedIv, iv)
		assert.Equal(t, expectedMetadata, metadata)
	})

	t.Run("not found", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			GetSecret(ctx, userID, dataType, name, version).
			Return(nil, nil, nil, nil, repository.ErrSecretNotFound)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, _, _, _, err := svc.GetSecret(ctx, userID, dataType, name, version)
		assert.ErrorIs(t, err, ErrSecretNotFound)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			GetSecret(ctx, userID, dataType, name, version).
			Return(nil, nil, nil, nil, repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		_, _, _, _, err := svc.GetSecret(ctx, userID, dataType, name, version)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestService_RollbackSecret(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	name := "mysecret"

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RollbackSecret(ctx, userID, name).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.RollbackSecret(ctx, userID, name)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RollbackSecret(ctx, userID, name).
			Return(repository.ErrSecretNotFound)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretNotFound)
	})

	t.Run("rollback not possible", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RollbackSecret(ctx, userID, name).
			Return(repository.ErrSecretRollbackNotPossible)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrRollbackNotPossible)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			RollbackSecret(ctx, userID, name).
			Return(repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.RollbackSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestService_DeleteSecret(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	name := "mysecret"

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			DeleteSecret(ctx, userID, name).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.DeleteSecret(ctx, userID, name)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			DeleteSecret(ctx, userID, name).
			Return(repository.ErrSecretNotFound)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.DeleteSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrSecretNotFound)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			DeleteSecret(ctx, userID, name).
			Return(repository.ErrInvalidArgument)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.DeleteSecret(ctx, userID, name)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	})
}

func TestService_UpdateSecretData(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	name := "mysecret"
	data := []byte("newdata")
	salt := []byte("newsalt")
	iv := []byte("newiv")

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			UpdateSecret(ctx, userID, name, data, salt, iv, mock.Anything).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.UpdateSecretData(ctx, userID, name, data, salt, iv)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			UpdateSecret(ctx, userID, name, data, salt, iv, mock.Anything).
			Return(repository.ErrSecretNotFound)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.UpdateSecretData(ctx, userID, name, data, salt, iv)
		assert.ErrorIs(t, err, ErrSecretNotFound)
	})
}

func TestService_UpdateSecretMetadata(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	name := "mysecret"
	metadata := map[string]string{"new": "meta"}

	t.Run("success", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			UpdateSecret(ctx, userID, name, mock.Anything, mock.Anything, mock.Anything, metadata).
			Return(nil)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.UpdateSecretMetadata(ctx, userID, name, metadata)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockStorage := NewMockStorage(t)
		mockStorage.EXPECT().
			UpdateSecret(ctx, userID, name, mock.Anything, mock.Anything, mock.Anything, metadata).
			Return(repository.ErrSecretNotFound)

		svc := NewService(mockStorage, testSecretKey, time.Hour, 24*time.Hour)
		err := svc.UpdateSecretMetadata(ctx, userID, name, metadata)
		assert.ErrorIs(t, err, ErrSecretNotFound)
	})
}
