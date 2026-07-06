package bootstrap

import (
	"context"
	"testing"
)

func TestWorkerStartStopAllowNilClient(t *testing.T) {
	worker := &Worker{}
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("start idle worker: %v", err)
	}
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatalf("stop idle worker: %v", err)
	}
}

func TestWorkerCloseRunsCleanupOnce(t *testing.T) {
	calls := 0
	worker := &Worker{
		close: func() {
			calls++
		},
	}

	worker.Close()
	worker.Close()

	if calls != 1 {
		t.Fatalf("expected cleanup once, got %d", calls)
	}
}
