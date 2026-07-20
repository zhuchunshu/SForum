package extensionsruntime

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	upgradedPlan := conflict
	upgradedPlan.SourceDigest = strings.Repeat("a", 64)
	upgradedPlan.CompatibleSourceDigests = []string{input.SourceDigest}
	upgradedPlan.CompatibleTargetDigests = []string{input.TargetDigest}
	if _, err := repository.PrepareLifecycleRegistryPublication(ctx, upgradedPlan); err != nil {
		t.Fatalf("explicit legacy in-flight plan compatibility = %v", err)
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

func TestPostgresLifecycleRegistryPublicationMoveLeavesMaxConnsOnePoolAvailableToApply(t *testing.T) {
	ctx, pool, journal, request, _ := newLifecyclePublicationIntegration(t)
	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	fence, err := lifecyclePublicationFenceFor(request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}

	poolConfig := pool.Config().Copy()
	poolConfig.MaxConns = 1
	poolConfig.MinConns = 0
	singlePool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(singlePool.Close)
	repository := NewPostgresLifecycleRegistryPublicationRepository(singlePool)
	ref, err := repository.PrepareLifecycleRegistryPublication(ctx, PrepareLifecycleRegistryPublicationInput{
		Fence: fence, SourceDigest: strings.Repeat("8", 64), TargetDigest: strings.Repeat("9", 64),
	})
	if err != nil {
		t.Fatal(err)
	}

	applyErr := errors.New("local registry apply failed")
	runApply := func(wantErr error) error {
		moveCtx, cancelMove := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancelMove()
		return repository.MoveLifecycleRegistryPublication(
			moveCtx,
			ref,
			LifecycleRegistryPublicationTarget,
			func() error {
				tx, beginErr := singlePool.BeginTx(moveCtx, pgx.TxOptions{IsoLevel: pgx.Serializable})
				if beginErr != nil {
					return beginErr
				}
				defer func() { _ = tx.Rollback(context.Background()) }()
				var value int
				if queryErr := tx.QueryRow(moveCtx, `SELECT 1`).Scan(&value); queryErr != nil {
					return queryErr
				}
				if value != 1 {
					return errors.New("unexpected callback query result")
				}
				if commitErr := tx.Commit(moveCtx); commitErr != nil {
					return commitErr
				}
				return wantErr
			},
		)
	}

	if err := runApply(applyErr); !errors.Is(err, applyErr) {
		t.Fatalf("failed apply error=%v", err)
	}
	if slots := len(repository.moveConnectionSlots); slots != 0 {
		t.Fatalf("failed apply retained %d move connection slots", slots)
	}
	if phase, err := repository.InspectLifecycleRegistryPublication(ctx, ref); err != nil || phase != LifecycleRegistryPublicationSource {
		t.Fatalf("phase after failed apply=%q err=%v", phase, err)
	}
	if err := runApply(nil); err != nil {
		t.Fatal(err)
	}
	if phase, err := repository.InspectLifecycleRegistryPublication(ctx, ref); err != nil || phase != LifecycleRegistryPublicationTarget {
		t.Fatalf("phase after successful apply=%q err=%v", phase, err)
	}
	if acquired := singlePool.Stat().AcquiredConns(); acquired != 0 {
		t.Fatalf("MaxConns=1 pool retained %d acquired connections", acquired)
	}
	if slots := len(repository.moveConnectionSlots); slots != 0 {
		t.Fatalf("successful apply retained %d move connection slots", slots)
	}
}
