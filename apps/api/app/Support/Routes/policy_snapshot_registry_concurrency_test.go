package routes

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRoutePolicyRegistryPublishAtomicallyTo64Readers(t *testing.T) {
	registry := NewRegistry()
	first := policySnapshotPublication("1.0.0", 'a', "policy.atomic.route@1", RouteExecutionPolicy{
		RateLimit: routePolicyRateLimitIPWrite, Idempotency: routePolicyDisabled,
	})
	second := policySnapshotPublication("2.0.0", 'b', "policy.atomic.route@2", RouteExecutionPolicy{
		RateLimit:   routePolicyRateLimitIPWrite,
		Idempotency: routePolicyIdempotencyRequired24h, IdempotencyRequired: true,
	})
	if _, err := registry.Publish(first); err != nil {
		t.Fatal(err)
	}

	const readers = 64
	start := make(chan struct{})
	errorsSeen := make(chan error, 1)
	var workers sync.WaitGroup
	var failed atomic.Bool
	var firstSeen atomic.Int64
	var secondSeen atomic.Int64
	report := func(err error) {
		failed.Store(true)
		select {
		case errorsSeen <- err:
		default:
		}
	}
	for reader := 0; reader < readers; reader++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for attempt := 0; attempt < 500 && !failed.Load(); attempt++ {
				plan, err := registry.BuildExecutionPlan("POST", "/policy-atomic")
				if err != nil {
					report(err)
					return
				}
				policy, bound := plan.ExecutionPolicy()
				if !bound {
					report(fmt.Errorf("revision %d has no bound policy", plan.Revision()))
					return
				}
				switch plan.Terminal().ContractVersion {
				case "policy.atomic.route@1":
					firstSeen.Add(1)
					if policy != first.Policies[0].Policy {
						report(fmt.Errorf("v1 plan observed policy %#v", policy))
						return
					}
				case "policy.atomic.route@2":
					secondSeen.Add(1)
					if policy != second.Policies[0].Policy {
						report(fmt.Errorf("v2 plan observed policy %#v", policy))
						return
					}
				default:
					report(fmt.Errorf("unknown terminal %#v", plan.Terminal()))
					return
				}
				if attempt%16 == 0 {
					runtime.Gosched()
				}
			}
		}()
	}
	close(start)
	for revision := 0; revision < 500 && !failed.Load(); revision++ {
		publication := first
		if revision%2 == 0 {
			publication = second
		}
		if _, err := registry.Publish(publication); err != nil {
			report(err)
			break
		}
		runtime.Gosched()
	}
	workers.Wait()
	select {
	case err := <-errorsSeen:
		t.Fatal(err)
	default:
	}
	if firstSeen.Load() == 0 || secondSeen.Load() == 0 {
		t.Fatalf("readers missed a revision: first=%d second=%d", firstSeen.Load(), secondSeen.Load())
	}
}
