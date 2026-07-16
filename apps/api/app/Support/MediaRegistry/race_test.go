package mediaregistry

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestConcurrentPublicationPlanningSnapshotsAndTraces(t *testing.T) {
	registry := registryWithMediaForTest()
	plugin := pluginPublicationForTest()
	traces := NewTraceRing(64)
	var wait sync.WaitGroup
	errorsFound := make(chan error, 16)
	wait.Add(1)
	go func() {
		defer wait.Done()
		for index := 0; index < 300; index++ {
			if _, removed, err := registry.Remove(plugin.Artifact); err != nil {
				errorsFound <- err
				return
			} else if !removed {
				errorsFound <- fmt.Errorf("iteration %d did not remove plugin", index)
				return
			}
			if _, err := registry.Publish(plugin); err != nil {
				errorsFound <- err
				return
			}
		}
	}()
	for worker := 0; worker < 6; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for index := 0; index < 400; index++ {
				snapshot := registry.Snapshot()
				if snapshot.SchemaVersion != SchemaVersion || snapshot.Digest == "" {
					errorsFound <- fmt.Errorf("worker %d invalid snapshot", worker)
					return
				}
				if len(snapshot.Publications) > 0 && len(snapshot.Publications[0].Policies) > 0 {
					snapshot.Publications[0].Policies[0].AllowedMIMEs = []string{"text/html"}
				}
				plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
				if err != nil && !errors.Is(err, ErrPlanStale) {
					errorsFound <- err
					return
				}
				if err == nil && plan.Source != plan.OriginalFallback {
					errorsFound <- fmt.Errorf("worker %d observed changed fallback", worker)
					return
				}
				if err == nil && plan.Digest != computePlanDigest(plan) {
					errorsFound <- fmt.Errorf("worker %d observed digest %s want %s at revision %d", worker, plan.Digest, computePlanDigest(plan), plan.Revision)
					return
				}
				traces.AppendMediaTrace(TraceEvent{Revision: max(registry.Revision(), 1), OperationKey: hexSequence(worker, index), PlanKind: PlanUpload, Stage: StageScan, StepID: "scan:race", Outcome: TraceSucceeded, Duration: time.Microsecond, Artifact: plugin.Artifact})
			}
		}(worker)
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
	if len(traces.MediaTraces(0)) != 64 {
		t.Fatalf("trace ring size=%d", len(traces.MediaTraces(0)))
	}
	if fresh := registry.Snapshot(); len(fresh.Publications) > 0 && len(fresh.Publications[0].Policies) > 0 && fresh.Publications[0].Policies[0].AllowedMIMEs[0] == "text/html" {
		t.Fatal("concurrent snapshot mutation escaped")
	}
}

func hexSequence(worker, index int) string { return fmt.Sprintf("%064x", worker*1000+index+1) }
