package app

import (
	"errors"
	"sync"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestContainer_ResolveInstance(t *testing.T) {
	c := NewContainer()
	if err := c.Instance("test", "hello"); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}

	val, err := c.Resolve("test")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if val.(string) != "hello" {
		t.Errorf("Expected 'hello', got %v", val)
	}
}

func TestContainer_ResolveFactory(t *testing.T) {
	c := NewContainer()
	if err := c.Factory("greet", func(c contracts.DIContainer) (interface{}, error) {
		return "Hello from factory!", nil
	}); err != nil {
		t.Errorf("Factor failed: %v", err)
	}

	val, err := c.Resolve("greet")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if val.(string) != "Hello from factory!" {
		t.Errorf("Expected factory result, got %v", val)
	}
}

func TestContainer_CircularDependency(t *testing.T) {
	c := NewContainer()
	if err := c.Factory("a", func(c contracts.DIContainer) (interface{}, error) {
		b, err := c.Resolve("b")
		if err != nil {
			return nil, err
		}
		return "a" + b.(string), nil
	}); err != nil {
		t.Fatalf("Factory failed: %v", err)
	}
	if err := c.Factory("b", func(c contracts.DIContainer) (interface{}, error) {
		a, err := c.Resolve("a")
		if err != nil {
			return nil, err
		}
		return "b" + a.(string), nil
	}); err != nil {
		t.Fatalf("Factor failed: %v", err)
	}

	_, err := c.Resolve("a")
	if err == nil {
		t.Fatal("Expected circular dependency error")
	}

	if !errors.Is(err, ErrCircularDep) {
		t.Errorf("Expected ErrCircularDep, got %v", err)
	}
}

func TestContainer_Has(t *testing.T) {
	c := NewContainer()

	if c.Has("nonexistent") {
		t.Error("Expected false for nonexistent key")
	}

	if err := c.Instance("instance", "value"); err != nil {
		t.Fatalf("Instance failed: %v", err)
	}
	if !c.Has("instance") {
		t.Error("Expected true for existing instance")
	}

	if err := c.Factory("factory", func(contracts.DIContainer) (interface{}, error) {
		return "factory_value", nil
	}); err != nil {
		t.Errorf("Factory failed: %v", err)
	}
	if !c.Has("factory") {
		t.Error("Expected true for existing factory")
	}

	if err := c.Instance("both", "instance_value"); err != nil {
		t.Errorf("Instance failed: %v", err)
	}
	if err := c.Factory("both_factory", func(contracts.DIContainer) (interface{}, error) {
		return "factory_value", nil
	}); err != nil {
		t.Errorf("Factory failed: %v", err)
	}
	if !c.Has("both") {
		t.Error("Expected true for existing instance in both maps")
	}
}

func TestContainer_DuplicateInstance(t *testing.T) {
	c := NewContainer()

	err := c.Instance("test", "value1")
	if err != nil {
		t.Fatalf("First Instance() failed: %v", err)
	}

	err = c.Instance("test", "value2")
	if err == nil {
		t.Error("Expected error for duplicate instance")
	}

	if !errors.Is(err, ErrDuplicateInstance) {
		t.Errorf("Expected ErrDuplicateInstance, got %v", err)
	}

	val, resolveErr := c.Resolve("test")
	if resolveErr != nil {
		t.Fatalf("Resolve failed: %v", resolveErr)
	}

	if val.(string) != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}
}

func TestContainer_DuplicateFactory(t *testing.T) {
	c := NewContainer()

	err := c.Factory("test", func(contracts.DIContainer) (interface{}, error) {
		return "factory1", nil
	})
	if err != nil {
		t.Fatalf("First Factory() failed: %v", err)
	}

	err = c.Factory("test", func(contracts.DIContainer) (interface{}, error) {
		return "factory2", nil
	})
	if err == nil {
		t.Error("Expected error for duplicate factory")
	}

	if !errors.Is(err, ErrDuplicateFactory) {
		t.Errorf("Expected ErrDuplicateFactory, got %v", err)
	}

	val, resolveErr := c.Resolve("test")
	if resolveErr != nil {
		t.Fatalf("Resolve failed: %v", resolveErr)
	}

	if val.(string) != "factory1" {
		t.Errorf("Expected 'factory1', got %v", val)
	}
}

func TestContainer_ResolveNonExistent(t *testing.T) {
	c := NewContainer()

	_, err := c.Resolve("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent key")
	}

	if !errors.Is(err, ErrValueNotFound) {
		t.Errorf("Expected ErrValueNotFound, got %v", err)
	}
}

func TestContainer_FactoryErrorPropagation(t *testing.T) {
	c := NewContainer()

	testErr := errors.New("factory error")
	if err := c.Factory("error_factory", func(contracts.DIContainer) (interface{}, error) {
		return nil, testErr
	}); err != nil {
		t.Errorf("Factory failed: %v", err)
	}

	_, err := c.Resolve("error_factory")
	if err == nil {
		t.Fatal("Expected error from factory")
	}

	if !errors.Is(err, testErr) {
		t.Errorf("Expected original error, got %v", err)
	}
}

func TestContainer_ConcurrentAccess(t *testing.T) {
	c := NewContainer()

	if err := c.Factory("counter", func(contracts.DIContainer) (interface{}, error) {
		return 1, nil
	}); err != nil {
		t.Errorf("Factory failed: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Resolve("counter")
			if err != nil {
				t.Errorf("Concurrent Resolve failed: %v", err)
			}
		}()
	}

	wg.Wait()
}
