package extensionsruntime

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPostgresLifecycleRegistryPublicationPrepareMoveRestartCASAndMarkerRefusal(t *testing.T) {
	ctx, pool, journal, request, _ := newLifecyclePublicationIntegration(t)
	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	fence, err := lifecyclePublicationFenceFor(request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	repository := NewPostgresLifecycleRegistryPublicationRepository(pool)
	input := PrepareLifecycleRegistryPublicationInput{
		Fence: fence, SourceDigest: strings.Repeat("d", 64), TargetDigest: strings.Repeat("e", 64),
	}
	ref, err := repository.PrepareLifecycleRegistryPublication(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	conflict := input
	conflict.TargetDigest = strings.Repeat("f", 64)
	if _, err := repository.PrepareLifecycleRegistryPublication(ctx, conflict); !errors.Is(err, ErrLifecycleRegistryPublicationConflict) {
		t.Fatalf("changed immutable plan = %v", err)
	}

	restarted := NewPostgresLifecycleRegistryPublicationRepository(pool)
	if phase, err := restarted.InspectLifecycleRegistryPublication(ctx, ref); err != nil || phase != LifecycleRegistryPublicationSource {
		t.Fatalf("prepared phase after restart = %q, %v", phase, err)
	}
	var applications atomic.Int32
	if err := restarted.MoveLifecycleRegistryPublication(ctx, ref, LifecycleRegistryPublicationTarget, func() error {
		applications.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if phase, err := repository.InspectLifecycleRegistryPublication(ctx, ref); err != nil || phase != LifecycleRegistryPublicationTarget {
		t.Fatalf("published phase = %q, %v", phase, err)
	}

	next := request
	next.Attempt++
	next.TargetBinding.RuntimeInstanceID = "registry-restart-runtime"
	if err := journal.PrepareLifecyclePublication(ctx, next, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	nextFence, err := lifecyclePublicationFenceFor(next, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	nextInput := input
	nextInput.Fence = nextFence
	nextRef, err := restarted.PrepareLifecycleRegistryPublication(ctx, nextInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.InspectLifecycleRegistryPublication(ctx, ref); !errors.Is(err, ErrLifecycleRegistryPublicationConflict) {
		t.Fatalf("stale attempt inspection = %v", err)
	}
	if err := restarted.MoveLifecycleRegistryPublication(ctx, nextRef, LifecycleRegistryPublicationTarget, func() error {
		applications.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if applications.Load() != 2 {
		t.Fatalf("local reconciliation count = %d", applications.Load())
	}
	if err := journal.CommitLifecyclePublication(ctx, next, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	if err := restarted.MoveLifecycleRegistryPublication(context.Background(), nextRef, LifecycleRegistryPublicationSource, func() error {
		t.Fatal("committed marker must reject before local restore")
		return nil
	}); !errors.Is(err, ErrLifecycleRegistryPublicationCommitted) {
		t.Fatalf("committed restore = %v", err)
	}
}
