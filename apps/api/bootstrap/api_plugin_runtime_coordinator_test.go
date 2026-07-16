package bootstrap

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestAPIRuntimeCoordinatorSupervisorClosesAdmissionBeforeReporting(t *testing.T) {
	release := make(chan struct{})
	coordinator := startWorkerRuntimeOwnerTestCoordinator(t, func(context.Context) error {
		<-release
		return errWorkerRuntimeCoordinatorTerminalTest
	})
	var closed atomic.Bool
	failures, done := superviseAPIPluginRuntimeCoordinator(coordinator, func() { closed.Store(true) })
	close(release)

	select {
	case runtimeErr, ok := <-failures:
		if !ok || !errors.Is(runtimeErr, errWorkerRuntimeCoordinatorTerminalTest) {
			t.Fatalf("terminal failure = %v, open=%v", runtimeErr, ok)
		}
		if !closed.Load() {
			t.Fatal("runtime admission remained open when failure became observable")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for API runtime failure")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("API runtime supervisor did not finish")
	}
}

func TestAPIRuntimeCoordinatorSupervisorSafeModeHasNoFailureSource(t *testing.T) {
	failures, done := superviseAPIPluginRuntimeCoordinator(newInactivePluginRuntimeCoordinatorRuntime(), func() {
		t.Fatal("Safe Mode closed a runtime that was never started")
	})
	if failures != nil || done != nil {
		t.Fatalf("Safe Mode supervisor channels = %#v, %#v", failures, done)
	}
}
