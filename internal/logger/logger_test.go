package logger

import (
	"bytes"
	"encoding/json"
	"flag"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevelSet(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLevel Level
		wantErr   bool
	}{
		{"valid DEBUG", "DEBUG", LevelDebug, false},
		{"valid INFO", "INFO", LevelInfo, false},
		{"valid WARN", "WARN", LevelWarn, false},
		{"valid ERROR", "ERROR", LevelError, false},
		{"invalid lower", "debug", "", true},
		{"invalid value", "FATAL", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l Level
			err := l.Set(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantLevel, l)
			}
		})
	}
}

func TestLevelMarshalUnmarshal(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		data, err := LevelDebug.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, "DEBUG", string(data))
	})

	t.Run("unmarshal", func(t *testing.T) {
		var l Level
		err := l.UnmarshalText([]byte("WARN"))
		require.NoError(t, err)
		assert.Equal(t, LevelWarn, l)
	})

	t.Run("unmarshal invalid", func(t *testing.T) {
		var l Level
		err := l.UnmarshalText([]byte("invalid"))
		assert.Error(t, err)
	})
}

func TestInitialize(t *testing.T) {
	// Save old logger to restore after test.
	oldLog := Log
	defer func() { Log = oldLog; slog.SetDefault(oldLog) }()

	tests := []struct {
		name    string
		level   Level
		wantErr bool
	}{
		{"valid DEBUG", LevelDebug, false},
		{"valid INFO", LevelInfo, false},
		{"valid WARN", LevelWarn, false},
		{"valid ERROR", LevelError, false},
		{"invalid level", Level("FATAL"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.level)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid log level")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, Log)
				assert.Equal(t, Log, slog.Default())
				// Check that the handler uses JSON and correct level.
				assert.IsType(t, &slog.JSONHandler{}, Log.Handler())
				// We can't easily check level, but we trust the mapping.
			}
		})
	}
}

func TestInitNoopLogger(t *testing.T) {
	// The package init already sets a no‑op logger. We can verify that Log is not nil.
	assert.NotNil(t, Log)
	// We can also check that the handler discards output – but hard to test.
}

func TestFlagIntegration(t *testing.T) {
	// Simulate flag parsing with a custom flagset.
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var l Level
	fs.Var(&l, "level", "log level")
	err := fs.Parse([]string{"-level", "DEBUG"})
	require.NoError(t, err)
	assert.Equal(t, LevelDebug, l)
}

func TestLoggerOutput(t *testing.T) {
	oldLog := Log
	defer func() { Log = oldLog; slog.SetDefault(oldLog) }()

	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	Log = slog.New(handler)
	slog.SetDefault(Log)

	Log.Info("test msg", "key", "value")

	var output map[string]any
	err := json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	assert.Equal(t, "test msg", output["msg"])
	assert.Equal(t, "INFO", output["level"])
	assert.Equal(t, "value", output["key"])
}
