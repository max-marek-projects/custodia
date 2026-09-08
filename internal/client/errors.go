package client

import "errors"

// Auth errors
var (
	ErrLoginAlreadyTaken   = errors.New("user with this login already exists")
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrRefreshTokenExpired = errors.New("refresh token expired, please login again")
)

// Session errors
var (
	ErrSessionRequired = errors.New("this operation requires an active session")
	ErrNoChanges       = errors.New("no changes were made")
)

// Secret errors
var (
	ErrSecretNotFound            = errors.New("secret not found")
	ErrSecretAlreadyExists       = errors.New("secret with this name already exists")
	ErrSecretRollbackNotPossible = errors.New("cannot rollback secret: only one version exists")
)
