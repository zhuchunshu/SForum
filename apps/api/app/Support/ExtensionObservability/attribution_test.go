package extensionobservability

import (
	"testing"
	"time"
)

func TestObserveSnapshotAttribution(t *testing.T) {
	rec := New(8)
	rec.Observe(Event{
		ExtensionID: "demo.a", PackageDigest: "aa", Surface: "route", Name: "demo.a.home",
		Duration: 10 * time.Millisecond,
	})
	rec.Observe(Event{
		ExtensionID: "demo.a", PackageDigest: "aa", Surface: "cache", Duration: 2 * time.Millisecond,
		ErrorClass: "timeout", Fallback: true,
	})
	rec.Observe(Event{
		ExtensionID: "demo.b", Surface: "hook", Duration: 5 * time.Millisecond,
	})
	snap := rec.Snapshot()
	if len(snap.Aggregates) != 2 || len(snap.Recent) != 3 {
		t.Fatalf("snap = %#v", snap)
	}
	var a Aggregate
	for _, row := range snap.Aggregates {
		if row.ExtensionID == "demo.a" {
			a = row
		}
	}
	if a.Events != 2 || a.Errors != 1 || a.Fallbacks != 1 || a.BySurface["route"] != 1 || a.BySurface["cache"] != 1 {
		t.Fatalf("demo.a = %#v", a)
	}
	if a.AvgLatency <= 0 {
		t.Fatalf("avg latency = %v", a.AvgLatency)
	}
}
