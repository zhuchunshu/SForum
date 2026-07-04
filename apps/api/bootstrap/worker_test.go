package bootstrap

import "testing"

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
