package repository

import (
	"errors"
)

// ErrUserNotFound is returned when a user login does not exist.
var ErrUserNotFound = errors.New("user not found in storage")

// ErrAlreadyInStorage is returned when attempting to insert a duplicate record.
var ErrAlreadyInStorage = errors.New("item already exists in storage")

// ErrRefreshTokenExpiredOrInvalid is returned when refresh token is expired or invalid.
var ErrRefreshTokenExpiredOrInvalid = errors.New("refresh token expired or invalid")

// ErrSecretRollbackNotPossible is returned when secret data has not enough versions for rollback.
var ErrSecretRollbackNotPossible = errors.New("no active version found to rollback")

// ErrSecretNotFound is returned when secret data was not found.
var ErrSecretNotFound = errors.New("secret not found")
