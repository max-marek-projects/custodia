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
	"google.golang.org/grpc/credentials"
)

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

func (c *client) Login(
	ctx context.Context,
	login string,
	password []byte,
	sessionTTL time.Duration,
) error {
	request := &proto.LoginRequest{}
	request.SetLogin(login)
	request.SetPassword(string(password))
	deviceName, err := c.getDeviceName()
	if err != nil {
		return err
	}
	request.SetDeviceName(deviceName)
	resp, err := c.client.LoginUser(
		ctx,
		request,
	)
	if err != nil {
		return err
	}
	c.session.ExpiresAt = time.Now().Add(sessionTTL)
	c.session.Password = password
	return c.storage.Save(
		&models.Tokens{
			UserID:       resp.GetUserId(),
			AccessToken:  resp.GetAccessToken(),
			RefreshToken: resp.GetRefreshToken(),
		})
}

func (c *client) Register(
	ctx context.Context,
	login string,
	password []byte,
	sessionTTL time.Duration,
) error {
	request := &proto.LoginRequest{}
	request.SetLogin(login)
	request.SetPassword(string(password))
	deviceName, err := c.getDeviceName()
	if err != nil {
		return err
	}
	request.SetDeviceName(deviceName)
	resp, err := c.client.RegisterUser(
		ctx,
		request,
	)
	if err != nil {
		return err
	}
	c.session.Password = password
	c.session.ExpiresAt = time.Now().Add(sessionTTL)
	return c.storage.Save(
		&models.Tokens{
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
	resp, err := c.client.Refresh(
		ctx,
		request,
	)
	if err != nil {
		return err
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
		return err
	}
	c.session.Clear()
	return c.storage.Clear()
}

// ========== SECRETS ==========

// ========== credentials ==========

func (c *client) CreateCredentials(
	ctx context.Context,
	name string,
	credentials *models.Credentials,
	metadata map[string]string,
) error {
	if !c.session.LoggedIn() {
		return fmt.Errorf("Can create credentials only from authorized TUI session")
	}
	request := &proto.CreateSecretRequest{}
	request.SetName(name)
	request.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
	request.SetMetadata(metadata)
	credentialsBytes, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to convert json struct to bytes: %w", err)
	}
	salt, iv, encryptedData, err := utils.EncryptData(credentialsBytes, c.session.Password)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}
	request.SetSalt(salt)
	request.SetIv(iv)
	request.SetData(encryptedData)
	_, err = c.client.CreateSecret(
		ctx,
		request,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) GetCredentials(
	ctx context.Context,
	name string,
) (credentials *models.Credentials, metadata map[string]string, err error) {
	if !c.session.LoggedIn() {
		return nil, nil, fmt.Errorf("Can get credentials only from authorized TUI session")
	}
	request := &proto.GetSecretRequest{}
	request.SetName(name)
	request.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
	resp, err := c.client.GetSecret(
		ctx,
		request,
	)
	if err != nil {
		return nil, nil, err
	}
	data, err := utils.DecryptData(resp.GetSalt(), resp.GetIv(), resp.GetData(), c.session.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	err = json.Unmarshal(data, &credentials)
	if err != nil {
		return nil, nil, fmt.Errorf("wrong data format: %w", err)
	}
	return credentials, resp.GetMetadata(), nil
}

// ========== secret text ==========

func (c *client) CreateSecretText(
	ctx context.Context,
	name string,
	secretText string,
	metadata map[string]string,
) error {
	if !c.session.LoggedIn() {
		return fmt.Errorf("Can create credentials only from authorized TUI session")
	}
	request := &proto.CreateSecretRequest{}
	request.SetName(name)
	request.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
	request.SetMetadata(metadata)
	salt, iv, encryptedData, err := utils.EncryptData([]byte(secretText), c.session.Password)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}
	request.SetSalt(salt)
	request.SetIv(iv)
	request.SetData(encryptedData)
	_, err = c.client.CreateSecret(
		ctx,
		request,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) GetSecretText(
	ctx context.Context,
	name string,
) (secretText string, metadata map[string]string, err error) {
	if !c.session.LoggedIn() {
		return "", nil, fmt.Errorf("Can get credentials only from authorized TUI session")
	}
	request := &proto.GetSecretRequest{}
	request.SetName(name)
	request.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
	resp, err := c.client.GetSecret(
		ctx,
		request,
	)
	if err != nil {
		return "", nil, err
	}
	data, err := utils.DecryptData(resp.GetSalt(), resp.GetIv(), resp.GetData(), c.session.Password)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	return string(data), resp.GetMetadata(), nil
}

// ========== secret binary ==========

func (c *client) CreateSecretBinary(
	ctx context.Context,
	name string,
	secretBinary []byte,
	metadata map[string]string,
) error {
	if !c.session.LoggedIn() {
		return fmt.Errorf("Can create credentials only from authorized TUI session")
	}
	request := &proto.CreateSecretRequest{}
	request.SetName(name)
	request.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
	request.SetMetadata(metadata)
	salt, iv, encryptedData, err := utils.EncryptData(secretBinary, c.session.Password)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}
	request.SetSalt(salt)
	request.SetIv(iv)
	request.SetData(encryptedData)
	_, err = c.client.CreateSecret(
		ctx,
		request,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) GetSecretBinary(
	ctx context.Context,
	name string,
) (secretBinary []byte, metadata map[string]string, err error) {
	if !c.session.LoggedIn() {
		return nil, nil, fmt.Errorf("Can get credentials only from authorized TUI session")
	}
	request := &proto.GetSecretRequest{}
	request.SetName(name)
	request.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
	resp, err := c.client.GetSecret(
		ctx,
		request,
	)
	if err != nil {
		return nil, nil, err
	}
	data, err := utils.DecryptData(resp.GetSalt(), resp.GetIv(), resp.GetData(), c.session.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	return data, resp.GetMetadata(), nil
}

// ========== card data ==========

func (c *client) CreateCardData(
	ctx context.Context,
	name string,
	cardData *models.CardData,
	metadata map[string]string,
) error {
	if !c.session.LoggedIn() {
		return fmt.Errorf("Can create credentials only from authorized TUI session")
	}
	request := &proto.CreateSecretRequest{}
	request.SetName(name)
	request.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
	request.SetMetadata(metadata)
	credentialsBytes, err := json.Marshal(cardData)
	if err != nil {
		return fmt.Errorf("failed to convert json struct to bytes: %w", err)
	}
	salt, iv, encryptedData, err := utils.EncryptData(credentialsBytes, c.session.Password)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}
	request.SetSalt(salt)
	request.SetIv(iv)
	request.SetData(encryptedData)
	_, err = c.client.CreateSecret(
		ctx,
		request,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) GetCardData(
	ctx context.Context,
	name string,
) (cardData *models.CardData, metadata map[string]string, err error) {
	if !c.session.LoggedIn() {
		return nil, nil, fmt.Errorf("Can get credentials only from authorized TUI session")
	}
	request := &proto.GetSecretRequest{}
	request.SetName(name)
	request.SetType(proto.DataType_DATA_TYPE_CREDENTIALS)
	resp, err := c.client.GetSecret(
		ctx,
		request,
	)
	if err != nil {
		return nil, nil, err
	}
	data, err := utils.DecryptData(resp.GetSalt(), resp.GetIv(), resp.GetData(), c.session.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	err = json.Unmarshal(data, &cardData)
	if err != nil {
		return nil, nil, fmt.Errorf("wrong data format: %w", err)
	}
	return cardData, resp.GetMetadata(), nil
}
