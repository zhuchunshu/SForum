package extensionsruntime

import (
	"context"
	"errors"
	"testing"
)

func TestRuntimeAdmissionResumeRestoresOrdinaryCallsAfterAbortedDrain(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	gate.BeginDrain()
	if _, err := gate.Acquire(context.Background(), RuntimeCallRoute); !errors.Is(err, ErrRuntimeAdmissionDraining) {
		t.Fatalf("draining acquire error = %v", err)
	}
	snapshot, err := gate.Resume()
	if err != nil || snapshot.Draining || snapshot.Forced {
		t.Fatalf("resumed snapshot = %#v, %v", snapshot, err)
	}
	lease := acquireRuntimeAdmission(t, gate, RuntimeCallRoute)
	lease.Release()
}

func TestRuntimeAdmissionResumeRejectsCleanupAndForcedGate(t *testing.T) {
	gate := newRuntimeAdmissionTestGate(t)
	gate.BeginDrain()
	cleanup := acquireRuntimeAdmission(t, gate, RuntimeCallLifecycleCleanup)
	if _, err := gate.Resume(); !errors.Is(err, ErrRuntimeAdmissionBusy) {
		t.Fatalf("cleanup resume error = %v", err)
	}
	cleanup.Release()
	cause := errors.New("forced drain")
	gate.ForceCancel(cause)
	if _, err := gate.Resume(); !errors.Is(err, ErrRuntimeAdmissionForced) || !errors.Is(err, cause) {
		t.Fatalf("forced resume error = %v", err)
	}
}

func TestManagerResumeOnlyReopensExactActiveInstance(t *testing.T) {
	manager := newTwoInstanceRuntimeManager(t, "resume.plugin")
	oldIdentity := RuntimeInstanceIdentity{ExtensionID: "resume.plugin", InstanceID: "instance-1"}
	activeIdentity := RuntimeInstanceIdentity{ExtensionID: "resume.plugin", InstanceID: "instance-2"}
	if _, err := manager.ResumeRuntimeInstance(oldIdentity); !errors.Is(err, ErrRuntimeInstanceNotActive) {
		t.Fatalf("retained resume error = %v", err)
	}
	if _, err := manager.BeginDrain(activeIdentity); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ResumeRuntimeInstance(activeIdentity); err != nil {
		t.Fatal(err)
	}
	lease, err := manager.AcquireRuntimeCall(context.Background(), activeIdentity, RuntimeCallRoute)
	if err != nil {
		t.Fatal(err)
	}
	lease.Release()
	if _, err := manager.AcquireRuntimeCall(context.Background(), oldIdentity, RuntimeCallRoute); !errors.Is(err, ErrRuntimeInstanceNotActive) {
		t.Fatalf("old runtime reopened error = %v", err)
	}
}
