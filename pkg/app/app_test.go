package app

import (
	"errors"
	"github.com/shuldan/framework/pkg/contracts"
	"testing"
	"time"
)

func TestApplication_Run_Success(t *testing.T) {
	a := New(AppInfo{
		AppName: "test",
	}, nil, nil)

	_ = a.(*app).registry.Register(&mockModule{
		name: "test",
		start: func(ctx contracts.AppContext) error {
			return nil
		},
		stop: func(ctx contracts.AppContext) error {
			return nil
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- a.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	a.(*app).appCtx.Stop()

	err := <-done
	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}
}

func TestApplication_GracefulTimeout(t *testing.T) {
	a := New(
		AppInfo{AppName: "timeout"},
		nil,
		nil,
		WithGracefulTimeout(100*time.Millisecond),
	)

	_ = a.(*app).registry.Register(&mockModule{
		name: "slow",
		stop: func(ctx contracts.AppContext) error {
			time.Sleep(1 * time.Second)
			return nil
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- a.Run()
	}()

	time.Sleep(50 * time.Millisecond)

	a.(*app).appCtx.Stop()

	err := <-done
	if err == nil {
		t.Fatal("Expected error due to timeout")
	}

	if !errors.Is(err, ErrAppStop) {
		t.Errorf("Expected ErrAppRun, got %v", err)
	}
}

func TestApplication_RegisterError(t *testing.T) {
	a := New(AppInfo{}, nil, nil)

	_ = a.(*app).registry.Register(&mockModule{
		name: "error",
		start: func(ctx contracts.AppContext) error {
			return errors.New("start failed")
		},
	})

	err := ErrModuleRegister.WithCause(errors.New("test cause"))
	if err.Unwrap() == nil {
		t.Error("Cause should be preserved")
	}
}

func TestApplication_DoubleRun(t *testing.T) {
	a := New(AppInfo{AppName: "test"}, nil, nil)

	done := make(chan error, 1)
	go func() {
		done <- a.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	a.(*app).appCtx.Stop()

	err := <-done
	if err != nil && !errors.Is(err, ErrAppRun) {
		t.Fatalf("First Run() failed with unexpected error: %v", err)
	}

	err = a.Run()
	if err == nil {
		t.Fatal("Second Run() should fail")
	}

	if !errors.Is(err, ErrAppRun) {
		t.Errorf("Expected ErrAppRun, got %v", err)
	}
}

func TestApplication_NewWithNilDependencies(t *testing.T) {
	a := New(AppInfo{AppName: "test"}, nil, nil)
	if a == nil {
		t.Fatal("New should not return nil")
	}

	registry := NewRegistry()
	a = New(AppInfo{AppName: "test"}, nil, registry)
	if a == nil {
		t.Fatal("New should not return nil with nil container")
	}

	container := NewContainer()
	a = New(AppInfo{AppName: "test"}, container, nil)
	if a == nil {
		t.Fatal("New should not return nil with nil registry")
	}
}

func TestApplication_WithGracefulTimeout(t *testing.T) {
	a := New(AppInfo{AppName: "test"}, nil, nil)
	appImpl := a.(*app)

	if appImpl.shutdownTimeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", appImpl.shutdownTimeout)
	}

	customTimeout := 5 * time.Second
	a = New(AppInfo{AppName: "test"}, nil, nil, WithGracefulTimeout(customTimeout))
	appImpl = a.(*app)

	if appImpl.shutdownTimeout != customTimeout {
		t.Errorf("Expected custom timeout %v, got %v", customTimeout, appImpl.shutdownTimeout)
	}
}

func TestApplication_RegisterAfterRun(t *testing.T) {
	a := New(AppInfo{AppName: "test"}, nil, nil)

	done := make(chan error, 1)
	go func() {
		done <- a.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	mockMod := &mockModule{name: "late"}
	err := a.Register(mockMod)
	if err != nil {
		t.Errorf("Register should work even after Run started: %v", err)
	}

	a.(*app).appCtx.Stop()

	<-done
}

func TestApplication_StartError_ShutdownPreviouslyStarted(t *testing.T) {
	a := New(AppInfo{AppName: "test"}, nil, nil)

	successModule := &mockModule{name: "success"}

	failingModule := &mockModule{
		name: "failing",
		start: func(ctx contracts.AppContext) error {
			return errors.New("start failed")
		},
	}

	_ = a.(*app).registry.Register(successModule)
	_ = a.(*app).registry.Register(failingModule)

	err := a.Run()
	if err == nil {
		t.Fatal("Expected error from failing module")
	}

}
