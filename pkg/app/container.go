package app

import (
	"sync"

	"github.com/shuldan/framework/pkg/contracts"
)

type container struct {
	mu        sync.RWMutex
	factories map[string]func(c contracts.DIContainer) (interface{}, error)
	instances map[string]interface{}
}

func NewContainer() contracts.DIContainer {
	return &container{
		factories: make(map[string]func(c contracts.DIContainer) (interface{}, error)),
		instances: make(map[string]interface{}),
	}
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
	return c.resolveWithStack(name, make(map[string]bool))
}

func (c *container) resolveWithStack(name string, resolving map[string]bool) (interface{}, error) {
	c.mu.RLock()
	if instance, exists := c.instances[name]; exists {
		c.mu.RUnlock()
		return instance, nil
	}
	c.mu.RUnlock()

	if resolving[name] {
		return nil, ErrCircularDep.WithDetail("name", name)
	}

	c.mu.RLock()
	factory, exists := c.factories[name]
	c.mu.RUnlock()

	if !exists {
		return nil, ErrValueNotFound.WithDetail("name", name)
	}

	resolving[name] = true
	defer func() {
		delete(resolving, name)
	}()

	proxy := &containerProxy{
		container: c,
		resolving: resolving,
	}

	instance, err := factory(proxy)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, exists := c.instances[name]; exists {
		return existing, nil
	}

	c.instances[name] = instance
	return instance, nil
}

type containerProxy struct {
	container contracts.DIContainer
	resolving map[string]bool
}

func (cp *containerProxy) Has(name string) bool {
	return cp.container.Has(name)
}

func (cp *containerProxy) Instance(name string, value interface{}) error {
	return cp.container.Instance(name, value)
}

func (cp *containerProxy) Factory(name string, factory func(c contracts.DIContainer) (interface{}, error)) error {
	return cp.container.Factory(name, factory)
}

func (cp *containerProxy) Resolve(name string) (interface{}, error) {
	if containerImpl, ok := cp.container.(*container); ok {
		return containerImpl.resolveWithStack(name, cp.resolving)
	}
	return cp.container.Resolve(name)
}
