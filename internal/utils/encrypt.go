// Package utils provides cryptographic helpers for encrypting and decrypting
// arbitrary data using a password (PBKDF2 + AES-GCM).
package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptData encrypts the given data using a password.
// It generates a random 16-byte salt and a random 12-byte nonce (IV).
// The encryption uses PBKDF2 (SHA‑256, 200,000 iterations) to derive a 32-byte key,
// then AES‑GCM to encrypt the data, producing a ciphertext that includes a 16‑byte
// authentication tag.
//
// Parameters:
//   - data: the plaintext to encrypt (as a byte slice).
//   - password: the password used for key derivation.
//
// Returns:
//   - salt: 16‑byte random salt.
//   - iv: 12‑byte random nonce (IV).
//   - cipherData: the encrypted data (includes the authentication tag).
//   - err: non‑nil if random generation or encryption fails.
func EncryptData(data []byte, password []byte) (salt, iv, cipherData []byte, err error) {
	// Generate salt
	salt = make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate IV
	iv = make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key(password, salt, 200000, 32, sha256.New)

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt and seal
	cipherData = gcm.Seal(nil, iv, data, nil)
	return salt, iv, cipherData, nil
}

// DecryptData decrypts data that was previously encrypted with EncryptData.
// It verifies the authentication tag and returns the original plaintext.
//
// Parameters:
//   - salt: the salt used during encryption.
//   - iv: the nonce (IV) used during encryption (must be 12 bytes).
//   - cipherData: the encrypted data (including the authentication tag).
//   - password: the password used for key derivation.
//
// Returns:
//   - []byte: the decrypted plaintext.
//   - error: non‑nil if the salt is empty, IV length is invalid,
//     key derivation fails, or authentication fails.
func DecryptData(salt, iv, cipherData, password []byte) ([]byte, error) {
	if len(salt) == 0 {
		return nil, errors.New("salt is empty")
	}
	if len(iv) != 12 {
		return nil, fmt.Errorf("invalid IV length: expected 12, got %d", len(iv))
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key(password, salt, 200000, 32, sha256.New)

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, iv, cipherData, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return plaintext, nil
}
