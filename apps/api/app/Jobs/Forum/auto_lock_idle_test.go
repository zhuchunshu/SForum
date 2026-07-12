package forumjobs

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
)

type fakeLocker struct {
	calls    int
	idleDays int
	limit    int
	result   int
	err      error
}

func (f *fakeLocker) AutoLockIdleTopics(_ context.Context, idleDays int, limit int) (int, error) {
	f.calls++
	f.idleDays = idleDays
	f.limit = limit
	return f.result, f.err
}

func TestAutoLockIdleWorkerSkipsWhenDisabled(t *testing.T) {
	locker := &fakeLocker{}
	w := &AutoLockIdleWorker{
		Locker: locker,
		IdleDays: func(context.Context) (int, error) {
			return 0, nil
		},
	}
	if err := w.Work(context.Background(), &river.Job[AutoLockIdleArgs]{}); err != nil {
		t.Fatalf("work: %v", err)
	}
	if locker.calls != 0 {
		t.Fatalf("expected no lock call when idleDays=0, got %d", locker.calls)
	}
}

func TestAutoLockIdleWorkerLocksWhenEnabled(t *testing.T) {
	locker := &fakeLocker{result: 3}
	w := &AutoLockIdleWorker{
		Locker:    locker,
		BatchSize: 50,
		IdleDays: func(context.Context) (int, error) {
			return 30, nil
		},
	}
	if err := w.Work(context.Background(), &river.Job[AutoLockIdleArgs]{}); err != nil {
		t.Fatalf("work: %v", err)
	}
	if locker.calls != 1 || locker.idleDays != 30 || locker.limit != 50 {
		t.Fatalf("unexpected call: calls=%d idle=%d limit=%d", locker.calls, locker.idleDays, locker.limit)
	}
}
