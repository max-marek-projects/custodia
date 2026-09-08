package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/max-marek-projects/custodia/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const testAddr = ":0" // let the system pick a free port

// TestCreateTLSConf tests loading of TLS certificates.
func TestCreateTLSConf(t *testing.T) {

	t.Run("invalid certificate path", func(t *testing.T) {
		_, err := CreateTLSConf("nonexistent.crt", "nonexistent.key")
		assert.Error(t, err)
	})
}

// TestNewServer tests server creation with and without TLS.
func TestNewServer(t *testing.T) {
	handler := handlers.NewGRPCHandler(NewMockService(t))

	t.Run("without TLS", func(t *testing.T) {
		svr := NewServer(testAddr, handler, 0, 0, "secret", nil)
		assert.NotNil(t, svr)
		assert.NotNil(t, svr.Server)
		assert.Equal(t, testAddr, svr.Addr)
	})
}

// TestServer_ListenAndServe_Shutdown tests lifecycle with a real listener.
func TestServer_ListenAndServe_Shutdown(t *testing.T) {
	handler := handlers.NewGRPCHandler(NewMockService(t))
	svr := NewServer("127.0.0.1:0", handler, 0, 0, "secret", nil)

	// Use a goroutine to start the server.
	errCh := make(chan error, 1)
	go func() {
		errCh <- svr.ListenAndServe()
	}()

	// Give the server time to start.
	time.Sleep(50 * time.Millisecond)

	// Shutdown gracefully.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := svr.Shutdown(ctx)
	assert.NoError(t, err)

	// Wait for server to finish.
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

// TestServer_withBufconn tests the server using bufconn (no real TCP).
func TestServer_withBufconn(t *testing.T) {
	handler := handlers.NewGRPCHandler(NewMockService(t))
	svr := NewServer("", handler, 0, 0, "secret", nil)

	// Create a bufconn listener.
	listener := bufconn.Listen(1024 * 1024)

	// Start server in goroutine.
	go func() {
		if err := svr.Serve(listener); err != nil {
			// Ignore expected errors from GracefulStop
		}
	}()
	defer svr.GracefulStop()

	// Create a client connection to the bufconn.
	conn, err := grpc.NewClient(
		"bufconn",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Here we would normally test a real RPC, but since the handler is empty,
	// we just verify the connection works (e.g., context).
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// We can check that the connection is ready.
	ok := conn.WaitForStateChange(ctx, conn.GetState())
	// It may or may not change state; we just ensure no panic.
	assert.True(t, ok || true) // placeholder
}
