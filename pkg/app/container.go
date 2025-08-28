package app

import (
	"sync"

	"github.com/shuldan/framework/pkg/contracts"
)

type container struct {
	mu        sync.RWMutex
	factories map[string]func(c contracts.DIContainer) (interface{}, error)
	instances map[string]interface{}
	resolving map[string]bool
}

func (c *container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, hasFactory := c.factories[name]
	_, hasInstance := c.instances[name]
	return hasFactory || hasInstance
}

func (c *container) Instance(name string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.instances[name]; exists {
		return ErrDuplicateInstance.WithDetail("name", name)
	}
	c.instances[name] = value
	return nil
}

func (c *container) Factory(name string, factory func(c contracts.DIContainer) (interface{}, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.factories[name]; exists {
		return ErrDuplicateFactory.WithDetail("name", name)
	}
	c.factories[name] = factory
	return nil
}

func (c *container) Resolve(name string) (interface{}, error) {
	c.mu.Lock()

	if instance, exists := c.instances[name]; exists {
		c.mu.Unlock()
		return instance, nil
	}

	if c.resolving[name] {
		c.mu.Unlock()
		return nil, ErrCircularDep.WithDetail("name", name)
	}

	factory, exists := c.factories[name]
	if !exists {
		c.mu.Unlock()
		return nil, ErrValueNotFound.WithDetail("name", name)
	}

	c.resolving[name] = true
	c.mu.Unlock()

	instance, err := factory(c)

	c.mu.Lock()

	delete(c.resolving, name)

	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	if existing, exists := c.instances[name]; exists {
		c.mu.Unlock()
		return existing, nil
	}

	c.instances[name] = instance
	c.mu.Unlock()
	return instance, nil
}
