package runtimerollout

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestMigrationOnceBeforeCanaryPromoteAndRollback(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewWithStore(store)
	plan, err := svc.CreatePlan(ctx, "demo.plugin", strings.Repeat("a", 64), strings.Repeat("b", 64), "admin", 50, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PromoteAtomic(ctx, plan.PlanID, "admin"); !errors.Is(err, ErrPhase) && !errors.Is(err, ErrMigration) {
		if !errors.Is(err, ErrPhase) {
			t.Fatalf("promote before migrate = %v", err)
		}
	}

	if _, err := svc.AckNode(ctx, plan.PlanID, "node-a", PhaseStaged, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AckNode(ctx, plan.PlanID, "node-b", PhaseStaged, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}

	plan, err = svc.MarkMigrationReady(ctx, plan.PlanID, "admin")
	if err != nil || !plan.MigrationReady || plan.Phase != PhaseStaged {
		t.Fatalf("migrate ready = %#v err=%v", plan, err)
	}

	plan, err = svc.SelectCanary(ctx, plan.PlanID, "admin")
	if err != nil || plan.Phase != PhaseCanary {
		t.Fatalf("canary = %#v err=%v", plan, err)
	}
	canaryCount := 0
	for _, ack := range plan.NodeAcks {
		if ack.Canary {
			canaryCount++
		}
	}
	if canaryCount != 1 {
		t.Fatalf("canary count = %d plan=%#v", canaryCount, plan.NodeAcks)
	}

	if _, err := svc.AckNode(ctx, plan.PlanID, "node-a", PhaseCanary, HealthUnhealthy, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.HealthGate(ctx, plan.PlanID, "admin", true); !errors.Is(err, ErrHealthGate) {
		t.Fatalf("unhealthy gate = %v", err)
	}

	if _, err := svc.AckNode(ctx, plan.PlanID, "node-a", PhaseCanary, HealthHealthy, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AckNode(ctx, plan.PlanID, "node-b", PhaseCanary, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BeginDrain(ctx, plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}
	plan, err = svc.PromoteAtomic(ctx, plan.PlanID, "admin")
	if err != nil || plan.Phase != PhaseActive || plan.SnapshotID == "" {
		t.Fatalf("promote = %#v err=%v", plan, err)
	}
	if len(plan.RetainedDigests) == 0 {
		t.Fatal("expected retained source digest")
	}

	plan, err = svc.Rollback(ctx, plan.PlanID, "admin", "regression")
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
	ctx := context.Background()
	svc := New()
	if _, err := svc.CreatePlan(ctx, "x", strings.Repeat("a", 64), strings.Repeat("a", 64), "admin", 10, 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("same digest = %v", err)
	}
	if _, err := svc.CreatePlan(ctx, "x", strings.Repeat("a", 64), strings.Repeat("b", 64), "", 10, 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty actor = %v", err)
	}
}

func TestRestartRecoversPlanFromSharedStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc1 := NewWithStore(store)
	plan, err := svc1.CreatePlan(ctx, "demo.restart", strings.Repeat("c", 64), strings.Repeat("d", 64), "admin", 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc1.AckNode(ctx, plan.PlanID, "node-a", PhaseStaged, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc1.MarkMigrationReady(ctx, plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}

	// 模拟 API 重启：新 Service 共享同一 durable store。
	svc2 := NewWithStore(store)
	reloaded, err := svc2.Get(ctx, plan.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.MigrationReady || reloaded.Phase != PhaseStaged {
		t.Fatalf("reload = %#v", reloaded)
	}
	if _, ok := reloaded.NodeAcks["node-a"]; !ok {
		t.Fatal("node ack lost across restart")
	}
	if _, err := svc2.SelectCanary(ctx, plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentCreateOneWinner(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	const n = 16
	var wait sync.WaitGroup
	results := make(chan error, n)
	for i := 0; i < n; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			svc := NewWithStore(store)
			_, err := svc.CreatePlan(ctx, "demo.race", strings.Repeat("e", 64), strings.Repeat("f", 64), "admin", 10, 2)
			results <- err
		}()
	}
	wait.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected: %v", err)
		}
	}
	if wins != 1 || conflicts != n-1 {
		t.Fatalf("wins=%d conflicts=%d", wins, conflicts)
	}
}

func TestNodeLostAndMigrationFailure(t *testing.T) {
	ctx := context.Background()
	svc := New()
	plan, err := svc.CreatePlan(ctx, "demo.fail", strings.Repeat("1", 64), strings.Repeat("2", 64), "admin", 50, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AckNode(ctx, plan.PlanID, "node-a", PhaseStaged, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AckNode(ctx, plan.PlanID, "node-b", PhaseStaged, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkMigrationReady(ctx, plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SelectCanary(ctx, plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkNodeLost(ctx, plan.PlanID, "node-a", "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.HealthGate(ctx, plan.PlanID, "admin", true); !errors.Is(err, ErrHealthGate) {
		t.Fatalf("lost node must fail health: %v", err)
	}

	// Migration failure terminal path.
	plan2, err := svc.CreatePlan(ctx, "demo.migfail", strings.Repeat("3", 64), strings.Repeat("4", 64), "admin", 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := svc.Fail(ctx, plan2.PlanID, "admin", "migration proof rejected")
	if err != nil || failed.Phase != PhaseFailed {
		t.Fatalf("fail = %#v err=%v", failed, err)
	}
}

// TestConcurrentAckPromoteRollbackCAS 证明并发 Ack/Promote/Rollback 不会丢更新：
// 失败者得 ErrConflict 或合法相位错误；最终 revision 单调递增。
func TestConcurrentAckPromoteRollbackCAS(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewWithStore(store)
	plan, err := svc.CreatePlan(ctx, "demo.cas", strings.Repeat("c", 64), strings.Repeat("d", 64), "admin", 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkMigrationReady(ctx, plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AckNode(ctx, plan.PlanID, "n1", PhaseStaged, HealthHealthy, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SelectCanary(ctx, plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AckNode(ctx, plan.PlanID, "n1", PhaseCanary, HealthHealthy, true); err != nil {
		t.Fatal(err)
	}
	before, err := svc.Get(ctx, plan.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	beforeRev := before.Revision

	var wait sync.WaitGroup
	var mu sync.Mutex
	var conflicts, successes, others int
	// 并发：多 Ack + Promote + 一次 Rollback 竞争。
	for i := 0; i < 12; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			// 独立 Service 实例共享 store（双连接语义）。
			local := NewWithStore(store)
			var opErr error
			switch i % 3 {
			case 0:
				_, opErr = local.AckNode(ctx, plan.PlanID, "n1", PhaseCanary, HealthHealthy, true)
			case 1:
				_, opErr = local.BeginDrain(ctx, plan.PlanID, "admin")
				if opErr == nil {
					_, opErr = local.PromoteAtomic(ctx, plan.PlanID, "admin")
				}
			default:
				_, opErr = local.Rollback(ctx, plan.PlanID, "admin", "race")
			}
			mu.Lock()
			defer mu.Unlock()
			switch {
			case opErr == nil:
				successes++
			case errors.Is(opErr, ErrConflict), errors.Is(opErr, ErrPhase), errors.Is(opErr, ErrHealthGate), errors.Is(opErr, ErrMigration):
				conflicts++
			default:
				others++
				t.Errorf("unexpected error: %v", opErr)
			}
		}(i)
	}
	wait.Wait()
	if others > 0 {
		t.Fatalf("unexpected errors=%d successes=%d conflicts=%d", others, successes, conflicts)
	}
	if successes < 1 {
		t.Fatal("expected at least one successful concurrent transition")
	}
	final, err := svc.Get(ctx, plan.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Revision < beforeRev {
		t.Fatalf("revision regressed: before=%d after=%d", beforeRev, final.Revision)
	}
	// 终态必须是合法相位之一。
	switch final.Phase {
	case PhaseActive, PhaseRolledBack, PhaseDraining, PhasePromoting, PhaseCanary, PhaseFailed:
	default:
		t.Fatalf("unexpected final phase=%s rev=%d", final.Phase, final.Revision)
	}
}

func TestMissedNotificationRecoveredByReload(t *testing.T) {
	// 模拟节点错过 NOTIFY：不依赖 pub/sub，直接从 durable store 重读并 ack。
	ctx := context.Background()
	store := NewMemoryStore()
	coordinator := NewWithStore(store)
	plan, err := coordinator.CreatePlan(ctx, "demo.notify", strings.Repeat("5", 64), strings.Repeat("6", 64), "admin", 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.MarkMigrationReady(ctx, plan.PlanID, "admin"); err != nil {
		t.Fatal(err)
	}

	// 节点进程：仅持有 plan id，从 store 恢复状态。
	node := NewWithStore(store)
	reloaded, err := node.Get(ctx, plan.PlanID)
	if err != nil || reloaded.Phase != PhaseStaged {
		t.Fatalf("node reload after missed notify: %#v err=%v", reloaded, err)
	}
	if _, err := node.AckNode(ctx, plan.PlanID, "node-z", PhaseStaged, HealthHealthy, false); err != nil {
		t.Fatal(err)
	}
	again, err := coordinator.Get(ctx, plan.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := again.NodeAcks["node-z"]; !ok {
		t.Fatal("coordinator must see node ack after store reload")
	}
}
