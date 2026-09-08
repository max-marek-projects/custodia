package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientConf_Defaults(t *testing.T) {
	// Use a temporary directory and set CUSTODIA_CONFIG_DIR.
	tmpDir := t.TempDir()
	t.Setenv(configEnvName, tmpDir)

	cfg, err := NewClientConf()
	require.NoError(t, err)

	assert.Equal(t, "custodia", cfg.ConfigFolder)
	assert.Equal(t, "config.json", cfg.ConfigFilename)
	assert.Equal(t, "localhost:3200", cfg.ServerAddr)
	assert.Equal(t, "tokens.json", cfg.TokenFilename)
	assert.Equal(t, models.Duration(10*time.Second), cfg.RequestTimeout)
	assert.Equal(t, models.Duration(15*time.Minute), cfg.SessionTTL)
	assert.Equal(t, logger.LevelInfo, cfg.LoggerLevel)
	assert.Equal(t, models.Duration(30*time.Second), cfg.SecretsTTL)
}

func TestNewClientConf_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(configEnvName, tmpDir)

	// Create a configuration file.
	cfgFile := filepath.Join(tmpDir, "config.json")
	content := `{
		"server_addr": "example.com:8080",
		"tokens_filename": "my_tokens.json",
		"requests_timeout": "5s",
		"session_ttl": "30m",
		"logger_level": "DEBUG",
		"secrets_ttl": "1m"
	}`
	err := os.WriteFile(cfgFile, []byte(content), 0600)
	require.NoError(t, err)

	cfg, err := NewClientConf()
	require.NoError(t, err)

	assert.Equal(t, "example.com:8080", cfg.ServerAddr)
	assert.Equal(t, "my_tokens.json", cfg.TokenFilename)
	assert.Equal(t, models.Duration(5*time.Second), cfg.RequestTimeout)
	assert.Equal(t, models.Duration(30*time.Minute), cfg.SessionTTL)
	assert.Equal(t, logger.LevelDebug, cfg.LoggerLevel)
	assert.Equal(t, models.Duration(1*time.Minute), cfg.SecretsTTL)
	// Fields that are not in JSON should remain defaults.
	assert.Equal(t, "custodia", cfg.ConfigFolder)
	assert.Equal(t, "config.json", cfg.ConfigFilename)
}

func TestNewClientConf_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(configEnvName, tmpDir)

	cfgFile := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(cfgFile, []byte(`{invalid json}`), 0600)
	require.NoError(t, err)

	_, err = NewClientConf()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestNewClientConf_EnvOverride(t *testing.T) {
	// Use a custom directory via environment variable.
	tmpDir := t.TempDir()
	t.Setenv(configEnvName, tmpDir)

	// Create a config file with custom server address.
	cfgFile := filepath.Join(tmpDir, "config.json")
	content := `{"server_addr": "env-override:9999"}`
	err := os.WriteFile(cfgFile, []byte(content), 0600)
	require.NoError(t, err)

	cfg, err := NewClientConf()
	require.NoError(t, err)
	assert.Equal(t, "env-override:9999", cfg.ServerAddr)
}

func TestClientConf_UnmarshalDuration(t *testing.T) {
	// Test that models.Duration unmarshals correctly from JSON.
	jsonData := `{
		"requests_timeout": "5s",
		"session_ttl": "30m",
		"secrets_ttl": "1h"
	}`
	var cfg ClientConf
	err := json.Unmarshal([]byte(jsonData), &cfg)
	require.NoError(t, err)

	assert.Equal(t, models.Duration(5*time.Second), cfg.RequestTimeout)
	assert.Equal(t, models.Duration(30*time.Minute), cfg.SessionTTL)
	assert.Equal(t, models.Duration(1*time.Hour), cfg.SecretsTTL)
}
