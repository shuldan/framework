package framework

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/shuldan/cli"
	"github.com/shuldan/config"

	"github.com/shuldan/framework/logger"
)

func TestNewKernel_WithConfig(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{
		"app": map[string]any{"name": "testapp", "version": "1.0.0"},
	})
	k, err := NewKernel(WithConfig(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k.Config() != cfg {
		t.Fatal("config mismatch")
	}
	if k.Logger() == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewKernel_WithLogger(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{})
	log := logger.New(logger.Config{Level: "debug"})
	k, err := NewKernel(WithConfig(cfg), WithLogger(log))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k.Logger() != log {
		t.Fatal("logger mismatch")
	}
}

func TestNewKernel_WithLoggerNil(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{})
	k, err := NewKernel(WithConfig(cfg), WithLogger(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k.Logger() == nil {
		t.Fatal("expected non-nil logger from config fallback")
	}
}

func TestNewKernel_EmptyConfig(t *testing.T) {
	t.Parallel()
	k, err := NewKernel(WithConfigFile())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name := k.Config().GetString("app.name", "default")
	if name != "default" {
		t.Fatalf("expected 'default', got %q", name)
	}
}

func TestNewKernel_WithEnvPrefix(t *testing.T) {
	os.Setenv("TESTFWK_APP_NAME", "envapp")
	t.Cleanup(func() { os.Unsetenv("TESTFWK_APP_NAME") })
	k, err := NewKernel(WithConfigFile(), WithEnvPrefix("TESTFWK"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k.Config() == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestNewKernel_WithProfileEnv(t *testing.T) {
	os.Setenv("APP_PROFILE_TEST", "test")
	t.Cleanup(func() { os.Unsetenv("APP_PROFILE_TEST") })
	_, err := NewKernel(
		WithConfigFile("testdata/nonexistent.yaml"),
		WithProfileEnv("APP_PROFILE_TEST"),
	)
	if err == nil {
		t.Log("no error even with nonexistent file (may be expected)")
	}
}

func TestNewKernel_WithRealConfigFile(t *testing.T) {
	const path = "testdata/test_config.yaml"
	if err := os.MkdirAll("testdata", 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := []byte("app:\n  name: fromfile\n  version: \"3.0\"\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })
	k, err := NewKernel(WithConfigFile(path))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name := k.Config().GetString("app.name", "")
	if name != "fromfile" {
		t.Fatalf("expected 'fromfile', got %q", name)
	}
}

func TestKernel_Command_And_Run(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{"app": map[string]any{"name": "test"}})
	k, err := NewKernel(WithConfig(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	executed := false
	k.Command(newStubCommand("greet", func() error {
		executed = true
		return nil
	}))
	var buf bytes.Buffer
	err = k.RunWith(context.Background(), emptyReader(), &buf, []string{"greet"})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !executed {
		t.Fatal("command was not executed")
	}
}

func TestKernel_Command_PanicOnDuplicate(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{"app": map[string]any{"name": "test"}})
	k, err := NewKernel(WithConfig(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	k.Command(newStubCommand("dup", nil))
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate command")
		}
	}()
	k.Command(newStubCommand("dup", nil))
}

func TestKernel_OnShutdown_RunsInReverse(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{"app": map[string]any{"name": "test"}})
	k, err := NewKernel(WithConfig(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var order []int
	k.OnShutdown(func() { order = append(order, 1) })
	k.OnShutdown(func() { order = append(order, 2) })
	k.OnShutdown(func() { order = append(order, 3) })
	k.Command(newStubCommand("noop", nil))
	_ = k.RunWith(context.Background(), emptyReader(), io.Discard, []string{"noop"})
	if len(order) != 3 || order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Fatalf("expected [3,2,1], got %v", order)
	}
}

func TestKernel_Run_HelpByDefault(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{"app": map[string]any{"name": "myapp"}})
	k, err := NewKernel(WithConfig(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var buf bytes.Buffer
	err = k.RunWith(context.Background(), emptyReader(), &buf, []string{})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected help output")
	}
}

func TestKernel_Run_UsesStdio(t *testing.T) {
	cfg := config.FromMap(map[string]any{"app": map[string]any{"name": "test"}})
	k, err := NewKernel(WithConfig(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	k.Command(newStubCommand("runtest", nil))
	err = k.Run(context.Background(), []string{"runtest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKernel_ConfigWithVersion(t *testing.T) {
	t.Parallel()
	cfg := config.FromMap(map[string]any{
		"app": map[string]any{"name": "vapp", "version": "2.0.0"},
	})
	k, err := NewKernel(WithConfig(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k == nil {
		t.Fatal("expected non-nil kernel")
	}
}

func TestBuildFileOpts_NoFiles(t *testing.T) {
	t.Parallel()
	o := &kernelOptions{configFiles: nil}
	opts := buildFileOpts(o)
	if len(opts) != 0 {
		t.Fatalf("expected 0 opts, got %d", len(opts))
	}
}

func TestBuildFileOpts_WithProfile(t *testing.T) {
	t.Parallel()
	o := &kernelOptions{configFiles: []string{"a.yaml", "b.yaml"}, profileEnvVar: "MY_PROFILE"}
	opts := buildFileOpts(o)
	if len(opts) != 2 {
		t.Fatalf("expected 2 opts, got %d", len(opts))
	}
}

func TestBuildFileOpts_WithoutProfile(t *testing.T) {
	t.Parallel()
	o := &kernelOptions{configFiles: []string{"a.yaml"}, profileEnvVar: ""}
	opts := buildFileOpts(o)
	if len(opts) != 1 {
		t.Fatalf("expected 1 opt, got %d", len(opts))
	}
}

func TestBuildProfileOpts(t *testing.T) {
	t.Parallel()
	o := &kernelOptions{configFiles: []string{"c.yaml"}, profileEnvVar: "PROF"}
	opts := buildProfileOpts(o)
	if len(opts) != 1 {
		t.Fatalf("expected 1 opt, got %d", len(opts))
	}
}

func TestDefaultKernelOptions(t *testing.T) {
	t.Parallel()
	o := defaultKernelOptions()
	if len(o.configFiles) != 1 || o.configFiles[0] != "config.yaml" {
		t.Fatalf("unexpected default config files: %v", o.configFiles)
	}
}

func emptyReader() io.Reader { return &bytes.Buffer{} }

type stubCommand struct {
	name string
	fn   func() error
}

func newStubCommand(name string, fn func() error) *stubCommand {
	return &stubCommand{name: name, fn: fn}
}

func (c *stubCommand) Name() string          { return c.name }
func (c *stubCommand) Description() string   { return "test command" }
func (c *stubCommand) Group() string         { return "test" }
func (c *stubCommand) Args() []cli.Arg       { return nil }
func (c *stubCommand) Options() []cli.Option { return nil }

func (c *stubCommand) Execute(
	_ context.Context, _ io.Reader, _ io.Writer, _ *cli.Input,
) error {
	if c.fn != nil {
		return c.fn()
	}
	return nil
}
