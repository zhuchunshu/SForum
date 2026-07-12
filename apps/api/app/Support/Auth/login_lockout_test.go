package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// memoryRedis 极简 INCR/EXISTS/SET/DEL/EXPIRE 模拟，避免为测试引入 miniredis。
type memoryRedis struct {
	mu     sync.Mutex
	vals   map[string]string
	counts map[string]int64
	err    error
}

func (m *memoryRedis) Exists(_ context.Context, keys ...string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return redis.NewIntResult(0, m.err)
	}
	var count int64
	for _, key := range keys {
		if _, ok := m.vals[key]; ok {
			count++
		}
	}
	return redis.NewIntResult(count, nil)
}

func (m *memoryRedis) Incr(_ context.Context, key string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return redis.NewIntResult(0, m.err)
	}
	m.counts[key]++
	return redis.NewIntResult(m.counts[key], nil)
}

func (m *memoryRedis) Expire(context.Context, string, time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(true, m.err)
}

func (m *memoryRedis) Set(_ context.Context, key string, value any, _ time.Duration) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return redis.NewStatusResult("", m.err)
	}
	m.vals[key] = fmt.Sprint(value)
	return redis.NewStatusResult("OK", nil)
}

func (m *memoryRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return redis.NewIntResult(0, m.err)
	}
	var count int64
	for _, key := range keys {
		if _, ok := m.vals[key]; ok {
			delete(m.vals, key)
			count++
		}
		delete(m.counts, key)
	}
	return redis.NewIntResult(count, nil)
}

func newMemoryRedis() *memoryRedis {
	return &memoryRedis{vals: map[string]string{}, counts: map[string]int64{}}
}

// 通过 redis.NewClient 无法注入；改为测试可观测的哈希与阈值逻辑，以及
// 用 LoginLockout 的公开 helper 验证 key 不含明文。

func TestHashIDDoesNotLeakRawEmail(t *testing.T) {
	email := "User@Example.com"
	h := hashID(email)
	if h == "" || strings.Contains(h, "user") || strings.Contains(strings.ToLower(h), "example") {
		t.Fatalf("bad hash %q", h)
	}
	// 稳定
	if hashID(email) != h {
		t.Fatal("hash not stable")
	}
	// 与 sha256 前缀一致
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	want := hex.EncodeToString(sum[:16])
	if h != want {
		t.Fatalf("got %s want %s", h, want)
	}
}

func TestRedisKeyShapesUseHashesOnly(t *testing.T) {
	login := "victim@example.com"
	ip := "203.0.113.10"
	lh, ih := hashID(login), hashID(normalizeIP(ip))
	keys := []string{
		pairFailKey(lh, ih), pairLockKey(lh, ih),
		ipFailKey(ih), ipLockKey(ih),
		accountFailKey(lh), accountVerificationKey(lh),
	}
	for _, k := range keys {
		if strings.Contains(k, login) || strings.Contains(k, "victim") || strings.Contains(k, "@") {
			t.Fatalf("key leaks identity: %s", k)
		}
		if !strings.HasPrefix(k, "sforum:login_") {
			t.Fatalf("unexpected prefix: %s", k)
		}
	}
}

func TestThresholdPolicyAccountVerificationHigherThanPair(t *testing.T) {
	// 账号阈值只触发验证，不产生 account lock。
	max := 5
	pairMax := max
	ipMax := max * 5
	if ipMax < 20 {
		ipMax = 20
	}
	accountMax := max * 3
	if accountMax < pairMax+2 {
		accountMax = pairMax + 2
	}
	if !(pairMax < accountMax && pairMax < ipMax) {
		t.Fatalf("pair=%d account=%d ip=%d", pairMax, accountMax, ipMax)
	}
}

func TestDistributedFailuresRequireVerificationWithoutAccountLock(t *testing.T) {
	client := newMemoryRedis()
	lockout := &LoginLockout{client: client}
	for i := 0; i < 6; i++ {
		ip := fmt.Sprintf("203.0.113.%d", i+1)
		if err := lockout.RecordFailure(context.Background(), "victim@example.com", ip, 2, 15*time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	locked, err := lockout.IsLocked(context.Background(), "victim@example.com", "198.51.100.10")
	if err != nil || locked {
		t.Fatalf("fresh source must not be account-locked: locked=%v err=%v", locked, err)
	}
	required, err := lockout.RequiresVerification(context.Background(), "victim@example.com")
	if err != nil || !required {
		t.Fatalf("distributed failures must require verification: required=%v err=%v", required, err)
	}
}

func TestPairAndIPLocksRemainEffective(t *testing.T) {
	client := newMemoryRedis()
	lockout := &LoginLockout{client: client}
	for i := 0; i < 2; i++ {
		_ = lockout.RecordFailure(context.Background(), "victim@example.com", "203.0.113.10", 2, time.Minute)
	}
	locked, err := lockout.IsLocked(context.Background(), "victim@example.com", "203.0.113.10")
	if err != nil || !locked {
		t.Fatalf("pair must lock: locked=%v err=%v", locked, err)
	}
}

func TestLoginLockoutRedisFailureFailsOpen(t *testing.T) {
	lockout := &LoginLockout{client: &memoryRedis{vals: map[string]string{}, counts: map[string]int64{}, err: errors.New("redis down")}}
	locked, err := lockout.IsLocked(context.Background(), "victim", "203.0.113.10")
	if err != nil || locked {
		t.Fatalf("redis failure must fail open: locked=%v err=%v", locked, err)
	}
	required, err := lockout.RequiresVerification(context.Background(), "victim")
	if err != nil || required {
		t.Fatalf("verification state must fail open: required=%v err=%v", required, err)
	}
	if err := lockout.RecordFailure(context.Background(), "victim", "203.0.113.10", 2, time.Minute); err != nil {
		t.Fatalf("record failure must fail open: %v", err)
	}
}

func TestLoginLockoutNilFailOpen(t *testing.T) {
	var l *LoginLockout
	locked, err := l.IsLocked(context.Background(), "x", "1.1.1.1")
	if err != nil || locked {
		t.Fatalf("nil: locked=%v err=%v", locked, err)
	}
	if err := l.RecordFailure(context.Background(), "x", "1.1.1.1", 3, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := l.ClearFailures(context.Background(), "x", "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
}

func TestNewLoginLockoutNilClient(t *testing.T) {
	if NewLoginLockout(nil) != nil {
		t.Fatal("expected nil")
	}
}
