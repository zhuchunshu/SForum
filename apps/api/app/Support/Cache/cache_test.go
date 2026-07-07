package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCacheSetGetDelete(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	if _, found, _ := c.Get(ctx, "missing"); found {
		t.Fatal("expected miss on missing key")
	}

	if err := c.Set(ctx, "k1", []byte("v1"), time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	val, found, err := c.Get(ctx, "k1")
	if err != nil || !found || string(val) != "v1" {
		t.Fatalf("get k1: found=%v val=%q err=%v", found, val, err)
	}

	// 返回的应是拷贝，修改不影响缓存内部。
	val[0] = 'x'
	again, _, _ := c.Get(ctx, "k1")
	if string(again) != "v1" {
		t.Fatalf("cache mutated by caller: %q", again)
	}

	if err := c.Delete(ctx, "k1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, found, _ := c.Get(ctx, "k1"); found {
		t.Fatal("expected miss after delete")
	}
}

func TestMemoryCacheTTLExpiry(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	if err := c.Set(ctx, "k", []byte("v"), 20*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, found, _ := c.Get(ctx, "k"); !found {
		t.Fatal("expected hit before expiry")
	}
	time.Sleep(30 * time.Millisecond)
	if _, found, _ := c.Get(ctx, "k"); found {
		t.Fatal("expected miss after expiry")
	}
}

func TestMemoryCacheIncrement(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()

	v1, err := c.Increment(ctx, "gen")
	if err != nil || v1 != 1 {
		t.Fatalf("first increment: v=%d err=%v", v1, err)
	}
	v2, err := c.Increment(ctx, "gen")
	if err != nil || v2 != 2 {
		t.Fatalf("second increment: v=%d err=%v", v2, err)
	}
	v3, err := c.Increment(ctx, "other")
	if err != nil || v3 != 1 {
		t.Fatalf("other key increment: v=%d err=%v", v3, err)
	}
}

func TestNoopCache(t *testing.T) {
	ctx := context.Background()
	var c Cache = NoopCache{}
	if _, found, _ := c.Get(ctx, "x"); found {
		t.Fatal("noop should always miss")
	}
	if err := c.Set(ctx, "x", []byte("v"), time.Minute); err != nil {
		t.Fatalf("noop set: %v", err)
	}
	if n, _ := c.Increment(ctx, "x"); n != 0 {
		t.Fatalf("noop increment: %d", n)
	}
}
