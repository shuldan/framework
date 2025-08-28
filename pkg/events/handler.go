package events

import (
	"log/slog"

	"github.com/shuldan/framework/pkg/contracts"
)

type defaultPanicHandler struct {
	logger contracts.Logger
}

func (d *defaultPanicHandler) Handle(event any, listener any, panicValue any, stack []byte) {
	if d.logger == nil {
		slog.Error(
			"event bus panic",
			"event", event,
			"listener", listener,
			"panic_value", panicValue,
			"stack", string(stack),
		)
		return
	}
	d.logger.Critical("event bus panic",
		"event", event,
		"listener", listener,
		"panic_value", panicValue,
		"stack", string(stack),
	)
}

type defaultErrorHandler struct {
	logger contracts.Logger
}

func (d *defaultErrorHandler) Handle(event any, listener any, err error) {
	if d.logger == nil {
		slog.Error(
			"event bus error",
			"event", event,
			"listener", listener,
			"error", err,
		)
		return
	}
	d.logger.Error("event bus error",
		"event", event,
		"listener", listener,
		"error", err,
	)
}
