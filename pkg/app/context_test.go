package app

import (
	"testing"
)

func TestAppContext_Stop(t *testing.T) {
	ctx := newAppContext(Info{AppName: "test"}, NewContainer(), nil)

	if !ctx.IsRunning() {
		t.Error("ParentContext should be isRunning after creation")
	}

	ctx.Stop()

	if ctx.IsRunning() {
		t.Error("ParentContext should not be isRunning after Stop()")
	}

	select {
	case <-ctx.ParentContext().Done():
	default:
		t.Error("ParentContext should be cancelled after Stop()")
	}

	if ctx.StopTime().IsZero() {
		t.Error("StopTime should be set after Stop()")
	}
}
