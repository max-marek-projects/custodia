// Package auth provides helpers for generating and validation of tokens.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
)

// GenerateRefreshToken создаёт новый сырой refresh-токен
func GenerateRefreshToken() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func HashRefreshToken(rawToken string) []byte {
	h := sha256.Sum256([]byte(rawToken))
	return h[:]
}
