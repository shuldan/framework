package commandbus

import "github.com/shuldan/commands"

// Config хранит настройки командной шины.
type Config struct {
	Async      bool
	Workers    int
	BufferSize int
	Ordered    bool
}

func buildOpts(cfg Config) []commands.Option {
	opts := make([]commands.Option, 0, 4) //nolint:mnd

	if cfg.Async {
		opts = append(opts, commands.WithAsyncMode())
	} else {
		opts = append(opts, commands.WithSyncMode())
	}

	if cfg.Workers > 0 {
		opts = append(opts, commands.WithWorkerCount(cfg.Workers))
	}

	if cfg.BufferSize > 0 {
		opts = append(opts, commands.WithBufferSize(cfg.BufferSize))
	}

	if cfg.Ordered {
		opts = append(opts, commands.WithOrderedDelivery())
	}

	return opts
}
