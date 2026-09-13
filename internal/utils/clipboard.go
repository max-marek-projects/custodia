// Package utils provides helper functions for system clipboard operations.
package utils

import (
	"context"
	"fmt"
	"time"

	"golang.design/x/clipboard"
)

// CopyToClipboard writes the given text data to the system clipboard.
// It initializes the clipboard backend on the first call.
// The write operation can be canceled via ctx.
// After the specified ttl (time-to-live) expires, the clipboard content is
// cleared in a background goroutine.
//
// Parameters:
//   - ctx: context for cancellation (if canceled before the write completes, an error is returned).
//   - data: the text data to copy (must be a UTF‑8 encoded byte slice).
//   - ttl: duration after which the clipboard content is automatically cleared.
//
// Returns:
//   - error: nil on success, or an error if clipboard initialization fails
//     or the write operation fails.
func CopyToClipboard(ctx context.Context, data []byte, ttl time.Duration) error {
	if err := clipboard.Init(); err != nil {
		return fmt.Errorf("clipboard init failed: %w", err)
	}
	_, err := clipboard.Write(ctx, clipboard.FmtText, data)
	if err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}
	go func() {
		time.Sleep(ttl)
		_, _ = clipboard.Write(ctx, clipboard.FmtText, []byte(""))
	}()
	return nil
}
