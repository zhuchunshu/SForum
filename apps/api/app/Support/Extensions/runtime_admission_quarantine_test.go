package extensionsruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRuntimeAdmissionQuarantineIsPermanentAndKeepsCleanupAvailable(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	route := acquireRuntimeAdmission(t, gate, RuntimeCallRoute)
	firstCause := errors.New("route incident")
	snapshot := gate.Quarantine(firstCause)
	if !snapshot.Draining || !snapshot.Quarantined || snapshot.Forced ||
		!errors.Is(snapshot.QuarantineCause, firstCause) || snapshot.ActiveTotal != 1 {
		t.Fatalf("quarantine snapshot=%#v", snapshot)
	}
	select {
	case <-route.Context.Done():
		t.Fatal("quarantine cancelled an existing lease")
	default:
	}
	if _, err := gate.Acquire(context.Background(), RuntimeCallRoute); !errors.Is(err, ErrRuntimeAdmissionQuarantined) || !errors.Is(err, firstCause) {
		t.Fatalf("ordinary acquire error=%v", err)
	}
	cleanup := acquireRuntimeAdmission(t, gate, RuntimeCallLifecycleCleanup)
	if _, err := gate.Resume(); !errors.Is(err, ErrRuntimeAdmissionQuarantined) || !errors.Is(err, firstCause) {
		t.Fatalf("quarantined resume error=%v", err)
	}
	second := gate.Quarantine(errors.New("later incident"))
	if !errors.Is(second.QuarantineCause, firstCause) {
		t.Fatalf("repeated quarantine replaced first cause: %v", second.QuarantineCause)
	}
	route.Release()
	cleanup.Release()
}

func TestRuntimeAdmissionQuarantineAndForcePreserveBothIncidents(t *testing.T) {
	for _, forceFirst := range []bool{false, true} {
		name := "quarantine_first"
		if forceFirst {
			name = "force_first"
		}
		t.Run(name, func(t *testing.T) {
			gate := newRuntimeAdmissionTestGate(t)
			quarantineCause := errors.New("route incident")
			forceCause := errors.New("forced uninstall")
			if forceFirst {
				gate.ForceCancel(forceCause)
				gate.Quarantine(quarantineCause)
			} else {
				gate.Quarantine(quarantineCause)
				gate.ForceCancel(forceCause)
			}
			snapshot := gate.Snapshot()
			if !snapshot.Quarantined || !snapshot.Forced || !errors.Is(snapshot.QuarantineCause, quarantineCause) ||
				!errors.Is(snapshot.ForceCause, forceCause) {
				t.Fatalf("snapshot=%#v", snapshot)
			}
			if _, err := gate.Resume(); !errors.Is(err, ErrRuntimeAdmissionForced) || !errors.Is(err, forceCause) {
				t.Fatalf("resume error=%v", err)
			}
		})
	}
}

func TestManagerQuarantineUsesExactArtifactWithoutWaitingForTransitions(t *testing.T) {
	manager := newTwoInstanceRuntimeManager(t, "quarantine.plugin")
	active, err := manager.ActiveRuntimeInstance("quarantine.plugin")
	if err != nil {
		t.Fatal(err)
	}
	exact := RuntimeInstanceArtifactIdentity{
		RuntimeInstanceIdentity: active.Identity,
		ExtensionVersion:        active.ExtensionVersion,
		ArtifactDigest:          active.ArtifactDigest,
	}

	barrierUnlock, err := manager.lockRuntimeSetTransition(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	lifecycleUnlock := manager.lockRuntimeLifecycle(active.Identity.ExtensionID)
	locksHeld := true
	defer func() {
		if locksHeld {
			lifecycleUnlock()
			barrierUnlock()
		}
	}()
	done := make(chan error, 1)
	go func() {
		_, quarantineErr := manager.QuarantineRuntimeInstance(exact, errors.New("route incident"))
		done <- quarantineErr
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("quarantine waited for runtime transition locks")
	}
	lifecycleUnlock()
	barrierUnlock()
	locksHeld = false
	if manager.RuntimeInstanceAvailable(active.Identity) {
		t.Fatal("quarantined runtime remained available")
	}
	if _, err := manager.ResumeRuntimeInstance(active.Identity); !errors.Is(err, ErrRuntimeAdmissionQuarantined) {
		t.Fatalf("resume error=%v", err)
	}
}

func TestManagerQuarantineRejectsArtifactDriftAndNeverFallsBackToReplacement(t *testing.T) {
	manager := newTwoInstanceRuntimeManager(t, "quarantine.stale")
	active, err := manager.ActiveRuntimeInstance("quarantine.stale")
	if err != nil {
		t.Fatal(err)
	}
	stale := RuntimeInstanceArtifactIdentity{
		RuntimeInstanceIdentity: active.Identity,
		ExtensionVersion:        active.ExtensionVersion,
		ArtifactDigest:          "wrong-digest",
	}
	if _, err := manager.QuarantineRuntimeInstance(stale, errors.New("stale incident")); !errors.Is(err, ErrRuntimeInstanceConflict) {
		t.Fatalf("artifact drift error=%v", err)
	}
	if !manager.RuntimeInstanceAvailable(active.Identity) {
		t.Fatal("artifact drift quarantined the active replacement")
	}
	old, err := manager.InspectRuntimeInstance(RuntimeInstanceIdentity{ExtensionID: "quarantine.stale", InstanceID: "instance-1"})
	if err != nil {
		t.Fatal(err)
	}
	oldExact := RuntimeInstanceArtifactIdentity{
		RuntimeInstanceIdentity: old.Identity,
		ExtensionVersion:        old.ExtensionVersion,
		ArtifactDigest:          old.ArtifactDigest,
	}
	if snapshot, err := manager.QuarantineRuntimeInstance(oldExact, errors.New("old incident")); err != nil || !snapshot.Quarantined {
		t.Fatalf("old quarantine snapshot=%#v err=%v", snapshot, err)
	}
	if !manager.RuntimeInstanceAvailable(active.Identity) {
		t.Fatal("old exact incident quarantined the replacement")
	}
}
