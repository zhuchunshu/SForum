package extensions

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPostgresLifecycleCoordinatorFencesActionAndHostSteps(t *testing.T) {
	ctx, _, repository, extensionID := newLifecycleRepositoryIntegration(t)
	acquire := lifecycleAcquireTestInput(extensionID, LifecycleOperationEnable)
	extension := Extension{
		ID: extensionID, Version: acquire.ExtensionVersion, Type: TypePlugin,
		Status: StatusEnabled, PackageDigest: acquire.PackageDigest,
	}
	input := LifecycleCoordinatorRunInput{Extension: extension, Acquire: acquire}
	release := make(chan struct{})
	runtime := &lifecycleCoordinatorLeaseTestRuntime{started: make(chan struct{}), release: release}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})

	first := make(chan lifecycleCoordinatorTestRun, 1)
	go func() {
		result, err := coordinator.Run(ctx, input)
		first <- lifecycleCoordinatorTestRun{result: result, err: err}
	}()
	select {
	case <-runtime.started:
	case early := <-first:
		close(release)
		t.Fatalf("first coordinator stopped before leased action: %#v, %v", early.result, early.err)
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("first coordinator did not reach leased action")
	}
	blocked, err := coordinator.Run(ctx, input)
	close(release)
	completed := <-first
	if !errors.Is(err, ErrLifecycleStepLeaseConflict) || blocked.Operation.CompletedAt != nil {
		t.Fatalf("concurrent coordinator = %#v, %v", blocked, err)
	}
	if completed.err != nil || completed.result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("first coordinator = %#v, %v", completed.result, completed.err)
	}
	attempts, err := repository.ListStepAttempts(ctx, completed.result.Operation.ID)
	path, _ := RecommendedLifecyclePath(LifecycleMachineEnable)
	if err != nil || len(attempts) != len(path)-1 {
		t.Fatalf("attempts = %#v, %v", attempts, err)
	}
	seenHost := false
	seenAction := false
	for _, attempt := range attempts {
		if attempt.Status != LifecycleStepSucceeded || attempt.LeaseOwnerToken != "" ||
			attempt.LeaseExpiresAt != nil || attempt.LeaseHeartbeatAt != nil || attempt.LeaseRevision < 2 {
			t.Fatalf("terminal attempt retained lease = %#v", attempt)
		}
		switch attempt.LifecycleAction {
		case lifecycleCoordinatorHostGateAction:
			seenHost = true
		case string(LifecycleMachineEnableAction):
			seenAction = true
		default:
			t.Fatalf("unexpected lifecycle action %q", attempt.LifecycleAction)
		}
		if !strings.HasPrefix(attempt.StepID, "lifecycle.enable.") {
			t.Fatalf("unstable step id %q", attempt.StepID)
		}
	}
	if !seenHost || !seenAction || len(runtime.requestsSnapshot()) != 1 {
		t.Fatalf("host=%t action=%t requests=%#v", seenHost, seenAction, runtime.requestsSnapshot())
	}
}
