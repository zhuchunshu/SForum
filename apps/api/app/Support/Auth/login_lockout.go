package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoginLockout 基于 Redis 的登录失败计数与临时锁定。
// key 形态：sforum:login_fail:{login} / sforum:login_lock:{login}
type LoginLockout struct {
	client *redis.Client
}

func NewLoginLockout(client *redis.Client) *LoginLockout {
	if client == nil {
		return nil
	}
	return &LoginLockout{client: client}
}

func (l *LoginLockout) IsLocked(ctx context.Context, key string) (bool, error) {
	if l == nil || l.client == nil || key == "" {
		return false, nil
	}
	n, err := l.client.Exists(ctx, lockKey(key)).Result()
	if err != nil {
		return false, fmt.Errorf("login lockout exists: %w", err)
	}
	return n > 0, nil
}

func (l *LoginLockout) RecordFailure(ctx context.Context, key string, maxFailures int, lockout time.Duration) error {
	if l == nil || l.client == nil || key == "" || maxFailures <= 0 || lockout <= 0 {
		return nil
	}
	fail := failKey(key)
	count, err := l.client.Incr(ctx, fail).Result()
	if err != nil {
		return fmt.Errorf("login lockout incr: %w", err)
	}
	// 首次失败设置计数 TTL，避免永久堆积。
	if count == 1 {
		_ = l.client.Expire(ctx, fail, lockout).Err()
	}
	if int(count) >= maxFailures {
		if err := l.client.Set(ctx, lockKey(key), "1", lockout).Err(); err != nil {
			return fmt.Errorf("login lockout set: %w", err)
		}
		_ = l.client.Del(ctx, fail).Err()
	}
	return nil
}

func (l *LoginLockout) ClearFailures(ctx context.Context, key string) error {
	if l == nil || l.client == nil || key == "" {
		return nil
	}
	return l.client.Del(ctx, failKey(key), lockKey(key)).Err()
}

func failKey(login string) string {
	return "sforum:login_fail:" + login
}

func lockKey(login string) string {
	return "sforum:login_lock:" + login
}
