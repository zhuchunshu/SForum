package humanverify

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreMarksTokenSingleUse(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	if err := store.MarkUsed(ctx, "abc", time.Minute); err != nil {
		t.Fatalf("first MarkUsed returned error: %v", err)
	}
	if err := store.MarkUsed(ctx, "abc", time.Minute); !errors.Is(err, ErrReplayed) {
		t.Fatalf("expected replay error, got %v", err)
	}
}

func TestMemoryStoreExpiresUsedToken(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	if err := store.MarkUsed(ctx, "abc", time.Nanosecond); err != nil {
		t.Fatalf("first MarkUsed returned error: %v", err)
	}
	time.Sleep(time.Millisecond)
	if err := store.MarkUsed(ctx, "abc", time.Minute); err != nil {
		t.Fatalf("expected expired token key to be reusable in store, got %v", err)
	}
}

func TestMemoryStoreRateLimitsAfterLimit(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		limited, err := store.IncrementRate(ctx, "ip:127.0.0.1", time.Minute, 3)
		if err != nil {
			t.Fatalf("IncrementRate returned error: %v", err)
		}
		if limited {
			t.Fatalf("attempt %d should not be limited", i+1)
		}
	}
	limited, err := store.IncrementRate(ctx, "ip:127.0.0.1", time.Minute, 3)
	if err != nil {
		t.Fatalf("IncrementRate returned error: %v", err)
	}
	if !limited {
		t.Fatal("expected fourth attempt to be rate limited")
	}
}
