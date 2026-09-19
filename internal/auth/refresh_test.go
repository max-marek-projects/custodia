package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRefreshToken(t *testing.T) {
	t.Run("returns 32 bytes and no error", func(t *testing.T) {
		token, err := GenerateRefreshToken()
		require.NoError(t, err, "unexpected error")
		assert.Len(t, token, 32, "wrong refresh token length")
	})

	t.Run("tokens are different on consecutive calls", func(t *testing.T) {
		token1, err := GenerateRefreshToken()
		require.NoError(t, err, "unexpected error")
		token2, err := GenerateRefreshToken()
		require.NoError(t, err, "unexpected error")
		assert.NotEqual(t, token1, token2, "two generated tokens are equal, expected them to be different")
	})
}

func TestHashRefreshToken(t *testing.T) {
	t.Run("returns 32-byte hash", func(t *testing.T) {
		hash := HashRefreshToken("some-token")
		assert.Len(t, hash, 32, "hash length should be 32 bytes")
	})

	t.Run("same input yields same hash", func(t *testing.T) {
		raw := "constant-token"
		hash1 := HashRefreshToken(raw)
		hash2 := HashRefreshToken(raw)
		assert.Equal(t, hash1, hash2, "hash of the same token should be identical")
	})

	t.Run("different inputs yield different hashes", func(t *testing.T) {
		hash1 := HashRefreshToken("token-a")
		hash2 := HashRefreshToken("token-b")
		assert.NotEqual(t, hash1, hash2, "hashes of different tokens should differ")
	})

	t.Run("hash is not equal to raw token", func(t *testing.T) {
		raw := "my-secret"
		hash := HashRefreshToken(raw)
		assert.NotEqual(t, hash, []byte(raw), "hash should not equal the raw token")
	})

	t.Run("empty string input", func(t *testing.T) {
		hash := HashRefreshToken("")
		assert.Len(t, hash, 32, "empty string hash should be 32 bytes")

		// SHA-256 of empty string (known value)
		expected := []byte{
			0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14,
			0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24,
			0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c,
			0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55,
		}
		assert.Equal(t, expected, hash, "hash of empty string should match known SHA-256 value")
	})
}
