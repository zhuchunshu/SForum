package bootstrap

import (
	"context"
	"testing"

	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestNewWorkerWithoutRegisteredJobsStartsIdle(t *testing.T) {
	worker, err := NewWorker(context.Background(), config.Config{
		DatabaseURL:            "://not-used-by-idle-worker",
		WorkerDatabaseMaxConns: 1,
	}, nil)
	if err != nil {
		t.Fatalf("new idle worker: %v", err)
	}
	if worker.Client != nil {
		t.Fatal("expected idle worker to skip River client setup")
	}

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
