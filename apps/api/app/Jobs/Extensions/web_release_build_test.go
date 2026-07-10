package extensionjobs

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

func TestWebReleaseBuildArgsUseSerializedThemeQueue(t *testing.T) {
	args := WebReleaseBuildArgs{ReleaseID: 42}
	if args.Kind() != "extension.web_release_build" {
		t.Fatalf("unexpected job kind %q", args.Kind())
	}
	opts := args.EnqueueOptions()
	if opts.Queue != supportjobs.QueueTheme || opts.MaxAttempts != 3 || !opts.Unique.ByArgs {
		t.Fatalf("unexpected web release options: %#v", opts)
	}
}

func TestWebReleaseBuildDispatcherAdapterForwardsCallerTransaction(t *testing.T) {
	dispatcher := &fakeWebReleaseBuildDispatcher{}
	adapter := WebReleaseBuildDispatcherAdapter{Dispatcher: dispatcher}
	var tx pgx.Tx = &fakeWebReleaseBuildTx{}

	if err := adapter.EnqueueWebReleaseBuildTx(context.Background(), tx, 77); err != nil {
		t.Fatalf("enqueue web release build: %v", err)
	}
	args, ok := dispatcher.args.(WebReleaseBuildArgs)
	if !ok || args.ReleaseID != 77 {
		t.Fatalf("unexpected job args: %#v", dispatcher.args)
	}
	if dispatcher.tx != tx {
		t.Fatal("adapter did not forward the caller transaction")
	}
}

type fakeWebReleaseBuildDispatcher struct {
	tx   pgx.Tx
	args river.JobArgs
	opts supportjobs.EnqueueOptions
}

func (d *fakeWebReleaseBuildDispatcher) EnqueueTx(_ context.Context, tx pgx.Tx, args river.JobArgs, opts supportjobs.EnqueueOptions) (*rivertype.JobInsertResult, error) {
	d.tx = tx
	d.args = args
	d.opts = opts
	return &rivertype.JobInsertResult{}, nil
}

type fakeWebReleaseBuildTx struct {
	pgx.Tx
}
