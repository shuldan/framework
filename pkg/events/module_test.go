package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/app"
	"github.com/shuldan/framework/pkg/config"
	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

func TestModule_Register(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(contracts.DIContainer)
		expectError bool
	}{
		{
			name: "register without logger",
			setupFunc: func(c contracts.DIContainer) {

			},
			expectError: false,
		},
		{
			name: "register with logger",
			setupFunc: func(c contracts.DIContainer) {
				logger := &mockLogger{}
				_ = c.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)
			},
			expectError: false,
		},
		{
			name: "register with invalid logger",
			setupFunc: func(c contracts.DIContainer) {
				_ = c.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), "not a logger")
			},
			expectError: false,
		},
		{
			name: "register with eventBusConfig from file",
			setupFunc: func(c contracts.DIContainer) {
				cfg := config.NewMapConfig(map[string]interface{}{
					"events": map[string]interface{}{
						"async_mode":   true,
						"worker_count": 5,
					},
				})
				_ = c.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), cfg)
			},
			expectError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := app.NewContainer()
			tt.setupFunc(container)
			m := NewModule()
			err := m.Register(container)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError {
				if !container.Has(reflect.TypeOf((*contracts.EventBus)(nil)).Elem()) {
					t.Error("event bus should be registered")
				}
			}
		})
	}
}

func TestModule_GetEventBusOptions(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(contracts.DIContainer)
		expectedAsync  bool
		expectedWorker int
	}{
		{
			name: "options from eventBusConfig file",
			setupFunc: func(c contracts.DIContainer) {
				cfg := config.NewMapConfig(map[string]interface{}{
					"events": map[string]interface{}{
						"async_mode":   true,
						"worker_count": 10,
					},
				})
				_ = c.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), cfg)
			},
			expectedAsync:  true,
			expectedWorker: 10,
		},
		{
			name: "default options when no eventBusConfig",
			setupFunc: func(c contracts.DIContainer) {

			},
			expectedAsync:  false,
			expectedWorker: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := app.NewContainer()
			tt.setupFunc(container)

			m := &module{}
			options := m.getEventBusOptions(container, nil)

			testConfig := &eventBusConfig{
				asyncMode:   false,
				workerCount: 1,
			}

			for _, opt := range options {
				opt(testConfig)
			}

			if testConfig.asyncMode != tt.expectedAsync {
				t.Errorf("expected asyncMode %v, got %v", tt.expectedAsync, testConfig.asyncMode)
			}

			if testConfig.workerCount != tt.expectedWorker {
				t.Errorf("expected workerCount %d, got %d", tt.expectedWorker, testConfig.workerCount)
			}
		})
	}
}

func TestModule_Start(t *testing.T) {
	container := app.NewContainer()
	m := NewModule()

	err := m.Register(container)
	if err != nil {
		t.Fatalf("failed to register module: %v", err)
	}

	ctx := newMockAppContext(container)

	err = m.Start(ctx)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}
}

func TestModule_Stop(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(contracts.DIContainer)
		expectError bool
		errorType   error
	}{
		{
			name: "stop with valid bus",
			setupFunc: func(c contracts.DIContainer) {
				bus := New()
				_ = c.Instance(reflect.TypeOf((*contracts.EventBus)(nil)).Elem(), bus)
			},
			expectError: false,
		},
		{
			name: "stop without bus",
			setupFunc: func(c contracts.DIContainer) {

			},
			expectError: true,
			errorType:   ErrBusNotFound,
		},
		{
			name: "stop with invalid bus type",
			setupFunc: func(c contracts.DIContainer) {
				_ = c.Instance(reflect.TypeOf((*contracts.EventBus)(nil)).Elem(), "not a bus")
			},
			expectError: true,
			errorType:   ErrInvalidBusInstance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := app.NewContainer()
			tt.setupFunc(container)

			m := NewModule()
			ctx := newMockAppContext(container)

			err := m.Stop(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error %v, got %v", tt.errorType, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestModule_FullLifecycle(t *testing.T) {
	container := app.NewContainer()

	logger := &mockLogger{}
	_ = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)

	cfg := config.NewMapConfig(map[string]interface{}{
		"events": map[string]interface{}{
			"async_mode":   true,
			"worker_count": 2,
		},
	})
	_ = container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), cfg)

	m := NewModule()
	err := m.Register(container)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := newMockAppContext(container)

	err = m.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	bus, err := container.Resolve(reflect.TypeOf((*contracts.EventBus)(nil)).Elem())
	if err != nil {
		t.Fatalf("failed to resolve event bus: %v", err)
	}

	eventBus, ok := bus.(contracts.EventBus)
	if !ok {
		t.Fatal("bus is not EventBus type")
	}

	listener := &TestEventListener{}
	err = eventBus.Subscribe((*TestEvent)(nil), listener)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	event := TestEvent{Message: "test", Value: 42}
	err = eventBus.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	listener.Mutex.Lock()
	if !listener.Called {
		t.Error("listener was not called")
	}
	listener.Mutex.Unlock()

	err = m.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

type mockAppContext struct {
	ctx       context.Context
	container contracts.DIContainer
	appName   string
	version   string
	env       string
	startTime time.Time
	stopTime  time.Time
	running   bool
	mu        sync.RWMutex
}

func (m *mockAppContext) AppRegistry() contracts.AppRegistry {
	return nil
}

func newMockAppContext(container contracts.DIContainer) *mockAppContext {
	return &mockAppContext{
		ctx:       context.Background(),
		container: container,
		appName:   "test-app",
		version:   "1.0.0",
		env:       "test",
		startTime: time.Now(),
		running:   true,
	}
}

func (m *mockAppContext) ParentContext() context.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx
}

func (m *mockAppContext) Container() contracts.DIContainer {
	return m.container
}

func (m *mockAppContext) AppName() string {
	return m.appName
}

func (m *mockAppContext) Version() string {
	return m.version
}

func (m *mockAppContext) Environment() string {
	return m.env
}

func (m *mockAppContext) StartTime() time.Time {
	return m.startTime
}

func (m *mockAppContext) StopTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopTime
}

func (m *mockAppContext) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *mockAppContext) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	m.stopTime = time.Now()
}

func TestModule_ConcurrentOperations(t *testing.T) {
	bus, m, ctx := setupEventBus(t)
	defer func() {
		err := m.Stop(ctx)
		if err != nil {
			t.Fatalf("Stop failed: %v", err)
		}
	}()

	errs := runConcurrentOperations(t, bus)

	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("concurrent operation failed: %v", err)
		}
	}
}

func TestModule_ErrorScenarios(t *testing.T) {
	t.Run("stop with closed bus", func(t *testing.T) {
		container := app.NewContainer()
		bus := New()
		_ = container.Instance(reflect.TypeOf((*contracts.EventBus)(nil)).Elem(), bus)

		_ = bus.Close()

		m := NewModule()
		ctx := newMockAppContext(container)

		err := m.Stop(ctx)
		if err != nil {
			t.Errorf("Stop should handle closed bus gracefully: %v", err)
		}
	})

	t.Run("register with factory error", func(t *testing.T) {
		container := app.NewContainer()

		_ = container.Factory(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), func(c contracts.DIContainer) (interface{}, error) {
			return nil, fmt.Errorf("factory error")
		})

		m := NewModule()
		err := m.Register(container)

		if err != nil {
			t.Errorf("Register should not fail: %v", err)
		}
	})
}

func runConcurrentOperations(t *testing.T, bus contracts.EventBus) []error {
	t.Helper()

	var wg sync.WaitGroup
	errsCh := make(chan error, 100)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			listener := &TestEventListener{}
			err := bus.Subscribe((*TestEvent)(nil), listener)
			if err != nil {
				errsCh <- err
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			event := TestEvent{Message: "concurrent", Value: val}
			err := bus.Publish(context.Background(), event)
			if err != nil {
				errsCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errsCh)

	errs := make([]error, 0, cap(errsCh))
	for err := range errsCh {
		errs = append(errs, err)
	}

	return errs
}

func setupEventBus(t *testing.T) (contracts.EventBus, contracts.AppModule, *mockAppContext) {
	t.Helper()

	container := app.NewContainer()

	logger := &mockLogger{}
	_ = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)

	cfg := config.NewMapConfig(map[string]interface{}{
		"events": map[string]interface{}{
			"async_mode":   true,
			"worker_count": 5,
		},
	})
	_ = container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), cfg)

	m := NewModule()
	err := m.Register(container)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := newMockAppContext(container)
	err = m.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	bus, err := container.Resolve(reflect.TypeOf((*contracts.EventBus)(nil)).Elem())
	if err != nil {
		t.Fatalf("failed to resolve event bus: %v", err)
	}

	return bus.(contracts.EventBus), m, ctx
}
