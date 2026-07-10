package extensionjobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	webreleaseruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseRuntime"
)

type WebReleaseBuildArgs struct {
	ReleaseID int64 `json:"release_id" river:"unique"`
}

func (WebReleaseBuildArgs) Kind() string {
	return "extension.web_release_build"
}

func (WebReleaseBuildArgs) EnqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{
		Queue:       supportjobs.QueueTheme,
		MaxAttempts: 3,
		Unique:      river.UniqueOpts{ByArgs: true},
	}
}

type WebReleaseBuildDispatcher interface {
	EnqueueTx(context.Context, pgx.Tx, river.JobArgs, supportjobs.EnqueueOptions) (*rivertype.JobInsertResult, error)
}

type WebReleaseBuildDispatcherAdapter struct {
	Dispatcher WebReleaseBuildDispatcher
}

type WebReleaseBuildStore interface {
	WebRelease(context.Context, int64) (extensions.WebReleaseDetail, error)
	TransitionWebRelease(context.Context, extensions.WebReleaseTransitionInput) (extensions.WebRelease, error)
	RecordWebReleaseDependencySnapshot(context.Context, extensions.WebReleaseDependencySnapshotInput) error
}

type WebReleaseBuilder interface {
	Prepare(context.Context, extensions.WebReleaseDetail) (webreleaseruntime.PreparedRelease, error)
	Install(context.Context, webreleaseruntime.PreparedRelease) ([]webreleaseruntime.DependencySnapshot, string, error)
	Build(context.Context, webreleaseruntime.PreparedRelease, string) (webreleaseruntime.BuildResult, error)
	Verify(context.Context, webreleaseruntime.PreparedRelease, webreleaseruntime.BuildResult) (webreleaseruntime.BuildResult, error)
}

type WebReleaseBuildLocker interface {
	WithLock(context.Context, string, func(context.Context) error) error
}

type WebReleaseBuildWorker struct {
	river.WorkerDefaults[WebReleaseBuildArgs]
	Store   WebReleaseBuildStore
	Builder WebReleaseBuilder
	Locker  WebReleaseBuildLocker
}

const webReleaseBuildLockKey = "sforum.web_release.build"

func (w *WebReleaseBuildWorker) Timeout(*river.Job[WebReleaseBuildArgs]) time.Duration {
	return -1
}

func (w *WebReleaseBuildWorker) Work(ctx context.Context, job *river.Job[WebReleaseBuildArgs]) error {
	if w.Store == nil || w.Builder == nil || w.Locker == nil {
		return fmt.Errorf("web release build worker dependencies are unavailable")
	}
	if job == nil || job.Args.ReleaseID <= 0 {
		return fmt.Errorf("web release build requires a positive release id")
	}
	return w.Locker.WithLock(ctx, webReleaseBuildLockKey, func(lockCtx context.Context) error {
		return w.build(lockCtx, job.Args.ReleaseID)
	})
}

func (w *WebReleaseBuildWorker) build(ctx context.Context, releaseID int64) error {
	detail, err := w.Store.WebRelease(ctx, releaseID)
	if err != nil {
		return err
	}
	if detail.Status == extensions.WebReleaseReady || detail.Status == extensions.WebReleaseActivating || detail.Status == extensions.WebReleaseActive || detail.Status.IsFinal() {
		return nil
	}
	current := detail.Status
	if current == extensions.WebReleaseQueued {
		current, err = w.transition(ctx, releaseID, current, extensions.WebReleaseResolving, webreleaseruntime.BuildResult{})
		if err != nil {
			return w.staleIsDone(ctx, releaseID, err)
		}
	}
	if current != extensions.WebReleaseResolving && current != extensions.WebReleaseInstalling && current != extensions.WebReleaseBuilding && current != extensions.WebReleaseVerifying {
		return fmt.Errorf("web release %d cannot build from %s", releaseID, current)
	}

	prepared, err := w.Builder.Prepare(ctx, detail)
	if err != nil {
		return w.fail(ctx, releaseID, current, webreleaseruntime.BuildResult{}, err)
	}
	if current == extensions.WebReleaseResolving {
		current, err = w.transition(ctx, releaseID, current, extensions.WebReleaseInstalling, webreleaseruntime.BuildResult{})
		if err != nil {
			return w.staleIsDone(ctx, releaseID, err)
		}
	}
	snapshots, installLog, err := w.Builder.Install(ctx, prepared)
	if err != nil {
		return w.fail(ctx, releaseID, current, webreleaseruntime.BuildResult{BuildLog: installLog}, err)
	}
	for _, snapshot := range snapshots {
		if err := w.Store.RecordWebReleaseDependencySnapshot(ctx, extensions.WebReleaseDependencySnapshotInput{
			WebReleaseID: releaseID, ExtensionID: snapshot.ExtensionID,
			ResolvedDependencies: snapshot.Dependencies, Digest: snapshot.Digest,
		}); err != nil {
			return w.fail(ctx, releaseID, current, webreleaseruntime.BuildResult{BuildLog: installLog}, err)
		}
	}
	if current == extensions.WebReleaseInstalling {
		current, err = w.transition(ctx, releaseID, current, extensions.WebReleaseBuilding, webreleaseruntime.BuildResult{})
		if err != nil {
			return w.staleIsDone(ctx, releaseID, err)
		}
	}
	result, err := w.Builder.Build(ctx, prepared, installLog)
	if err != nil {
		return w.fail(ctx, releaseID, current, result, err)
	}
	if current == extensions.WebReleaseBuilding {
		current, err = w.transition(ctx, releaseID, current, extensions.WebReleaseVerifying, result)
		if err != nil {
			return w.staleIsDone(ctx, releaseID, err)
		}
	}
	result, err = w.Builder.Verify(ctx, prepared, result)
	if err != nil {
		return w.fail(ctx, releaseID, current, result, err)
	}
	_, err = w.transition(ctx, releaseID, current, extensions.WebReleaseReady, result)
	if err != nil {
		return w.staleIsDone(ctx, releaseID, err)
	}
	return nil
}

func (w *WebReleaseBuildWorker) transition(ctx context.Context, id int64, current extensions.WebReleaseStatus, next extensions.WebReleaseStatus, result webreleaseruntime.BuildResult) (extensions.WebReleaseStatus, error) {
	release, err := w.Store.TransitionWebRelease(ctx, extensions.WebReleaseTransitionInput{
		ID: id, ExpectedStatus: current, NextStatus: next,
		ArtifactPath: result.ArtifactPath, ArtifactDigest: result.ArtifactDigest,
		ServerEntry: result.ServerEntry, BuildLog: result.BuildLog,
		Reason: "web_release." + string(next),
	})
	return release.Status, err
}

func (w *WebReleaseBuildWorker) fail(ctx context.Context, id int64, current extensions.WebReleaseStatus, result webreleaseruntime.BuildResult, cause error) error {
	updateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, err := w.Store.TransitionWebRelease(updateCtx, extensions.WebReleaseTransitionInput{
		ID: id, ExpectedStatus: current, NextStatus: extensions.WebReleaseFailed,
		BuildLog: result.BuildLog, PublicReason: "extension.build_failed",
		PublicMessage: "Web release build failed.", Reason: "web_release.failed", Message: cause.Error(),
	})
	if errors.Is(err, extensions.ErrWebReleaseStale) {
		return w.staleIsDone(updateCtx, id, err)
	}
	if err != nil && !errors.Is(err, extensions.ErrWebReleaseStale) {
		return errors.Join(cause, err)
	}
	return cause
}

func (w *WebReleaseBuildWorker) staleIsDone(ctx context.Context, id int64, transitionErr error) error {
	if !errors.Is(transitionErr, extensions.ErrWebReleaseStale) {
		return transitionErr
	}
	detail, err := w.Store.WebRelease(ctx, id)
	if err == nil && (detail.Status.IsFinal() || detail.Status == extensions.WebReleaseReady || detail.Status == extensions.WebReleaseActivating || detail.Status == extensions.WebReleaseActive) {
		return nil
	}
	return transitionErr
}

func RegisterWebReleaseBuildWorker(registry *supportjobs.Registry, store WebReleaseBuildStore, builder WebReleaseBuilder, locker WebReleaseBuildLocker) {
	registry.Add(func(workers *river.Workers) error {
		river.AddWorker(workers, &WebReleaseBuildWorker{Store: store, Builder: builder, Locker: locker})
		return nil
	})
}

func (a WebReleaseBuildDispatcherAdapter) EnqueueWebReleaseBuildTx(ctx context.Context, tx pgx.Tx, releaseID int64) error {
	if releaseID <= 0 {
		return fmt.Errorf("web release build requires a positive release id")
	}
	if a.Dispatcher == nil {
		return fmt.Errorf("web release build dispatcher is unavailable")
	}
	args := WebReleaseBuildArgs{ReleaseID: releaseID}
	_, err := a.Dispatcher.EnqueueTx(ctx, tx, args, args.EnqueueOptions())
	return err
}
