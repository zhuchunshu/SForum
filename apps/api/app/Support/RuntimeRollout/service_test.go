package runtimerollout

import (
	"errors"
	"strings"
	"testing"
)

func TestMigrationOnceBeforeCanaryPromoteAndRollback(t *testing.T) {
	svc := New()
	plan, err := svc.CreatePlan("demo.plugin", strings.Repeat("a", 64), strings.Repeat("b", 64), "admin", 50, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Cannot promote without migration.
	if _, err := svc.PromoteAtomic(plan.PlanID, "admin"); !errors.Is(err, ErrPhase) && !errors.Is(err, ErrMigration) {
		// Phase pending → ErrPhase
		if !errors.Is(err, ErrPhase) {
			t.Fatalf("promote before migrate = %v", err)
		}
	}

	// Register nodes before canary.
	if _, err := svc.AckNode(plan.PlanID, "node-a", PhaseStaged, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AckNode(plan.PlanID, "node-b", PhaseStaged, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}

	plan, err = svc.MarkMigrationReady(plan.PlanID, "admin")
	if err != nil || !plan.MigrationReady || plan.Phase != PhaseStaged {
		t.Fatalf("migrate ready = %#v err=%v", plan, err)
	}

	plan, err = svc.SelectCanary(plan.PlanID, "admin")
	if err != nil || plan.Phase != PhaseCanary {
		t.Fatalf("canary = %#v err=%v", plan, err)
	}
	canaryCount := 0
	for _, ack := range plan.NodeAcks {
		if ack.Canary {
			canaryCount++
		}
	}
	if canaryCount != 1 { // 50% of 2 nodes
		t.Fatalf("canary count = %d plan=%#v", canaryCount, plan.NodeAcks)
	}

	// Unhealthy canary fails health gate.
	if _, err := svc.AckNode(plan.PlanID, "node-a", PhaseCanary, HealthUnhealthy, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.HealthGate(plan.PlanID, "admin", true); !errors.Is(err, ErrHealthGate) {
		t.Fatalf("unhealthy gate = %v", err)
	}

	// Restore health, drain, promote.
	if _, err := svc.AckNode(plan.PlanID, "node-a", PhaseCanary, HealthHealthy, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AckNode(plan.PlanID, "node-b", PhaseCanary, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BeginDrain(plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}
	plan, err = svc.PromoteAtomic(plan.PlanID, "admin")
	if err != nil || plan.Phase != PhaseActive || plan.SnapshotID == "" {
		t.Fatalf("promote = %#v err=%v", plan, err)
	}
	if len(plan.RetainedDigests) == 0 {
		t.Fatal("expected retained source digest")
	}

	plan, err = svc.Rollback(plan.PlanID, "admin", "regression")
	if err != nil || plan.Phase != PhaseRolledBack {
		t.Fatalf("rollback = %#v err=%v", plan, err)
	}
	foundTarget := false
	for _, d := range plan.RetainedDigests {
		if d == plan.TargetDigest {
			foundTarget = true
		}
	}
	if !foundTarget {
		t.Fatalf("target not retained after rollback: %#v", plan.RetainedDigests)
	}
}

func TestCreatePlanRejectsSameDigestAndEmptyActor(t *testing.T) {
	svc := New()
	if _, err := svc.CreatePlan("x", strings.Repeat("a", 64), strings.Repeat("a", 64), "admin", 10, 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("same digest = %v", err)
	}
	if _, err := svc.CreatePlan("x", strings.Repeat("a", 64), strings.Repeat("b", 64), "", 10, 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty actor = %v", err)
	}
}
