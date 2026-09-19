// Package utils provides errors definitions.
package utils

import "errors"

// ErrEmptySecretKey is returned when secret key is empty string.
var ErrEmptySecretKey = errors.New("empty secret key")
