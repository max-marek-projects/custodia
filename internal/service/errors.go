package service

import (
	"errors"
)

// ErrLoginAlreadyTaken is returned when registration login already exists.
var ErrLoginAlreadyTaken = errors.New("login already taken by other user")

// ErrWrongUsernamePassword is returned when login credentials are invalid.
var ErrWrongUsernamePassword = errors.New("wrong username or password")
