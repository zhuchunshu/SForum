package extensionsruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestLifecyclePublicationFenceAcceptsEveryPublicationOperation(t *testing.T) {
	tests := []struct {
		operation extensions.LifecycleMachineOperation
		position  int
		mode      LifecycleBoundaryPublicationMode
	}{
		{extensions.LifecycleMachineInstall, 8, LifecycleBoundaryActivate},
		{extensions.LifecycleMachineEnable, 5, LifecycleBoundaryActivate},
		{extensions.LifecycleMachineDisable, 3, LifecycleBoundaryDeactivate},
		{extensions.LifecycleMachineUpgrade, 8, LifecycleBoundaryActivate},
		{extensions.LifecycleMachineRollback, 6, LifecycleBoundaryActivate},
		{extensions.LifecycleMachineUninstall, 3, LifecycleBoundaryDeactivate},
	}
	for _, test := range tests {
		t.Run(string(test.operation), func(t *testing.T) {
			request := lifecyclePublicationTestRequest(t, test.operation, test.position)
			if test.mode == LifecycleBoundaryDeactivate {
				// Deactivation is fenced by the source runtime; target is an exact
				// artifact snapshot and does not have to own a live process.
				request.TargetBinding.RuntimeInstanceID = ""
			}
			fence, err := lifecyclePublicationFenceFor(request, test.mode)
			if err != nil {
				t.Fatal(err)
			}
			if fence.OperationID != request.OperationID || fence.StepID != request.StepID ||
				fence.Attempt != request.Attempt || fence.Target.PackageDigest != request.TargetExtension.PackageDigest {
				t.Fatalf("fence = %#v", fence)
			}
			if test.mode == LifecycleBoundaryDeactivate && !fence.Source.Present {
				t.Fatal("deactivation source artifact was not retained")
			}
		})
	}
}

func TestLifecyclePublicationFenceRejectsIncompleteOrCrossModeIdentity(t *testing.T) {
	base := lifecyclePublicationTestRequest(t, extensions.LifecycleMachineUpgrade, 8)
	tests := []struct {
		name   string
		mode   LifecycleBoundaryPublicationMode
		mutate func(*LifecycleBoundaryRequest)
	}{
		{"wrong mode", LifecycleBoundaryDeactivate, func(*LifecycleBoundaryRequest) {}},
		{"missing operation", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) { r.OperationID = 0 }},
		{"unstable step", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) { r.StepID = " step " }},
		{"missing attempt", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) { r.Attempt = 0 }},
		{"missing target runtime", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) { r.TargetBinding.RuntimeInstanceID = "" }},
		{"target version id", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) {
			r.TargetExtension.ActiveVersionID = 0
			r.TargetBinding.VersionID = 0
		}},
		{"target digest", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) {
			r.TargetExtension.PackageDigest = strings.Repeat("B", 64)
			r.TargetBinding.PackageDigest = r.TargetExtension.PackageDigest
		}},
		{"missing source", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) {
			r.SourceExtension = nil
			r.SourceBinding = extensions.LifecycleRuntimeBinding{}
		}},
		{"source runtime", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) { r.SourceBinding.RuntimeInstanceID = "" }},
		{"foreign source", LifecycleBoundaryActivate, func(r *LifecycleBoundaryRequest) {
			r.SourceExtension.ID = "other.extension"
			r.SourceExtension.Manifest.ID = "other.extension"
			r.SourceBinding.ExtensionID = "other.extension"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := cloneLifecycleBoundaryRequest(base)
			test.mutate(&request)
			if _, err := lifecyclePublicationFenceFor(request, test.mode); !errors.Is(err, ErrLifecyclePublicationJournalInvalid) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestLifecyclePublicationCommittedTreatsAbsentMarkerAsUncommitted(t *testing.T) {
	request := lifecyclePublicationTestRequest(t, extensions.LifecycleMachineEnable, 5)
	fence, err := lifecyclePublicationFenceFor(request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := lifecyclePublicationCommittedResult(lifecyclePublicationRecord{}, pgx.ErrNoRows, fence, true)
	if err != nil || committed {
		t.Fatalf("absent marker = %v, %v", committed, err)
	}

	record := lifecyclePublicationRecord{Fence: fence, LastAttempt: fence.Attempt, CommitMarker: true}
	committed, err = lifecyclePublicationCommittedResult(record, nil, fence, true)
	if err != nil || !committed {
		t.Fatalf("committed marker = %v, %v", committed, err)
	}
	record.Fence.Target.RuntimeInstanceID = "other-runtime"
	if _, err := lifecyclePublicationCommittedResult(record, nil, fence, true); !errors.Is(err, ErrLifecyclePublicationJournalConflict) {
		t.Fatalf("conflicting marker error = %v", err)
	}
}

func TestLifecyclePublicationOperationInspectionIgnoresOnlyPerStepAttempt(t *testing.T) {
	request := lifecyclePublicationTestRequest(t, extensions.LifecycleMachineUpgrade, 8)
	fence, err := lifecyclePublicationFenceFor(request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	record := lifecyclePublicationRecord{Fence: fence, LastAttempt: 1, CommitMarker: true}
	earlyGateFence := fence
	earlyGateFence.Attempt = 9
	committed, err := lifecyclePublicationCommittedResult(record, nil, earlyGateFence, false)
	if err != nil || !committed {
		t.Fatalf("operation marker = %v, %v", committed, err)
	}
	if _, err := lifecyclePublicationCommittedResult(record, nil, earlyGateFence, true); !errors.Is(err, ErrLifecyclePublicationJournalConflict) {
		t.Fatalf("strict attempt error = %v", err)
	}
	earlyGateFence.Target.RuntimeInstanceID = "foreign-target"
	if committed, err := lifecyclePublicationCommittedResult(record, nil, earlyGateFence, false); err != nil || !committed {
		t.Fatalf("restarted runtime marker = %v, %v", committed, err)
	}
	earlyGateFence.Target.VersionID++
	if _, err := lifecyclePublicationCommittedResult(record, nil, earlyGateFence, false); !errors.Is(err, ErrLifecyclePublicationJournalConflict) {
		t.Fatalf("operation artifact fence error = %v", err)
	}
}

func TestPostgresLifecyclePublicationJournalRejectsMissingPool(t *testing.T) {
	request := lifecyclePublicationTestRequest(t, extensions.LifecycleMachineEnable, 5)
	journal := NewPostgresLifecycleBoundaryPublicationJournal(nil)
	if err := journal.PrepareLifecyclePublication(t.Context(), request, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalInvalid) {
		t.Fatalf("prepare error = %v", err)
	}
	if _, err := journal.LifecyclePublicationCommitted(t.Context(), request, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalInvalid) {
		t.Fatalf("inspect error = %v", err)
	}
	if _, err := journal.LifecyclePublicationCommittedForOperation(t.Context(), request, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalInvalid) {
		t.Fatalf("operation inspect error = %v", err)
	}
	if err := journal.CommitLifecyclePublication(t.Context(), request, LifecycleBoundaryActivate); !errors.Is(err, ErrLifecyclePublicationJournalInvalid) {
		t.Fatalf("commit error = %v", err)
	}
}

func lifecyclePublicationTestRequest(
	t *testing.T,
	operation extensions.LifecycleMachineOperation,
	position int,
) LifecycleBoundaryRequest {
	t.Helper()
	gate := lifecycleHostTestRequest(t, operation, position)
	gate.ActionResults = make(map[extensions.LifecycleMachineAction]json.RawMessage)
	for _, action := range lifecycleBoundaryAllowedActions(operation, position) {
		gate.ActionResults[action] = json.RawMessage(fmt.Sprintf(`{"action":%q}`, action))
	}
	request, err := newLifecycleBoundaryRequest(gate)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
