// Package models defines data structures used across the client and server.
package models

// Tokens represents the authentication tokens and associated user information.
// It is used for storing and transmitting access and refresh tokens.
type Tokens struct {
	// UserID is the unique identifier of the authenticated user.
	UserID int64 `json:"user_id"`

	// AccessToken is the short‑lived JWT used for API requests.
	AccessToken string `json:"access_token"`

	// RefreshToken is a long‑lived, opaque token used to obtain new access tokens.
	// It is stored as a byte slice and will be base64‑encoded in JSON.
	RefreshToken []byte `json:"refresh_token"`
}
