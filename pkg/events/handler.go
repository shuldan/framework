package events

import (
	"log/slog"

	"github.com/shuldan/framework/pkg/contracts"
)

type defaultPanicHandler struct {
	logger contracts.Logger
}

func NewDefaultPanicHandler(logger contracts.Logger) PanicHandler {
	return &defaultPanicHandler{logger: logger}
}

func (d *defaultPanicHandler) Handle(event any, listener any, panicValue any, stack []byte) {
	if d.logger == nil {
		slog.Error(
			"event eventBus panic",
			"event", event,
			"listener", listener,
			"panic_value", panicValue,
			"stack", string(stack),
		)
		return
	}
	d.logger.Critical("event eventBus panic",
		"event", event,
		"listener", listener,
		"panic_value", panicValue,
		"stack", string(stack),
	)
}

type defaultErrorHandler struct {
	logger contracts.Logger
}

func NewDefaultErrorHandler(logger contracts.Logger) ErrorHandler {
	return &defaultErrorHandler{logger: logger}
}

func (d *defaultErrorHandler) Handle(event any, listener any, err error) {
	if d.logger == nil {
		slog.Error(
			"event eventBus error",
			"event", event,
			"listener", listener,
			"error", err,
		)
		return
	}
	d.logger.Error("event eventBus error",
		"event", event,
		"listener", listener,
		"error", err,
	)
}
