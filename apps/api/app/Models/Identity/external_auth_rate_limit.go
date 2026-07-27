package identity

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// 外部认证 start/callback 限流（M5）。
//
// Host 独占：插件不得自建公开限流旁路。start 额外受全局写限流保护；
// callback 为 GET，必须有专用 IP 限流，防止 state 爆破与上游滥用。
// Redis 故障 fail-open（与登录锁定一致），避免 Redis 抖动锁死登录。
//
// T8C：INCR 与 TTL 建立必须原子（Lua），禁止 EXPIRE 失败后留下永久 lockout key。

const (
	// ExternalAuthStartMaxPerIP 单 IP 每窗口 start 上限（默认）。
	ExternalAuthStartMaxPerIP = 20
	// ExternalAuthCallbackMaxPerIP 单 IP 每窗口 callback 上限（默认）。
	ExternalAuthCallbackMaxPerIP = 40
	// ExternalAuthRateWindow 限流窗口。
	ExternalAuthRateWindow = time.Minute

	externalAuthRateKeyPrefix = "sforum:extauth:"
)

// externalAuthRateIncrWithTTLLua 原子 INCR + 首击 PEXPIRE。
// 返回当前计数；TTL 与递增在同一 Redis 脚本内，避免永久无 TTL key。
// 若 key 已存在但无 TTL（历史 bug / 手工写入），脚本会重新施加 TTL 自愈。
const externalAuthRateIncrWithTTLLua = `
local count = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if count == 1 or ttl < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return count
`

// ExternalAuthRateLimiter 按 opaque key（通常为 hashed IP + 操作类）限流。
// max<=0 或 limiter 为 nil 时一律放行。
type ExternalAuthRateLimiter interface {
	Allow(ctx context.Context, key string, max int, window time.Duration) (bool, error)
}

// externalAuthRateRedis 限流所需的最小 Redis 面；生产用 *redis.Client，测试可注入。
type externalAuthRateRedis interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// RedisExternalAuthRateLimiter 生产限流后端。
type RedisExternalAuthRateLimiter struct {
	client externalAuthRateRedis
}

// NewRedisExternalAuthRateLimiter 构造 Redis 限流；client 为 nil 时返回 nil。
func NewRedisExternalAuthRateLimiter(client *redis.Client) *RedisExternalAuthRateLimiter {
	if client == nil {
		return nil
	}
	return &RedisExternalAuthRateLimiter{client: client}
}

// newRedisExternalAuthRateLimiterForTest 注入可观测 Redis 面（T8C 回归）。
func newRedisExternalAuthRateLimiterForTest(client externalAuthRateRedis) *RedisExternalAuthRateLimiter {
	if client == nil {
		return nil
	}
	return &RedisExternalAuthRateLimiter{client: client}
}

// Allow 在 window 内对 key 递增；超过 max 返回 false。
// Redis 错误 fail-open（返回 true, nil），避免基础设施故障阻断登录。
// INCR+TTL 经 Lua 原子建立；脚本失败时 fail-open，且尝试删除可能残留的 key。
func (l *RedisExternalAuthRateLimiter) Allow(ctx context.Context, key string, max int, window time.Duration) (bool, error) {
	if l == nil || l.client == nil || key == "" || max <= 0 || window <= 0 {
		return true, nil
	}
	redisKey := externalAuthRateKeyPrefix + key
	ms := window.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	raw, err := l.client.Eval(ctx, externalAuthRateIncrWithTTLLua, []string{redisKey}, ms).Result()
	if err != nil {
		// fail-open：基础设施故障不得阻断登录；尽力清理可能半写入的 key。
		_ = l.client.Del(ctx, redisKey).Err()
		return true, nil
	}
	count, ok := redisInt64(raw)
	if !ok {
		_ = l.client.Del(ctx, redisKey).Err()
		return true, nil
	}
	return int(count) <= max, nil
}

func redisInt64(raw interface{}) (int64, bool) {
	switch v := raw.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case uint64:
		return int64(v), true
	default:
		return 0, false
	}
}

// MemoryExternalAuthRateLimiter 进程内限流（测试与无 Redis 本地路径）。
type MemoryExternalAuthRateLimiter struct {
	mu     sync.Mutex
	counts map[string]memoryRateBucket
}

type memoryRateBucket struct {
	count     int
	expiresAt time.Time
}

// NewMemoryExternalAuthRateLimiter 构造内存限流。
func NewMemoryExternalAuthRateLimiter() *MemoryExternalAuthRateLimiter {
	return &MemoryExternalAuthRateLimiter{counts: map[string]memoryRateBucket{}}
}

// Allow 实现 ExternalAuthRateLimiter。
func (l *MemoryExternalAuthRateLimiter) Allow(_ context.Context, key string, max int, window time.Duration) (bool, error) {
	if l == nil || key == "" || max <= 0 || window <= 0 {
		return true, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.counts[key]
	if !ok || now.After(b.expiresAt) {
		l.counts[key] = memoryRateBucket{count: 1, expiresAt: now.Add(window)}
		return true, nil
	}
	b.count++
	l.counts[key] = b
	return b.count <= max, nil
}

// ExternalAuthRateKey 构造限流 key；scope 为 "start" 或 "callback"。
// ip 应已是客户端 IP 字符串；空 IP 使用 "unknown"。
func ExternalAuthRateKey(scope, ip string) string {
	if ip == "" {
		ip = "unknown"
	}
	if scope == "" {
		scope = "any"
	}
	return fmt.Sprintf("%s:%s", scope, ip)
}
