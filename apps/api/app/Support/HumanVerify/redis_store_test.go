package humanverify

import (
	"testing"
	"time"
)

// TestNewRedisClientAppliesOptions 验证 RedisClientOptions 被正确应用到 go-redis Options。
// 不连接真实 Redis，只检查配置字段（Options() 读取客户端配置）。
func TestNewRedisClientAppliesOptions(t *testing.T) {
	opts := RedisClientOptions{
		PoolSize:        32,
		MinIdleConns:    8,
		DialTimeout:     2 * time.Second,
		ReadTimeout:     1 * time.Second,
		WriteTimeout:    1 * time.Second,
		ConnMaxIdleTime: 15 * time.Minute,
		ConnMaxLifetime: 2 * time.Hour,
	}
	client := NewRedisClient("localhost:6379", "secret", opts)
	defer client.Close()

	got := client.Options()
	if got.PoolSize != 32 {
		t.Fatalf("expected PoolSize 32, got %d", got.PoolSize)
	}
	if got.MinIdleConns != 8 {
		t.Fatalf("expected MinIdleConns 8, got %d", got.MinIdleConns)
	}
	if got.DialTimeout != 2*time.Second {
		t.Fatalf("expected DialTimeout 2s, got %v", got.DialTimeout)
	}
	if got.ReadTimeout != 1*time.Second {
		t.Fatalf("expected ReadTimeout 1s, got %v", got.ReadTimeout)
	}
	if got.WriteTimeout != 1*time.Second {
		t.Fatalf("expected WriteTimeout 1s, got %v", got.WriteTimeout)
	}
	if got.ConnMaxIdleTime != 15*time.Minute {
		t.Fatalf("expected ConnMaxIdleTime 15m, got %v", got.ConnMaxIdleTime)
	}
	if got.ConnMaxLifetime != 2*time.Hour {
		t.Fatalf("expected ConnMaxLifetime 2h, got %v", got.ConnMaxLifetime)
	}
	if got.Addr != "localhost:6379" || got.Password != "secret" {
		t.Fatalf("unexpected addr/password: %s / %s", got.Addr, got.Password)
	}
}

// TestNewRedisClientZeroOptionsFallsBack 验证零值 opts 不覆盖默认行为（Addr/Password 仍生效）。
func TestNewRedisClientZeroOptionsFallsBack(t *testing.T) {
	client := NewRedisClient("redis:6379", "", RedisClientOptions{})
	defer client.Close()

	got := client.Options()
	if got.Addr != "redis:6379" {
		t.Fatalf("expected addr redis:6379, got %s", got.Addr)
	}
	// PoolSize 零值表示走 go-redis 默认（10*CPU），不应是 0
	if got.PoolSize == 0 {
		t.Fatal("expected non-zero default PoolSize")
	}
}
