package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type app struct {
	container       contracts.DIContainer
	registry        contracts.AppRegistry
	info            AppInfo
	appCtx          *appContext
	appCtxMu        sync.RWMutex
	isRunning       int32
	shutdownTimeout time.Duration
}

func WithGracefulTimeout(timeout time.Duration) func(*app) {
	return func(a *app) {
		a.shutdownTimeout = timeout
	}
}

func (a *app) Register(module contracts.AppModule) error {
	return a.registry.Register(module)
}

func (a *app) getAppCtx() *appContext {
	a.appCtxMu.RLock()
	defer a.appCtxMu.RUnlock()
	return a.appCtx
}

func (a *app) setAppCtx(ctx *appContext) {
	a.appCtxMu.Lock()
	defer a.appCtxMu.Unlock()
	a.appCtx = ctx
}

func (a *app) Run() error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 0, 1) {
		return ErrAppRun.WithDetail("reason", "application is already isRunning")
	}

	ctx := newAppContext(a.info, a.container)
	a.setAppCtx(ctx)

	for _, module := range a.registry.All() {
		if err := module.Register(a.container); err != nil {
			ctx.Stop()
			return ErrModuleRegister.
				WithDetail("module", module.Name()).
				WithCause(err)
		}
	}

	started := 0
	for _, module := range a.registry.All() {
		if err := module.Start(ctx); err != nil {
			ctx.Stop()
			a.shutdownStarted(ctx, started)
			return ErrModuleStart.
				WithDetail("module", module.Name()).
				WithCause(err)
		}
		started++
	}

	go setupSignalHandler(ctx)

	<-ctx.Ctx().Done()

	var err error
	if a.shutdownTimeout > 0 {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			if regErr := a.registry.Shutdown(ctx); regErr != nil {
				errCh <- regErr
			} else {
				errCh <- nil
			}
		}()

		select {
		case err = <-errCh:
		case <-shutdownCtx.Done():
			err = ErrAppStop.WithDetail("reason", "graceful shutdown timed out after "+a.shutdownTimeout.String())
		}
	} else {
		err = a.registry.Shutdown(ctx)
	}

	return err
}

func (a *app) shutdownStarted(appCtx contracts.AppContext, startedModulesCount int) {
	modules := a.registry.All()
	for i := startedModulesCount - 1; i >= 0; i-- {
		_ = modules[i].Stop(appCtx)
	}
}

func setupSignalHandler(ctx contracts.AppContext) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-sigChan:
		ctx.Stop()
	case <-ctx.Ctx().Done():
		return
	}
}
