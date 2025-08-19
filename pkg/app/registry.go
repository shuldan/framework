package app

import (
	"errors"
	"github.com/shuldan/framework/pkg/contracts"
	"sync"
)

type registry struct {
	modules []contracts.AppModule
	mu      sync.RWMutex
}

func (r *registry) Register(module contracts.AppModule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules = append(r.modules, module)
	return nil
}

func (r *registry) All() []contracts.AppModule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]contracts.AppModule, len(r.modules))
	copy(result, r.modules)
	return result
}

func (r *registry) Shutdown(ctx contracts.AppContext) error {
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
