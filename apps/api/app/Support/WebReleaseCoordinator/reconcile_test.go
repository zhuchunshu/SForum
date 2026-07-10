package webreleasecoordinator

import (
	"context"
	"testing"
	"time"
)

func TestCoordinatorStartStopAreIdempotent(t *testing.T) {
	store := newCoordinatorStore("failed", CheckpointPending)
	coordinator := New(store, &fakeCoordinatorRuntime{}, &fakePointerStore{}, directLocker{})
	coordinator.interval = time.Millisecond
	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Start(ctx); err != nil {
		t.Fatal(err)
	}
	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := coordinator.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
}
