package framework

import "sync"

type Lazy[T any] struct {
	factory func() (T, error)
	value   T
	err     error
	once    sync.Once
	created bool
}

func NewLazy[T any](factory func() (T, error)) *Lazy[T] {
	return &Lazy[T]{factory: factory}
}

func (l *Lazy[T]) Get() (T, error) {
	l.once.Do(func() {
		l.value, l.err = l.factory()
		if l.err == nil {
			l.created = true
		}
	})

	return l.value, l.err
}

func (l *Lazy[T]) MustGet() T {
	v, err := l.Get()
	if err != nil {
		panic("framework: lazy init failed: " + err.Error())
	}

	return v
}

func (l *Lazy[T]) IsCreated() bool {
	return l.created
}

func (l *Lazy[T]) IfCreated(fn func(T)) {
	if l.created {
		fn(l.value)
	}
}
