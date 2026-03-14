package commandbus

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/commands"
	"github.com/shuldan/queue/broker/memory"
)

func TestIntegration_SendAndReceive(t *testing.T) {
	t.Parallel()
	br := memory.New()
	t.Cleanup(func() { br.Close() })

	sender := NewCommandSender(br, nil, WithReplyTo("test-svc"), WithSender("origin"))
	sender.Forward("test.cmd")

	receiver := NewCommandReceiver(br, nil)
	var mu sync.Mutex
	var received commands.Command
	_ = receiver.Handle("test.cmd", stubDeserializer, func(ctx context.Context, cmd commands.Command) (commands.Result, error) {
		mu.Lock()
		received = cmd
		mu.Unlock()
		return &stubResult{BaseResult: commands.BaseResult{Name: "ok"}, Value: "done"}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = br.Consume(ctx, "commands.test.cmd", func(data []byte) error {
			return receiver.handleMessage(ctx, data)
		})
	}()

	time.Sleep(50 * time.Millisecond)

	cmd := &stubCommand{Name: "test.cmd", Payload: "integration"}
	cmd.IdemKey = "int-key-1"
	err := sender.Send(context.Background(), cmd)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if received == nil {
		t.Fatal("expected command to be received")
	}
}

func TestIntegration_SendReceiveReply(t *testing.T) {
	t.Parallel()
	br := memory.New()
	t.Cleanup(func() { br.Close() })

	sender := NewCommandSender(br, nil, WithReplyTo("reply-svc"), WithSender("origin"))
	sender.Forward("test.cmd")

	receiver := NewCommandReceiver(br, nil)
	_ = receiver.Handle("test.cmd", stubDeserializer, func(_ context.Context, _ commands.Command) (commands.Result, error) {
		return &stubResult{BaseResult: commands.BaseResult{Name: "res"}, Value: "hello"}, nil
	})

	listener := NewReplyListener(br, nil, WithListenerServiceName("reply-svc"))
	var mu sync.Mutex
	var gotResult commands.Result
	listener.OnResult("test.cmd", stubResultDeserializer, func(_ context.Context, r commands.Result, _ error) error {
		mu.Lock()
		gotResult = r
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = br.Consume(ctx, "commands.test.cmd", func(data []byte) error {
			return receiver.handleMessage(ctx, data)
		})
	}()

	go func() {
		_ = listener.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	cmd := &stubCommand{Name: "test.cmd", Payload: "e2e"}
	cmd.IdemKey = "e2e-key"
	err := sender.Send(context.Background(), cmd)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if gotResult == nil {
		t.Fatal("expected result in reply listener")
	}
}

func TestIntegration_Idempotency(t *testing.T) {
	t.Parallel()
	br := memory.New()
	t.Cleanup(func() { br.Close() })

	store := newStubIdempotencyStore()
	receiver := NewCommandReceiver(br, nil, WithIdempotencyStore(store))

	var callCount int
	var mu sync.Mutex
	_ = receiver.Handle("test.cmd", stubDeserializer, func(_ context.Context, _ commands.Command) (commands.Result, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return &stubResult{BaseResult: commands.BaseResult{Name: "r"}}, nil
	})

	data := makeCommandEnvelopeBytes("test.cmd", "same-key", "", 0)

	ctx := context.Background()
	_ = receiver.handleMessage(ctx, data)
	_ = receiver.handleMessage(ctx, data)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestIntegration_ExpiredCommand_ReplyError(t *testing.T) {
	t.Parallel()
	br := memory.New()
	t.Cleanup(func() { br.Close() })

	receiver := NewCommandReceiver(br, nil)
	_ = receiver.Handle("test.cmd", stubDeserializer, stubHandler)

	listener := NewReplyListener(br, nil, WithListenerServiceName("reply-svc"))
	var mu sync.Mutex
	var gotErr error
	listener.OnResult("test.cmd", stubResultDeserializer, func(_ context.Context, _ commands.Result, err error) error {
		mu.Lock()
		gotErr = err
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = listener.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	data := makeExpiredEnvelopeBytes("test.cmd", "exp-key", "reply-svc")
	_ = receiver.handleMessage(ctx, data)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if gotErr == nil {
		t.Fatal("expected error for expired command")
	}
	if !errContains(gotErr, "command expired") {
		t.Errorf("expected 'command expired' error, got %v", gotErr)
	}
}

func TestIntegration_HandlerError_ReplyError(t *testing.T) {
	t.Parallel()
	br := memory.New()
	t.Cleanup(func() { br.Close() })

	receiver := NewCommandReceiver(br, nil)
	_ = receiver.Handle("test.cmd", stubDeserializer, failHandler)

	listener := NewReplyListener(br, nil, WithListenerServiceName("reply-svc"))
	var mu sync.Mutex
	var gotErr error
	listener.OnResult("test.cmd", stubResultDeserializer, func(_ context.Context, _ commands.Result, err error) error {
		mu.Lock()
		gotErr = err
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = listener.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	data := makeCommandEnvelopeBytes("test.cmd", "fail-key", "reply-svc", 0)
	_ = receiver.handleMessage(ctx, data)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if gotErr == nil {
		t.Fatal("expected error from failed handler")
	}
	if !errContains(gotErr, "handler error") {
		t.Errorf("expected 'handler error', got %v", gotErr)
	}
}

func TestCommandReceiver_SendResult_NilResult(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	cmdEnv := &CommandEnvelope{
		CorrelationID: "c1",
		CommandName:   "test.cmd",
		ReplyTo:       "svc",
	}
	handlerErr := errors.New("some error")
	err := r.sendResult(context.Background(), cmdEnv, nil, handlerErr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("replies.svc")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var env ResultEnvelope
	_ = json.Unmarshal(msgs[0], &env)
	if env.Error == nil {
		t.Fatal("expected error in result envelope")
	}
	if *env.Error != "some error" {
		t.Errorf("expected %q, got %q", "some error", *env.Error)
	}
}
