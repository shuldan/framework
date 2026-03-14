package commandbus

import "time"

// SenderOption настраивает CommandSender при создании.
type SenderOption func(*senderConfig)

type senderConfig struct {
	replyTo string
	sender  string
	timeout time.Duration
}

// WithReplyTo задаёт имя сервиса для получения ответов.
func WithReplyTo(service string) SenderOption {
	return func(c *senderConfig) { c.replyTo = service }
}

// WithSender задаёт имя сервиса-отправителя.
func WithSender(service string) SenderOption {
	return func(c *senderConfig) { c.sender = service }
}

// WithDefaultTimeout задаёт таймаут по умолчанию для всех команд.
func WithDefaultTimeout(d time.Duration) SenderOption {
	return func(c *senderConfig) { c.timeout = d }
}

// SendOption настраивает отправку конкретной команды.
type SendOption func(*sendOptions)

type sendOptions struct {
	timeout time.Duration
	replyTo string
	headers map[string]string
}

// WithTimeout переопределяет таймаут для команды.
func WithTimeout(d time.Duration) SendOption {
	return func(o *sendOptions) { o.timeout = d }
}

// WithoutReply отключает ответ для команды (fire-and-forget).
func WithoutReply() SendOption {
	return func(o *sendOptions) { o.replyTo = "" }
}

// WithHeaders добавляет заголовки к команде.
func WithHeaders(h map[string]string) SendOption {
	return func(o *sendOptions) { o.headers = h }
}
