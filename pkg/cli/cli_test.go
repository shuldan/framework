package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/app"
	"github.com/shuldan/framework/pkg/contracts"
)

type mockApplicationContext struct {
	ctx         context.Context
	container   contracts.DIContainer
	appName     string
	version     string
	environment string
	startTime   time.Time
	stopTime    time.Time
	running     bool
	stopped     bool
}

func (m *mockApplicationContext) Ctx() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *mockApplicationContext) Container() contracts.DIContainer {
	return m.container
}

func (m *mockApplicationContext) AppName() string {
	return m.appName
}

func (m *mockApplicationContext) Version() string {
	return m.version
}

func (m *mockApplicationContext) Environment() string {
	return m.environment
}

func (m *mockApplicationContext) StartTime() time.Time {
	return m.startTime
}

func (m *mockApplicationContext) StopTime() time.Time {
	return m.stopTime
}

func (m *mockApplicationContext) IsRunning() bool {
	return m.running
}

func (m *mockApplicationContext) Stop() {
	m.stopped = true
	m.running = false
}

func (m *mockApplicationContext) AppRegistry() contracts.AppRegistry { return nil }

type testCommand struct {
	name        string
	description string
	group       string
	validateErr error
	executeErr  error
	executed    bool
	validated   bool
	flags       *flag.FlagSet
}

func (t *testCommand) Name() string {
	return t.name
}

func (t *testCommand) Description() string {
	return t.description
}

func (t *testCommand) Group() string {
	return t.group
}

func (t *testCommand) Configure(flags *flag.FlagSet) {
	t.flags = flags
}

func (t *testCommand) Validate(contracts.CliContext) error {
	time.Sleep(15 * time.Millisecond)
	t.validated = true
	return t.validateErr
}

func (t *testCommand) Execute(contracts.CliContext) error {
	t.executed = true
	return t.executeErr
}

func TestConsole_Register(t *testing.T) {
	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = c.Register(cmd)
	if err == nil {
		t.Error("Expected error when registering duplicate command")
	}
}

func TestConsole_Run_NoArgs(t *testing.T) {
	registry := NewRegistry()
	_, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	appCtx := &mockApplicationContext{}

	var args []string
	ctx := NewContext(appCtx, nil, nil, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected error for no command specified")
	}

	if !errors.Is(err, ErrNoCommandSpecified) {
		t.Errorf("Expected ErrNoCommandSpecified, got %v", err)
	}
}

func TestConsole_Run_UnknownCommand(t *testing.T) {
	registry := NewRegistry()
	_, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	appCtx := &mockApplicationContext{}

	args := []string{"unknown"}
	ctx := NewContext(appCtx, nil, nil, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	if err != nil && !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected unknown command error, got %v", err)
	}
}

func TestConsole_Run_ValidCommand(t *testing.T) {
	registry := NewRegistry()
	console, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	testCmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = console.Register(testCmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockApplicationContext{}
	args := []string{"test"}

	var output bytes.Buffer
	ctx := NewContext(appCtx, nil, &output, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !testCmd.validated {
		t.Error("Expected command to be validated")
	}

	if !testCmd.executed {
		t.Error("Expected command to be executed")
	}
}

func TestConsole_Run_CommandValidationFailure(t *testing.T) {
	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	testCmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
		validateErr: fmt.Errorf("validation failed"),
	}

	err = c.Register(testCmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockApplicationContext{}
	args := []string{"test"}
	ctx := NewContext(appCtx, nil, nil, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected validation error")
	}

	if err != nil && !strings.Contains(err.Error(), "command validation failed") {
		t.Errorf("Expected validation error, got %v", err)
	}
}

func TestConsole_Run_CommandExecutionFailure(t *testing.T) {
	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	testCmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
		executeErr:  fmt.Errorf("execution failed"),
	}

	err = c.Register(testCmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockApplicationContext{}
	args := []string{"test"}
	ctx := NewContext(appCtx, nil, nil, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected execution error")
	}

	if err != nil && !strings.Contains(err.Error(), "command execution failed") {
		t.Errorf("Expected execution error, got %v", err)
	}
}

type mockContainer struct {
	mu        sync.RWMutex
	instances map[reflect.Type]interface{}
	factories map[reflect.Type]func(contracts.DIContainer) (interface{}, error)
}

func newMockContainer() *mockContainer {
	return &mockContainer{
		instances: make(map[reflect.Type]interface{}),
		factories: make(map[reflect.Type]func(contracts.DIContainer) (interface{}, error)),
	}
}

func (m *mockContainer) Has(abstract reflect.Type) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, hasInstance := m.instances[abstract]
	_, hasFactory := m.factories[abstract]
	return hasInstance || hasFactory
}

func (m *mockContainer) Instance(abstract reflect.Type, concrete interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, exists := m.instances[abstract]; exists {
		return app.ErrDuplicateInstance.WithDetail("type", abstract.String())
	}
	m.instances[abstract] = concrete
	return nil
}

func (m *mockContainer) Factory(abstract reflect.Type, factory func(contracts.DIContainer) (interface{}, error)) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, exists := m.factories[abstract]; exists {
		return app.ErrDuplicateFactory.WithDetail("type", abstract.String())
	}
	m.factories[abstract] = factory
	return nil
}

func (m *mockContainer) Resolve(abstract reflect.Type) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if instance, exists := m.instances[abstract]; exists {
		return instance, nil
	}

	if factory, exists := m.factories[abstract]; exists {
		return factory(m)
	}

	return nil, app.ErrValueNotFound.WithDetail("type", abstract.String())
}

type mockAppContextWithCancel struct {
	ctx       context.Context
	container contracts.DIContainer
	stopped   bool
}

func (m *mockAppContextWithCancel) AppRegistry() contracts.AppRegistry {
	return nil
}

func (m *mockAppContextWithCancel) Ctx() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *mockAppContextWithCancel) Container() contracts.DIContainer {
	return m.container
}

func (m *mockAppContextWithCancel) AppName() string      { return testAppName }
func (m *mockAppContextWithCancel) Version() string      { return testVersion }
func (m *mockAppContextWithCancel) Environment() string  { return testEnv }
func (m *mockAppContextWithCancel) StartTime() time.Time { return time.Now() }
func (m *mockAppContextWithCancel) StopTime() time.Time  { return time.Time{} }
func (m *mockAppContextWithCancel) IsRunning() bool      { return !m.stopped }

func (m *mockAppContextWithCancel) Stop() {
	m.stopped = true
}

func TestCli_RegisterNilCommand(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	err = c.Register(nil)
	if err == nil {
		t.Error("Expected error when registering nil command")
	}
}

func TestCli_RegisterWithNilRegistry(t *testing.T) {
	t.Parallel()

	c, err := New(nil)
	if err != nil {
		t.Fatalf("Expected no error with nil registry, got %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCli_RunWithOSArgs(t *testing.T) {
	t.Parallel()

	args := []string{"program", "test"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockAppContextWithCancel{
		container: newMockContainer(),
	}

	err = c.Run(
		NewContext(appCtx, nil, nil, args[1:]),
	)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !cmd.executed {
		t.Error("Expected command to be executed")
	}
}

func TestCli_RunWithCancelledContext(t *testing.T) {
	t.Parallel()

	args := []string{"program", "test"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	appCtx := &mockAppContextWithCancel{
		ctx:       ctx,
		container: newMockContainer(),
	}

	err = c.Run(NewContext(appCtx, nil, nil, args[1:]))
	if err == nil {
		t.Error("Expected error for cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestCli_RunWithTimeout(t *testing.T) {
	t.Parallel()

	args := []string{"program", "slow"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	slowCmd := &testCommand{
		name:        "slow",
		description: "Slow command",
		group:       "test",
	}

	err = c.Register(slowCmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	appCtx := &mockAppContextWithCancel{
		ctx:       ctx,
		container: newMockContainer(),
	}

	time.Sleep(2 * time.Millisecond)

	err = c.Run(NewContext(appCtx, nil, nil, args[1:]))
	if err == nil {
		t.Error("Expected error for timed out context")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestCli_RunWithEmptyArgs(t *testing.T) {
	t.Parallel()

	args := []string{"program"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	appCtx := &mockAppContextWithCancel{
		container: newMockContainer(),
	}

	err = c.Run(NewContext(appCtx, nil, nil, args[1:]))
	if err == nil {
		t.Error("Expected error for empty args")
	}

	if !errors.Is(err, ErrNoCommandSpecified) {
		t.Errorf("Expected ErrNoCommandSpecified, got %v", err)
	}
}

func TestCli_RunPreservesStdio(t *testing.T) {
	args := []string{"program", "test"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockAppContextWithCancel{
		container: newMockContainer(),
	}

	originalStdin := os.Stdin
	originalStdout := os.Stdout
	defer func() {
		os.Stdin = originalStdin
		os.Stdout = originalStdout
	}()

	err = c.Run(NewContext(appCtx, originalStdin, originalStdout, args[1:]))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if os.Stdin != originalStdin {
		t.Error("Expected os.Stdin to be preserved")
	}

	if os.Stdout != originalStdout {
		t.Error("Expected os.Stdout to be preserved")
	}
}

func TestCli_RunWithValidationError(t *testing.T) {
	t.Parallel()

	args := []string{"program", "test"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
		validateErr: errors.New("validation failed"),
	}

	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockAppContextWithCancel{
		container: newMockContainer(),
	}

	err = c.Run(NewContext(appCtx, nil, nil, args[1:]))
	if err == nil {
		t.Error("Expected validation error")
	}

	if !errors.Is(err, ErrCommandValidation) {
		t.Errorf("Expected ErrCommandValidation wrapper, got %v", err)
	}
}

func TestCli_RunWithExecutionError(t *testing.T) {
	t.Parallel()

	args := []string{"program", "test"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
		executeErr:  errors.New("execution failed"),
	}

	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockAppContextWithCancel{
		container: newMockContainer(),
	}

	err = c.Run(NewContext(appCtx, nil, nil, args[1:]))
	if err == nil {
		t.Error("Expected execution error")
	}

	if !errors.Is(err, ErrCommandExecution) {
		t.Errorf("Expected ErrCommandExecution wrapper, got %v", err)
	}
}

type panicCommand struct {
	name        string
	description string
	group       string
	shouldPanic bool
}

func (p *panicCommand) Name() string        { return p.name }
func (p *panicCommand) Description() string { return p.description }
func (p *panicCommand) Group() string       { return p.group }

func (p *panicCommand) Configure(*flag.FlagSet) {}

func (p *panicCommand) Validate(contracts.CliContext) error {
	if p.shouldPanic {
		panic("validation panic")
	}
	return nil
}

func (p *panicCommand) Execute(contracts.CliContext) error {
	if p.shouldPanic {
		panic("execution panic")
	}
	return nil
}

func TestCli_RunHandlesPanic(t *testing.T) {
	t.Parallel()

	args := []string{"program", "panic"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &panicCommand{
		name:        "panic",
		description: "Panic command",
		group:       "test",
		shouldPanic: true,
	}

	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockAppContextWithCancel{
		container: newMockContainer(),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected panic to be recovered, but got panic: %v", r)
		}
	}()

	err = c.Run(NewContext(appCtx, nil, nil, args[1:]))
	if err == nil {
		t.Error("Expected error from panic")
	}
}

func TestCli_NewCreatesProperComponents(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cliImpl, ok := c.(*cli)
	if !ok {
		t.Error("Expected cli implementation")
	}

	if cliImpl.registry == nil {
		t.Error("Expected registry to be set")
	}

	if cliImpl.cmdParser == nil {
		t.Error("Expected cmdParser to be set")
	}

	if cliImpl.cmdExecutor == nil {
		t.Error("Expected cmdExecutor to be set")
	}
}

func TestCli_RunWithOutput(t *testing.T) {
	t.Parallel()

	originalStdout := os.Stdout
	_, w, cleanup := setupPipe(t)
	defer cleanup()

	os.Stdout = w
	defer func() { os.Stdout = originalStdout }()

	c, cmd := setupTestCLI(t, "test")
	appCtx := &mockAppContextWithCancel{container: newMockContainer()}

	done := make(chan error, 1)
	go func() {
		args := []string{"test"}
		done <- c.Run(NewContext(appCtx, nil, nil, args))
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Command execution timed out")
	}

	if !cmd.executed {
		t.Error("Expected command to be executed")
	}
}

func setupPipe(t *testing.T) (*os.File, *os.File, func()) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	return r, w, func() {
		if err := r.Close(); err != nil {
			t.Errorf("Failed to close pipe: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Errorf("Failed to close pipe: %v", err)
		}
	}
}

func setupTestCLI(t *testing.T, cmdName string) (contracts.Cli, *testCommand) {
	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}
	cmd := &testCommand{
		name:        cmdName,
		description: "Test command",
		group:       "test",
	}
	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}
	return c, cmd
}

func TestCli_RegisterErrorWrapping(t *testing.T) {
	t.Parallel()

	registry := &failingRegistry{}
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err == nil {
		t.Error("Expected error from failing registry")
	}
}

func TestCli_RunWithComplexArgs(t *testing.T) {
	t.Parallel()

	args := []string{"program", "test", "-value=value", "arg1", "arg2"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &flagTestCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockAppContextWithCancel{
		container: newMockContainer(),
	}

	err = c.Run(NewContext(appCtx, nil, nil, args[1:]))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !cmd.executed {
		t.Error("Expected command to be executed")
	}
}

type failingRegistry struct{}

func (f *failingRegistry) Register(contracts.CliCommand) error {
	return errors.New("registry failure")
}

func (f *failingRegistry) Get(string) (contracts.CliCommand, bool) {
	return nil, false
}

func (f *failingRegistry) Groups() map[string][]contracts.CliCommand {
	return nil
}

func TestCli_RunWithNilAppContext(t *testing.T) {
	t.Parallel()

	args := []string{"program", "test"}

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil app context")
		}
	}()

	_ = c.Run(NewContext(nil, nil, nil, args[1:]))
	t.Error("Should have panicked before reaching this line")
}

func TestCli_MultipleRegistrations(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	for i := 0; i < 10; i++ {
		cmd := &testCommand{
			name:        "test" + string(rune('0'+i)),
			description: "Test command " + string(rune('0'+i)),
			group:       "test",
		}

		err = c.Register(cmd)
		if err != nil {
			t.Errorf("Failed to register command %d: %v", i, err)
		}
	}

	allGroups := registry.Groups()
	if len(allGroups["test"]) != 10 {
		t.Errorf("Expected 10 commands registered, got %d", len(allGroups["test"]))
	}
}

func TestCli_RunInternalComponents(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create cli: %v", err)
	}

	cliImpl := c.(*cli)

	if cliImpl.registry != registry {
		t.Error("Expected registry to match provided registry")
	}

	if cliImpl.cmdParser == nil {
		t.Error("Expected cmdParser to be initialized")
	}

	if cliImpl.cmdExecutor == nil {
		t.Error("Expected cmdExecutor to be initialized")
	}

	if cliImpl.cmdParser.registry != registry {
		t.Error("Expected parser registry to match")
	}

	if cliImpl.cmdExecutor.parser != cliImpl.cmdParser {
		t.Error("Expected executor parser to match")
	}
}

func TestCli_ContextCancellationAtDifferentStages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cancelAfter  time.Duration
		expectCancel bool
	}{
		{name: "immediate_cancel", cancelAfter: 0, expectCancel: true},
		{name: "delayed_cancel", cancelAfter: 10 * time.Millisecond, expectCancel: true},
		{name: "no_cancel", cancelAfter: -1, expectCancel: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testContextCancellation(t, tt.cancelAfter, tt.expectCancel)
		})
	}
}

func testContextCancellation(t *testing.T, cancelAfter time.Duration, expectCancel bool) {
	c, cmd := setupTestCLI(t, "test")
	appCtx, cancel := setupTestContext(t, cancelAfter)
	defer cancel()

	err := c.Run(NewContext(appCtx, nil, nil, []string{"test"}))
	if expectCancel {
		if err == nil {
			t.Error("Expected error for cancelled context")
		} else if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	} else {
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !cmd.executed {
			t.Error("Expected command to be executed")
		}
	}
}

func setupTestContext(_ *testing.T, cancelAfter time.Duration) (*mockAppContextWithCancel, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	if cancelAfter == 0 {
		cancel()
	} else if cancelAfter > 0 {
		time.AfterFunc(cancelAfter, cancel)
	}

	appCtx := &mockAppContextWithCancel{
		ctx:       ctx,
		container: newMockContainer(),
	}

	return appCtx, cancel
}
