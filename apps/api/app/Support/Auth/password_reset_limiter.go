package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PasswordResetLimiter 按邮箱哈希 / IP 限制密码重置请求频率。
// key：sforum:pwreset:email:{hash} / sforum:pwreset:ip:{hash}
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
		return false, fmt.Errorf("password reset rate incr: %w", err)
	}
	if count == 1 {
		if err := l.client.Expire(ctx, redisKey, window).Err(); err != nil {
			return false, fmt.Errorf("password reset rate expire: %w", err)
		}
	}
	return int(count) <= max, nil
}
