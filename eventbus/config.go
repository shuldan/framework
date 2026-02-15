package eventbus

import "github.com/shuldan/events"

// Config holds event dispatcher settings.
type Config struct {
	Async      bool
	Workers    int
	BufferSize int
	Ordered    bool
}

func buildOpts(cfg Config) []events.Option {
	opts := make([]events.Option, 0, 4) //nolint:mnd

	if cfg.Async {
		opts = append(opts, events.WithAsyncMode())
	} else {
		opts = append(opts, events.WithSyncMode())
	}

	if cfg.Workers > 0 {
		opts = append(opts, events.WithWorkerCount(cfg.Workers))
	}

	if cfg.BufferSize > 0 {
		opts = append(opts, events.WithBufferSize(cfg.BufferSize))
	}

	if cfg.Ordered {
		opts = append(opts, events.WithOrderedDelivery())
	}

	return opts
}
