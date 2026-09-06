// Package config handles application configuration from flags, env, and .env.
package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all configuration parameters for the application.
// Values are populated from environment variables, .env file, and command-line flags.
// Flags take precedence over environment variables.
type Config struct {
	RunAddr              string        `env:"SERVER_ADDRESS" json:"server_address"`                 // address and port to run http server
	ShowAddr             string        `env:"BASE_URL" json:"base_url"`                             // address and port to show for short urls
	ReadTimeout          time.Duration `env:"READ_TIMEOUT" json:"read_timeout"`                     // server read timeout in seconds
	AccessTokenLifespan  time.Duration `env:"ACCESS_TOKEN_LIFESPAN" json:"access_token_lifespan"`   // access token lifespan in minutes
	RefreshTokenLifespan time.Duration `env:"REFRESH_TOKEN_LIFESPAN" json:"refresh_token_lifespan"` // refresh token lifespan in hours
	WriteTimeout         time.Duration `env:"WRITE_TIMEOUT" json:"write_timeout"`                   // server write timeout in seconds
	LoggerLevel          string        `env:"LOGGER_LEVEL" json:"logger_level"`                     // logger level DEBUG / INFO / WARNING / ERROR / FATAL
	DatabaseURI          string        `env:"DATABASE_URI" json:"database_uri"`                     // database connection url
	ForceMigrations      bool          `env:"FORCE_MIGRATIONS" json:"force_migrations"`             // force database migrations in case of dirty versions
	CookieSecret         string        `env:"COOKIE_SECRET" json:"cookie_secret"`                   // secret for cookie signature
	MigrationsPath       string        `env:"MIGRATIONS" json:"migrations_path"`                    // path to migrations
	EnableHTTPS          bool          `env:"ENABLE_HTTPS" json:"enable_https"`                     // enable https protocol
	ConfigFilePath       string        `env:"CONFIG" json:"-"`                                      // path to config file (ignored in JSON)
}

// LoadConfig parses configuration from .env file, environment variables,
// and command-line flags. Flags take precedence over environment variables.
// Returns a pointer to the populated Config struct.
// If .env is missing, it continues with environment variables and flags.
// If parsing fails, it logs a fatal error.
func LoadConfig() *Config {
	// default configuration
	config := &Config{
		RunAddr:              ":3200",
		LoggerLevel:          "INFO",
		MigrationsPath:       "./migrations",
		ReadTimeout:          time.Duration(30) * time.Second,
		WriteTimeout:         time.Duration(30) * time.Second,
		AccessTokenLifespan:  time.Duration(15) * time.Minute,
		RefreshTokenLifespan: time.Duration(3) * time.Hour,
	}

	// parse .env file
	err := godotenv.Load()
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("No .env file found, using environment variables and flags")
		} else {
			log.Fatalf("Failed to load .env file: %v", err)
		}
	}
	if err := env.Parse(config); err != nil {
		log.Printf("warning: failed to parse env: %v", err)
	}

	// parse config file
	if config.ConfigFilePath != "" {
		// #nosec G703 -- configuration file path is intentionally provided by the user.
		data, err := os.ReadFile(config.ConfigFilePath)
		if err != nil {
			log.Fatalf("failed to read config file: %v", err)
		}
		if err := json.Unmarshal(data, config); err != nil {
			log.Fatalf("failed to parse config file: %v", err)
		}
	}

	// read flags directly to config
	flag.StringVar(&config.RunAddr, "a", config.RunAddr, "address and port to run http server")
	flag.StringVar(&config.ShowAddr, "b", config.ShowAddr, "address and port to show for short urls")
	flag.StringVar(&config.LoggerLevel, "l", config.LoggerLevel, "logger level")
	flag.StringVar(&config.DatabaseURI, "d", config.DatabaseURI, "database connection url")
	flag.BoolVar(&config.ForceMigrations, "f", config.ForceMigrations, "force database migrations in case of last dirty versions")
	flag.StringVar(&config.CookieSecret, "cookie-secret", config.CookieSecret, "cookie signing secret")
	flag.StringVar(&config.MigrationsPath, "migrations", config.MigrationsPath, "path to database migrations")
	flag.BoolVar(&config.EnableHTTPS, "s", config.EnableHTTPS, "enable https protocol")
	flag.StringVar(&config.ConfigFilePath, "c", config.ConfigFilePath, "config file path")
	flag.StringVar(&config.ConfigFilePath, "config", config.ConfigFilePath, "config file path")
	// read flags to temp vars
	var readSec, writeSec, accessTokenLifespan, refreshTokenLifespan float64
	flag.Float64Var(&readSec, "r", config.ReadTimeout.Seconds(), "server read timeout in seconds")
	flag.Float64Var(&writeSec, "w", config.WriteTimeout.Seconds(), "server write timeout in seconds")
	flag.Float64Var(&accessTokenLifespan, "access-token-lifespan", config.AccessTokenLifespan.Minutes(), "access token lifespan in minutes")
	flag.Float64Var(&refreshTokenLifespan, "refresh-token-lifespan", config.RefreshTokenLifespan.Hours(), "refresh token lifespan in hours")
	// parse flags
	flag.Parse()
	// parse temp vars to config struct
	config.ReadTimeout = time.Duration(readSec) * time.Second
	config.WriteTimeout = time.Duration(writeSec) * time.Second
	config.AccessTokenLifespan = time.Duration(accessTokenLifespan) * time.Minute
	config.RefreshTokenLifespan = time.Duration(refreshTokenLifespan) * time.Hour

	return config
}
