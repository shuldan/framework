package framework

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestLazy_Get_ReturnsValue(t *testing.T) {
	t.Parallel()
	lazy := NewLazy(func() (string, error) { return "hello", nil })
	val, err := lazy.Get()
	assertNoError(t, err)
	assertEqual(t, "hello", val)
}

func TestLazy_Get_ReturnsError(t *testing.T) {
	t.Parallel()
	expected := errors.New("init failed")
	lazy := NewLazy(func() (string, error) { return "", expected })
	_, err := lazy.Get()
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestLazy_Get_CallsFactoryOnce(t *testing.T) {
	t.Parallel()
	var calls int32
	lazy := NewLazy(func() (string, error) {
		atomic.AddInt32(&calls, 1)
		return "value", nil
	})
	for range 5 {
		val, err := lazy.Get()
		assertNoError(t, err)
		assertEqual(t, "value", val)
	}
	if c := atomic.LoadInt32(&calls); c != 1 {
		t.Fatalf("factory called %d times, expected 1", c)
	}
}

func TestLazy_Get_CachesError(t *testing.T) {
	t.Parallel()
	var calls int32
	lazy := NewLazy(func() (string, error) {
		atomic.AddInt32(&calls, 1)
		return "", errors.New("fail")
	})
	for range 3 {
		_, err := lazy.Get()
		if err == nil {
			t.Fatal("expected error")
		}
	}
	if c := atomic.LoadInt32(&calls); c != 1 {
		t.Fatalf("factory called %d times, expected 1", c)
	}
}

func TestLazy_Get_Concurrent(t *testing.T) {
	t.Parallel()
	var calls int32
	lazy := NewLazy(func() (int, error) {
		atomic.AddInt32(&calls, 1)
		return 42, nil
	})
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := lazy.Get()
			assertNoError(t, err)
			assertEqual(t, 42, val)
		}()
	}
	wg.Wait()
	if c := atomic.LoadInt32(&calls); c != 1 {
		t.Fatalf("factory called %d times, expected 1", c)
	}
}

func TestLazy_MustGet_ReturnsValue(t *testing.T) {
	t.Parallel()
	lazy := NewLazy(func() (string, error) { return "ok", nil })
	assertEqual(t, "ok", lazy.MustGet())
}

func TestLazy_MustGet_PanicsOnError(t *testing.T) {
	t.Parallel()
	lazy := NewLazy(func() (string, error) { return "", errors.New("boom") })
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	lazy.MustGet()
}

func TestLazy_IsCreated(t *testing.T) {
	t.Parallel()
	lazy := NewLazy(func() (string, error) { return "ok", nil })
	assertEqual(t, false, lazy.IsCreated())
	_, _ = lazy.Get()
	assertEqual(t, true, lazy.IsCreated())
}

func TestLazy_IsCreated_FalseOnError(t *testing.T) {
	t.Parallel()
	lazy := NewLazy(func() (string, error) { return "", errors.New("fail") })
	_, _ = lazy.Get()
	assertEqual(t, false, lazy.IsCreated())
}

func TestLazy_IfCreated_Executes(t *testing.T) {
	t.Parallel()
	lazy := NewLazy(func() (string, error) { return "val", nil })
	_, _ = lazy.Get()
	called := false
	lazy.IfCreated(func(v string) {
		called = true
		assertEqual(t, "val", v)
	})
	if !called {
		t.Fatal("IfCreated callback was not called")
	}
}

func TestLazy_IfCreated_SkipsWhenNotCreated(t *testing.T) {
	t.Parallel()
	lazy := NewLazy(func() (string, error) { return "val", nil })
	called := false
	lazy.IfCreated(func(_ string) { called = true })
	if called {
		t.Fatal("IfCreated should not call fn before Get")
	}
}

func TestLazy_IfCreated_SkipsOnError(t *testing.T) {
	t.Parallel()
	lazy := NewLazy(func() (string, error) { return "", errors.New("fail") })
	_, _ = lazy.Get()
	called := false
	lazy.IfCreated(func(_ string) { called = true })
	if called {
		t.Fatal("IfCreated should not call fn on error")
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}
