package queue

import (
	"fmt"
	"github.com/shuldan/framework/pkg/contracts"
)

type PanicHandler interface {
	Handle(job any, consumer any, panicValue any, stack []byte)
}

type ErrorHandler interface {
	Handle(job any, consumer any, err error)
}

type defaultPanicHandler struct{ logger contracts.Logger }

func (d *defaultPanicHandler) Handle(job any, consumer any, panicValue any, stack []byte) {
	if d.logger == nil {
		panic(fmt.Sprintf("queue panic: job=%v, consumer=%v, panic=%v, stack=%s", job, consumer, panicValue, string(stack)))
		return
	}
	if job == nil {
		d.logger.Critical("queue panic", "consumer", consumer, "panic", panicValue, "stack", string(stack))
		return
	}
	d.logger.Critical("queue panic", "job", job, "consumer", consumer, "panic", panicValue, "stack", string(stack))
}

type defaultErrorHandler struct{ logger contracts.Logger }

func (d *defaultErrorHandler) Handle(job any, consumer any, err error) {
	if d.logger == nil {
		panic(fmt.Sprintf("queue error: job=%v, consumer=%v, error=%v", job, consumer, err))
		return
	}
	if job == nil {
		d.logger.Error("queue error", "consumer", consumer, "error", err)
		return
	}
	d.logger.Error("queue error", "job", job, "consumer", consumer, "error", err)
}
