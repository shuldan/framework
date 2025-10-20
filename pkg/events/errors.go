package events

import "github.com/shuldan/framework/pkg/errors"

var newEventCode = errors.WithPrefix("EVENTS")

var (
	ErrInvalidListener         = newEventCode().New("listener must be func(context.ParentContext, T) error or have Handle(context.ParentContext, T) error method")
	ErrInvalidListenerFunction = newEventCode().New("listener function must have signature func(context.ParentContext, T) error")
	ErrInvalidListenerMethod   = newEventCode().New("Handle method must have signature Handle(context.ParentContext, T) error")
	ErrInvalidEventType        = newEventCode().New("eventType must be a pointer to struct, e.g. (*MyEvent)(nil)")
	ErrBusClosed               = newEventCode().New("cannot subscribe: event eventBus is closed")
	ErrPublishOnClosedBus      = newEventCode().New("cannot publish: event eventBus is closed")
	ErrInvalidBusInstance      = newEventCode().New("events eventBus instance must be a Bus interface")
	ErrBusNotFound             = newEventCode().New("events eventBus not found")
	ErrLoggerNotFound          = newEventCode().New("logger not found")
	ErrLoggerRequired          = newEventCode().New("logger required")
	ErrInvalidLoggerInstance   = newEventCode().New("logger instance must be a Logger interface")
	ErrEventChannelBlocked     = newEventCode().New("event channel blocked")
)
