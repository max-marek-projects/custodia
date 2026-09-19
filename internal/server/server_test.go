package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/max-marek-projects/custodia/internal/handlers"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const testAddr = ":0"                                              // let the system pick a free port
const testSecretKey = "test-secret-key-abcdefghijklmnopqrstuvwxyz" // at least 32 characters long

// TestCreateTLSConf tests loading of TLS certificates.
func TestCreateTLSConf(t *testing.T) {

	t.Run("invalid certificate path", func(t *testing.T) {
		_, err := CreateTLSConf("nonexistent.crt", "nonexistent.key", logger.NewNop())
		assert.Error(t, err)
	})
}

// TestNewServer tests server creation with and without TLS.
func TestNewServer(t *testing.T) {
	handler := handlers.NewGRPCHandler(NewMockService(t), logger.NewNop())

	t.Run("without TLS", func(t *testing.T) {
		svr, err := NewServer(testAddr, handler, 0, 0, testSecretKey, nil, logger.NewNop())
		require.NoError(t, err)
		assert.NotNil(t, svr)
		assert.NotNil(t, svr.Server)
		assert.Equal(t, testAddr, svr.Addr)
	})

	t.Run("empty secret key", func(t *testing.T) {
		_, err := NewServer(testAddr, handler, 0, 0, "", nil, logger.NewNop())
		assert.Error(t, err)
	})

	t.Run("too short secret key", func(t *testing.T) {
		_, err := NewServer(testAddr, handler, 0, 0, "too-short-secret-key", nil, logger.NewNop())
		assert.Error(t, err)
	})
}

// TestServer_ListenAndServe_Shutdown tests lifecycle with a real listener.
func TestServer_ListenAndServe_Shutdown(t *testing.T) {
	mockService := NewMockService(t)
	mockService.EXPECT().Close(mock.Anything).Return(nil)
	handler := handlers.NewGRPCHandler(mockService, logger.NewNop())
	svr, err := NewServer("127.0.0.1:0", handler, 0, 0, testSecretKey, nil, logger.NewNop())
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- svr.ListenAndServe()
	}()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = svr.Shutdown(ctx)
	assert.NoError(t, err)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			assert.NoError(t, err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop in time")
	}
	mockService.AssertExpectations(t)
}
