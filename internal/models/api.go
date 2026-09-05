// Package models defines data structures used across the loyalty system.
package models

// LoginRequest represents the JSON payload for user registration and login.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// UserData holds user credentials for internal storage (login and password hash).
type UserData struct {
	Login        string
	PasswordHash string
}
