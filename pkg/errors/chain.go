package errors

import (
	"context"

	"github.com/shuldan/framework/pkg/contracts"
)

type ChainErrorHandler struct {
	handlers []contracts.ErrorHandler
}

func NewChainErrorHandler(handlers ...contracts.ErrorHandler) *ChainErrorHandler {
	return &ChainErrorHandler{
		handlers: handlers,
	}
}

func (c *ChainErrorHandler) Add(handler contracts.ErrorHandler) *ChainErrorHandler {
	c.handlers = append(c.handlers, handler)
	return c
}

func (c *ChainErrorHandler) Handle(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	for _, handler := range c.handlers {
		if handler.Handle(ctx, err) == nil {
			return nil
		}
	}
	return err
}
