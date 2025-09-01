package contracts

import "context"

type ErrorHandler interface {
	Handle(ctx context.Context, err error) error
}

type ErrorHandlerConfig interface {
	StatusCodeMap() map[string]int
	UserMessageMap() map[string]string
	LogLevel() string
	ShowStackTrace() bool
	ShowDetails() bool
}
