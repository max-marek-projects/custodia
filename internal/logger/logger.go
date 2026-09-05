// Package logger provides a global slog.Logger instance with configurable log level.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Log is the global logger instance. Initially a no-op logger.
var Log *slog.Logger

// Initialize sets up the global logger with the specified log level.
// Parameters:
//   - level: log level from slog library.
//
// Returns an error if parsing the level or building the logger fails.

var logLevels = map[string]slog.Level{
	"DEBUG": slog.LevelDebug,
	"INFO":  slog.LevelInfo,
	"WARN":  slog.LevelWarn,
	"ERROR": slog.LevelError,
}

func Initialize(level string) error {
	lvl, exists := logLevels[level]
	if !exists {
		return fmt.Errorf("failed to initialize logger. error level is invalid: %s", level)
	}
	opts := &slog.HandlerOptions{Level: lvl}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Log = slog.New(handler)
	slog.SetDefault(Log)
	return nil
}

// initialize default logger
func init() {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	handler := slog.NewJSONHandler(io.Discard, opts)
	Log = slog.New(handler)
}
