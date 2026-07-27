package identity

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// t8cRateRedis 可观测 Redis 面：模拟原子脚本成功、脚本失败、以及无 TTL 自愈路径。
type t8cRateRedis struct {
	mu       sync.Mutex
	counts   map[string]int64
	ttls     map[string]time.Duration // <0 表示无 TTL；0 表示未设置
	evalErr  error
	delCalls []string
	// expireFail 在脚本路径内模拟 PEXPIRE 失败（整段脚本失败）。
	expireFail bool
}

func newT8CRateRedis() *t8cRateRedis {
	return &t8cRateRedis{
		counts: map[string]int64{},
		ttls:   map[string]time.Duration{},
	}
}

func (m *t8cRateRedis) Eval(_ context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.evalErr != nil {
		return redis.NewCmdResult(nil, m.evalErr)
	}
	if len(keys) != 1 {
		return redis.NewCmdResult(nil, errors.New("expected one key"))
	}
	key := keys[0]
	// 模拟 Lua：INCR + 条件 PEXPIRE。
	m.counts[key]++
	count := m.counts[key]
	ttl, hasTTL := m.ttls[key]
	needExpire := count == 1 || !hasTTL || ttl < 0
	if needExpire {
		if m.expireFail {
			// 脚本整体失败：不提交无 TTL 的永久 key（模拟事务性失败）。
			// 若此前已存在无 TTL 计数，调用方应 DEL 清理。
			return redis.NewCmdResult(nil, errors.New("PEXPIRE failed"))
		}
		var ms int64
		if len(args) > 0 {
			switch v := args[0].(type) {
			case int64:
				ms = v
			case int:
				ms = int64(v)
			}
		}
		m.ttls[key] = time.Duration(ms) * time.Millisecond
	}
	return redis.NewCmdResult(count, nil)
}

func (m *t8cRateRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, key := range keys {
		m.delCalls = append(m.delCalls, key)
		if _, ok := m.counts[key]; ok {
			delete(m.counts, key)
			delete(m.ttls, key)
			n++
		}
	}
	return redis.NewIntResult(n, nil)
}

func (m *t8cRateRedis) hasKey(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.counts[key]
	return ok
}

func (m *t8cRateRedis) ttl(key string) (time.Duration, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.ttls[key]
	return v, ok
}

func TestT8C_RedisRateLimit_AtomicIncrWithTTL(t *testing.T) {
	backend := newT8CRateRedis()
	lim := newRedisExternalAuthRateLimiterForTest(backend)
	ctx := context.Background()
	key := "start:10.0.0.1"

	ok, err := lim.Allow(ctx, key, 2, time.Minute)
	if err != nil || !ok {
		t.Fatalf("first allow: ok=%v err=%v", ok, err)
	}
	redisKey := externalAuthRateKeyPrefix + key
	ttl, has := backend.ttl(redisKey)
	if !has || ttl <= 0 {
		t.Fatalf("first hit must establish positive TTL, has=%v ttl=%v", has, ttl)
	}

	ok, err = lim.Allow(ctx, key, 2, time.Minute)
	if err != nil || !ok {
		t.Fatalf("second allow: ok=%v err=%v", ok, err)
	}
	ok, err = lim.Allow(ctx, key, 2, time.Minute)
	if err != nil {
		t.Fatalf("third allow err: %v", err)
	}
	if ok {
		t.Fatal("third allow must be rate limited")
	}
}

// T8C：脚本/过期失败时 fail-open，且不得留下永久 lockout key。
func TestT8C_RedisRateLimit_ExpiryFailureFailOpenAndNoPermanentKey(t *testing.T) {
	backend := newT8CRateRedis()
	backend.expireFail = true
	lim := newRedisExternalAuthRateLimiterForTest(backend)
	ctx := context.Background()
	key := "callback:10.0.0.2"

	ok, err := lim.Allow(ctx, key, 1, time.Minute)
	if err != nil {
		t.Fatalf("allow err: %v", err)
	}
	if !ok {
		t.Fatal("expiry/script failure must fail-open (allow request)")
	}
	redisKey := externalAuthRateKeyPrefix + key
	if backend.hasKey(redisKey) {
		t.Fatal("failed rate-limit op must not leave a permanent lockout key")
	}
	if len(backend.delCalls) == 0 {
		t.Fatal("expected Del cleanup after script failure")
	}
}

// T8C：Eval 整体错误同样 fail-open 并清理。
func TestT8C_RedisRateLimit_EvalErrorFailOpen(t *testing.T) {
	backend := newT8CRateRedis()
	backend.evalErr = errors.New("redis down")
	// 预置一个可能残留的 key，验证失败路径会 DEL。
	backend.counts[externalAuthRateKeyPrefix+"start:x"] = 99
	backend.ttls[externalAuthRateKeyPrefix+"start:x"] = -1

	lim := newRedisExternalAuthRateLimiterForTest(backend)
	ok, err := lim.Allow(context.Background(), "start:x", 1, time.Minute)
	if err != nil || !ok {
		t.Fatalf("redis error must fail-open: ok=%v err=%v", ok, err)
	}
	if backend.hasKey(externalAuthRateKeyPrefix + "start:x") {
		t.Fatal("eval error path must delete residual key")
	}
}

// T8C：历史无 TTL key 在成功路径上被重新施加 TTL（自愈）。
func TestT8C_RedisRateLimit_HealsKeyWithoutTTL(t *testing.T) {
	backend := newT8CRateRedis()
	redisKey := externalAuthRateKeyPrefix + "start:heal"
	backend.counts[redisKey] = 5
	backend.ttls[redisKey] = -1 // 无 TTL

	lim := newRedisExternalAuthRateLimiterForTest(backend)
	ok, err := lim.Allow(context.Background(), "start:heal", 20, 30*time.Second)
	if err != nil || !ok {
		t.Fatalf("heal allow: ok=%v err=%v", ok, err)
	}
	ttl, has := backend.ttl(redisKey)
	if !has || ttl != 30*time.Second {
		t.Fatalf("expected healed TTL=30s, has=%v ttl=%v", has, ttl)
	}
}
