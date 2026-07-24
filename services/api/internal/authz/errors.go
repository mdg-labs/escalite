package authz

import "errors"

var (
	// ErrForbidden is returned when the user is authenticated but lacks access.
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound is returned when the requested resource does not exist in the user's org.
	ErrNotFound = errors.New("not found")
)
