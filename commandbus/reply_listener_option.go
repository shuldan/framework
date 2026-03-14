package commandbus

// ReplyListenerOption настраивает ReplyListener при создании.
type ReplyListenerOption func(*replyListenerConfig)

type replyListenerConfig struct {
	serviceName string
}

// WithListenerServiceName задаёт имя сервиса для определения топика ответов.
func WithListenerServiceName(name string) ReplyListenerOption {
	return func(c *replyListenerConfig) { c.serviceName = name }
}
