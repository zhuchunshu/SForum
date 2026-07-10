package extensions

import (
	"errors"
	"testing"
)

func TestWebReleaseTransitionAllowsApprovedEdges(t *testing.T) {
	tests := []struct {
		from WebReleaseStatus
		to   WebReleaseStatus
	}{
		{WebReleaseQueued, WebReleaseResolving},
		{WebReleaseQueued, WebReleaseFailed},
		{WebReleaseQueued, WebReleaseSuperseded},
		{WebReleaseResolving, WebReleaseInstalling},
		{WebReleaseResolving, WebReleaseFailed},
		{WebReleaseResolving, WebReleaseSuperseded},
		{WebReleaseInstalling, WebReleaseBuilding},
		{WebReleaseInstalling, WebReleaseFailed},
		{WebReleaseInstalling, WebReleaseSuperseded},
		{WebReleaseBuilding, WebReleaseVerifying},
		{WebReleaseBuilding, WebReleaseFailed},
		{WebReleaseBuilding, WebReleaseSuperseded},
		{WebReleaseVerifying, WebReleaseReady},
		{WebReleaseVerifying, WebReleaseFailed},
		{WebReleaseVerifying, WebReleaseSuperseded},
		{WebReleaseReady, WebReleaseActivating},
		{WebReleaseReady, WebReleaseFailed},
		{WebReleaseReady, WebReleaseSuperseded},
		{WebReleaseActivating, WebReleaseActive},
		{WebReleaseActivating, WebReleaseFailed},
		{WebReleaseActive, WebReleaseInactive},
	}

	for _, test := range tests {
		t.Run(string(test.from)+"_to_"+string(test.to), func(t *testing.T) {
			if err := ValidateWebReleaseTransition(test.from, test.to); err != nil {
				t.Fatalf("expected transition %s -> %s to be valid: %v", test.from, test.to, err)
			}
		})
	}
}

func TestWebReleaseTransitionRequiresCompensationForRollback(t *testing.T) {
	if err := ValidateWebReleaseTransition(WebReleaseActive, WebReleaseRolledBack); !errors.Is(err, ErrInvalidWebReleaseTransition) {
		t.Fatalf("expected ordinary active -> rolled_back to fail, got %v", err)
	}
	if err := ValidateWebReleaseTransitionWithOptions(
		WebReleaseActive,
		WebReleaseRolledBack,
		TransitionOptions{Compensation: true},
	); err != nil {
		t.Fatalf("expected compensation rollback to be valid: %v", err)
	}
}

func TestWebReleaseTransitionTreatsSameKnownStateAsIdempotent(t *testing.T) {
	for _, status := range allWebReleaseStatuses() {
		if err := ValidateWebReleaseTransition(status, status); err != nil {
			t.Fatalf("expected %s -> %s to be idempotent: %v", status, status, err)
		}
	}
}

func TestWebReleaseTransitionRejectsSkipsAndFinalStateMutation(t *testing.T) {
	tests := []struct {
		name string
		from WebReleaseStatus
		to   WebReleaseStatus
	}{
		{name: "queued cannot skip to active", from: WebReleaseQueued, to: WebReleaseActive},
		{name: "ready cannot skip to active", from: WebReleaseReady, to: WebReleaseActive},
		{name: "activating cannot be superseded", from: WebReleaseActivating, to: WebReleaseSuperseded},
		{name: "active cannot fail", from: WebReleaseActive, to: WebReleaseFailed},
		{name: "failed is immutable", from: WebReleaseFailed, to: WebReleaseBuilding},
		{name: "superseded is immutable", from: WebReleaseSuperseded, to: WebReleaseQueued},
		{name: "inactive is immutable", from: WebReleaseInactive, to: WebReleaseActive},
		{name: "rolled back is immutable", from: WebReleaseRolledBack, to: WebReleaseQueued},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateWebReleaseTransition(test.from, test.to); !errors.Is(err, ErrInvalidWebReleaseTransition) {
				t.Fatalf("expected transition %s -> %s to fail, got %v", test.from, test.to, err)
			}
		})
	}
}

func TestWebReleaseTransitionRejectsUnknownStates(t *testing.T) {
	for _, test := range []struct {
		from WebReleaseStatus
		to   WebReleaseStatus
	}{
		{WebReleaseStatus("unknown"), WebReleaseQueued},
		{WebReleaseQueued, WebReleaseStatus("unknown")},
		{WebReleaseStatus("unknown"), WebReleaseStatus("unknown")},
	} {
		if err := ValidateWebReleaseTransition(test.from, test.to); !errors.Is(err, ErrInvalidWebReleaseTransition) {
			t.Fatalf("expected unknown transition %q -> %q to fail, got %v", test.from, test.to, err)
		}
	}
}

func TestWebReleaseStatusClassifiesFinalAndLiveStates(t *testing.T) {
	for _, status := range []WebReleaseStatus{
		WebReleaseFailed,
		WebReleaseSuperseded,
		WebReleaseInactive,
		WebReleaseRolledBack,
	} {
		if !status.IsFinal() || status.IsLive() {
			t.Fatalf("expected %s to be final and not live", status)
		}
	}

	for _, status := range []WebReleaseStatus{
		WebReleaseQueued,
		WebReleaseResolving,
		WebReleaseInstalling,
		WebReleaseBuilding,
		WebReleaseVerifying,
		WebReleaseReady,
		WebReleaseActivating,
		WebReleaseActive,
	} {
		if status.IsFinal() || !status.IsLive() {
			t.Fatalf("expected %s to be live and not final", status)
		}
	}

	unknown := WebReleaseStatus("unknown")
	if unknown.IsFinal() || unknown.IsLive() {
		t.Fatalf("unknown status must be neither final nor live")
	}
}

func allWebReleaseStatuses() []WebReleaseStatus {
	return []WebReleaseStatus{
		WebReleaseQueued,
		WebReleaseResolving,
		WebReleaseInstalling,
		WebReleaseBuilding,
		WebReleaseVerifying,
		WebReleaseReady,
		WebReleaseActivating,
		WebReleaseActive,
		WebReleaseInactive,
		WebReleaseFailed,
		WebReleaseSuperseded,
		WebReleaseRolledBack,
	}
}
