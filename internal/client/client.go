package client

import (
	"context"
	"crypto/tls"
	"os"

	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Handler handles grpc endpoints for URL shortening and redirection.
type client struct {
	connection *grpc.ClientConn
	client     proto.CustodiaClient
	storage    *tokenStorage
}

// NewHandler creates a new GRPCHandler.
func NewClient(addr, folder, filename string) (*client, error) {
	clientItem := &client{storage: newTokenStorage(folder, filename)}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithChainUnaryInterceptor(clientItem.authInterceptor()),
	)
	if err != nil {
		return nil, err
	}
	clientItem.connection = conn
	clientItem.client = proto.NewCustodiaClient(conn)
	return clientItem, nil
}

func (c *client) getDeviceName() (string, error) {
	return os.Hostname()
}

func (c *client) Close() error {
	return c.connection.Close()
}

// ========== endpoints ==========

func (c *client) Login(
	ctx context.Context,
	login string,
	password string,
) error {
	request := &proto.LoginRequest{}
	request.SetLogin(login)
	request.SetPassword(password)
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
	password string,
) error {
	request := &proto.LoginRequest{}
	request.SetLogin(login)
	request.SetPassword(password)
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
	return c.storage.Clear()
}
