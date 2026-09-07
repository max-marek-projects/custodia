package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.design/x/clipboard"
)

// skipIfNoClipboard skips the test if clipboard initialization fails
// (e.g., no GUI or DISPLAY available in CI).
func skipIfNoClipboard(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("skipping test: clipboard not available (%v)", err)
	}
}

func TestCopyToClipboard(t *testing.T) {
	t.Run("successful copy and read back", func(t *testing.T) {
		skipIfNoClipboard(t)

		ctx := context.Background()
		data := []byte("Hello, clipboard!")
		ttl := 10 * time.Second

		err := CopyToClipboard(ctx, data, ttl)
		require.NoError(t, err, "CopyToClipboard failed")

		// Wait a moment for the write to be effective
		time.Sleep(100 * time.Millisecond)

		// Read back from clipboard and compare
		readData, err := clipboard.Read(ctx, clipboard.FmtText)
		require.NoError(t, err, "failed to read from clipboard")
		assert.Equal(t, data, readData, "copied data does not match read data")
	})

	t.Run("context cancellation", func(t *testing.T) {
		skipIfNoClipboard(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		data := []byte("should not be copied")
		ttl := time.Minute

		err := CopyToClipboard(ctx, data, ttl)
		assert.Error(t, err, "expected error when context is canceled")
		assert.Contains(t, err.Error(), "copy failed", "error should mention copy failure")
	})
}
