// Package auth provides helpers for generating and validating tokens.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// GenerateRefreshToken generates a new raw refresh token.
//
// Returns:
//   - []byte: a 32-byte cryptographically secure random slice.
//   - error: If reading from the random source fails, a non-nil error is returned.
func GenerateRefreshToken() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// HashRefreshToken computes a SHA-256 hash.
// This function is typically used to store a hashed version of the token
// in a database for secure comparison during validation.
//
// Parameters:
//   - rawToken: raw token string.
//
// Returns:
//   - []byte: the hash as a 32-byte slice.
func HashRefreshToken(rawToken string) []byte {
	h := sha256.Sum256([]byte(rawToken))
	return h[:]
}
