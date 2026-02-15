package httpserver

import "errors"

var (
	ErrEmptyBody    = errors.New("request body is empty")
	ErrBodyTooLarge = errors.New("request body too large")
	ErrInvalidJSON  = errors.New("invalid JSON")
)
