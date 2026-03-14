package commandbus

import (
	"testing"
	"time"
)

func TestWithReplyTo(t *testing.T) {
	t.Parallel()
	cfg := &senderConfig{}
	WithReplyTo("svc-a")(cfg)
	if cfg.replyTo != "svc-a" {
		t.Errorf("expected %q, got %q", "svc-a", cfg.replyTo)
	}
}

func TestWithSender(t *testing.T) {
	t.Parallel()
	cfg := &senderConfig{}
	WithSender("sender-x")(cfg)
	if cfg.sender != "sender-x" {
		t.Errorf("expected %q, got %q", "sender-x", cfg.sender)
	}
}

func TestWithDefaultTimeout(t *testing.T) {
	t.Parallel()
	cfg := &senderConfig{}
	WithDefaultTimeout(42 * time.Second)(cfg)
	if cfg.timeout != 42*time.Second {
		t.Errorf("expected %v, got %v", 42*time.Second, cfg.timeout)
	}
}

func TestWithTimeout_SendOption(t *testing.T) {
	t.Parallel()
	so := &sendOptions{}
	WithTimeout(15 * time.Second)(so)
	if so.timeout != 15*time.Second {
		t.Errorf("expected %v, got %v", 15*time.Second, so.timeout)
	}
}

func TestWithoutReply_SendOption(t *testing.T) {
	t.Parallel()
	so := &sendOptions{replyTo: "svc"}
	WithoutReply()(so)
	if so.replyTo != "" {
		t.Errorf("expected empty replyTo, got %q", so.replyTo)
	}
}

func TestWithHeaders_SendOption(t *testing.T) {
	t.Parallel()
	so := &sendOptions{}
	h := map[string]string{"k": "v"}
	WithHeaders(h)(so)
	if so.headers["k"] != "v" {
		t.Errorf("expected header k=v, got %v", so.headers)
	}
}
