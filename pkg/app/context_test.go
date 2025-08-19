package app

import (
	"testing"
)

func TestAppContext_Stop(t *testing.T) {
	ctx := newAppContext(AppInfo{AppName: "test"}, NewContainer())

	if !ctx.IsRunning() {
		t.Error("Context should be isRunning after creation")
	}

	ctx.Stop()

	if ctx.IsRunning() {
		t.Error("Context should not be isRunning after Stop()")
	}

	select {
	case <-ctx.Ctx().Done():
	default:
		t.Error("Context should be cancelled after Stop()")
	}

	if ctx.StopTime().IsZero() {
		t.Error("StopTime should be set after Stop()")
	}
}
