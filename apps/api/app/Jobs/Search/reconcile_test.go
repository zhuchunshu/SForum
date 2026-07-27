package searchjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
)

type fakeReconciler struct {
	calls int
	err   error
}

func (r *fakeReconciler) Reconcile(context.Context) error {
	r.calls++
	return r.err
}

func TestReconcileWorkerDelegatesAndRetriesFailures(t *testing.T) {
	expected := errors.New("temporary failure")
	reconciler := &fakeReconciler{err: expected}
	worker := &ReconcileWorker{Reconciler: reconciler}
	if err := worker.Work(context.Background(), &river.Job[ReconcileArgs]{}); !errors.Is(err, expected) {
		t.Fatalf("error = %v", err)
	}
	if reconciler.calls != 1 {
		t.Fatalf("calls = %d", reconciler.calls)
	}
}

func TestReconcileArgsUseMaintenanceQueueAndActiveUniqueness(t *testing.T) {
	opts := (ReconcileArgs{}).EnqueueOptions()
	if opts.Queue != "maintenance" || opts.MaxAttempts != 5 {
		t.Fatalf("options = %+v", opts)
	}
	if !opts.Unique.ByArgs || len(opts.Unique.ByState) == 0 {
		t.Fatalf("unique options = %+v", opts.Unique)
	}
}
