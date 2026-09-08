package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/max-marek-projects/custodia/internal/config"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/utils"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

//go:generate mockery --name=CustodiaClient --srcpkg=github.com/max-marek-projects/custodia/pkg/proto --output=. --outpkg=client --filename=mock_client.gen_test.go --with-expecter --structname=MockClient

// Handler handles grpc endpoints for URL shortening and redirection.
type client struct {
	connection *grpc.ClientConn
	client     proto.CustodiaClient
	storage    *tokenStorage
	session    *Session
}

// NewHandler creates a new GRPCHandler.
func NewClient(session *Session) (*client, *config.ClientConf, error) {
	configuration, err := config.NewClientConf()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize configuration: %w", err)
	}
	err = logger.Initialize(configuration.LoggerLevel)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	clientItem := &client{storage: newTokenStorage(configuration.ConfigFolder, configuration.TokenFilename), session: session}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := grpc.NewClient(
		configuration.ServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithChainUnaryInterceptor(clientItem.authInterceptor()),
	)
	if err != nil {
		return nil, nil, err
	}
	clientItem.connection = conn
	clientItem.client = proto.NewCustodiaClient(conn)
	return clientItem, configuration, nil
}

func (c *client) getDeviceName() (string, error) {
	return os.Hostname()
}

func (c *client) Close() error {
	return c.connection.Close()
}

// ========== AUTH ==========

func (c *client) Login(ctx context.Context, login string, password []byte, sessionTTL time.Duration) error {
	request := &proto.LoginRequest{}
	request.SetLogin(login)
	request.SetPassword(string(password))
	deviceName, err := c.getDeviceName()
	if err != nil {
		return err
	}
	request.SetDeviceName(deviceName)
	resp, err := c.client.LoginUser(ctx, request)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return fmt.Errorf("invalid login or password: %s", st.Message())
			case codes.PermissionDenied:
				return ErrInvalidCredentials
			case codes.Unauthenticated:
				return ErrInvalidCredentials
			default:
				return fmt.Errorf("login failed: %w", err)
			}
		}
		return fmt.Errorf("login failed: %w", err)
	}
	c.session.ExpiresAt = time.Now().Add(sessionTTL)
	c.session.Password = password
	return c.storage.Save(&models.Tokens{
		UserID:       resp.GetUserId(),
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

func (c *client) Register(ctx context.Context, login string, password []byte, sessionTTL time.Duration) error {
	request := &proto.LoginRequest{}
	request.SetLogin(login)
	request.SetPassword(string(password))
	deviceName, err := c.getDeviceName()
	if err != nil {
		return err
	}
	request.SetDeviceName(deviceName)
	resp, err := c.client.RegisterUser(ctx, request)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
			return ErrLoginAlreadyTaken
		}
		return fmt.Errorf("registration failed: %w", err)
	}
	c.session.ExpiresAt = time.Now().Add(sessionTTL)
	c.session.Password = password
	return c.storage.Save(&models.Tokens{
		UserID:       resp.GetUserId(),
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
	})
}

func (c *client) RefreshAccess(
	ctx context.Context,
) error {
	request := &proto.RefreshRequest{}
	tokenData, err := c.storage.Read()
	if err != nil {
		return err
	}
	request.SetUserId(tokenData.UserID)
	request.SetRefreshToken(tokenData.RefreshToken)
	deviceName, err := c.getDeviceName()
	if err != nil {
		return err
	}
	request.SetDeviceName(deviceName)
	resp, err := c.client.Refresh(ctx, request)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.PermissionDenied {
			return ErrRefreshTokenExpired
		}
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	return c.storage.Save(
		&models.Tokens{
			UserID:       tokenData.UserID,
			AccessToken:  resp.GetAccessToken(),
			RefreshToken: tokenData.RefreshToken,
		})
}

func (c *client) Logout(
	ctx context.Context,
	allDevices bool,
) error {
	var err error
	if allDevices {
		request := &proto.LogoutAllDevicesRequest{}
		_, err = c.client.LogoutAllDevices(
			ctx,
			request,
		)
	} else {
		request := &proto.LogoutDeviceRequest{}
		deviceName, dErr := c.getDeviceName()
		if dErr != nil {
			return dErr
		}
		request.SetDeviceName(deviceName)
		_, err = c.client.LogoutDevice(
			ctx,
			request,
		)
	}
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("logout failed: %w", err)
	}
	c.session.Clear()
	return c.storage.Clear()
}

// ========== SECRETS ==========

// ========== credentials ==========

func (c *client) createSecret(
	ctx context.Context,
	name string,
	data []byte,
	metadata map[string]string,
	dataType proto.DataType,
) error {
	if !c.session.LoggedIn() {
		return ErrSessionRequired
	}
	request := &proto.CreateSecretRequest{}
	request.SetName(name)
	request.SetType(dataType)
	request.SetMetadata(metadata)
	salt, iv, encryptedData, err := utils.EncryptData(data, c.session.Password)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}
	request.SetSalt(salt)
	request.SetIv(iv)
	request.SetData(encryptedData)
	_, err = c.client.CreateSecret(ctx, request)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.AlreadyExists:
				return fmt.Errorf("%w: secret '%s' already exists", ErrSecretAlreadyExists, name)
			case codes.InvalidArgument:
				return fmt.Errorf("invalid secret data: %s", st.Message())
			default:
				return fmt.Errorf("failed to create secret: %w", err)
			}
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}

func (c *client) CreateCredentials(
	ctx context.Context,
	name string,
	credentials *models.Credentials,
	metadata map[string]string,
) error {
	credentialsBytes, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to convert json struct to bytes: %w", err)
	}
	return c.createSecret(ctx, name, credentialsBytes, metadata, proto.DataType_DATA_TYPE_CREDENTIALS)
}

func (c *client) getSecret(
	ctx context.Context,
	name string,
	version uint64,
	dataType proto.DataType,
) (data []byte, metadata map[string]string, err error) {
	if !c.session.LoggedIn() {
		return nil, nil, ErrSessionRequired
	}
	request := &proto.GetSecretRequest{}
	request.SetName(name)
	request.SetVersion(version)
	request.SetType(dataType)
	resp, err := c.client.GetSecret(
		ctx,
		request,
	)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, nil, fmt.Errorf("%w: secret '%s' not found", ErrSecretNotFound, name)
			case codes.InvalidArgument:
				return nil, nil, fmt.Errorf("invalid request: %s", st.Message())
			default:
				return nil, nil, fmt.Errorf("failed to get secret: %w", err)
			}
		}
		return nil, nil, fmt.Errorf("failed to get secret: %w", err)
	}
	data, err = utils.DecryptData(resp.GetSalt(), resp.GetIv(), resp.GetData(), c.session.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	return data, resp.GetMetadata(), nil
}

func (c *client) GetCredentials(
	ctx context.Context,
	name string,
	version uint64,
) (credentials *models.Credentials, metadata map[string]string, err error) {
	data, metadata, err := c.getSecret(ctx, name, version, proto.DataType_DATA_TYPE_CREDENTIALS)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get secret data: %w", err)
	}
	err = json.Unmarshal(data, &credentials)
	if err != nil {
		return nil, nil, fmt.Errorf("wrong data format: %w", err)
	}
	return credentials, metadata, nil
}

func (c *client) updateSecret(
	ctx context.Context,
	name string,
	data []byte,
) error {
	if !c.session.LoggedIn() {
		return ErrSessionRequired
	}
	request := &proto.UpdateSecretDataRequest{}
	request.SetName(name)
	salt, iv, encryptedData, err := utils.EncryptData(data, c.session.Password)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}
	request.SetSalt(salt)
	request.SetIv(iv)
	request.SetData(encryptedData)
	_, err = c.client.UpdateSecretData(ctx, request)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return fmt.Errorf("%w: secret '%s' not found", ErrSecretNotFound, name)
			case codes.InvalidArgument:
				return fmt.Errorf("invalid secret data: %s", st.Message())
			default:
				return fmt.Errorf("failed to update secret: %w", err)
			}
		}
		return fmt.Errorf("failed to update secret: %w", err)
	}
	return nil
}

func (c *client) UpdateCredentials(
	ctx context.Context,
	name string,
	credentials *models.Credentials,
) error {
	credentialsBytes, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to convert json struct to bytes: %w", err)
	}
	return c.updateSecret(ctx, name, credentialsBytes)
}

// ========== secret text ==========

func (c *client) CreateSecretText(
	ctx context.Context,
	name string,
	secretText string,
	metadata map[string]string,
) error {
	return c.createSecret(ctx, name, []byte(secretText), metadata, proto.DataType_DATA_TYPE_TEXT)
}

func (c *client) GetSecretText(
	ctx context.Context,
	name string,
	version uint64,
) (secretText string, metadata map[string]string, err error) {
	data, metadata, err := c.getSecret(ctx, name, version, proto.DataType_DATA_TYPE_TEXT)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get secret data: %w", err)
	}
	return string(data), metadata, nil
}

func (c *client) UpdateSecretText(
	ctx context.Context,
	name string,
	secretText string,
) error {
	return c.updateSecret(ctx, name, []byte(secretText))
}

// ========== secret binary ==========

func (c *client) CreateSecretBinary(
	ctx context.Context,
	name string,
	secretBinary []byte,
	metadata map[string]string,
) error {
	return c.createSecret(ctx, name, secretBinary, metadata, proto.DataType_DATA_TYPE_BINARY)
}

func (c *client) GetSecretBinary(
	ctx context.Context,
	name string,
	version uint64,
) (secretBinary []byte, metadata map[string]string, err error) {
	data, metadata, err := c.getSecret(ctx, name, version, proto.DataType_DATA_TYPE_BINARY)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get secret data: %w", err)
	}
	return data, metadata, nil
}

func (c *client) UpdateSecretBinary(
	ctx context.Context,
	name string,
	secretBinary []byte,
) error {
	return c.updateSecret(ctx, name, secretBinary)
}

// ========== card data ==========

func (c *client) CreateCardData(
	ctx context.Context,
	name string,
	cardData *models.CardData,
	metadata map[string]string,
) error {
	cardBytes, err := json.Marshal(cardData)
	if err != nil {
		return fmt.Errorf("failed to convert json struct to bytes: %w", err)
	}
	return c.createSecret(ctx, name, cardBytes, metadata, proto.DataType_DATA_TYPE_CARD)
}

func (c *client) GetCardData(
	ctx context.Context,
	name string,
	version uint64,
) (cardData *models.CardData, metadata map[string]string, err error) {
	data, metadata, err := c.getSecret(ctx, name, version, proto.DataType_DATA_TYPE_CARD)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get secret data: %w", err)
	}
	err = json.Unmarshal(data, &cardData)
	if err != nil {
		return nil, nil, fmt.Errorf("wrong data format: %w", err)
	}
	return cardData, metadata, nil
}

func (c *client) UpdateCardData(
	ctx context.Context,
	name string,
	cardData *models.CardData,
) error {
	cardBytes, err := json.Marshal(cardData)
	if err != nil {
		return fmt.Errorf("failed to convert json struct to bytes: %w", err)
	}
	return c.updateSecret(ctx, name, cardBytes)
}

// ========== mutual ==========

func (c *client) RollbackSecret(
	ctx context.Context,
	name string,
) error {
	request := &proto.RollbackSecretRequest{}
	request.SetName(name)
	_, err := c.client.RollbackSecret(
		ctx,
		request,
	)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return fmt.Errorf("%w: secret '%s' not found", ErrSecretNotFound, name)
			case codes.FailedPrecondition, codes.InvalidArgument:
				return ErrSecretRollbackNotPossible
			default:
				return fmt.Errorf("failed to rollback secret: %w", err)
			}
		}
		return fmt.Errorf("failed to rollback secret: %w", err)
	}
	return nil
}

func (c *client) DeleteSecret(
	ctx context.Context,
	name string,
) error {
	request := &proto.DeleteSecretRequest{}
	request.SetName(name)
	_, err := c.client.DeleteSecret(
		ctx,
		request,
	)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return fmt.Errorf("%w: secret '%s' not found", ErrSecretNotFound, name)
			default:
				return fmt.Errorf("failed to delete secret: %w", err)
			}
		}
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

func (c *client) UpdateMetadata(
	ctx context.Context,
	name string,
	metadata map[string]string,
) error {
	request := &proto.UpdateSecretMetadataRequest{}
	request.SetName(name)
	request.SetMetadata(metadata)
	_, err := c.client.UpdateSecretMetadata(
		ctx,
		request,
	)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return fmt.Errorf("%w: secret '%s' not found", ErrSecretNotFound, name)
			case codes.InvalidArgument:
				return fmt.Errorf("invalid metadata: %s", st.Message())
			default:
				return fmt.Errorf("failed to update metadata: %w", err)
			}
		}
		return fmt.Errorf("failed to update metadata: %w", err)
	}
	return nil
}
