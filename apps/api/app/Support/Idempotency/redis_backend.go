package idempotency

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBackend 基于 go-redis 的生产实现。
type RedisBackend struct {
	client *redis.Client
}

func NewRedisBackend(client *redis.Client) *RedisBackend {
	return &RedisBackend{client: client}
}

func (b *RedisBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if b == nil || b.client == nil {
		return nil, false, nil
	}
	value, err := b.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (b *RedisBackend) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	if b == nil || b.client == nil {
		return true, nil
	}
	return b.client.SetNX(ctx, key, value, ttl).Result()
}

func (b *RedisBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if b == nil || b.client == nil {
		return nil
	}
	return b.client.Set(ctx, key, value, ttl).Err()
}

func (b *RedisBackend) Delete(ctx context.Context, keys ...string) error {
	if b == nil || b.client == nil || len(keys) == 0 {
		return nil
	}
	return b.client.Del(ctx, keys...).Err()
}
