package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, ":3200", cfg.RunAddr)
	assert.Equal(t, 30*time.Second, cfg.ReadTimeout.Duration())
	assert.Equal(t, 15*time.Minute, cfg.AccessTokenLifespan.Duration())
	assert.Equal(t, 3*time.Hour, cfg.RefreshTokenLifespan.Duration())
	assert.Equal(t, 30*time.Second, cfg.WriteTimeout.Duration())
	assert.Equal(t, logger.LevelInfo, cfg.LoggerLevel)
	assert.Empty(t, cfg.DatabaseURI)
	assert.False(t, cfg.ForceMigrations)
	assert.Empty(t, cfg.CookieSecret)
	assert.Equal(t, "./migrations", cfg.MigrationsPath)
	assert.False(t, cfg.EnableHTTPS)
	assert.Empty(t, cfg.ConfigFilePath)
}

func TestLoadConfig_Flags(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{
		"cmd",
		"-a=:9090",
		"-r=5s",
		"-w=10s",
		"-access-token-lifespan=30m",
		"-refresh-token-lifespan=6h",
		"-l=ERROR",
		"-d=postgres://flag:pass@localhost:5432/db",
		"-f=true",
		"-cookie-secret=flag_secret",
		"-migrations=./flag_migrations",
		"-s=true",
	}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, ":9090", cfg.RunAddr)
	assert.Equal(t, 5*time.Second, cfg.ReadTimeout.Duration())
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout.Duration())
	assert.Equal(t, 30*time.Minute, cfg.AccessTokenLifespan.Duration())
	assert.Equal(t, 6*time.Hour, cfg.RefreshTokenLifespan.Duration())
	assert.Equal(t, logger.LevelError, cfg.LoggerLevel)
	assert.Equal(t, "postgres://flag:pass@localhost:5432/db", cfg.DatabaseURI)
	assert.True(t, cfg.ForceMigrations)
	assert.Equal(t, "flag_secret", cfg.CookieSecret)
	assert.Equal(t, "./flag_migrations", cfg.MigrationsPath)
	assert.True(t, cfg.EnableHTTPS)
	assert.Empty(t, cfg.ConfigFilePath)
}

func TestLoadConfig_Env(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	envVars := map[string]string{
		"SERVER_ADDRESS":         ":8080",
		"READ_TIMEOUT":           "15s",
		"WRITE_TIMEOUT":          "20s",
		"ACCESS_TOKEN_LIFESPAN":  "45m",
		"REFRESH_TOKEN_LIFESPAN": "8h",
		"LOGGER_LEVEL":           "DEBUG",
		"DATABASE_URI":           "postgres://env:pass@localhost:5432/db",
		"FORCE_MIGRATIONS":       "true",
		"COOKIE_SECRET":          "env_secret",
		"MIGRATIONS":             "./env_migrations",
		"ENABLE_HTTPS":           "true",
	}
	for k, v := range envVars {
		err := os.Setenv(k, v)
		require.NoError(t, err)
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, ":8080", cfg.RunAddr)
	assert.Equal(t, 15*time.Second, cfg.ReadTimeout.Duration())
	assert.Equal(t, 20*time.Second, cfg.WriteTimeout.Duration())
	assert.Equal(t, 45*time.Minute, cfg.AccessTokenLifespan.Duration())
	assert.Equal(t, 8*time.Hour, cfg.RefreshTokenLifespan.Duration())
	assert.Equal(t, logger.LevelDebug, cfg.LoggerLevel)
	assert.Equal(t, "postgres://env:pass@localhost:5432/db", cfg.DatabaseURI)
	assert.True(t, cfg.ForceMigrations)
	assert.Equal(t, "env_secret", cfg.CookieSecret)
	assert.Equal(t, "./env_migrations", cfg.MigrationsPath)
	assert.True(t, cfg.EnableHTTPS)
	assert.Empty(t, cfg.ConfigFilePath)
}

func TestLoadConfig_FlagsOverrideEnv(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	err := os.Setenv("SERVER_ADDRESS", ":9999")
	require.NoError(t, err)
	err = os.Setenv("LOGGER_LEVEL", "WARN")
	require.NoError(t, err)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{
		"cmd",
		"-a=:8080",
		"-l=ERROR",
	}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, ":8080", cfg.RunAddr)
	assert.Equal(t, logger.LevelError, cfg.LoggerLevel)
	assert.Empty(t, cfg.ConfigFilePath)
}

func TestLoadConfig_ConfigFile(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	// Create a temporary JSON config file
	tmpFile, err := os.CreateTemp(".", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `{
		"server_address": ":7000",
		"read_timeout": "5s",
		"write_timeout": "10s",
		"access_token_lifespan": "20m",
		"refresh_token_lifespan": "4h",
		"logger_level": "DEBUG",
		"database_uri": "postgres://file:pass@localhost:5432/db",
		"force_migrations": true,
		"cookie_secret": "file_secret",
		"migrations_path": "./file_migrations",
		"enable_https": true
	}`
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "-c=" + tmpFile.Name()}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, ":7000", cfg.RunAddr)
	assert.Equal(t, 5*time.Second, cfg.ReadTimeout.Duration())
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout.Duration())
	assert.Equal(t, 20*time.Minute, cfg.AccessTokenLifespan.Duration())
	assert.Equal(t, 4*time.Hour, cfg.RefreshTokenLifespan.Duration())
	assert.Equal(t, logger.LevelDebug, cfg.LoggerLevel)
	assert.Equal(t, "postgres://file:pass@localhost:5432/db", cfg.DatabaseURI)
	assert.True(t, cfg.ForceMigrations)
	assert.Equal(t, "file_secret", cfg.CookieSecret)
	assert.Equal(t, "./file_migrations", cfg.MigrationsPath)
	assert.True(t, cfg.EnableHTTPS)
	assert.Equal(t, tmpFile.Name(), cfg.ConfigFilePath)
}

// Test that config file is loaded via CONFIG env variable
func TestLoadConfig_ConfigFileFromEnv(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	tmpFile, err := os.CreateTemp(".", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `{
		"server_address": ":7777",
		"logger_level": "WARN"
	}`
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)

	err = os.Setenv("CONFIG", tmpFile.Name())
	require.NoError(t, err)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, ":7777", cfg.RunAddr)
	assert.Equal(t, logger.LevelWarn, cfg.LoggerLevel)
	assert.Equal(t, tmpFile.Name(), cfg.ConfigFilePath)
}

// Test that flags override config file
func TestLoadConfig_FlagsOverrideConfigFile(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	tmpFile, err := os.CreateTemp(".", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `{
		"server_address": ":7000",
		"logger_level": "DEBUG",
		"database_uri": "postgres://file:pass@localhost/db"
	}`
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{
		"cmd",
		"-c=" + tmpFile.Name(),
		"-a=:9000",
		"-l=ERROR",
	}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	// Flags should override file
	assert.Equal(t, ":9000", cfg.RunAddr)
	assert.Equal(t, logger.LevelError, cfg.LoggerLevel)
	// But database_uri should come from file (since no flag)
	assert.Equal(t, "postgres://file:pass@localhost/db", cfg.DatabaseURI)
	assert.Equal(t, tmpFile.Name(), cfg.ConfigFilePath)
}

// Test invalid duration in flags
func TestLoadConfig_InvalidFlagDuration(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "-r=invalid"}

	_, err := LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration")
}

// Test that config file with numbers (seconds) as duration works
func TestLoadConfig_ConfigFileWithNumberDuration(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
		os.Unsetenv("CONFIG")
		os.Clearenv()
	}()
	os.Unsetenv("CONFIG")
	os.Clearenv()

	tmpFile, err := os.CreateTemp(".", "config*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `{
		"read_timeout": 10,
		"write_timeout": 20.5
	}`
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd", "-c=" + tmpFile.Name()}

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 10*time.Second, cfg.ReadTimeout.Duration())
	assert.Equal(t, 20*time.Second, cfg.WriteTimeout.Duration())
}
