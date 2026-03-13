package eventbus

// InboundOption настраивает InboundRelay при создании.
type InboundOption func(*inboundConfig)

type inboundConfig struct {
	service string
}

// WithServiceName задаёт имя текущего сервиса для фильтрации собственных событий.
func WithServiceName(name string) InboundOption {
	return func(c *inboundConfig) {
		c.service = name
	}
}
