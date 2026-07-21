package devhygiene

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestShouldEnableDevelopmentOrphanReaper(t *testing.T) {
	if !ShouldEnableDevelopmentOrphanReaper("development") {
		t.Fatal("expected development enabled")
	}
	if ShouldEnableDevelopmentOrphanReaper("production") {
		t.Fatal("expected production disabled")
	}
	if ShouldEnableDevelopmentOrphanReaper("test") {
		t.Fatal("expected test disabled")
	}
}

func TestStartOrphanPluginReaperNoopWhenDisabled(t *testing.T) {
	var calls atomic.Int32
	stop := StartOrphanPluginReaper(ReaperConfig{
		Enabled: false,
		Cleanup: func() (CleanupResult, error) {
			calls.Add(1)
			return CleanupResult{}, nil
		},
		Delays: []time.Duration{time.Millisecond},
	})
	stop()
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("disabled reaper must not cleanup, calls=%d", calls.Load())
	}
}

func TestStartOrphanPluginReaperRunsAfterDelays(t *testing.T) {
	var calls atomic.Int32
	// 用可立即触发的 after channel 模拟时钟。
	ch1 := make(chan time.Time, 1)
	ch2 := make(chan time.Time, 1)
	pass := 0
	stop := StartOrphanPluginReaper(ReaperConfig{
		Enabled: true,
		Delays:  []time.Duration{time.Millisecond, 2 * time.Millisecond},
		After: func(d time.Duration) <-chan time.Time {
			pass++
			if pass == 1 {
				return ch1
			}
			return ch2
		},
		Cleanup: func() (CleanupResult, error) {
			calls.Add(1)
			return CleanupResult{Selected: []int{1}, Signaled: []int{1}}, nil
		},
	})
	ch1 <- time.Now()
	// 等第一轮
	deadline := time.Now().Add(200 * time.Millisecond)
	for calls.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if calls.Load() != 1 {
		t.Fatalf("after first tick calls=%d", calls.Load())
	}
	ch2 <- time.Now()
	deadline = time.Now().Add(200 * time.Millisecond)
	for calls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if calls.Load() != 2 {
		t.Fatalf("after second tick calls=%d", calls.Load())
	}
	stop()
}

func TestStartOrphanPluginReaperStopCancels(t *testing.T) {
	var calls atomic.Int32
	blocked := make(chan time.Time)
	stop := StartOrphanPluginReaper(ReaperConfig{
		Enabled: true,
		Delays:  []time.Duration{time.Hour},
		After:   func(time.Duration) <-chan time.Time { return blocked },
		Cleanup: func() (CleanupResult, error) {
			calls.Add(1)
			return CleanupResult{}, nil
		},
	})
	stop()
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("stopped reaper must not run, calls=%d", calls.Load())
	}
}
