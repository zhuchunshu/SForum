package extensionjobs

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	webreleaseruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseRuntime"
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

func TestWebReleaseBuildWorkerBuildsThroughReadyWithoutActivating(t *testing.T) {
	store := &fakeWebReleaseBuildStore{detail: extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{ID: 42, Status: extensions.WebReleaseQueued}}}
	builder := &fakeWebReleaseBuilder{}
	locker := &fakeWebReleaseLocker{}
	worker := &WebReleaseBuildWorker{Store: store, Builder: builder, Locker: locker}

	if timeout := worker.Timeout(nil); timeout != -1 {
		t.Fatalf("expected River timeout disabled, got %s", timeout)
	}
	if err := worker.Work(context.Background(), &river.Job[WebReleaseBuildArgs]{Args: WebReleaseBuildArgs{ReleaseID: 42}}); err != nil {
		t.Fatal(err)
	}
	want := []extensions.WebReleaseStatus{
		extensions.WebReleaseResolving, extensions.WebReleaseInstalling,
		extensions.WebReleaseBuilding, extensions.WebReleaseVerifying, extensions.WebReleaseReady,
	}
	if len(store.transitions) != len(want) {
		t.Fatalf("unexpected transitions: %v", store.transitions)
	}
	for index, status := range want {
		if store.transitions[index] != status {
			t.Fatalf("transition %d: want %s, got %s", index, status, store.transitions[index])
		}
	}
	if store.detail.Status != extensions.WebReleaseReady || builder.verifyCalls != 1 || locker.key != webReleaseBuildLockKey {
		t.Fatalf("worker did not stop at ready: status=%s verify=%d lock=%q", store.detail.Status, builder.verifyCalls, locker.key)
	}
}

func TestWebReleaseBuildWorkerTreatsConcurrentSupersedeAsDone(t *testing.T) {
	store := &fakeWebReleaseBuildStore{detail: extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{ID: 43, Status: extensions.WebReleaseQueued}}, supersedeOn: extensions.WebReleaseBuilding}
	worker := &WebReleaseBuildWorker{Store: store, Builder: &fakeWebReleaseBuilder{}, Locker: &fakeWebReleaseLocker{}}

	if err := worker.Work(context.Background(), &river.Job[WebReleaseBuildArgs]{Args: WebReleaseBuildArgs{ReleaseID: 43}}); err != nil {
		t.Fatalf("superseded work should end cleanly: %v", err)
	}
	if store.detail.Status != extensions.WebReleaseSuperseded {
		t.Fatalf("expected superseded status, got %s", store.detail.Status)
	}
}

func TestWebReleaseBuildWorkerChecksSupersedeAfterBuildBeforeVerify(t *testing.T) {
	store := &fakeWebReleaseBuildStore{detail: extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{ID: 44, Status: extensions.WebReleaseQueued}}, supersedeOn: extensions.WebReleaseVerifying}
	builder := &fakeWebReleaseBuilder{}
	worker := &WebReleaseBuildWorker{Store: store, Builder: builder, Locker: &fakeWebReleaseLocker{}}

	if err := worker.Work(context.Background(), &river.Job[WebReleaseBuildArgs]{Args: WebReleaseBuildArgs{ReleaseID: 44}}); err != nil {
		t.Fatalf("superseded work should end cleanly: %v", err)
	}
	if builder.buildCalls != 1 || builder.verifyCalls != 0 {
		t.Fatalf("expected build then stale check before verify, build=%d verify=%d", builder.buildCalls, builder.verifyCalls)
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

type fakeWebReleaseBuildStore struct {
	detail      extensions.WebReleaseDetail
	transitions []extensions.WebReleaseStatus
	supersedeOn extensions.WebReleaseStatus
}

func (s *fakeWebReleaseBuildStore) WebRelease(context.Context, int64) (extensions.WebReleaseDetail, error) {
	return s.detail, nil
}

func (s *fakeWebReleaseBuildStore) TransitionWebRelease(_ context.Context, input extensions.WebReleaseTransitionInput) (extensions.WebRelease, error) {
	if s.supersedeOn != "" && input.NextStatus == s.supersedeOn {
		s.detail.Status = extensions.WebReleaseSuperseded
		return extensions.WebRelease{}, extensions.ErrWebReleaseStale
	}
	if s.detail.Status != input.ExpectedStatus {
		return extensions.WebRelease{}, extensions.ErrWebReleaseStale
	}
	s.detail.Status = input.NextStatus
	s.detail.ArtifactPath = input.ArtifactPath
	s.detail.ArtifactDigest = input.ArtifactDigest
	s.detail.ServerEntry = input.ServerEntry
	s.transitions = append(s.transitions, input.NextStatus)
	return s.detail.WebRelease, nil
}

func (s *fakeWebReleaseBuildStore) RecordWebReleaseDependencySnapshot(context.Context, extensions.WebReleaseDependencySnapshotInput) error {
	return nil
}

type fakeWebReleaseBuilder struct {
	buildCalls  int
	verifyCalls int
}

func (b *fakeWebReleaseBuilder) Prepare(_ context.Context, detail extensions.WebReleaseDetail) (webreleaseruntime.PreparedRelease, error) {
	return webreleaseruntime.PreparedRelease{Detail: detail}, nil
}

func (b *fakeWebReleaseBuilder) Install(context.Context, webreleaseruntime.PreparedRelease) ([]webreleaseruntime.DependencySnapshot, string, error) {
	return []webreleaseruntime.DependencySnapshot{{ExtensionID: "demo.plugin", Digest: "digest"}}, "install\n", nil
}

func (b *fakeWebReleaseBuilder) Build(context.Context, webreleaseruntime.PreparedRelease, string) (webreleaseruntime.BuildResult, error) {
	b.buildCalls++
	return webreleaseruntime.BuildResult{ArtifactPath: "/release/artifact", ServerEntry: "/release/artifact/server/index.mjs"}, nil
}

func (b *fakeWebReleaseBuilder) Verify(_ context.Context, _ webreleaseruntime.PreparedRelease, result webreleaseruntime.BuildResult) (webreleaseruntime.BuildResult, error) {
	b.verifyCalls++
	result.ArtifactDigest = "artifact-digest"
	return result, nil
}

type fakeWebReleaseLocker struct {
	key string
}

func (l *fakeWebReleaseLocker) WithLock(ctx context.Context, key string, action func(context.Context) error) error {
	l.key = key
	return action(ctx)
}
