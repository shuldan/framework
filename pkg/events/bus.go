package events

import (
	"context"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type listenerAdapter struct {
	listenerFunc func(context.Context, any) error
	eventType    reflect.Type
}

func (l *listenerAdapter) handleEvent(ctx context.Context, event any) error {
	eventValue := reflect.ValueOf(event)
	if !eventValue.Type().AssignableTo(l.eventType) {
		return ErrInvalidEventType.
			WithDetail("expected", l.eventType.String()).
			WithDetail("got", eventValue.Type().String())
	}

	return l.listenerFunc(ctx, event)
}

func adapterFromFunction(fn any) (*listenerAdapter, error) {
	fnType := reflect.TypeOf(fn)
	if fnType.NumIn() != 2 || fnType.NumOut() != 1 {
		return nil, ErrInvalidListenerFunction.WithDetail("signature", fnType.String())
	}

	ctxType, eventType := fnType.In(0), fnType.In(1)
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()

	if !ctxType.Implements(contextType) {
		return nil, ErrInvalidListenerFunction.WithDetail("reason", "first argument must implement context.ParentContext")
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
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()

	if !ctxType.Implements(contextType) {
		return nil, ErrInvalidListenerMethod.WithDetail("reason", "first argument must implement context.ParentContext")
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

type eventTask struct {
	ctx     context.Context
	event   any
	adapter *listenerAdapter
}

type eventBus struct {
	mu           sync.RWMutex
	listeners    map[reflect.Type][]*listenerAdapter
	closed       bool
	wg           sync.WaitGroup
	panicHandler PanicHandler
	errorHandler ErrorHandler
	eventChan    chan eventTask
	workerCount  int
	asyncMode    bool
}

func New(opts ...Option) contracts.EventBus {
	cfg := &eventBusConfig{
		asyncMode:   false,
		workerCount: 1,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.panicHandler == nil {
		cfg.panicHandler = NewDefaultPanicHandler(nil)
	}
	if cfg.errorHandler == nil {
		cfg.errorHandler = NewDefaultErrorHandler(nil)
	}

	b := &eventBus{
		listeners:    make(map[reflect.Type][]*listenerAdapter),
		panicHandler: cfg.panicHandler,
		errorHandler: cfg.errorHandler,
		asyncMode:    cfg.asyncMode,
		workerCount:  cfg.workerCount,
	}

	if cfg.asyncMode {
		b.startWorkers()
	}

	return b
}

func (b *eventBus) WithPanicHandler(h PanicHandler) contracts.EventBus {
	b.panicHandler = h
	return b
}

func (b *eventBus) WithErrorHandler(h ErrorHandler) contracts.EventBus {
	b.errorHandler = h
	return b
}

func (b *eventBus) Subscribe(eventTypeArg any, listener any) error {
	eventTypeOf := reflect.TypeOf(eventTypeArg)
	if eventTypeOf == nil {
		return ErrInvalidEventType.WithDetail("reason", "eventType is nil")
	}
	if eventTypeOf.Kind() != reflect.Ptr || eventTypeOf.Elem().Kind() != reflect.Struct {
		return ErrInvalidEventType.WithDetail("reason", "eventType must be a pointer to struct")
	}
	eventType := eventTypeOf.Elem()

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

func (b *eventBus) Publish(ctx context.Context, event any) error {
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

	if b.asyncMode {
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, adapter := range adapters {
			select {
			case b.eventChan <- eventTask{ctx: ctx, event: event, adapter: adapter}:
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return ErrEventChannelBlocked
			}
		}
	} else {
		for _, adapter := range adapters {
			if err := b.processEventSync(ctx, event, adapter); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *eventBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	if b.eventChan != nil {
		close(b.eventChan)
	}

	b.wg.Wait()
	return nil
}

func (b *eventBus) startWorkers() {
	b.eventChan = make(chan eventTask, b.workerCount*10)
	for i := 0; i < b.workerCount; i++ {
		b.wg.Add(1)
		go b.worker()
	}
}

func (b *eventBus) worker() {
	defer b.wg.Done()
	for task := range b.eventChan {
		b.processEvent(task)
	}
}

func (b *eventBus) processEvent(task eventTask) {
	defer func() {
		if r := recover(); r != nil {
			b.panicHandler.Handle(task.event, task.adapter, r, debug.Stack())
		}
	}()

	if err := task.adapter.handleEvent(task.ctx, task.event); err != nil {
		b.errorHandler.Handle(task.event, task.adapter, err)
	}
}

func (b *eventBus) processEventSync(ctx context.Context, event any, adapter *listenerAdapter) error {
	defer func() {
		if r := recover(); r != nil {
			b.panicHandler.Handle(event, adapter, r, debug.Stack())
		}
	}()

	if err := adapter.handleEvent(ctx, event); err != nil {
		b.errorHandler.Handle(event, adapter, err)
		return err
	}

	return nil
}
