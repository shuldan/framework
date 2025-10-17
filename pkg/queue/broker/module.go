package broker

import (
	"reflect"
	"time"

	rdClient "github.com/redis/go-redis/v9"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/queue/broker/memory"
	"github.com/shuldan/framework/pkg/queue/broker/redis"
)

type module struct{}

func NewModule() contracts.AppModule {
	return &module{}
}

func (m *module) Name() string {
	return "queue.broker"
}

func (m *module) Register(container contracts.DIContainer) error {
	config, err := container.Resolve(reflect.TypeOf((*contracts.Config)(nil)).Elem())
	if err != nil {
		return err
	}
	cfgInst, ok := config.(contracts.Config)
	if !ok {
		return ErrInvalidConfigInstance
	}

	logger, err := container.Resolve(reflect.TypeOf((*contracts.Logger)(nil)).Elem())
	if err != nil {
		return err
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	queueCfg, exists := cfgInst.GetSub("queue")
	if !exists {
		return ErrQueueBrokerConfigNotFound
	}

	driver := queueCfg.GetString("driver", "memory")
	switch driver {
	case "memory":
		return container.Instance(reflect.TypeOf((*contracts.Broker)(nil)).Elem(), memory.New(loggerInst))
	case "redis":
		redisCfg, exists := queueCfg.GetSub("drivers.redis")
		if !exists {
			return ErrRedisConfigNotFound
		}

		clientCfg, exists := redisCfg.GetSub("client")
		if !exists {
			return ErrRedisClientNotConfigured
		}

		client := rdClient.NewClient(&rdClient.Options{
			Addr:     clientCfg.GetString("address", ""),
			Username: clientCfg.GetString("username", ""),
			Password: clientCfg.GetString("password", ""),
		})

		return container.Instance(reflect.TypeOf((*contracts.Broker)(nil)).Elem(), m.createRedisBroker(redisCfg, client))
	default:
		return ErrUnsupportedQueueDriver
	}
}

func (m *module) Start(_ contracts.AppContext) error {
	return nil
}

func (m *module) Stop(ctx contracts.AppContext) error {
	broker, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.Broker)(nil)).Elem())
	if err != nil {
		// Если брокер не зарегистрирован, это не ошибка
		return nil
	}

	brokerInst, ok := broker.(contracts.Broker)
	if !ok {
		return nil
	}

	return brokerInst.Close()
}

func (m *module) createRedisBroker(cfg contracts.Config, client *rdClient.Client) contracts.Broker {
	var opts []redis.Option

	if prefix := cfg.GetString("prefix", ""); prefix != "" {
		opts = append(opts, redis.WithStreamKeyFormat(prefix+":%s"))
	}

	if group := cfg.GetString("consumer_group", ""); group != "" {
		opts = append(opts, redis.WithConsumerGroup(group))
	}

	if timeout := cfg.GetInt64("processing_timeout", 0); timeout > 0 {
		opts = append(opts, redis.WithProcessingTimeout(time.Duration(timeout)*time.Second))
	}

	if interval := cfg.GetInt64("claim_interval", 0); interval > 0 {
		opts = append(opts, redis.WithClaimInterval(time.Duration(interval)*time.Second))
	}

	if batch := cfg.GetInt("max_claim_batch", 0); batch > 0 {
		opts = append(opts, redis.WithMaxClaimBatch(batch))
	}

	if block := cfg.GetInt64("block_timeout", 0); block > 0 {
		opts = append(opts, redis.WithBlockTimeout(time.Duration(block)*time.Second))
	}

	if maxLen := cfg.GetInt64("max_stream_length", 0); maxLen > 0 {
		opts = append(opts, redis.WithMaxStreamLength(maxLen))
	}

	if trim := cfg.GetBool("approximate_trimming", true); !trim {
		opts = append(opts, redis.WithApproximateTrimming(trim))
	}

	if claim := cfg.GetBool("enable_claim", true); !claim {
		opts = append(opts, redis.WithClaim(claim))
	}

	if prefix := cfg.GetString("consumer_prefix", ""); prefix != "" {
		opts = append(opts, redis.WithConsumerPrefix(prefix))
	}

	return redis.New(client, opts...)
}
