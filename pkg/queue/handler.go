package queue

import (
	"log/slog"

	"github.com/shuldan/framework/pkg/contracts"
)

type PanicHandler interface {
	Handle(job any, consumer any, panicValue any, stack []byte)
}

type ErrorHandler interface {
	Handle(job any, consumer any, err error)
}

type defaultPanicHandler struct{ logger contracts.Logger }

func NewDefaultPanicHandler(logger contracts.Logger) PanicHandler {
	return &defaultPanicHandler{
		logger: logger,
	}
}

func (d *defaultPanicHandler) Handle(job any, consumer any, panicValue any, stack []byte) {
	if d.logger == nil {
		slog.Error(
			"queue panic",
			"job", job,
			"consumer", consumer,
			"panic", panicValue,
			"stack", string(stack),
		)
		return
	}
	if job == nil {
		d.logger.Critical("queue panic", "consumer", consumer, "panic", panicValue, "stack", string(stack))
		return
	}
	d.logger.Critical("queue panic", "job", job, "consumer", consumer, "panic", panicValue, "stack", string(stack))
}

type defaultErrorHandler struct{ logger contracts.Logger }

func NewDefaultErrorHandler(logger contracts.Logger) ErrorHandler {
	return &defaultErrorHandler{
		logger: logger,
	}
}

func (d *defaultErrorHandler) Handle(job any, consumer any, err error) {
	if d.logger == nil {
		slog.Error(
			"queue error: job=%v, consumer=%v, error=%v",
			"job", job,
			"consumer", consumer,
			"error", err,
		)
		return
	}
	if job == nil {
		d.logger.Error("queue error", "consumer", consumer, "error", err)
		return
	}
	d.logger.Error("queue error", "job", job, "consumer", consumer, "error", err)
}
