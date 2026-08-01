package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PasswordResetLimiter 为手动密码找回和邮箱验证发送提供共享 Redis 计数器。
// 业务层负责生成含场景、策略版本、目标或 IP 哈希的隔离 key。
type PasswordResetLimiter struct {
	client *redis.Client
}

func NewPasswordResetLimiter(client *redis.Client) *PasswordResetLimiter {
	if client == nil {
		return nil
	}
	return &PasswordResetLimiter{client: client}
}

// Allow 在 window 内对 key 递增计数；超过 max 返回 false。
// max<=0 或 limiter 未配置时一律放行。
func (l *PasswordResetLimiter) Allow(ctx context.Context, key string, max int, window time.Duration) (bool, error) {
	if l == nil || l.client == nil || key == "" || max <= 0 || window <= 0 {
		return true, nil
	}
	redisKey := "sforum:pwreset:" + key
	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("mail resend rate incr: %w", err)
	}
	if count == 1 {
		if err := l.client.Expire(ctx, redisKey, window).Err(); err != nil {
			return false, fmt.Errorf("mail resend rate expire: %w", err)
		}
	}
	return int(count) <= max, nil
}

func (l *PasswordResetLimiter) RetryAfter(ctx context.Context, key string) (time.Duration, error) {
	if l == nil || l.client == nil || key == "" {
		return 0, nil
	}
	remaining, err := l.client.TTL(ctx, "sforum:pwreset:"+key).Result()
	if err != nil {
		return 0, fmt.Errorf("mail resend rate ttl: %w", err)
	}
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}
