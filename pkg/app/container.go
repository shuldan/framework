package app

import (
	"reflect"
	"sync"

	"github.com/shuldan/framework/pkg/contracts"
)

type container struct {
	mu        sync.RWMutex
	factories map[reflect.Type]func(c contracts.DIContainer) (interface{}, error)
	instances map[reflect.Type]interface{}
}

func NewContainer() contracts.DIContainer {
	return &container{
		factories: make(map[reflect.Type]func(c contracts.DIContainer) (interface{}, error)),
		instances: make(map[reflect.Type]interface{}),
	}
}

func (c *container) Has(abstract reflect.Type) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, hasFactory := c.factories[abstract]
	_, hasInstance := c.instances[abstract]
	return hasFactory || hasInstance
}

func (c *container) Instance(abstract reflect.Type, concrete interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.instances[abstract]; exists {
		return ErrDuplicateInstance.WithDetail("type", abstract.String())
	}
	c.instances[abstract] = concrete
	return nil
}

func (c *container) Factory(abstract reflect.Type, factory func(c contracts.DIContainer) (interface{}, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.factories[abstract]; exists {
		return ErrDuplicateFactory.WithDetail("type", abstract.String())
	}
	c.factories[abstract] = factory
	return nil
}

func (c *container) Resolve(abstract reflect.Type) (interface{}, error) {
	return c.resolveWithStack(abstract, make(map[reflect.Type]bool))
}

func (c *container) resolveWithStack(abstract reflect.Type, resolving map[reflect.Type]bool) (interface{}, error) {
	c.mu.RLock()
	if instance, exists := c.instances[abstract]; exists {
		c.mu.RUnlock()
		return instance, nil
	}
	c.mu.RUnlock()

	if resolving[abstract] {
		return nil, ErrCircularDep.WithDetail("type", abstract.String())
	}

	c.mu.RLock()
	factory, exists := c.factories[abstract]
	c.mu.RUnlock()

	if !exists {
		return nil, ErrValueNotFound.WithDetail("type", abstract.String())
	}

	resolving[abstract] = true
	defer func() {
		delete(resolving, abstract)
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

	if existing, exists := c.instances[abstract]; exists {
		return existing, nil
	}

	c.instances[abstract] = instance
	return instance, nil
}

type containerProxy struct {
	container contracts.DIContainer
	resolving map[reflect.Type]bool
}

func (cp *containerProxy) Has(abstract reflect.Type) bool {
	return cp.container.Has(abstract)
}

func (cp *containerProxy) Instance(abstract reflect.Type, concrete interface{}) error {
	return cp.container.Instance(abstract, concrete)
}

func (cp *containerProxy) Factory(abstract reflect.Type, factory func(c contracts.DIContainer) (interface{}, error)) error {
	return cp.container.Factory(abstract, factory)
}

func (cp *containerProxy) Resolve(abstract reflect.Type) (interface{}, error) {
	if containerImpl, ok := cp.container.(*container); ok {
		return containerImpl.resolveWithStack(abstract, cp.resolving)
	}
	return cp.container.Resolve(abstract)
}
