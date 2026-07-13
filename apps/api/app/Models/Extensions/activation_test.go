package extensions

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestActivationCoordinatorPersistsFailureAndSkipsSameDigest(t *testing.T) {
	extension := installedExtension("crashing.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	extension.Status = StatusEnabled
	extension.PackageDigest = "digest-v1"
	store := &memoryActivationStore{}
	coordinator := NewActivationCoordinator(store)
	runtime := &activationRuntime{startErr: errors.New("plugin crashed during startup")}

	if err := coordinator.Start(context.Background(), runtime, extension, ActivationTriggerStartup, 0, "boot-1"); err == nil {
		t.Fatal("expected startup failure")
	}
	if store.latest.Status != ActivationStatusFailed || store.latest.FailureReason == "" {
		t.Fatalf("failed attempt=%#v", store.latest)
	}
	skip, err := coordinator.ShouldSkipStartup(context.Background(), extension, "boot-2")
	if err != nil || !skip {
		t.Fatalf("same digest skip=%v err=%v", skip, err)
	}
	if store.latest.Status != ActivationStatusSkipped || runtime.starts != 1 {
		t.Fatalf("skipped attempt=%#v starts=%d", store.latest, runtime.starts)
	}

	changed := extension
	changed.PackageDigest = "digest-v2"
	skip, err = coordinator.ShouldSkipStartup(context.Background(), changed, "boot-3")
	if err != nil || skip {
		t.Fatalf("changed digest skip=%v err=%v", skip, err)
	}
}

func TestActivationCoordinatorManualStartRetriesAfterSkip(t *testing.T) {
	extension := installedExtension("recovered.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	extension.Status = StatusEnabled
	extension.PackageDigest = "digest-v1"
	store := &memoryActivationStore{latest: ActivationAttempt{
		ID: 1, ExtensionID: extension.ID, PackageDigest: extension.PackageDigest,
		Status: ActivationStatusSkipped,
	}}
	coordinator := NewActivationCoordinator(store)
	runtime := &activationRuntime{}
	if err := coordinator.Start(context.Background(), runtime, extension, ActivationTriggerEnable, 42, "manual"); err != nil {
		t.Fatal(err)
	}
	if runtime.starts != 1 || store.latest.Status != ActivationStatusHealthy || store.latest.ActorUserID != 42 {
		t.Fatalf("manual recovery attempt=%#v starts=%d", store.latest, runtime.starts)
	}
}

func TestServiceEnableRecordsActivationAttempt(t *testing.T) {
	extension := exactTrustExtension(t, "tracked.enable")
	store := &fakeExtensionStore{items: map[string]Extension{extension.ID: extension}}
	attempts := &memoryActivationStore{}
	runtime := &activationRuntime{}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", runtime,
		WithActivationCoordinator(NewActivationCoordinator(attempts)),
	)
	if _, err := service.Enable(context.Background(), extensionManager(), extension.ID, EnableInput{ConfirmCapabilities: true}); err != nil {
		t.Fatal(err)
	}
	if attempts.latest.Status != ActivationStatusHealthy || attempts.latest.Trigger != ActivationTriggerEnable || attempts.latest.ActorUserID != extensionManager().ID {
		t.Fatalf("activation attempt=%#v", attempts.latest)
	}
}

type memoryActivationStore struct {
	latest ActivationAttempt
	nextID int64
}

func (s *memoryActivationStore) LatestActivationAttempt(_ context.Context, extensionID, packageDigest string) (ActivationAttempt, error) {
	if s.latest.ID == 0 || s.latest.ExtensionID != extensionID || s.latest.PackageDigest != packageDigest {
		return ActivationAttempt{}, ErrActivationAttemptNotFound
	}
	return s.latest, nil
}

func (s *memoryActivationStore) BeginActivationAttempt(_ context.Context, extension Extension, trigger, bootID string, actorUserID int64) (ActivationAttempt, error) {
	s.nextID++
	if s.nextID == 0 {
		s.nextID = 1
	}
	s.latest = ActivationAttempt{
		ID: s.nextID, ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, Trigger: trigger, BootID: bootID,
		Status: ActivationStatusStarting, ActorUserID: actorUserID, StartedAt: time.Now(),
	}
	return s.latest, nil
}

func (s *memoryActivationStore) CompleteActivationAttempt(_ context.Context, attemptID int64, status, reason string) error {
	if s.latest.ID != attemptID {
		return ErrActivationAttemptNotFound
	}
	now := time.Now()
	s.latest.Status = status
	s.latest.FailureReason = reason
	s.latest.CompletedAt = &now
	return nil
}

func (s *memoryActivationStore) RecordSkippedActivation(_ context.Context, extension Extension, bootID, reason string) error {
	s.nextID++
	if s.nextID == 0 {
		s.nextID = 1
	}
	now := time.Now()
	s.latest = ActivationAttempt{
		ID: s.nextID, ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, BootID: bootID,
		Trigger: ActivationTriggerStartup, Status: ActivationStatusSkipped,
		FailureReason: reason, StartedAt: now, CompletedAt: &now,
	}
	return nil
}

type activationRuntime struct {
	starts   int
	stops    int
	startErr error
}

func (*activationRuntime) Check(context.Context, Extension) error { return nil }
func (r *activationRuntime) Start(context.Context, Extension) error {
	r.starts++
	return r.startErr
}
func (r *activationRuntime) Stop(context.Context, Extension) error {
	r.stops++
	return nil
}
func (*activationRuntime) Status(context.Context, Extension) RuntimeStatus {
	return RuntimeStatus{State: RuntimeRunning}
}
func (*activationRuntime) EmitHook(context.Context, string, map[string]any) {}
