// Package config handles client configuration from configuration file.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// clientConf contains whole client configuration
type clientConf struct {
	ConfigFolder   string
	ConfigFilename string
	ServerAddr     string        `json:"server_addr"`
	TokenFilename  string        `json:"tokens_filename"`
	RequestTimeout time.Duration `json:"requests_timeout"`
	LoggerLevel    string        `json:"logger_level"`
}

// NewClientConf returns all client configuration including configuration from file
func NewClientConf() (*clientConf, error) {
	configuration := &clientConf{
		// constants
		ConfigFolder:   "custodia",
		ConfigFilename: "config.json",
		// can be overwritten by config file
		ServerAddr:     "localhost:3200",
		TokenFilename:  "tokens.json",
		RequestTimeout: 10 * time.Second,
		LoggerLevel:    "INFO",
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
