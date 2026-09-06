// Package models defines data structures used across the client.
package models

type Tokens struct {
	UserID       int64  `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken []byte `json:"refresh_token"`
}
