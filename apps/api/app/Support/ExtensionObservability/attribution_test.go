package extensionobservability

import (
	"errors"
	"testing"
	"time"
)

func TestProcessWritersFromRealSurfaces(t *testing.T) {
	ResetProcessForTest(New(64))
	ObserveRoute("demo.plugin", "aa", "GET /x", time.Millisecond, nil)
	ObserveHook("demo.plugin", "aa", "topic.created", 2*time.Millisecond, nil)
	ObserveSQL("demo.plugin", "aa", "SELECT", time.Microsecond, errors.New("timeout"))
	ObserveCache("demo.plugin", "aa", "remember", time.Microsecond, nil)
	ObserveRPC("demo.plugin", "aa", "Host.Call", time.Millisecond, nil)
	ObserveJob("demo.plugin", "aa", "cleanup", 5*time.Millisecond, nil)

	snap := Process().Snapshot()
	if len(snap.Aggregates) != 1 {
		t.Fatalf("aggregates = %#v", snap.Aggregates)
	}
	agg := snap.Aggregates[0]
	if agg.Events != 6 || agg.Errors != 1 {
		t.Fatalf("agg = %#v", agg)
	}
	for _, surface := range []string{SurfaceRoute, SurfaceHook, SurfaceSQL, SurfaceCache, SurfaceRPC, SurfaceJob} {
		if agg.BySurface[surface] == 0 {
			t.Fatalf("missing surface %s in %#v", surface, agg.BySurface)
		}
	}
	if len(snap.Recent) != 6 {
		t.Fatalf("recent = %d", len(snap.Recent))
	}
}
