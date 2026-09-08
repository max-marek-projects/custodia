// Package config handles application configuration from flags, env, and .env.
// It supports loading from environment variables, .env file, JSON config file,
// and command-line flags with the following precedence:
//
//  1. Command-line flags (highest)
//  2. JSON config file (if provided via -config or CONFIG env)
//  3. Environment variables
//  4. .env file
//  5. Default values (lowest)
//
// All fields are documented with their env and JSON tags.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
)

// Config holds all configuration parameters for the application.
type Config struct {
	// RunAddr is the address and port for the HTTP server (e.g., ":3200").
	RunAddr string `env:"SERVER_ADDRESS" json:"server_address"`
	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout models.Duration `env:"READ_TIMEOUT" json:"read_timeout"`
	// AccessTokenLifespan is the validity duration of JWT access tokens.
	AccessTokenLifespan models.Duration `env:"ACCESS_TOKEN_LIFESPAN" json:"access_token_lifespan"`
	// RefreshTokenLifespan is the validity duration of refresh tokens.
	RefreshTokenLifespan models.Duration `env:"REFRESH_TOKEN_LIFESPAN" json:"refresh_token_lifespan"`
	// WriteTimeout is the maximum duration before timing out writes of the response.
	WriteTimeout models.Duration `env:"WRITE_TIMEOUT" json:"write_timeout"`
	// LoggerLevel is the log level (DEBUG, INFO, WARN, ERROR).
	LoggerLevel logger.Level `env:"LOGGER_LEVEL" json:"logger_level"`
	// DatabaseURI is the connection string for the PostgreSQL database.
	DatabaseURI string `env:"DATABASE_URI" json:"database_uri"`
	// ForceMigrations forces database migrations even if the schema is dirty.
	ForceMigrations bool `env:"FORCE_MIGRATIONS" json:"force_migrations"`
	// CookieSecret is the secret key used for signing cookies (JWT).
	CookieSecret string `env:"COOKIE_SECRET" json:"cookie_secret"`
	// MigrationsPath is the directory containing migration files.
	MigrationsPath string `env:"MIGRATIONS" json:"migrations_path"`
	// EnableHTTPS enables TLS for the server.
	EnableHTTPS bool `env:"ENABLE_HTTPS" json:"enable_https"`
	// ConfigFilePath is the path to a JSON configuration file (ignored in JSON).
	ConfigFilePath string `env:"CONFIG" json:"-"`
}

// LoadConfig loads and returns the application configuration.
// It applies defaults, reads .env, environment variables, config file (if specified),
// and finally command-line flags. Flags take precedence over all other sources.
//
// Returns:
//   - *Config: populated configuration.
//   - error: non-nil if .env parsing fails, environment parsing fails,
//     config file reading or unmarshaling fails.
func LoadConfig() (config *Config, err error) {
	// default configuration
	config = &Config{
		RunAddr:              ":3200",
		LoggerLevel:          logger.LevelInfo,
		MigrationsPath:       "./migrations",
		ReadTimeout:          models.Duration(30 * time.Second),
		WriteTimeout:         models.Duration(30 * time.Second),
		AccessTokenLifespan:  models.Duration(15 * time.Minute),
		RefreshTokenLifespan: models.Duration(3 * time.Hour),
	}

	// parse .env file
	err = godotenv.Load()
	if err != nil {
		if os.IsNotExist(err) {
			logger.Log.Info("No .env file found, using environment variables and flags")
		} else {
			return nil, fmt.Errorf("Failed to load .env file: %w", err)
		}
	}
	if err := env.Parse(config); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	// parse config file path
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-c" || args[i] == "-config":
			if i+1 < len(args) {
				config.ConfigFilePath = args[i+1]
			}
		case strings.HasPrefix(args[i], "-c="):
			config.ConfigFilePath = strings.TrimPrefix(args[i], "-c=")
		case strings.HasPrefix(args[i], "-config="):
			config.ConfigFilePath = strings.TrimPrefix(args[i], "-config=")
		}
	}

	// parse config file
	if config.ConfigFilePath != "" {
		// #nosec G703 -- configuration file path is intentionally provided by the user.
		data, err := os.ReadFile(config.ConfigFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// read flags directly to config
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.StringVar(&config.RunAddr, "a", config.RunAddr, "address and port to run http server")
	fs.Var(&config.LoggerLevel, "l", "logger level")
	fs.StringVar(&config.DatabaseURI, "d", config.DatabaseURI, "database connection url")
	fs.BoolVar(&config.ForceMigrations, "f", config.ForceMigrations, "force database migrations in case of last dirty versions")
	fs.StringVar(&config.CookieSecret, "cookie-secret", config.CookieSecret, "cookie signing secret")
	fs.StringVar(&config.MigrationsPath, "migrations", config.MigrationsPath, "path to database migrations")
	fs.BoolVar(&config.EnableHTTPS, "s", config.EnableHTTPS, "enable https protocol")
	fs.StringVar(&config.ConfigFilePath, "c", config.ConfigFilePath, "config file path")
	fs.StringVar(&config.ConfigFilePath, "config", config.ConfigFilePath, "config file path")
	// read flags to temp vars
	fs.Var(&config.ReadTimeout, "r", "server read timeout in seconds")
	fs.Var(&config.WriteTimeout, "w", "server write timeout in seconds")
	fs.Var(&config.AccessTokenLifespan, "access-token-lifespan", "access token lifespan in minutes")
	fs.Var(&config.RefreshTokenLifespan, "refresh-token-lifespan", "refresh token lifespan in hours")
	// parse flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	return config, nil
}
