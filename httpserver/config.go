package httpserver

import "time"

type Config struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func (c Config) withDefaults() Config {
	if c.Port == 0 {
		c.Port = 8080
	}

	if c.ReadTimeout == 0 {
		c.ReadTimeout = 15 * time.Second
	}

	if c.WriteTimeout == 0 {
		c.WriteTimeout = 15 * time.Second
	}

	if c.IdleTimeout == 0 {
		c.IdleTimeout = 60 * time.Second
	}

	return c
}
