package app

import (
	"context"
	"sync"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type AppInfo struct {
	AppName     string
	Version     string
	Environment string
}

type appContext struct {
	ctx       context.Context
	container contracts.DIContainer
	cancel    context.CancelFunc
	info      AppInfo
	startTime time.Time
	stopTime  time.Time
	mu        sync.RWMutex
	isRunning bool
}

func newAppContext(info AppInfo, container contracts.DIContainer) *appContext {
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()
	return &appContext{
		ctx:       ctx,
		container: container,
		cancel:    cancel,
		info:      info,
		startTime: now,
		stopTime:  time.Time{},
		isRunning: true,
	}
}

func (c *appContext) Ctx() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

func (c *appContext) Container() contracts.DIContainer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.container
}

func (c *appContext) AppName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.info.AppName
}

func (c *appContext) Version() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.info.Version
}

func (c *appContext) Environment() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.info.Environment
}

func (c *appContext) StartTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.startTime
}

func (c *appContext) StopTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stopTime
}

func (c *appContext) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

func (c *appContext) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isRunning {
		c.cancel()
		c.stopTime = time.Now()
		c.isRunning = false
	}
}
