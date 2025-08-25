package events

import (
	"fmt"
	"github.com/shuldan/framework/pkg/contracts"
)

type defaultPanicHandler struct {
	logger contracts.Logger
}

func (d *defaultPanicHandler) Handle(event any, listener any, panicValue any, stack []byte) {
	if d.logger == nil {
		panic(fmt.Sprintf("event bus panic: event=%v, listener=%v, panic=%v, stack=%s",
			event, listener, panicValue, string(stack)))
		return
	}
	d.logger.Critical("event bus panic", "event", event, "listener", listener,
		"panic_value", panicValue, "stack", string(stack))
}

type defaultErrorHandler struct {
	logger contracts.Logger
}

func (d *defaultErrorHandler) Handle(event any, listener any, err error) {
	if d.logger == nil {
		panic(fmt.Sprintf("event bus error: event=%v, listener=%v, error=%v",
			event, listener, err))
		return
	}
	d.logger.Error("event bus error", "event", event, "listener", listener, "error", err)
}
