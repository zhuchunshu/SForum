package systemtier

import (
	"context"
	"errors"
	"testing"
)

func TestSafeModeBypassesBeforeLoad(t *testing.T) {
	ctx := context.Background()
	reg := New()
	if err := reg.Upsert(ctx, Member{ExtensionID: "sys.auth", Role: RoleAuth, Priority: 1, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	order, err := reg.LoadOrder(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if order != nil {
		t.Fatalf("safe mode must return nil before loading code: %#v", order)
	}
	order, err = reg.LoadOrder(ctx, false)
	if err != nil || len(order) != 1 {
		t.Fatalf("normal load = %#v err=%v", order, err)
	}
}

func TestCLIDisableWithoutLoadingCode(t *testing.T) {
	// 同一 durable store 跨进程：CLI Disable 后 API LoadOrder 不可见。
	ctx := context.Background()
	store := NewMemoryStore()
	cli := NewWithStore(store)
	api := NewWithStore(store)
	if err := cli.Upsert(ctx, Member{
		ExtensionID: "sys.cache", Role: RoleCache, Priority: 10, Enabled: true, UpdatedBy: "cli",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cli.Disable(ctx, "sys.cache", "recovery-cli"); err != nil {
		t.Fatal(err)
	}
	order, err := api.LoadOrder(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 0 {
		t.Fatalf("disabled member still loading: %#v", order)
	}
	snap, err := api.Snapshot(ctx)
	if err != nil || !snap.SafeModeBypass || len(snap.Members) != 1 || snap.Members[0].Enabled {
		t.Fatalf("snapshot = %#v err=%v", snap, err)
	}
}

func TestDisableMissing(t *testing.T) {
	if err := New().Disable(context.Background(), "missing.x", "cli"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}
