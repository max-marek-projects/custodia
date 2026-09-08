package client

import (
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnv sets environment variables so that os.UserConfigDir() points
// to a temporary directory. Uses t.Setenv for automatic cleanup.
func setupTestEnv(t *testing.T) string {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("APPDATA", tmpDir)
	return tmpDir
}

func TestTokenStorage_Save(t *testing.T) {
	setupTestEnv(t)

	storage := newTokenStorage("", "")
	tokens := &models.Tokens{
		AccessToken:  "access123",
		RefreshToken: []byte("refresh456"),
	}

	err := storage.Save(tokens)
	require.NoError(t, err)

	path, err := storage.getFilePath()
	require.NoError(t, err)
	_, err = os.Stat(path)
	assert.NoError(t, err)

	// Verify content
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var read models.Tokens
	err = json.Unmarshal(data, &read)
	require.NoError(t, err)
	assert.Equal(t, tokens, &read)
}

func TestTokenStorage_Save_NilTokens(t *testing.T) {
	setupTestEnv(t)

	storage := newTokenStorage("", "")
	err := storage.Save(nil)
	assert.Error(t, err)
	assert.Equal(t, "tokens is nil", err.Error())
}

func TestTokenStorage_Read(t *testing.T) {
	setupTestEnv(t)

	storage := newTokenStorage("", "")
	tokens := &models.Tokens{
		AccessToken:  "access123",
		RefreshToken: []byte("refresh456"),
	}

	err := storage.Save(tokens)
	require.NoError(t, err)

	read, err := storage.Read()
	require.NoError(t, err)
	assert.Equal(t, tokens, read)
}

func TestTokenStorage_Read_FileNotExists(t *testing.T) {
	setupTestEnv(t)

	storage := newTokenStorage("", "")

	// Ensure file does not exist
	path, err := storage.getFilePath()
	require.NoError(t, err)
	_ = os.Remove(path)

	read, err := storage.Read()
	require.NoError(t, err)
	assert.Empty(t, read.AccessToken)
	assert.Empty(t, read.RefreshToken)
}

func TestTokenStorage_Read_InvalidJSON(t *testing.T) {
	setupTestEnv(t)

	storage := newTokenStorage("", "")
	path, err := storage.getFilePath()
	require.NoError(t, err)

	err = os.WriteFile(path, []byte(`{"invalid":`), 0600)
	require.NoError(t, err)

	_, err = storage.Read()
	assert.Error(t, err)
}

func TestTokenStorage_Clear(t *testing.T) {
	setupTestEnv(t)

	storage := newTokenStorage("", "")
	tokens := &models.Tokens{AccessToken: "access123"}
	err := storage.Save(tokens)
	require.NoError(t, err)

	err = storage.Clear()
	require.NoError(t, err)

	path, err := storage.getFilePath()
	require.NoError(t, err)
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestTokenStorage_Clear_FileNotExists(t *testing.T) {
	setupTestEnv(t)

	storage := newTokenStorage("", "")
	path, err := storage.getFilePath()
	require.NoError(t, err)
	_ = os.Remove(path)

	err = storage.Clear()
	assert.NoError(t, err)
}

func TestTokenStorage_ConcurrentAccess(t *testing.T) {
	setupTestEnv(t)

	storage := newTokenStorage("", "")
	tokens := &models.Tokens{AccessToken: "access123"}

	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := storage.Save(tokens); err != nil {
				errCh <- err
			}
		}()
	}
	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := storage.Read(); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		assert.NoError(t, err)
	}
}
