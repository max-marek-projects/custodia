package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	t.Run("successful roundtrip", func(t *testing.T) {
		plaintext := []byte("Hello, world! This is a secret.")
		password := []byte("my-secure-password")

		salt, iv, cipherData, err := EncryptData(plaintext, password)
		require.NoError(t, err, "EncryptData failed")
		require.NotEmpty(t, salt, "salt is empty")
		require.Len(t, iv, 12, "IV length should be 12")
		require.NotEmpty(t, cipherData, "cipherData is empty")

		decrypted, err := DecryptData(salt, iv, cipherData, password)
		require.NoError(t, err, "DecryptData failed")
		assert.Equal(t, plaintext, decrypted, "decrypted data does not match original")
	})

	t.Run("different passwords fail", func(t *testing.T) {
		plaintext := []byte("secret data")
		password := []byte("correct-password")
		wrongPassword := []byte("wrong-password")

		salt, iv, cipherData, err := EncryptData(plaintext, password)
		require.NoError(t, err)

		_, err = DecryptData(salt, iv, cipherData, wrongPassword)
		assert.Error(t, err, "decryption with wrong password should fail")
		assert.Contains(t, err.Error(), "decryption failed", "error should mention decryption failure")
	})

	t.Run("tampered ciphertext fails", func(t *testing.T) {
		plaintext := []byte("tamper me")
		password := []byte("password")

		salt, iv, cipherData, err := EncryptData(plaintext, password)
		require.NoError(t, err)

		// Corrupt the ciphertext by flipping a byte
		if len(cipherData) > 0 {
			cipherData[0] ^= 0xFF
		}
		_, err = DecryptData(salt, iv, cipherData, password)
		assert.Error(t, err, "tampered ciphertext should fail authentication")
	})

	t.Run("invalid IV length", func(t *testing.T) {
		salt := []byte("some-salt-16bytes") // 16 bytes
		iv := []byte("short")               // 5 bytes
		cipherData := []byte("some data")
		password := []byte("pass")

		_, err := DecryptData(salt, iv, cipherData, password)
		assert.Error(t, err, "should reject invalid IV length")
		assert.Contains(t, err.Error(), "invalid IV length")
	})

	t.Run("empty salt", func(t *testing.T) {
		salt := []byte{}
		iv := make([]byte, 12)
		cipherData := []byte("data")
		password := []byte("pass")

		_, err := DecryptData(salt, iv, cipherData, password)
		assert.Error(t, err, "should reject empty salt")
		assert.Equal(t, "salt is empty", err.Error())
	})

	t.Run("empty data", func(t *testing.T) {
		plaintext := []byte{}
		password := []byte("password")

		salt, iv, cipherData, err := EncryptData(plaintext, password)
		require.NoError(t, err)

		decrypted, err := DecryptData(salt, iv, cipherData, password)
		require.NoError(t, err)
		assert.Empty(t, decrypted, "decrypted empty data should be empty")
	})
}
