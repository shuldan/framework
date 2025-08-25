package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strings"
	"sync"
	"time"
)

type broker struct {
	client      redis.UniversalClient
	consumers   map[string][]context.CancelFunc
	consumersMu sync.RWMutex
	config      *config
	wg          sync.WaitGroup
}

func (b *broker) Produce(ctx context.Context, topic string, data []byte) error {
	msg := redisStreamMessage{
		Data:       data,
		EnqueuedAt: time.Now().UTC().Format(time.RFC3339),
	}

	values, err := b.encodeMessage(msg)
	if err != nil {
		return ErrEncodeFailed.
			WithDetail("topic", topic).
			WithDetail("err", err)
	}

	stream := fmt.Sprintf(b.config.streamKeyFormat, topic)

	xAddArgs := &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}

	if b.config.maxStreamLength > 0 {
		xAddArgs.MaxLen = b.config.maxStreamLength
		xAddArgs.Approx = b.config.approximateTrim
	}

	_, err = b.client.XAdd(ctx, xAddArgs).Result()
	if err != nil {
		return ErrProduceFailed.
			WithDetail("topic", topic).
			WithDetail("stream", stream).
			WithDetail("err", err)
	}

	return nil
}

func (b *broker) Consume(ctx context.Context, topic string, handler func([]byte) error) error {
	stream := fmt.Sprintf(b.config.streamKeyFormat, topic)
	group := fmt.Sprintf("%s:%s", b.config.consumerGroup, topic)
	consumer := b.newConsumerID(topic)

	exists, err := b.groupExists(ctx, stream, group)
	if err != nil {
		return ErrGroupCheckFailed.
			WithDetail("topic", topic).
			WithDetail("stream", stream).
			WithDetail("group", group).
			WithDetail("err", err)
	}

	if !exists {
		if err := b.client.XGroupCreateMkStream(ctx, stream, group, "0").Err(); err != nil {
			if !isGroupExists(err) {
				return ErrConsumeSetupFailed.
					WithDetail("topic", topic).
					WithDetail("stream", stream).
					WithDetail("group", group).
					WithDetail("err", err)
			}
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	b.trackConsumer(topic, cancel)

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer cancel()
		b.consumeLoop(ctx, stream, group, consumer, handler)
	}()

	return nil
}

func (b *broker) consumeLoop(
	ctx context.Context,
	stream, group, consumer string,
	handler func([]byte) error,
) {
	ticker := time.NewTicker(b.config.claimInterval)
	if !b.config.enableClaim {
		ticker.Stop()
	}
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if b.config.enableClaim {
				b.claimStalledMessages(ctx, stream, group, consumer, handler)
			}
		default:
			b.processNewMessage(ctx, stream, group, consumer, handler)
		}
	}
}

func (b *broker) processNewMessage(
	ctx context.Context,
	stream, group, consumer string,
	handler func([]byte) error,
) {
	result, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    b.config.blockTimeout,
		NoAck:    false,
	}).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) || ctx.Err() != nil {
			select {
			case <-time.After(10 * time.Millisecond):
			case <-ctx.Done():
			}
			return
		}
		return // Логируем ошибку?
	}

	if len(result) == 0 || len(result[0].Messages) == 0 {
		return
	}

	msg := result[0].Messages[0]
	var body redisStreamMessage
	if err := b.decodeMessage(msg.Values, &body); err != nil {
		_ = b.client.XAck(ctx, stream, group, msg.ID)
		return
	}

	err = handler(body.Data)
	if err == nil {
		_ = b.client.XAck(ctx, stream, group, msg.ID)
	}
	// Если ошибка, сообщение останется в группе для повторной обработки
}

func (b *broker) claimStalledMessages(ctx context.Context, stream, group, consumer string, handler func([]byte) error) {
	ids, err := b.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   stream,
		Group:    group,
		Start:    "-",
		End:      "+",
		Count:    int64(b.config.maxClaimBatch),
		Idle:     b.config.processingTimeout,
		Consumer: "",
	}).Result()
	if err != nil {
		return
	}

	for _, id := range ids {
		msgs, _ := b.client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   stream,
			Group:    group,
			Consumer: consumer,
			MinIdle:  b.config.processingTimeout,
			Messages: []string{id.ID},
		}).Result()

		for _, msg := range msgs {
			var body redisStreamMessage
			if err := b.decodeMessage(msg.Values, &body); err != nil {
				_ = b.client.XAck(ctx, stream, group, msg.ID)
				continue
			}
			// Обрабатываем сообщение через handler
			err := handler(body.Data)
			if err == nil {
				_ = b.client.XAck(ctx, stream, group, msg.ID)
			}
			// Если ошибка, сообщение останется для следующей попытки
		}
	}
}

func (b *broker) Close() error {
	b.consumersMu.Lock()
	defer b.consumersMu.Unlock()

	for topic, cancels := range b.consumers {
		for _, cancel := range cancels {
			if cancel != nil {
				cancel()
			}
		}
		b.consumers[topic] = nil
	}

	b.wg.Wait()
	return nil
}

func (b *broker) groupExists(ctx context.Context, stream, group string) (bool, error) {
	groups, err := b.client.XInfoGroups(ctx, stream).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	for _, g := range groups {
		if g.Name == group {
			return true, nil
		}
	}

	return false, nil
}

func (b *broker) trackConsumer(topic string, cancel context.CancelFunc) {
	b.consumersMu.Lock()
	b.consumers[topic] = append(b.consumers[topic], cancel)
	b.consumersMu.Unlock()
}

func (b *broker) newConsumerID(topic string) string {
	prefix := b.config.consumerPrefix
	if prefix != "" {
		prefix = prefix + "-"
	}
	return fmt.Sprintf("consumer-%s%s-%s", prefix, topic, uuid.New().String())
}

func (b *broker) encodeMessage(msg redisStreamMessage) (map[string]interface{}, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"payload": string(data)}, nil
}

func (b *broker) decodeMessage(values map[string]interface{}, msg *redisStreamMessage) error {
	if payload, ok := values["payload"].(string); ok {
		return json.Unmarshal([]byte(payload), msg)
	}
	return ErrInvalidPayload
}

func isGroupExists(err error) bool {
	return err != nil && (strings.HasPrefix(err.Error(), "BUSYGROUP") ||
		strings.Contains(err.Error(), "already exists"))
}

type redisStreamMessage struct {
	Data       []byte `json:"data"`
	EnqueuedAt string `json:"enqueued_at"`
}
