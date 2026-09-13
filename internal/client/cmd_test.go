package client

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to capture stdout and restore it after test
func captureStdout(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	buf := &bytes.Buffer{}
	go func() {
		_, _ = io.Copy(buf, r)
	}()
	return buf, func() {
		w.Close()
		os.Stdout = old
	}
}

// helper to simulate stdin input
func withStdin(t *testing.T, input string) func() {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r
	go func() {
		defer w.Close()
		_, _ = w.Write([]byte(input))
	}()
	return func() {
		r.Close()
		os.Stdin = oldStdin
	}
}

func TestReadStringValue(t *testing.T) {
	// Test with flag value provided
	t.Run("from flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("name", "", "")
		err := cmd.Flags().Set("name", "flag-value")
		require.NoError(t, err)

		val, err := ReadStringValue(cmd, "name")
		assert.NoError(t, err)
		assert.Equal(t, "flag-value", val)
	})

	// Test with flag empty, read from stdin
	t.Run("from stdin", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("name", "", "")

		// Simulate stdin input
		cleanup := withStdin(t, "stdin-value\n")
		defer cleanup()

		// Capture stdout to avoid printing prompt
		_, restore := captureStdout(t)
		defer restore()

		val, err := ReadStringValue(cmd, "name")
		assert.NoError(t, err)
		assert.Equal(t, "stdin-value", val)
	})

	// Test error when flag does not exist
	t.Run("invalid flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("name", "", "")
		_, err := ReadStringValue(cmd, "nonexistent")
		assert.Error(t, err)
	})
}

func TestReadMapValue(t *testing.T) {
	// Test with valid JSON from flag
	t.Run("from flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("meta", "", "")
		err := cmd.Flags().Set("meta", `{"key":"value"}`)
		require.NoError(t, err)

		meta, err := ReadMapValue(cmd, "meta")
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"key": "value"}, meta)
	})

	// Test with empty flag, read JSON from stdin
	t.Run("from stdin", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("meta", "", "")

		cleanup := withStdin(t, `{"foo":"bar"}`+"\n")
		defer cleanup()

		_, restore := captureStdout(t)
		defer restore()

		meta, err := ReadMapValue(cmd, "meta")
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"foo": "bar"}, meta)
	})

	// Test invalid JSON
	t.Run("invalid JSON", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("meta", "", "")
		err := cmd.Flags().Set("meta", "not-json")
		require.NoError(t, err)

		_, err = ReadMapValue(cmd, "meta")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse metadata")
	})

	// Test invalid flag
	t.Run("invalid flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		_, err := ReadMapValue(cmd, "nonexistent")
		assert.Error(t, err)
	})
}
