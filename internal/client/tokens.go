// Package client provides file-based storage for authentication tokens.
package client

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
)

// tokenStorage manages persistent storage of tokens in a JSON file.
type tokenStorage struct {
	folder   string
	filename string
	mu       sync.RWMutex
}

// newTokenStorage creates a new tokenStorage instance.
func newTokenStorage(folder, filename string) *tokenStorage {
	return &tokenStorage{folder: folder, filename: filename}
}

// getFilePath returns the full path to the token file, creating the directory if needed.
// It uses the user's configuration directory.
func (t *tokenStorage) getFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appConfigDir := filepath.Join(configDir, "custodia")
	err = os.MkdirAll(appConfigDir, 0700)
	if err != nil {
		return "", err
	}
	tokenFilepath := filepath.Join(appConfigDir, "tokens.json")
	logger.Log.Debug("token filepath", slog.String("filepath", tokenFilepath))
	return tokenFilepath, nil
}

// Save writes the tokens to the file in JSON format.
// It acquires a write lock and creates the file with 0600 permissions.
//
// Parameters:
//   - tokens: the token data to store (must be non‑nil).
//
// Returns:
//   - error: nil on success, or an error if the file path cannot be obtained,
//     marshaling fails, or writing the file fails.
func (t *tokenStorage) Save(tokens *models.Tokens) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	tokensFilePath, err := t.getFilePath()
	if err != nil {
		return err
	}
	jsonData, err := json.MarshalIndent(tokens, "", "    ")
	if err != nil {
		return err
	}
	err = os.WriteFile(tokensFilePath, jsonData, 0600)
	if err != nil {
		return err
	}
	return nil
}

// Read loads tokens from the file.
// If the file does not exist, it returns an empty Tokens object (no error).
//
// Returns:
//   - *models.Tokens: the loaded tokens, or an empty object if file is missing.
//   - error: nil on success, or an error if reading or unmarshaling fails.
func (t *tokenStorage) Read() (*models.Tokens, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	path, err := t.getFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.Tokens{}, nil
		}
		return nil, err
	}
	var s models.Tokens
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Clear removes the token file.
// If the file does not exist, it returns nil (no error).
//
// Returns:
//   - error: nil on success, or an error if removal fails (except for non‑existent).
func (t *tokenStorage) Clear() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	path, err := t.getFilePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
