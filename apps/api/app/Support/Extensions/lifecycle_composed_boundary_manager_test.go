package extensionsruntime

import (
	"context"
	"errors"
	"testing"
)

func TestManagerPublishDrainedKeepsEveryOrdinaryClassClosedUntilResume(t *testing.T) {
	starter := newManagerStagedStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	sourceExtension := managerStagedExtension("composed.closed", "1.0.0", "digest-1")
	if err := manager.Start(context.Background(), sourceExtension); err != nil {
		t.Fatal(err)
	}
	source, _ := manager.ActiveRuntimeInstance(sourceExtension.ID)
	targetExtension := managerStagedExtension(sourceExtension.ID, "2.0.0", "digest-2")
	target, err := manager.StageRuntimeInstance(context.Background(), targetExtension)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.HealthRuntimeInstance(context.Background(), target.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.BeginDrain(source.Identity); err != nil {
		t.Fatal(err)
	}
	if err := manager.WaitDrain(context.Background(), source.Identity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.BeginDrain(target.Identity); err != nil {
		t.Fatal(err)
	}
	if err := manager.WaitDrain(context.Background(), target.Identity); err != nil {
		t.Fatal(err)
	}
	published, err := manager.PublishDrainedRuntimeInstance(context.Background(), target.Identity)
	if err != nil || !published.Active || !published.Admission.Draining || published.Admission.ActiveTotal != 0 {
		t.Fatalf("published = %#v, %v", published, err)
	}

	for _, class := range []RuntimeCallClass{
		RuntimeCallRoute, RuntimeCallPage, RuntimeCallHook, RuntimeCallProvider,
		RuntimeCallService, RuntimeCallHost, RuntimeCallJob, RuntimeCallSchedule,
	} {
		if _, _, err := manager.AcquireActiveRuntimeCall(context.Background(), sourceExtension.ID, class); !errors.Is(err, ErrRuntimeAdmissionDraining) {
			t.Fatalf("class %q admission = %v", class, err)
		}
	}
	cleanup := acquireManagerRuntimeCall(t, manager, target.Identity, RuntimeCallLifecycleCleanup)
	cleanup.Release()
	if _, err := manager.ResumeRuntimeInstance(target.Identity); err != nil {
		t.Fatal(err)
	}
	_, route, err := manager.AcquireActiveRuntimeCall(context.Background(), sourceExtension.ID, RuntimeCallRoute)
	if err != nil {
		t.Fatal(err)
	}
	route.Release()
}

func TestManagerActiveDrainedRepublishBlocksCleanupDuringProtocolSwitch(t *testing.T) {
	starter := newManagerStagedStarter()
	manager := NewManager(ManagerConfig{Starter: starter})
	extension := managerStagedExtension("composed.replay", "1.0.0", "digest-1")
	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatal(err)
	}
	active, _ := manager.ActiveRuntimeInstance(extension.ID)
	if _, err := manager.BeginDrain(active.Identity); err != nil {
		t.Fatal(err)
	}
	if err := manager.WaitDrain(context.Background(), active.Identity); err != nil {
		t.Fatal(err)
	}
	starter.publishStarted = make(chan struct{}, 1)
	starter.publishContinue = make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := manager.PublishDrainedRuntimeInstance(context.Background(), active.Identity)
		result <- err
	}()
	<-starter.publishStarted
	if _, err := manager.AcquireRuntimeCall(context.Background(), active.Identity, RuntimeCallLifecycleCleanup); !errors.Is(err, ErrRuntimeInstanceBusy) {
		t.Fatalf("cleanup during active replay = %v", err)
	}
	close(starter.publishContinue)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}
