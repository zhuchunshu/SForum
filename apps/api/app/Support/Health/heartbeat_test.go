package health

import (
	"testing"
	"time"
)

func TestObserveHeartbeatUnknownWhenMissing(t *testing.T) {
	got := ObserveHeartbeat(time.Time{}, false, time.Now().UTC(), DefaultHeartbeatStaleAfter)
	if !got.Stale || got.Status != "unknown" {
		t.Fatalf("got %#v", got)
	}
}

func TestObserveHeartbeatOKAndStale(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-10 * time.Second)
	ok := ObserveHeartbeat(fresh, true, now, 45*time.Second)
	if ok.Stale || ok.Status != "ok" || ok.AgeSeconds == nil || *ok.AgeSeconds != 10 {
		t.Fatalf("fresh: %#v", ok)
	}

	old := now.Add(-2 * time.Minute)
	stale := ObserveHeartbeat(old, true, now, 45*time.Second)
	if !stale.Stale || stale.Status != "stale" {
		t.Fatalf("stale: %#v", stale)
	}
}
