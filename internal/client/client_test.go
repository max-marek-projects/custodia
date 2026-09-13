package client

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// setupTest creates a client instance with mock gRPC client, temporary storage, and session.
func setupTest(t *testing.T) (*client, *MockClient, *tokenStorage, *Session) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("APPDATA", tmpDir)

	storage := newTokenStorage("", "", logger.NewNop())
	session := &Session{}
	mockClient := NewMockClient(t)

	cache, err := newSecretCache("", logger.NewNop())
	require.NoError(t, err)

	cl := &client{
		client:  mockClient,
		storage: storage,
		session: session,
		cache:   cache,
	}
	return cl, mockClient, storage, session
}

// ---------- Auth tests ----------

func TestClient_Login(t *testing.T) {
	cl, mockClient, storage, session := setupTest(t)
	ctx := context.Background()
	login := "testuser"
	password := []byte("secret")
	sessionTTL := time.Hour

	mockClient.EXPECT().
		LoginUser(ctx, mock.MatchedBy(func(req *proto.LoginRequest) bool {
			return req.GetLogin() == login && req.GetPassword() == string(password)
		}), mock.Anything).
		Return(func() *proto.LoginResponse {
			resp := &proto.LoginResponse{}
			resp.SetUserId(42)
			resp.SetAccessToken("access")
			resp.SetRefreshToken([]byte("refresh"))
			return resp
		}(), nil)

	err := cl.Login(ctx, login, password, sessionTTL)
	require.NoError(t, err)

	assert.True(t, session.LoggedIn())
	assert.Equal(t, password, session.Password)

	tokens, err := storage.Read()
	require.NoError(t, err)
	assert.Equal(t, int64(42), tokens.UserID)
	assert.Equal(t, "access", tokens.AccessToken)
	assert.Equal(t, []byte("refresh"), tokens.RefreshToken)
}

func TestClient_Login_InvalidCredentials(t *testing.T) {
	cl, mockClient, _, _ := setupTest(t)
	ctx := context.Background()

	mockClient.EXPECT().
		LoginUser(ctx, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.PermissionDenied, "invalid credentials"))

	err := cl.Login(ctx, "user", []byte("pass"), time.Hour)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestClient_Login_InvalidArgument(t *testing.T) {
	cl, mockClient, _, _ := setupTest(t)
	ctx := context.Background()

	mockClient.EXPECT().
		LoginUser(ctx, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.InvalidArgument, "missing login"))

	err := cl.Login(ctx, "", []byte("pass"), time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid login or password")
}

func TestClient_Register(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()

	mockClient.EXPECT().
		RegisterUser(ctx, mock.Anything, mock.Anything).
		Return(func() *proto.LoginResponse {
			resp := &proto.LoginResponse{}
			resp.SetUserId(100)
			resp.SetAccessToken("access")
			resp.SetRefreshToken([]byte("refresh"))
			return resp
		}(), nil)

	err := cl.Register(ctx, "newuser", []byte("secret"), time.Hour)
	require.NoError(t, err)
	assert.True(t, session.LoggedIn())
}

func TestClient_Register_AlreadyExists(t *testing.T) {
	cl, mockClient, _, _ := setupTest(t)
	ctx := context.Background()

	mockClient.EXPECT().
		RegisterUser(ctx, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.AlreadyExists, "user exists"))

	err := cl.Register(ctx, "existing", []byte("pass"), time.Hour)
	assert.ErrorIs(t, err, ErrLoginAlreadyTaken)
}

func TestClient_RefreshAccess(t *testing.T) {
	cl, mockClient, storage, _ := setupTest(t)
	ctx := context.Background()

	existingTokens := &models.Tokens{
		UserID:       42,
		AccessToken:  "old_access",
		RefreshToken: []byte("refresh"),
	}
	err := storage.Save(existingTokens)
	require.NoError(t, err)

	mockClient.EXPECT().
		Refresh(ctx, mock.MatchedBy(func(req *proto.RefreshRequest) bool {
			return req.GetUserId() == 42 && string(req.GetRefreshToken()) == "refresh"
		}), mock.Anything).
		Return(func() *proto.RefreshResponse {
			resp := &proto.RefreshResponse{}
			resp.SetAccessToken("new_access")
			return resp
		}(), nil)

	err = cl.RefreshAccess(ctx)
	require.NoError(t, err)

	tokens, err := storage.Read()
	require.NoError(t, err)
	assert.Equal(t, "new_access", tokens.AccessToken)
	assert.Equal(t, []byte("refresh"), tokens.RefreshToken)
}

func TestClient_RefreshAccess_Expired(t *testing.T) {
	cl, mockClient, storage, _ := setupTest(t)
	ctx := context.Background()

	existingTokens := &models.Tokens{
		UserID:       42,
		AccessToken:  "old",
		RefreshToken: []byte("refresh"),
	}
	err := storage.Save(existingTokens)
	require.NoError(t, err)

	mockClient.EXPECT().
		Refresh(ctx, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.PermissionDenied, "refresh expired"))

	err = cl.RefreshAccess(ctx)
	assert.ErrorIs(t, err, ErrRefreshTokenExpired)
}

func TestClient_Logout_Device(t *testing.T) {
	cl, mockClient, storage, session := setupTest(t)
	ctx := context.Background()

	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)
	tokens := &models.Tokens{UserID: 42}
	err := storage.Save(tokens)
	require.NoError(t, err)

	mockClient.EXPECT().
		LogoutDevice(ctx, mock.Anything, mock.Anything).
		Return(&proto.LogoutDeviceResponse{}, nil)

	err = cl.Logout(ctx, false)
	require.NoError(t, err)
	assert.False(t, session.LoggedIn())
	assert.Nil(t, session.Password)

	readTokens, err := storage.Read()
	require.NoError(t, err)
	assert.Empty(t, readTokens.AccessToken)
	assert.Empty(t, readTokens.RefreshToken)
}

func TestClient_Logout_AllDevices(t *testing.T) {
	cl, mockClient, storage, session := setupTest(t)
	ctx := context.Background()

	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)
	tokens := &models.Tokens{UserID: 42}
	err := storage.Save(tokens)
	require.NoError(t, err)

	mockClient.EXPECT().
		LogoutAllDevices(ctx, mock.Anything, mock.Anything).
		Return(&proto.LogoutAllDevicesResponse{}, nil)

	err = cl.Logout(ctx, true)
	require.NoError(t, err)
	assert.False(t, session.LoggedIn())
	assert.Nil(t, session.Password)

	readTokens, err := storage.Read()
	require.NoError(t, err)
	assert.Empty(t, readTokens.AccessToken)
	assert.Empty(t, readTokens.RefreshToken)
}

// ---------- Secrets tests ----------

func TestClient_CreateCredentials(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()

	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mycreds"
	creds := &models.Credentials{Login: "user", Password: "pass"}
	metadata := map[string]string{"env": "prod"}

	mockClient.EXPECT().
		CreateSecret(ctx, mock.MatchedBy(func(req *proto.CreateSecretRequest) bool {
			return req.GetName() == name && req.GetType() == proto.DataType_DATA_TYPE_CREDENTIALS
		}), mock.Anything).
		Return(&proto.CreateSecretResponse{}, nil)

	err := cl.CreateCredentials(ctx, name, creds, metadata)
	require.NoError(t, err)
}

func TestClient_CreateCredentials_AlreadyExists(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()

	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	mockClient.EXPECT().
		CreateSecret(ctx, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.AlreadyExists, "secret exists"))

	err := cl.CreateCredentials(ctx, "mycreds", &models.Credentials{}, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSecretAlreadyExists)
}

func TestClient_CreateCredentials_SessionRequired(t *testing.T) {
	cl, _, _, _ := setupTest(t)
	ctx := context.Background()
	// session is inactive (no password)

	err := cl.CreateCredentials(ctx, "name", &models.Credentials{}, nil)
	assert.ErrorIs(t, err, ErrSessionRequired)
}

func TestClient_GetCredentials(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()

	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mycreds"
	version := uint64(0)
	creds := &models.Credentials{Login: "user", Password: "pass"}
	credsBytes, _ := json.Marshal(creds)
	salt, iv, encData, _ := utils.EncryptData(credsBytes, session.Password)

	mockClient.EXPECT().
		GetSecret(ctx, mock.MatchedBy(func(req *proto.GetSecretRequest) bool {
			return req.GetName() == name && req.GetVersion() == version && req.GetType() == proto.DataType_DATA_TYPE_CREDENTIALS
		}), mock.Anything).
		Return(func() *proto.GetSecretResponse {
			resp := &proto.GetSecretResponse{}
			resp.SetData(encData)
			resp.SetSalt(salt)
			resp.SetIv(iv)
			resp.SetMetadata(map[string]string{"env": "prod"})
			return resp
		}(), nil)

	gotCreds, gotMeta, err := cl.GetCredentials(ctx, name, version)
	require.NoError(t, err)
	assert.Equal(t, creds, gotCreds)
	assert.Equal(t, map[string]string{"env": "prod"}, gotMeta)
}

func TestClient_GetCredentials_NotFound(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()

	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	mockClient.EXPECT().
		GetSecret(ctx, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "secret not found"))

	_, _, err := cl.GetCredentials(ctx, "missing", 0)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestClient_UpdateCredentials(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()

	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mycreds"
	creds := &models.Credentials{Login: "newuser", Password: "newpass"}

	mockClient.EXPECT().
		UpdateSecretData(ctx, mock.MatchedBy(func(req *proto.UpdateSecretDataRequest) bool {
			return req.GetName() == name
		}), mock.Anything).
		Return(&proto.UpdateSecretDataResponse{}, nil)

	err := cl.UpdateCredentials(ctx, name, creds)
	require.NoError(t, err)
}

func TestClient_UpdateCredentials_NotFound(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()

	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	mockClient.EXPECT().
		UpdateSecretData(ctx, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "not found"))

	err := cl.UpdateCredentials(ctx, "missing", &models.Credentials{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

// ---------- Text secrets ----------

func TestClient_CreateSecretText(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mytext"
	text := "hello world"
	metadata := map[string]string{"key": "value"}

	mockClient.EXPECT().
		CreateSecret(ctx, mock.MatchedBy(func(req *proto.CreateSecretRequest) bool {
			return req.GetName() == name && req.GetType() == proto.DataType_DATA_TYPE_TEXT
		}), mock.Anything).
		Return(&proto.CreateSecretResponse{}, nil)

	err := cl.CreateSecretText(ctx, name, text, metadata)
	require.NoError(t, err)
}

func TestClient_GetSecretText(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mytext"
	version := uint64(0)
	text := "hello world"
	salt, iv, encData, _ := utils.EncryptData([]byte(text), session.Password)

	mockClient.EXPECT().
		GetSecret(ctx, mock.MatchedBy(func(req *proto.GetSecretRequest) bool {
			return req.GetName() == name && req.GetVersion() == version && req.GetType() == proto.DataType_DATA_TYPE_TEXT
		}), mock.Anything).
		Return(func() *proto.GetSecretResponse {
			resp := &proto.GetSecretResponse{}
			resp.SetData(encData)
			resp.SetSalt(salt)
			resp.SetIv(iv)
			resp.SetMetadata(map[string]string{"key": "value"})
			return resp
		}(), nil)

	gotText, gotMeta, err := cl.GetSecretText(ctx, name, version)
	require.NoError(t, err)
	assert.Equal(t, text, gotText)
	assert.Equal(t, map[string]string{"key": "value"}, gotMeta)
}

func TestClient_UpdateSecretText(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mytext"
	newText := "new text"

	mockClient.EXPECT().
		UpdateSecretData(ctx, mock.MatchedBy(func(req *proto.UpdateSecretDataRequest) bool {
			return req.GetName() == name
		}), mock.Anything).
		Return(&proto.UpdateSecretDataResponse{}, nil)

	err := cl.UpdateSecretText(ctx, name, newText)
	require.NoError(t, err)
}

// ---------- Binary secrets ----------

func TestClient_CreateSecretBinary(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mybinary"
	data := []byte{0x01, 0x02, 0x03}
	metadata := map[string]string{"key": "value"}

	mockClient.EXPECT().
		CreateSecret(ctx, mock.MatchedBy(func(req *proto.CreateSecretRequest) bool {
			return req.GetName() == name && req.GetType() == proto.DataType_DATA_TYPE_BINARY
		}), mock.Anything).
		Return(&proto.CreateSecretResponse{}, nil)

	err := cl.CreateSecretBinary(ctx, name, data, metadata)
	require.NoError(t, err)
}

func TestClient_GetSecretBinary(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mybinary"
	version := uint64(0)
	data := []byte{0x01, 0x02, 0x03}
	salt, iv, encData, _ := utils.EncryptData(data, session.Password)

	mockClient.EXPECT().
		GetSecret(ctx, mock.MatchedBy(func(req *proto.GetSecretRequest) bool {
			return req.GetName() == name && req.GetVersion() == version && req.GetType() == proto.DataType_DATA_TYPE_BINARY
		}), mock.Anything).
		Return(func() *proto.GetSecretResponse {
			resp := &proto.GetSecretResponse{}
			resp.SetData(encData)
			resp.SetSalt(salt)
			resp.SetIv(iv)
			resp.SetMetadata(map[string]string{"key": "value"})
			return resp
		}(), nil)

	gotData, gotMeta, err := cl.GetSecretBinary(ctx, name, version)
	require.NoError(t, err)
	assert.Equal(t, data, gotData)
	assert.Equal(t, map[string]string{"key": "value"}, gotMeta)
}

func TestClient_UpdateSecretBinary(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mybinary"
	newData := []byte{0x04, 0x05, 0x06}

	mockClient.EXPECT().
		UpdateSecretData(ctx, mock.MatchedBy(func(req *proto.UpdateSecretDataRequest) bool {
			return req.GetName() == name
		}), mock.Anything).
		Return(&proto.UpdateSecretDataResponse{}, nil)

	err := cl.UpdateSecretBinary(ctx, name, newData)
	require.NoError(t, err)
}

// ---------- Card data ----------

func TestClient_CreateCardData(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mycard"
	card := &models.CardData{Number: "1234", ExpirationDate: "12/25"}
	metadata := map[string]string{"key": "value"}

	mockClient.EXPECT().
		CreateSecret(ctx, mock.MatchedBy(func(req *proto.CreateSecretRequest) bool {
			return req.GetName() == name && req.GetType() == proto.DataType_DATA_TYPE_CARD
		}), mock.Anything).
		Return(&proto.CreateSecretResponse{}, nil)

	err := cl.CreateCardData(ctx, name, card, metadata)
	require.NoError(t, err)
}

func TestClient_GetCardData(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mycard"
	version := uint64(0)
	card := &models.CardData{Number: "1234", ExpirationDate: "12/25"}
	cardBytes, _ := json.Marshal(card)
	salt, iv, encData, _ := utils.EncryptData(cardBytes, session.Password)

	mockClient.EXPECT().
		GetSecret(ctx, mock.MatchedBy(func(req *proto.GetSecretRequest) bool {
			return req.GetName() == name && req.GetVersion() == version && req.GetType() == proto.DataType_DATA_TYPE_CARD
		}), mock.Anything).
		Return(func() *proto.GetSecretResponse {
			resp := &proto.GetSecretResponse{}
			resp.SetData(encData)
			resp.SetSalt(salt)
			resp.SetIv(iv)
			resp.SetMetadata(map[string]string{"key": "value"})
			return resp
		}(), nil)

	gotCard, gotMeta, err := cl.GetCardData(ctx, name, version)
	require.NoError(t, err)
	assert.Equal(t, card, gotCard)
	assert.Equal(t, map[string]string{"key": "value"}, gotMeta)
}

func TestClient_UpdateCardData(t *testing.T) {
	cl, mockClient, _, session := setupTest(t)
	ctx := context.Background()
	session.Password = []byte("secret")
	session.ExpiresAt = time.Now().Add(time.Hour)

	name := "mycard"
	newCard := &models.CardData{Number: "5678", ExpirationDate: "01/26"}

	mockClient.EXPECT().
		UpdateSecretData(ctx, mock.MatchedBy(func(req *proto.UpdateSecretDataRequest) bool {
			return req.GetName() == name
		}), mock.Anything).
		Return(&proto.UpdateSecretDataResponse{}, nil)

	err := cl.UpdateCardData(ctx, name, newCard)
	require.NoError(t, err)
}

// ---------- Mutual operations ----------

func TestClient_RollbackSecret(t *testing.T) {
	cl, mockClient, _, _ := setupTest(t)
	ctx := context.Background()

	name := "mysecret"

	mockClient.EXPECT().
		RollbackSecret(ctx, mock.MatchedBy(func(req *proto.RollbackSecretRequest) bool {
			return req.GetName() == name
		}), mock.Anything).
		Return(&proto.RollbackSecretResponse{}, nil)

	err := cl.RollbackSecret(ctx, name)
	require.NoError(t, err)
}

func TestClient_RollbackSecret_NotFound(t *testing.T) {
	cl, mockClient, _, _ := setupTest(t)
	ctx := context.Background()

	mockClient.EXPECT().
		RollbackSecret(ctx, mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "not found"))

	err := cl.RollbackSecret(ctx, "missing")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestClient_DeleteSecret(t *testing.T) {
	cl, mockClient, _, _ := setupTest(t)
	ctx := context.Background()

	name := "mysecret"

	mockClient.EXPECT().
		DeleteSecret(ctx, mock.MatchedBy(func(req *proto.DeleteSecretRequest) bool {
			return req.GetName() == name
		}), mock.Anything).
		Return(&proto.DeleteSecretResponse{}, nil)

	err := cl.DeleteSecret(ctx, name)
	require.NoError(t, err)
}

func TestClient_UpdateMetadata(t *testing.T) {
	cl, mockClient, _, _ := setupTest(t)
	ctx := context.Background()

	name := "mysecret"
	metadata := map[string]string{"new": "meta"}

	mockClient.EXPECT().
		UpdateSecretMetadata(ctx, mock.MatchedBy(func(req *proto.UpdateSecretMetadataRequest) bool {
			return req.GetName() == name && req.GetMetadata()["new"] == "meta"
		}), mock.Anything).
		Return(&proto.UpdateSecretMetadataResponse{}, nil)

	err := cl.UpdateMetadata(ctx, name, metadata)
	require.NoError(t, err)
}

func TestClient_ListSecrets(t *testing.T) {
	t.Run("success without filter", func(t *testing.T) {
		cl, mockClient, _, session := setupTest(t)
		ctx := context.Background()

		session.Password = []byte("secret")
		session.ExpiresAt = time.Now().Add(time.Hour)

		// Prepare expected response.
		info1 := &proto.SecretInfo{}
		info1.SetName("login1")
		info1.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
		info1.SetMetadata(map[string]string{"env": "prod"})
		info1.SetLatestVersion(3)

		info2 := &proto.SecretInfo{}
		info2.SetName("note1")
		info2.SetType(proto.DataType_DATA_TYPE_TEXT)
		info2.SetMetadata(map[string]string{})
		info2.SetLatestVersion(1)

		resp := &proto.ListSecretsResponse{}
		resp.SetSecrets([]*proto.SecretInfo{info1, info2})

		mockClient.EXPECT().
			ListSecrets(ctx, mock.MatchedBy(func(req *proto.ListSecretsRequest) bool {
				return len(req.GetMetadata()) == 0
			})).
			Return(resp, nil)

		result, err := cl.ListSecrets(ctx, nil)
		require.NoError(t, err)
		require.Len(t, result, 2)

		assert.Equal(t, "login1", result[0].Name)
		assert.Equal(t, models.DataTypeCredentials, result[0].Type)
		assert.Equal(t, map[string]string{"env": "prod"}, result[0].Metadata)
		assert.Equal(t, uint64(3), result[0].LatestVersion)

		assert.Equal(t, "note1", result[1].Name)
		assert.Equal(t, models.DataTypeText, result[1].Type)
		assert.Equal(t, uint64(1), result[1].LatestVersion)

		mockClient.AssertExpectations(t)
	})

	t.Run("success with metadata filter", func(t *testing.T) {
		cl, mockClient, _, session := setupTest(t)
		ctx := context.Background()

		session.Password = []byte("secret")
		session.ExpiresAt = time.Now().Add(time.Hour)

		info := &proto.SecretInfo{}
		info.SetName("login1")
		info.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
		info.SetMetadata(map[string]string{"env": "prod"})
		info.SetLatestVersion(2)

		resp := &proto.ListSecretsResponse{}
		resp.SetSecrets([]*proto.SecretInfo{info})

		filter := map[string]string{"env": "prod"}

		mockClient.EXPECT().
			ListSecrets(ctx, mock.MatchedBy(func(req *proto.ListSecretsRequest) bool {
				return req.GetMetadata()["env"] == "prod"
			})).
			Return(resp, nil)

		result, err := cl.ListSecrets(ctx, filter)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "login1", result[0].Name)
		assert.Equal(t, uint64(2), result[0].LatestVersion)

		mockClient.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		cl, mockClient, _, session := setupTest(t)
		ctx := context.Background()

		session.Password = []byte("secret")
		session.ExpiresAt = time.Now().Add(time.Hour)

		mockClient.EXPECT().
			ListSecrets(ctx, mock.Anything).
			Return(&proto.ListSecretsResponse{}, nil)

		result, err := cl.ListSecrets(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("invalid argument from server", func(t *testing.T) {
		cl, mockClient, _, session := setupTest(t)
		ctx := context.Background()

		session.Password = []byte("secret")
		session.ExpiresAt = time.Now().Add(time.Hour)

		mockClient.EXPECT().
			ListSecrets(ctx, mock.Anything).
			Return(nil, status.Error(codes.InvalidArgument, "invalid filter"))

		_, err := cl.ListSecrets(ctx, map[string]string{"bad": "filter"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid filter")
	})

	t.Run("internal error from server", func(t *testing.T) {
		cl, mockClient, _, session := setupTest(t)
		ctx := context.Background()

		session.Password = []byte("secret")
		session.ExpiresAt = time.Now().Add(time.Hour)

		mockClient.EXPECT().
			ListSecrets(ctx, mock.Anything).
			Return(nil, status.Error(codes.Internal, "db down"))

		_, err := cl.ListSecrets(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list secrets")
	})

	t.Run("unknown data type in response", func(t *testing.T) {
		cl, mockClient, _, session := setupTest(t)
		ctx := context.Background()

		session.Password = []byte("secret")
		session.ExpiresAt = time.Now().Add(time.Hour)

		// Unknown data type (999).
		info := &proto.SecretInfo{}
		info.SetName("weird")
		info.SetType(proto.DataType(999))
		info.SetLatestVersion(1)

		resp := &proto.ListSecretsResponse{}
		resp.SetSecrets([]*proto.SecretInfo{info})

		mockClient.EXPECT().
			ListSecrets(ctx, mock.Anything).
			Return(resp, nil)

		_, err := cl.ListSecrets(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown data type")
	})
}

// ---------- Error cases: no active session ----------

func TestClient_OperationsRequireSession(t *testing.T) {
	cl, _, _, _ := setupTest(t)
	ctx := context.Background()
	// Session is inactive (password is empty)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"CreateCredentials", func() error { return cl.CreateCredentials(ctx, "name", &models.Credentials{}, nil) }},
		{"GetCredentials", func() error { _, _, err := cl.GetCredentials(ctx, "name", 0); return err }},
		{"UpdateCredentials", func() error { return cl.UpdateCredentials(ctx, "name", &models.Credentials{}) }},
		{"CreateSecretText", func() error { return cl.CreateSecretText(ctx, "name", "text", nil) }},
		{"GetSecretText", func() error { _, _, err := cl.GetSecretText(ctx, "name", 0); return err }},
		{"UpdateSecretText", func() error { return cl.UpdateSecretText(ctx, "name", "text") }},
		{"CreateSecretBinary", func() error { return cl.CreateSecretBinary(ctx, "name", []byte{1}, nil) }},
		{"GetSecretBinary", func() error { _, _, err := cl.GetSecretBinary(ctx, "name", 0); return err }},
		{"UpdateSecretBinary", func() error { return cl.UpdateSecretBinary(ctx, "name", []byte{1}) }},
		{"CreateCardData", func() error { return cl.CreateCardData(ctx, "name", &models.CardData{}, nil) }},
		{"GetCardData", func() error { _, _, err := cl.GetCardData(ctx, "name", 0); return err }},
		{"UpdateCardData", func() error { return cl.UpdateCardData(ctx, "name", &models.CardData{}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			assert.ErrorIs(t, err, ErrSessionRequired)
		})
	}
}
