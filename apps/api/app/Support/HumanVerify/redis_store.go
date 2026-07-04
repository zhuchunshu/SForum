package humanverify

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	prefix string
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client, prefix: "humanverify:"}
}

func NewRedisClient(addr string, password string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})
}

func (s *RedisStore) MarkUsed(ctx context.Context, key string, ttl time.Duration) error {
	ok, err := s.client.SetNX(ctx, s.prefix+"used:"+key, "1", ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrReplayed
	}
	return nil
}

func (s *RedisStore) IncrementRate(ctx context.Context, key string, window time.Duration, limit int) (bool, error) {
	if limit <= 0 {
		return false, nil
	}

	redisKey := s.prefix + "rate:" + key
	count, err := s.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := s.client.Expire(ctx, redisKey, window).Err(); err != nil {
			return false, err
		}
	}
	return count > int64(limit), nil
}

func (s *RedisStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}
