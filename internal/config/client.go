// Package config handles client configuration from configuration file.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ClientConf contains whole client configuration
type ClientConf struct {
	ConfigFolder   string
	ConfigFilename string
	ServerAddr     string        `json:"server_addr"`
	TokenFilename  string        `json:"tokens_filename"`
	RequestTimeout time.Duration `json:"requests_timeout"`
	SessionTTL     time.Duration `json:"session_ttl"`
	LoggerLevel    string        `json:"logger_level"`
	SecretsTTL     time.Duration `json:"secrets_ttl"`
}

// NewClientConf returns all client configuration including configuration from file
func NewClientConf() (*ClientConf, error) {
	configuration := &ClientConf{
		// constants
		ConfigFolder:   "custodia",
		ConfigFilename: "config.json",
		// can be overwritten by config file
		ServerAddr:     "localhost:3200",
		TokenFilename:  "tokens.json",
		RequestTimeout: 10 * time.Second,
		SessionTTL:     15 * time.Minute,
		LoggerLevel:    "INFO",
		SecretsTTL:     30 * time.Second,
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	appConfigDir := filepath.Join(configDir, configuration.ConfigFolder)
	err = os.MkdirAll(appConfigDir, 0700)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(appConfigDir, configuration.ConfigFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return configuration, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &configuration); err != nil {
		return nil, err
	}
	return configuration, nil
}
