package command

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shuldan/cli"
	"github.com/shuldan/config"
)

func TestHealth_AllHealthy(t *testing.T) {
	t.Parallel()
	checkers := []HealthChecker{
		&mockHealthChecker{name: "db", err: nil},
		&mockHealthChecker{name: "redis", err: nil},
	}
	output, err := runCommand(t, Health(checkers...))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, output, "✓ db")
	assertContains(t, output, "✓ redis")
	assertContains(t, output, "All services healthy")
}

func TestHealth_SomeFailing(t *testing.T) {
	t.Parallel()
	checkers := []HealthChecker{
		&mockHealthChecker{name: "db", err: nil},
		&mockHealthChecker{name: "redis", err: errors.New("connection refused")},
	}
	output, err := runCommand(t, Health(checkers...))
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, output, "✗ redis")
}

func TestHealth_NoCheckers(t *testing.T) {
	t.Parallel()
	output, err := runCommand(t, Health())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, output, "All services healthy")
}

func TestHealth_Metadata(t *testing.T) {
	t.Parallel()
	cmd := Health()
	assertCliCommand(t, cmd, "health", "debug")
	if cmd.Description() == "" {
		t.Error("description should not be empty")
	}
	if cmd.Args() != nil {
		t.Error("expected nil args")
	}
	if cmd.Options() != nil {
		t.Error("expected nil options")
	}
}

func TestConfigDump_MasksSensitiveKeys(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{
		"app":      map[string]any{"name": "test"},
		"database": map[string]any{"dsn": "postgres://secret"},
		"api":      map[string]any{"token": "abc123"},
	})
	output, err := runCommand(t, ConfigDump(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, output, "***")
	assertNotContains(t, output, "postgres://secret")
	assertNotContains(t, output, "abc123")
}

func TestConfigDump_NoMask(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{
		"database": map[string]any{"dsn": "postgres://real"},
	})
	output, err := runCommand(t, ConfigDump(cfg), "--no-mask")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, output, "postgres://real")
}

func TestConfigDump_Metadata(t *testing.T) {
	t.Parallel()
	cmd := ConfigDump(nil)
	assertCliCommand(t, cmd, "config:dump", "debug")
	if cmd.Description() == "" {
		t.Error("description should not be empty")
	}
	if cmd.Args() != nil {
		t.Error("expected nil args")
	}
}

func TestConfigDump_NonSensitiveKey(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{
		"app": map[string]any{"name": "myapp"},
	})
	output, err := runCommand(t, ConfigDump(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, output, "myapp")
}

func TestConfigDump_EmptyMap(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{})
	output, err := runCommand(t, ConfigDump(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "" {
		t.Log("output with empty config:", output)
	}
}

func TestIsSensitiveKey_AllKeywords(t *testing.T) {
	t.Parallel()
	tests := []struct {
		key      string
		expected bool
	}{
		{"db.password", true},
		{"app.secret", true},
		{"api.token", true},
		{"ssh.key", true},
		{"database.dsn", true},
		{"auth.credential", true},
		{"app.name", false},
		{"server.port", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			got := isSensitiveKey(tt.key)
			if got != tt.expected {
				t.Errorf("isSensitiveKey(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestFormatValue_Masked(t *testing.T) {
	t.Parallel()
	result := formatValue("db.password", "secret123", false)
	if result != "***" {
		t.Errorf("expected '***', got %q", result)
	}
}

func TestFormatValue_Unmasked(t *testing.T) {
	t.Parallel()
	result := formatValue("db.password", "secret123", true)
	if result != "secret123" {
		t.Errorf("expected 'secret123', got %q", result)
	}
}

func TestFormatValue_NonSensitive(t *testing.T) {
	t.Parallel()
	result := formatValue("app.name", "myapp", false)
	if result != "myapp" {
		t.Errorf("expected 'myapp', got %q", result)
	}
}

func TestSortedMapKeys(t *testing.T) {
	t.Parallel()
	m := map[string]any{"c": 1, "a": 2, "b": 3}
	keys := sortedMapKeys(m)
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("expected [a b c], got %v", keys)
	}
}

type mockHealthChecker struct {
	name string
	err  error
}

func (m *mockHealthChecker) Name() string                   { return m.name }
func (m *mockHealthChecker) Health(_ context.Context) error { return m.err }

func runCommand(t *testing.T, cmd cli.Command, args ...string) (string, error) {
	t.Helper()
	c := cli.New()
	if err := c.Register(cmd); err != nil {
		t.Fatalf("failed to register command: %v", err)
	}
	var buf bytes.Buffer
	fullArgs := append([]string{cmd.Name()}, args...)
	err := c.Run(context.Background(), strings.NewReader(""), &buf, fullArgs)
	return buf.String(), err
}

func assertCliCommand(t *testing.T, cmd cli.Command, name, group string) {
	t.Helper()
	if cmd.Name() != name {
		t.Errorf("expected name %q, got %q", name, cmd.Name())
	}
	if cmd.Group() != group {
		t.Errorf("expected group %q, got %q", group, cmd.Group())
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q in:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("did not expect %q in:\n%s", substr, s)
	}
}

var (
	_ cli.Command = Health()
	_ cli.Command = ConfigDump(nil)
)
