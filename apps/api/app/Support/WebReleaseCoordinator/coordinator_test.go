package webreleasecoordinator

import (
	"context"
	"errors"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	webreleaseruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseRuntime"
)

func TestCoordinatorActivatesOnlyAfterMatchingSupervisorAcknowledgement(t *testing.T) {
	store := newCoordinatorStore(extensions.WebReleaseReady, CheckpointPending)
	store.detail.Effects = pluginEnableEffects()
	runtime := &fakeCoordinatorRuntime{}
	pointers := &fakePointerStore{}
	coordinator := New(store, runtime, pointers, directLocker{})

	if err := coordinator.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.detail.Status != extensions.WebReleaseActivating || store.detail.ActivationCheckpoint != CheckpointPointerWritten {
		t.Fatalf("expected pointer wait, got status=%s checkpoint=%s", store.detail.Status, store.detail.ActivationCheckpoint)
	}
	if pointers.writes != 1 || runtime.prepare != 1 || store.forward != 1 {
		t.Fatalf("unexpected activation work: pointer=%d prepare=%d effects=%d", pointers.writes, runtime.prepare, store.forward)
	}

	pointers.active = webreleaseruntime.ActiveRelease{
		ReleaseID: store.detail.ID, CompositionHash: store.detail.CompositionHash,
		ArtifactDigest: store.detail.ArtifactDigest,
	}
	if err := coordinator.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.detail.Status != extensions.WebReleaseActive || runtime.finalize != 1 || store.revocations != 1 {
		t.Fatalf("release was not finalized: status=%s runtime=%d revocations=%d", store.detail.Status, runtime.finalize, store.revocations)
	}
}

func TestCoordinatorCompensatesFailureExactlyOnce(t *testing.T) {
	store := newCoordinatorStore(extensions.WebReleaseActivating, CheckpointPointerWritten)
	store.detail.Effects = pluginEnableEffects()
	runtime := &fakeCoordinatorRuntime{}
	pointers := &fakePointerStore{failure: webreleaseruntime.Failure{ReleaseID: 7, Reason: "web_release.start_failed", Message: "unhealthy"}}
	coordinator := New(store, runtime, pointers, directLocker{})

	if err := coordinator.Reconcile(context.Background()); err == nil {
		t.Fatal("expected supervisor failure")
	}
	if store.detail.Status != extensions.WebReleaseFailed || store.backward != 1 || runtime.compensate != 1 || pointers.restores != 1 {
		t.Fatalf("failure was not compensated exactly once: %#v runtime=%#v pointers=%#v", store, runtime, pointers)
	}
	if err := coordinator.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.backward != 1 || runtime.compensate != 1 {
		t.Fatal("final release was compensated more than once")
	}
}

func pluginEnableEffects() []extensions.WebReleaseExtensionEffect {
	return []extensions.WebReleaseExtensionEffect{{
		ExtensionID: "demo.plugin", PreviousStatus: extensions.StatusDisabled, TargetStatus: extensions.StatusEnabled,
	}}
}

func TestCoordinatorResumesAfterEffectsCommitWithoutRepeatingSideEffects(t *testing.T) {
	store := newCoordinatorStore(extensions.WebReleaseActivating, CheckpointEffectsCommitted)
	runtime := &fakeCoordinatorRuntime{}
	pointers := &fakePointerStore{}
	coordinator := New(store, runtime, pointers, directLocker{})

	if err := coordinator.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.prepare != 0 || store.forward != 0 || pointers.writes != 1 {
		t.Fatalf("crash recovery repeated committed work: runtime=%d effects=%d pointers=%d", runtime.prepare, store.forward, pointers.writes)
	}
}

func TestCoordinatorIgnoresMismatchedAcknowledgement(t *testing.T) {
	store := newCoordinatorStore(extensions.WebReleaseActivating, CheckpointPointerWritten)
	runtime := &fakeCoordinatorRuntime{}
	pointers := &fakePointerStore{active: webreleaseruntime.ActiveRelease{
		ReleaseID: store.detail.ID, CompositionHash: "other", ArtifactDigest: store.detail.ArtifactDigest,
	}}
	coordinator := New(store, runtime, pointers, directLocker{})

	if err := coordinator.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.detail.Status != extensions.WebReleaseActivating || runtime.finalize != 0 || store.revocations != 0 {
		t.Fatalf("mismatched acknowledgement finalized release: status=%s runtime=%d revocations=%d", store.detail.Status, runtime.finalize, store.revocations)
	}
}

func TestCoordinatorRuntimePrepareFailureDoesNotApplyEffects(t *testing.T) {
	store := newCoordinatorStore(extensions.WebReleaseReady, CheckpointPending)
	runtime := &fakeCoordinatorRuntime{prepareErr: errors.New("start failed")}
	pointers := &fakePointerStore{}
	coordinator := New(store, runtime, pointers, directLocker{})

	if err := coordinator.Reconcile(context.Background()); err == nil {
		t.Fatal("expected runtime prepare failure")
	}
	if store.detail.Status != extensions.WebReleaseFailed || store.forward != 0 || store.backward != 0 || runtime.compensate != 0 || pointers.restores != 0 {
		t.Fatalf("prepare failure performed effects or compensation: store=%#v runtime=%#v pointers=%#v", store, runtime, pointers)
	}
}

func TestCoordinatorResumesSupervisorActiveCheckpoint(t *testing.T) {
	store := newCoordinatorStore(extensions.WebReleaseActivating, CheckpointSupervisorActive)
	runtime := &fakeCoordinatorRuntime{}
	pointers := &fakePointerStore{active: webreleaseruntime.ActiveRelease{
		ReleaseID: store.detail.ID, CompositionHash: store.detail.CompositionHash, ArtifactDigest: store.detail.ArtifactDigest,
	}}
	coordinator := New(store, runtime, pointers, directLocker{})

	if err := coordinator.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.detail.Status != extensions.WebReleaseActive || runtime.prepare != 0 || store.forward != 0 || pointers.writes != 0 || runtime.finalize != 1 {
		t.Fatalf("supervisor checkpoint recovery repeated work: store=%#v runtime=%#v pointers=%#v", store, runtime, pointers)
	}
}

func TestCoordinatorDefersDisableEffectsUntilSupervisorAcknowledges(t *testing.T) {
	store := newCoordinatorStore(extensions.WebReleaseReady, CheckpointPending)
	store.detail.Effects = []extensions.WebReleaseExtensionEffect{{
		ExtensionID: "demo.plugin", PreviousStatus: extensions.StatusEnabled, TargetStatus: extensions.StatusDisabled,
	}}
	runtime := &fakeCoordinatorRuntime{}
	pointers := &fakePointerStore{}
	coordinator := New(store, runtime, pointers, directLocker{})

	if err := coordinator.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.forward != 0 {
		t.Fatalf("disable effect committed before supervisor acknowledgement: %d", store.forward)
	}
	pointers.active = webreleaseruntime.ActiveRelease{
		ReleaseID: store.detail.ID, CompositionHash: store.detail.CompositionHash, ArtifactDigest: store.detail.ArtifactDigest,
	}
	if err := coordinator.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.forward != 1 || runtime.finalize != 1 || store.detail.Status != extensions.WebReleaseActive {
		t.Fatalf("disable effect was not finalized after acknowledgement: store=%#v runtime=%#v", store, runtime)
	}
}

type fakeCoordinatorStore struct {
	detail      extensions.WebReleaseDetail
	forward     int
	backward    int
	revocations int
}

func newCoordinatorStore(status extensions.WebReleaseStatus, checkpoint string) *fakeCoordinatorStore {
	return &fakeCoordinatorStore{detail: extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{
		ID: 7, Status: status, ActivationCheckpoint: checkpoint,
		CompositionHash: "composition", ArtifactDigest: "artifact", ArtifactPath: "/release/artifact",
		ServerEntry: "/release/artifact/server/index.mjs", ActiveThemeID: "demo.theme", ThemeVersion: "1.0.0",
		ReloadMode: extensions.WebReleaseReloadPrompt,
	}}}
}

func (s *fakeCoordinatorStore) NextActivation(context.Context) (extensions.WebReleaseDetail, error) {
	if s.detail.Status != extensions.WebReleaseReady && s.detail.Status != extensions.WebReleaseActivating {
		return extensions.WebReleaseDetail{}, ErrNoActivation
	}
	return s.detail, nil
}

func (s *fakeCoordinatorStore) Transition(_ context.Context, input extensions.WebReleaseTransitionInput) (extensions.WebRelease, error) {
	if s.detail.Status != input.ExpectedStatus {
		return extensions.WebRelease{}, extensions.ErrWebReleaseStale
	}
	s.detail.Status = input.NextStatus
	if input.ActivationCheckpoint != "" {
		s.detail.ActivationCheckpoint = input.ActivationCheckpoint
	}
	return s.detail.WebRelease, nil
}

func (s *fakeCoordinatorStore) SetCheckpoint(_ context.Context, _ int64, expected, next string) error {
	if s.detail.ActivationCheckpoint != expected {
		return extensions.ErrWebReleaseStale
	}
	s.detail.ActivationCheckpoint = next
	return nil
}

func (s *fakeCoordinatorStore) ApplyEffects(_ context.Context, _ extensions.WebReleaseDetail, forward bool) error {
	if forward {
		s.forward++
	} else {
		s.backward++
	}
	return nil
}

func (s *fakeCoordinatorStore) FinalizeRevocations(context.Context, int64) error {
	s.revocations++
	return nil
}

type fakeCoordinatorRuntime struct {
	prepare, finalize, compensate int
	prepareErr                    error
}

func (r *fakeCoordinatorRuntime) Prepare(context.Context, extensions.WebReleaseDetail) error {
	r.prepare++
	return r.prepareErr
}
func (r *fakeCoordinatorRuntime) Finalize(context.Context, extensions.WebReleaseDetail) error {
	r.finalize++
	return nil
}
func (r *fakeCoordinatorRuntime) Compensate(context.Context, extensions.WebReleaseDetail) error {
	r.compensate++
	return nil
}

type fakePointerStore struct {
	writes, restores int
	active           webreleaseruntime.ActiveRelease
	failure          webreleaseruntime.Failure
}

func (p *fakePointerStore) WriteCurrent(context.Context, webreleaseruntime.CurrentRelease) error {
	p.writes++
	return nil
}
func (p *fakePointerStore) ReadActive(context.Context) (webreleaseruntime.ActiveRelease, error) {
	if p.active.ReleaseID == 0 {
		return webreleaseruntime.ActiveRelease{}, errors.New("not found")
	}
	return p.active, nil
}
func (p *fakePointerStore) ReadFailure(context.Context, int64) (webreleaseruntime.Failure, error) {
	if p.failure.ReleaseID == 0 {
		return webreleaseruntime.Failure{}, errors.New("not found")
	}
	return p.failure, nil
}
func (p *fakePointerStore) RestorePrevious(context.Context, extensions.WebReleaseDetail) error {
	p.restores++
	return nil
}

type directLocker struct{}

func (directLocker) WithLock(ctx context.Context, _ string, action func(context.Context) error) error {
	return action(ctx)
}
