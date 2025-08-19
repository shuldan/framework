package events

import (
	"context"
	"github.com/shuldan/framework/pkg/contracts"
	"reflect"
	"runtime/debug"
	"sync"
)

type defaultPanicHandler struct{}

func (d *defaultPanicHandler) Handle(event any, listener any, panicValue any, stack []byte) {
	// TODO: log panic
}

type defaultErrorHandler struct{}

func (d *defaultErrorHandler) Handle(event any, listener any, err error) {
	// TODO: log error
}

type listenerAdapter struct {
	listenerFunc func(context.Context, any) error
	eventType    reflect.Type
}

func (l *listenerAdapter) handleEvent(ctx context.Context, event any) error {
	return l.listenerFunc(ctx, event)
}

func adapterFromFunction(fn any) (*listenerAdapter, error) {
	fnType := reflect.TypeOf(fn)
	if fnType.NumIn() != 2 || fnType.NumOut() != 1 {
		return nil, ErrInvalidListenerFunction.WithDetail("signature", fnType.String())
	}

	ctxType, eventType := fnType.In(0), fnType.In(1)
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	if !ctxType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil, ErrInvalidListenerFunction.WithDetail("reason", "first argument must be context.Context")
	}
	if fnType.Out(0) != errorType {
		return nil, ErrInvalidListenerFunction.WithDetail("reason", "return type must be error")
	}

	return &listenerAdapter{
		eventType: eventType,
		listenerFunc: func(ctx context.Context, event any) error {
			results := reflect.ValueOf(fn).Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(event),
			})
			if err := results[0].Interface(); err != nil {
				return err.(error)
			}
			return nil
		},
	}, nil
}

func adapterFromMethod(receiver reflect.Value, method reflect.Method) (*listenerAdapter, error) {
	fnType := method.Type
	if fnType.NumIn() != 3 {
		return nil, ErrInvalidListenerMethod.WithDetail("reason", "must have two arguments (receiver, ctx, event)")
	}

	ctxType := fnType.In(1)
	eventType := fnType.In(2)
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	if !ctxType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil, ErrInvalidListenerMethod.WithDetail("reason", "first argument must be context.Context")
	}
	if fnType.NumOut() != 1 || fnType.Out(0) != errorType {
		return nil, ErrInvalidListenerMethod.WithDetail("reason", "must return error")
	}

	return &listenerAdapter{
		eventType: eventType,
		listenerFunc: func(ctx context.Context, event any) error {
			results := method.Func.Call([]reflect.Value{
				receiver,
				reflect.ValueOf(ctx),
				reflect.ValueOf(event),
			})
			if err := results[0].Interface(); err != nil {
				return err.(error)
			}
			return nil
		},
	}, nil
}

func newListenerAdapter(listener any) (*listenerAdapter, error) {
	listenerVal := reflect.ValueOf(listener)
	if !listenerVal.IsValid() {
		return nil, ErrInvalidListener
	}

	if listenerVal.Kind() == reflect.Func {
		return adapterFromFunction(listener)
	}

	if method, ok := listenerVal.Type().MethodByName("Handle"); ok {
		return adapterFromMethod(listenerVal, method)
	}

	return nil, ErrInvalidListener
}

type bus struct {
	mu           sync.RWMutex
	listeners    map[reflect.Type][]*listenerAdapter
	closed       bool
	wg           sync.WaitGroup
	panicHandler contracts.EventPanicHandler
	errorHandler contracts.EventErrorHandler
}

func (b *bus) WithPanicHandler(h contracts.EventPanicHandler) contracts.Bus {
	b.panicHandler = h
	return b
}

func (b *bus) WithErrorHandler(h contracts.EventErrorHandler) contracts.Bus {
	b.errorHandler = h
	return b
}

func (b *bus) Subscribe(eventTypeArg any, listener any) error {
	eventTypeOf := reflect.TypeOf(eventTypeArg)
	if eventTypeOf == nil {
		return ErrInvalidEventType.
			WithDetail("reason", "eventType is nil")
	}
	if eventTypeOf.Kind() != reflect.Ptr || eventTypeOf.Elem().Kind() != reflect.Struct {
		return ErrInvalidEventType.
			WithDetail("reason", "eventType must be a pointer to struct, e.g. (*MyEvent)(nil)")
	}
	eventType := eventTypeOf.Elem()

	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	b.mu.RUnlock()

	adapter, err := newListenerAdapter(listener)
	if err != nil {
		return err
	}

	if adapter.eventType != eventType {
		return ErrInvalidListener.
			WithDetail("expected_type", eventType.String()).
			WithDetail("actual_type", adapter.eventType.String())
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrBusClosed
	}

	b.listeners[eventType] = append(b.listeners[eventType], adapter)

	return nil
}

func (b *bus) Publish(ctx context.Context, event any) error {
	if event == nil {
		return nil
	}

	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrPublishOnClosedBus
	}

	eventType := reflect.TypeOf(event)
	adapters, ok := b.listeners[eventType]
	b.mu.RUnlock()

	if !ok || len(adapters) == 0 {
		return nil
	}

	for _, adapter := range adapters {
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					b.panicHandler.Handle(event, adapter, r, debug.Stack())
				}
			}()

			if err := adapter.listenerFunc(ctx, event); err != nil {
				b.errorHandler.Handle(event, adapter, err)
			}
		}()
	}

	return nil
}

func (b *bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	listeners := make([]*listenerAdapter, 0)
	for _, ls := range b.listeners {
		listeners = append(listeners, ls...)
	}
	b.mu.Unlock()

	b.wg.Wait()
	return nil
}
