package app

import (
	"errors"
	"github.com/shuldan/framework/pkg/contracts"
	"testing"
)

type mockModule struct {
	name  string
	start func(ctx contracts.AppContext) error
	stop  func(ctx contracts.AppContext) error
}

func (m *mockModule) Name() string                           { return m.name }
func (m *mockModule) Register(c contracts.DIContainer) error { return nil }
func (m *mockModule) Start(ctx contracts.AppContext) error {
	if m.start == nil {
		return nil
	}
	return m.start(ctx)
}
func (m *mockModule) Stop(ctx contracts.AppContext) error {
	if m.stop == nil {
		return nil
	}
	return m.stop(ctx)
}

func TestRegistry_ShutdownWithError(t *testing.T) {
	reg := NewRegistry().(*registry)

	mod1 := &mockModule{
		name: "mod1",
		stop: func(ctx contracts.AppContext) error {
			return errors.New("stop failed")
		},
	}

	_ = reg.Register(mod1)

	ctx := newAppContext(AppInfo{}, NewContainer())

	err := reg.Shutdown(ctx)
	if err == nil {
		t.Fatal("Expected error from Shutdown")
	}

	if !errors.Is(err, ErrModuleStop) {
		t.Errorf("Expected ErrModuleStop, got %v", err)
	}
}
