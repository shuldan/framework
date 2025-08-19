package application

import (
	"errors"
	"sync"
)

type registry struct {
	modules []Module
	mu      sync.RWMutex
}

func (r *registry) Register(module Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules = append(r.modules, module)
	return nil
}

func (r *registry) All() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Module, len(r.modules))
	copy(result, r.modules)
	return result
}

func (r *registry) Shutdown(ctx Context) error {
	var errs []error
	modules := r.All()
	for i := len(modules) - 1; i >= 0; i-- {
		if err := modules[i].Stop(ctx); err != nil {
			wrapped := ErrModuleStop.
				WithDetail("module", modules[i].Name()).
				WithCause(err)
			errs = append(errs, wrapped)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
