package events

import "github.com/shuldan/framework/pkg/errors"

var newEventCode = errors.WithPrefix("EVENTS")

var (
	ErrInvalidListener         = newEventCode().New("listener must be func(context.Context, T) error or have Handle(context.Context, T) error method")
	ErrInvalidListenerFunction = newEventCode().New("listener function must have signature func(context.Context, T) error")
	ErrInvalidListenerMethod   = newEventCode().New("Handle method must have signature Handle(context.Context, T) error")
	ErrInvalidEventType        = newEventCode().New("eventType must be a pointer to struct, e.g. (*MyEvent)(nil)")
	ErrBusClosed               = newEventCode().New("cannot subscribe: event bus is closed")
	ErrPublishOnClosedBus      = newEventCode().New("cannot publish: event bus is closed")
	ErrInvalidBusInstance      = newEventCode().New("events bus instance must be a Bus interface")
	ErrBusNotFound             = newEventCode().New("events bus not found")
	ErrLoggerNotFound          = newEventCode().New("logger not found")
	ErrLoggerRequired          = newEventCode().New("logger required")
	ErrInvalidLoggerInstance   = newEventCode().New("logger instance must be a Logger interface")
)
