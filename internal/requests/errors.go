package requests

import (
	"errors"
)

// ErrMissingMetadata is returned when request metadata is missing
var ErrMissingMetadata = errors.New("missing metadata")

// ErrAuthorizationDataMissing is returned when metadata is present
// but does not contain authorization data
var ErrAuthorizationDataMissing = errors.New("authorization is missing")

// ErrInvalidAuthorizationData is returned when metadata is present
// but contains invalid authorization data
var ErrInvalidAuthorizationData = errors.New("invalid authorization scheme")

// ErrInvalidAuthorizationData is returned when metadata is present
// but contains empty auth token
var ErrEmptyToken = errors.New("token is empty")
