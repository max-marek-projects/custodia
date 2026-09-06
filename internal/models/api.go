// Package models defines data structures used across the custodia system.
package models

// LoginRequest represents the JSON payload for user registration and login.
type LoginRequest struct {
	Login      string
	Password   string
	DeviceName string
}

// UserData holds user credentials for internal storage (login and password hash).
type UserData struct {
	Login        string
	PasswordHash []byte
}

// LoginResponse represents the response for user registration and login.
type LoginResponse struct {
	UserID       int64
	AccessToken  string
	RefreshToken []byte
}
