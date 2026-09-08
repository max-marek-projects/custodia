// Package logger provides a global slog.Logger instance with configurable log level.
// It defines a custom Level type that implements flag.Value and TextMarshaler/Unmarshaler
// for seamless integration with command-line flags, environment variables, and JSON.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Log is the global logger instance used throughout the application.
// It is initialized with a no‑op logger (discarding all output) until
// Initialize is called with a valid level.
var Log *slog.Logger

// Level represents a log level string (DEBUG, INFO, WARN, ERROR).
type Level string

// Level constants.
const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// logLevels maps Level constants to slog.Level values.
var logLevels = map[Level]slog.Level{
	LevelDebug: slog.LevelDebug,
	LevelInfo:  slog.LevelInfo,
	LevelWarn:  slog.LevelWarn,
	LevelError: slog.LevelError,
}

// String returns the string representation of the level.
func (l Level) String() string {
	return string(l)
}

// Set implements flag.Value interface.
// It validates that the given string is one of the predefined levels.
func (l *Level) Set(value string) error {
	switch Level(value) {
	case LevelDebug, LevelInfo, LevelWarn, LevelError:
		*l = Level(value)
		return nil
	default:
		return fmt.Errorf("invalid log level: %s", value)
	}
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON and environment variables.
func (l *Level) UnmarshalText(text []byte) error {
	return l.Set(string(text))
}

// MarshalText implements encoding.TextMarshaler for JSON serialization.
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l), nil
}

// Initialize sets up the global logger (Log) with the specified log level.
// It configures a JSON handler writing to os.Stdout and also sets the
// default slog logger for package‑level functions.
//
// Parameters:
//   - level: the log level (one of LevelDebug, LevelInfo, LevelWarn, LevelError).
//
// Returns:
//   - error: nil on success, or an error if the level is invalid.
func Initialize(level Level) error {
	lvl, exists := logLevels[level]
	if !exists {
		return fmt.Errorf("failed to initialize logger: invalid log level: %s", level)
	}
	opts := &slog.HandlerOptions{Level: lvl}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Log = slog.New(handler)
	slog.SetDefault(Log)
	return nil
}

// init sets up a no‑op logger (writing to io.Discard) as the initial default.
func init() {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	handler := slog.NewJSONHandler(io.Discard, opts)
	Log = slog.New(handler)
}
