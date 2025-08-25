package events

type PanicHandler interface {
	Handle(event any, listener any, panicValue any, stack []byte)
}

type ErrorHandler interface {
	Handle(event any, listener any, err error)
}

type Option func(*config)

type config struct {
	panicHandler PanicHandler
	errorHandler ErrorHandler
	asyncMode    bool
	workerCount  int
}

func WithPanicHandler(h PanicHandler) Option {
	return func(c *config) {
		c.panicHandler = h
	}
}

func WithErrorHandler(h ErrorHandler) Option {
	return func(c *config) {
		c.errorHandler = h
	}
}

func WithAsyncMode(async bool) Option {
	return func(c *config) {
		c.asyncMode = async
	}
}

func WithWorkerCount(count int) Option {
	return func(c *config) {
		if count < 1 {
			count = 1
		}
		c.workerCount = count
	}
}
