package api

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrUnauthorized   = errors.New("authentication required")
	ErrInvalidToken   = errors.New("invalid token")
)
