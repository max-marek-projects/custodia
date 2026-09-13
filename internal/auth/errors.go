package auth

import "errors"

// ErrUnexpectedSigningMethod is returned when a unexpected signing method was received.
var ErrUnexpectedSigningMethod = errors.New("unexpected signing method")

// ErrInvalidToken is returned when token is not valid.
var ErrInvalidToken = errors.New("invalid token")

// ErrEmptyUserID is returned when user id extracted from token is empty.
var ErrEmptyUserID = errors.New("userID is empty")
