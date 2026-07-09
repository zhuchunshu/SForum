package identityjobs

import (
	"context"
	"log/slog"
	"testing"
)

type fakeCleanupStore struct {
	calledKeepDays int
	deleted        int
	err            error
}

func (f *fakeCleanupStore) DeleteOldRevokedSessions(_ context.Context, keepDays int) (int, error) {
	f.calledKeepDays = keepDays
	if f.err != nil {
		return 0, f.err
	}
	return f.deleted, nil
}

func TestCleanupSessionsWorkerDeletesWithResolvedKeepDays(t *testing.T) {
	store := &fakeCleanupStore{deleted: 7}
	worker := &CleanupSessionsWorker{
		Store:    store,
		KeepDays: func(_ context.Context) (int, error) { return 45, nil },
		Logger:   slog.New(slog.NewTextHandler(&discardWriter{}, nil)),
	}
	if err := worker.Work(context.Background(), nil); err != nil {
		t.Fatalf("Work returned error: %v", err)
	}
	if store.calledKeepDays != 45 {
		t.Fatalf("expected keepDays 45, got %d", store.calledKeepDays)
	}
}

func TestCleanupSessionsWorkerFallsBackToDefaultKeepDays(t *testing.T) {
	store := &fakeCleanupStore{deleted: 0}
	worker := &CleanupSessionsWorker{
		Store:  store,
		Logger: slog.New(slog.NewTextHandler(&discardWriter{}, nil)),
	}
	// 无 KeepDays resolver，应回退到 30。
	if err := worker.Work(context.Background(), nil); err != nil {
		t.Fatalf("Work returned error: %v", err)
	}
	if store.calledKeepDays != 30 {
		t.Fatalf("expected fallback keepDays 30, got %d", store.calledKeepDays)
	}
}

func TestCleanupSessionsWorkerRequiresStore(t *testing.T) {
	worker := &CleanupSessionsWorker{}
	if err := worker.Work(context.Background(), nil); err == nil {
		t.Fatal("expected error when store is nil")
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
