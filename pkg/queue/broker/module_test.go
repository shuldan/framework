package broker

import (
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

func TestModule_Register_Success_MemoryDriver(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "memory",
			},
		},
	}
	err := container.Instance(contracts.ConfigModuleName, config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(contracts.LoggerModuleName, logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err != nil {
		t.Fatalf("Module registration failed: %v", err)
	}

	broker, err := container.Resolve(contracts.QueueBrokerModuleName)
	if err != nil {
		t.Fatalf("Failed to resolve queue broker: %v", err)
	}

	if broker == nil {
		t.Fatal("Queue broker should not be nil")
	}
}

func TestModule_Register_Success_RedisDriver(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "redis",
				"drivers": map[string]interface{}{
					"redis": map[string]interface{}{
						"prefix":               "testapp",
						"consumer_group":       "testgroup",
						"processing_timeout":   int64(30),
						"claim_interval":       int64(1),
						"max_claim_batch":      10,
						"block_timeout":        int64(500),
						"max_stream_length":    int64(10000),
						"approximate_trimming": true,
						"enable_claim":         true,
						"consumer_prefix":      "testnode",
						"client": map[string]interface{}{
							"address":  "localhost:6379",
							"username": "testuser",
							"password": "testpass",
						},
					},
				},
			},
		},
	}
	err := container.Instance(contracts.ConfigModuleName, config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(contracts.LoggerModuleName, logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err != nil {
		t.Fatalf("Module registration failed: %v", err)
	}

	broker, err := container.Resolve(contracts.QueueBrokerModuleName)
	if err != nil {
		t.Fatalf("Failed to resolve queue broker: %v", err)
	}

	if broker == nil {
		t.Fatal("Queue broker should not be nil")
	}
}

func TestModule_Register_MissingConfig(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	logger := &mockLogger{}

	if err := container.Instance(contracts.LoggerModuleName, logger); err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	config := &mockConfig{}

	if err := container.Instance(contracts.ConfigModuleName, config); err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	err := module.Register(container)
	if err == nil {
		t.Fatal("Expected error when config is missing, got nil")
	}

	if !errors.Is(err, ErrQueueBrokerConfigNotFound) {
		t.Errorf("Expected ErrQueueBrokerConfigNotFound, got %v", err)
	}
}

func TestModule_Register_InvalidConfigInstance(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	err := container.Instance(contracts.ConfigModuleName, "invalid config")
	if err != nil {
		t.Fatalf("Failed to register invalid config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(contracts.LoggerModuleName, logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error for invalid config instance, got nil")
	}

	if !errors.Is(err, ErrInvalidConfigInstance) {
		t.Errorf("Expected ErrInvalidConfigInstance, got %v", err)
	}
}

func TestModule_Register_InvalidLoggerInstance(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "memory",
			},
		},
	}
	err := container.Instance(contracts.ConfigModuleName, config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	err = container.Instance(contracts.LoggerModuleName, "invalid logger")
	if err != nil {
		t.Fatalf("Failed to register invalid logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error for invalid logger instance, got nil")
	}

	if !errors.Is(err, ErrInvalidLoggerInstance) {
		t.Errorf("Expected ErrInvalidLoggerInstance, got %v", err)
	}
}

func TestModule_Register_MissingQueueConfig(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"app": map[string]interface{}{
				"name": "testapp",
			},
		},
	}
	err := container.Instance(contracts.ConfigModuleName, config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(contracts.LoggerModuleName, logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error when queue config is missing, got nil")
	}

	if !errors.Is(err, ErrQueueBrokerConfigNotFound) {
		t.Errorf("Expected ErrQueueBrokerConfigNotFound, got %v", err)
	}
}

func TestModule_Register_UnsupportedDriver(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "unsupported",
			},
		},
	}
	err := container.Instance(contracts.ConfigModuleName, config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(contracts.LoggerModuleName, logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error for unsupported driver, got nil")
	}

	if !errors.Is(err, ErrUnsupportedQueueDriver) {
		t.Errorf("Expected ErrUnsupportedQueueDriver, got %v", err)
	}
}

func TestModule_Register_RedisDriver_MissingRedisConfig(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "redis",
			},
		},
	}
	err := container.Instance(contracts.ConfigModuleName, config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(contracts.LoggerModuleName, logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error when redis config is missing, got nil")
	}

	if !errors.Is(err, ErrRedisConfigNotFound) {
		t.Errorf("Expected ErrRedisConfigNotFound, got %v", err)
	}
}

func TestModule_Register_RedisDriver_MissingClientConfig(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "redis",
				"drivers": map[string]interface{}{
					"redis": map[string]interface{}{
						"prefix": "testapp",
					},
				},
			},
		},
	}
	err := container.Instance(contracts.ConfigModuleName, config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(contracts.LoggerModuleName, logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error when redis client config is missing, got nil")
	}

	if !errors.Is(err, ErrRedisClientNotConfigured) {
		t.Errorf("Expected ErrRedisClientNotConfigured, got %v", err)
	}
}

func TestModule_Name(t *testing.T) {
	module := NewModule()
	expectedName := contracts.QueueBrokerModuleName

	if module.Name() != expectedName {
		t.Errorf("Expected module name %s, got %s", expectedName, module.Name())
	}
}

func TestModule_Start(t *testing.T) {
	module := NewModule()
	err := module.Start(nil)
	if err != nil {
		t.Errorf("Start should return nil, got %v", err)
	}
}
