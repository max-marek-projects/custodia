package client

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	ErrSecretTooLarge            = errors.New("secret data is too large")
)

// isConnectionError reports whether err is a gRPC error caused by the
// server being unreachable, as opposed to the server rejecting the request.
// Only these errors trigger the offline cache fallback.
func isConnectionError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
		return true
	default:
		return false
	}
}
