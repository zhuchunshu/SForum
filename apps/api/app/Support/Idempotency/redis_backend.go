package idempotency

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var errRedisBackendUnavailable = errors.New("idempotency: redis backend is unavailable")

var compareAndSwapScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if not current or current ~= ARGV[1] then
  return 0
end
if ARGV[2] == "delete" then
  redis.call("DEL", KEYS[1])
else
  redis.call("SET", KEYS[1], ARGV[3], "PX", ARGV[4])
end
return 1
`)

// RedisBackend 基于 go-redis 的生产实现。
type RedisBackend struct {
	client *redis.Client
}

func NewRedisBackend(client *redis.Client) *RedisBackend {
	return &RedisBackend{client: client}
}

func (b *RedisBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if b == nil || b.client == nil {
		return nil, false, errRedisBackendUnavailable
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
		return false, errRedisBackendUnavailable
	}
	return b.client.SetNX(ctx, key, value, ttl).Result()
}

func (b *RedisBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if b == nil || b.client == nil {
		return errRedisBackendUnavailable
	}
	return b.client.Set(ctx, key, value, ttl).Err()
}

func (b *RedisBackend) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if b == nil || b.client == nil {
		return errRedisBackendUnavailable
	}
	return b.client.Del(ctx, keys...).Err()
}

func (b *RedisBackend) CompareAndSwap(
	ctx context.Context,
	key string,
	expected, replacement []byte,
	ttl time.Duration,
) (bool, error) {
	if b == nil || b.client == nil {
		return false, errRedisBackendUnavailable
	}
	mode := "set"
	if replacement == nil {
		mode = "delete"
	}
	result, err := compareAndSwapScript.Run(
		ctx,
		b.client,
		[]string{key},
		expected,
		mode,
		replacement,
		strconv.FormatInt(ttl.Milliseconds(), 10),
	).Int()
	return result == 1, err
}
