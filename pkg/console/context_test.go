package console

import (
	"bytes"
	"context"
	"github.com/shuldan/framework/pkg/application"
	"strings"
	"testing"
	"time"
)

type simpleAppContext struct {
	ctx context.Context
}

func (s *simpleAppContext) Ctx() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *simpleAppContext) Container() application.Container {
	return nil
}

func (s *simpleAppContext) AppName() string {
	return "test"
}

func (s *simpleAppContext) Version() string {
	return "1.0.0"
}

func (s *simpleAppContext) Environment() string {
	return "test"
}

func (s *simpleAppContext) StartTime() time.Time {
	return time.Now()
}

func (s *simpleAppContext) StopTime() time.Time {
	return time.Time{}
}

func (s *simpleAppContext) IsRunning() bool {
	return true
}

func (s *simpleAppContext) Stop() {}

func TestContext(t *testing.T) {
	appCtx := &simpleAppContext{}
	input := strings.NewReader("test input")
	output := &bytes.Buffer{}
	args := []string{"arg1", "arg2"}

	ctx := newContext(appCtx, input, output, args)

	if ctx.Ctx() != appCtx {
		t.Error("Expected same application context")
	}

	if ctx.Input() != input {
		t.Error("Expected same input reader")
	}

	if ctx.Output() != output {
		t.Error("Expected same output writer")
	}

	if len(ctx.Args()) != len(args) {
		t.Errorf("Expected %d args, got %d", len(args), len(ctx.Args()))
	}

	for i, arg := range args {
		if ctx.Args()[i] != arg {
			t.Errorf("Expected arg %s at index %d, got %s", arg, i, ctx.Args()[i])
		}
	}
}

func TestContext_ArgsImmutability(t *testing.T) {
	appCtx := &simpleAppContext{}
	input := strings.NewReader("test")
	output := &bytes.Buffer{}
	args := []string{"arg1", "arg2"}

	ctx := newContext(appCtx, input, output, args)

	args[0] = "modified"

	ctxArgs := ctx.Args()
	if ctxArgs[0] != "arg1" {
		t.Error("Context args should be immutable copy")
	}
}
