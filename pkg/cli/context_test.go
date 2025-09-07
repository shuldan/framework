package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

const (
	testAppName = "test"
	testVersion = "1.0.0"
	testEnv     = "test"
)

type simpleAppContext struct {
	ctx context.Context
}

func (s *simpleAppContext) AppRegistry() contracts.AppRegistry {
	return nil
}

func (s *simpleAppContext) Ctx() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *simpleAppContext) Container() contracts.DIContainer {
	return nil
}

func (s *simpleAppContext) AppName() string {
	return testAppName
}

func (s *simpleAppContext) Version() string {
	return testVersion
}

func (s *simpleAppContext) Environment() string {
	return testEnv
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

	ctx := NewContext(appCtx, input, output, args)

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

	ctx := NewContext(appCtx, input, output, args)

	args[0] = "modified"

	ctxArgs := ctx.Args()
	if ctxArgs[0] != "arg1" {
		t.Error("AppContext args should be immutable copy")
	}
}
