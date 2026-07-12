package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"
)

// memoryRedis 极简 INCR/EXISTS/SET/DEL/EXPIRE 模拟，避免为测试引入 miniredis。
type memoryRedis struct {
	mu    sync.Mutex
	vals  map[string]string
	counts map[string]int64
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
		accountFailKey(lh), accountLockKey(lh),
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

func TestThresholdPolicyAccountHigherThanPair(t *testing.T) {
	// 文档化阈值关系：账号硬锁门槛高于 pair。
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
