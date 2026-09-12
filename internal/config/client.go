// Package config provides client configuration management.
// It reads configuration from a JSON file located in the user's config directory.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
)

var configEnvName = "CUSTODIA_CONFIG_DIR"

// ClientConf holds the client configuration parameters.
type ClientConf struct {
	// ConfigFolder is the name of the application's configuration folder.
	ConfigFolder string `json:"-"`
	// ConfigFilename is the name of the configuration file.
	ConfigFilename string `json:"-"`

	// ServerAddr is the address of the server (e.g., "localhost:3200").
	ServerAddr string `json:"server_addr"`
	// TokenFilename is the name of the file where tokens are stored.
	TokenFilename string `json:"tokens_filename"`
	// RequestTimeout is the timeout for HTTP requests to the server.
	RequestTimeout models.Duration `json:"requests_timeout"`
	// SessionTTL is the duration for which the session is valid.
	SessionTTL models.Duration `json:"session_ttl"`
	// LoggerLevel is the logging level (DEBUG, INFO, WARN, ERROR).
	LoggerLevel logger.Level `json:"logger_level"`
	// SecretsTTL is the duration for which secrets are cached locally.
	SecretsTTL models.Duration `json:"secrets_ttl"`
	// CertPath is the path to a certificate file for client tls configuration.
	CertPath string `json:"cert_path"`
}

// NewClientConf creates a new ClientConf with default values,
// then overrides them with values from the configuration file if it exists.
// The configuration file is expected to be in the user's configuration directory
// under the folder "custodia" with the name "config.json".
//
// Returns:
//   - *ClientConf: populated configuration.
//   - error: non-nil if the config directory cannot be created,
//     the config file cannot be read, or JSON unmarshaling fails.
func NewClientConf() (*ClientConf, error) {
	configuration := &ClientConf{
		// constants
		ConfigFolder:   "custodia",
		ConfigFilename: "config.json",
		// can be overwritten by config file
		ServerAddr:     "localhost:3200",
		TokenFilename:  "tokens.json",
		RequestTimeout: models.Duration(10 * time.Second),
		SessionTTL:     models.Duration(15 * time.Minute),
		LoggerLevel:    logger.LevelInfo,
		SecretsTTL:     models.Duration(30 * time.Second),
		CertPath:       "server.pem",
	}
	// Determine the configuration directory.
	var appConfigDir string
	if envDir := os.Getenv(configEnvName); envDir != "" {
		appConfigDir = envDir
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user config dir: %w", err)
		}
		appConfigDir = filepath.Join(configDir, configuration.ConfigFolder)
	}

	// create configuration directory
	if err := os.MkdirAll(appConfigDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(appConfigDir, configuration.ConfigFilename)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("config file not found (%s), using defaults\n", configPath)
			return configuration, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, configuration); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	fmt.Printf("configuration loaded (%s)\n", configPath)
	return configuration, nil
}
