package eventbus

import "github.com/shuldan/events"

// OutboundConfig настраивает OutboundRelay при создании.
type OutboundConfig func(*outboundConfig)

type outboundConfig struct {
	source string
}

// WithSource задаёт имя сервиса-источника, записываемое в envelope.
func WithSource(service string) OutboundConfig {
	return func(c *outboundConfig) {
		c.source = service
	}
}

// OutboundOption настраивает отдельный маршрут пересылки.
type OutboundOption func(*outboundEntry)

// WithFilter задаёт фильтр — событие пересылается только если fn вернёт true.
func WithFilter(fn func(events.Event) bool) OutboundOption {
	return func(e *outboundEntry) {
		e.filter = fn
	}
}

// WithTransform задаёт пользовательскую сериализацию payload.
// Если задан — envelope НЕ используется, данные отправляются как есть.
func WithTransform(
	fn func(events.Event) ([]byte, error),
) OutboundOption {
	return func(e *outboundEntry) {
		e.transform = fn
	}
}
