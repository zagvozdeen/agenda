package api

import "errors"

var (
	ErrUnauthorized = errors.New("authentication required")
	ErrInvalidToken = errors.New("invalid token")
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func invalid(message string) error   { return &Error{Code: "invalid_arguments", Message: message} }
func conflict(message string) error  { return &Error{Code: "conflict", Message: message} }
func forbidden(message string) error { return &Error{Code: "forbidden", Message: message} }
