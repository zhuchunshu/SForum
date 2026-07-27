package extensionsruntime

import (
	"context"
	"testing"
)

func newRuntimeAdmissionTestGate(t *testing.T) *RuntimeAdmissionGate {
	t.Helper()
	gate, err := NewRuntimeAdmissionGate(RuntimeInstanceIdentity{ExtensionID: "demo.plugin", InstanceID: "runtime-1"})
	if err != nil {
		t.Fatal(err)
	}
	return gate
}

func acquireRuntimeAdmission(t *testing.T, gate *RuntimeAdmissionGate, class RuntimeCallClass) *RuntimeAdmissionLease {
	t.Helper()
	lease, err := gate.Acquire(context.Background(), class)
	if err != nil {
		t.Fatal(err)
	}
	return lease
}
