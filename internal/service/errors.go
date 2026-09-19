package service

import (
	"errors"
)

// ErrLoginAlreadyTaken is returned when registration login already exists.
var ErrLoginAlreadyTaken = errors.New("login already taken by other user")

// ErrWrongUsernamePassword is returned when login credentials are invalid.
var ErrWrongUsernamePassword = errors.New("wrong username or password")

// ErrRefreshTokenExpiredOrInvalid is returned when refresh token is expired or invalid.
var ErrRefreshTokenExpiredOrInvalid = errors.New("refresh token expired or invalid")

// ErrRollbackNotPossible is returned when credential rollback is not possible.
var ErrRollbackNotPossible = errors.New("rollback is not possible")

// ErrNoChanges is returned when no changes applied.
var ErrNoChanges = errors.New("no changes")

// ErrNoActionPerformed is returned when secret was not found.
var ErrSecretNotFound = errors.New("secret not found")

// ErrInvalidArgument is returned when invalid argument value was received
var ErrInvalidArgument = errors.New("invalid argument received")

// ErrSecretConflict is returned when secret name is taken.
var ErrSecretConflict = errors.New("secret name is taken")
