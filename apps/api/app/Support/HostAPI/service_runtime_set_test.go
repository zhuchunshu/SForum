package hostapi

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestServiceRegistryRuntimeSetReadersSeeOnlyWholeGraphs(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "runtime-set"}
	oldSet := serviceRuntimePair(provider, "old", "1.0.0")
	newSet := serviceRuntimePair(provider, "new", "2.0.0")
	if err := registry.ReplaceRuntimeSet(oldSet); err != nil {
		t.Fatalf("seed old runtime set: %v", err)
	}

	stop := make(chan struct{})
	errCh := make(chan error, 1)
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				services, err := registry.List("", "")
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				if len(services) != 2 {
					select {
					case errCh <- fmt.Errorf("partial service graph: %#v", services):
					default:
					}
					return
				}
				left := services[0].Winner.InstanceID
				right := services[1].Winner.InstanceID
				if !((left == "a-old" && right == "b-old") || (left == "a-new" && right == "b-new")) {
					select {
					case errCh <- fmt.Errorf("mixed service graph: %s / %s", left, right):
					default:
					}
					return
				}
			}
		}()
	}

	for index := 0; index < 200; index++ {
		desired := newSet
		if index%2 == 1 {
			desired = oldSet
		}
		if err := registry.ReplaceRuntimeSet(desired); err != nil {
			close(stop)
			readers.Wait()
			t.Fatalf("replace runtime set %d: %v", index, err)
		}
	}
	close(stop)
	readers.Wait()
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}
}

func TestServiceRegistryRuntimeSetPrepareFailureAndAbortLeaveOldGraph(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "runtime-set"}
	oldSet := serviceRuntimePair(provider, "old", "1.0.0")
	if err := registry.ReplaceRuntimeSet(oldSet); err != nil {
		t.Fatalf("seed old runtime set: %v", err)
	}
	revision := registry.Revision()

	invalid := serviceRuntimePair(provider, "invalid", "2.0.0")
	invalid[1].ExtensionVersion = "2"
	if _, err := registry.PrepareRuntimeSet(invalid); err == nil {
		t.Fatal("expected invalid complete runtime set to fail preparation")
	}
	if registry.Revision() != revision {
		t.Fatalf("failed preparation advanced revision: %d -> %d", revision, registry.Revision())
	}

	transaction, err := registry.PrepareRuntimeSet(serviceRuntimePair(provider, "new", "2.0.0"))
	if err != nil {
		t.Fatalf("prepare runtime set: %v", err)
	}
	transaction.Abort()
	if err := transaction.Commit(); err == nil {
		t.Fatal("aborted transaction committed")
	}
	assertServiceRuntimePair(t, registry, "old")
}

func TestServiceRegistryStaleRuntimeSetCommitPreservesNewerGraph(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "runtime-set"}
	if err := registry.ReplaceRuntimeSet(serviceRuntimePair(provider, "old", "1.0.0")); err != nil {
		t.Fatalf("seed old runtime set: %v", err)
	}
	stale, err := registry.PrepareRuntimeSet(serviceRuntimePair(provider, "stale", "2.0.0"))
	if err != nil {
		t.Fatalf("prepare stale runtime set: %v", err)
	}
	if err := registry.ReplaceRuntimeSet(serviceRuntimePair(provider, "winner", "3.0.0")); err != nil {
		t.Fatalf("publish winner runtime set: %v", err)
	}
	if err := stale.Commit(); !errors.Is(err, ErrServiceRuntimeSetConflict) {
		t.Fatalf("stale commit error = %v", err)
	}
	assertServiceRuntimePair(t, registry, "winner")
}

func TestServiceRegistryRuntimeSetResolvesDependenciesAgainstSameSnapshot(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "runtime-set"}
	oldSet := serviceRuntimePair(provider, "old", "1.0.0")
	oldSet[0].Dependencies = []ServiceDependency{{
		ExtensionID: "set.b", VersionConstraint: "^1.0.0", Kind: ServiceDependencyRequired,
	}}
	if err := registry.ReplaceRuntimeSet(oldSet); err != nil {
		t.Fatalf("seed old runtime set: %v", err)
	}

	newSet := serviceRuntimePair(provider, "new", "2.0.0")
	newSet[0].Dependencies = []ServiceDependency{{
		ExtensionID: "set.b", VersionConstraint: "^2.0.0", Kind: ServiceDependencyRequired,
	}}
	if err := registry.ReplaceRuntimeSet(newSet); err != nil {
		t.Fatalf("publish new runtime set: %v", err)
	}
	resolved, err := registry.ResolveExact("set.b.lookup", "1.0.0")
	if err != nil {
		t.Fatalf("resolve new provider: %v", err)
	}
	if decision := resolved.AuthorizeDependency(testServiceCaller(newSet[0])); !decision.Allowed || decision.ProviderVersion != "2.0.0" {
		t.Fatalf("new dependency decision = %#v", decision)
	}
	if decision := resolved.AuthorizeDependency(testServiceCaller(oldSet[0])); decision.Allowed || decision.Reason != "caller_stale" {
		t.Fatalf("old caller dependency decision = %#v", decision)
	}
}

func TestServiceRegistryRuntimeSetMatchesExactCompleteGraph(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "runtime-set"}
	desired := serviceRuntimePair(provider, "current", "1.0.0")
	if err := registry.ReplaceRuntimeSet(desired); err != nil {
		t.Fatalf("publish runtime set: %v", err)
	}
	revision := registry.Revision()
	if matches, err := registry.RuntimeSetMatches([]ServiceRuntimePublication{desired[1], desired[0]}); err != nil || !matches {
		t.Fatalf("reordered exact runtime set matches=%v err=%v", matches, err)
	}
	if registry.Revision() != revision {
		t.Fatalf("read-only match advanced revision: %d -> %d", revision, registry.Revision())
	}

	changed := serviceRuntimePair(provider, "changed", "1.0.0")
	if matches, err := registry.RuntimeSetMatches(changed); err != nil || matches {
		t.Fatalf("changed runtime set matches=%v err=%v", matches, err)
	}
	if !registry.UnregisterProtocolV2ServiceInstance("set.a", "a-current") {
		t.Fatal("remove current runtime owner")
	}
	if matches, err := registry.RuntimeSetMatches(desired); err != nil || matches {
		t.Fatalf("compensated runtime set matches=%v err=%v", matches, err)
	}
	if _, err := registry.RuntimeSetMatches([]ServiceRuntimePublication{desired[0], desired[0]}); err == nil {
		t.Fatal("duplicate desired runtime owner was accepted")
	}
}

func TestServiceRegistryRuntimeSetRemovesOmittedOwnersAndCanClearAll(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "runtime-set"}
	desired := serviceRuntimePair(provider, "current", "1.0.0")
	if err := registry.ReplaceRuntimeSet(desired); err != nil {
		t.Fatalf("publish runtime set: %v", err)
	}
	if err := registry.ReplaceRuntimeSet(desired[:1]); err != nil {
		t.Fatalf("remove omitted owner: %v", err)
	}
	if _, ok, err := registry.ExtensionSnapshot("set.b"); err != nil || ok {
		t.Fatalf("omitted owner snapshot ok=%v err=%v", ok, err)
	}
	if _, err := registry.ResolveExact("set.b.lookup", "1.0.0"); !errors.Is(err, ErrServiceNotFound) {
		t.Fatalf("omitted service resolve error = %v", err)
	}
	if err := registry.ReplaceRuntimeSet(nil); err != nil {
		t.Fatalf("clear runtime set: %v", err)
	}
	services, err := registry.List("", "")
	if err != nil || len(services) != 0 {
		t.Fatalf("cleared services = %#v err=%v", services, err)
	}
	if matches, err := registry.RuntimeSetMatches(nil); err != nil || !matches {
		t.Fatalf("empty runtime set matches=%v err=%v", matches, err)
	}
}

func TestServiceRegistryRuntimeSetLeaseBlocksWritersAndRejectsDrift(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "runtime-set"}
	oldSet := serviceRuntimePair(provider, "old", "1.0.0")
	newSet := serviceRuntimePair(provider, "new", "2.0.0")
	if err := registry.ReplaceRuntimeSet(oldSet); err != nil {
		t.Fatalf("seed runtime set: %v", err)
	}
	if _, err := registry.AcquireRuntimeSet(newSet); !errors.Is(err, ErrServiceRuntimeSetConflict) {
		t.Fatalf("drifted lease error = %v", err)
	}
	lease, err := registry.AcquireRuntimeSet(oldSet)
	if err != nil {
		t.Fatalf("acquire runtime set lease: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- registry.ReplaceRuntimeSet(newSet) }()
	select {
	case err := <-done:
		t.Fatalf("writer crossed runtime set lease: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	lease.Release()
	lease.Release()
	if err := <-done; err != nil {
		t.Fatalf("writer after lease release: %v", err)
	}
	assertServiceRuntimePair(t, registry, "new")
}

func TestServiceRegistryRuntimeSetCommitWaitHonorsContext(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{id: "runtime-set"}
	oldSet := serviceRuntimePair(provider, "old", "1.0.0")
	newSet := serviceRuntimePair(provider, "new", "2.0.0")
	if err := registry.ReplaceRuntimeSet(oldSet); err != nil {
		t.Fatalf("seed runtime set: %v", err)
	}
	lease, err := registry.AcquireRuntimeSet(oldSet)
	if err != nil {
		t.Fatalf("acquire blocking lease: %v", err)
	}
	defer lease.Release()

	transaction, err := registry.PrepareRuntimeSet(newSet)
	if err != nil {
		t.Fatalf("prepare queued runtime set: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := transaction.CommitContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued commit error = %v", err)
	}
	assertServiceRuntimePair(t, registry, "old")

	lease.Release()
	if err := transaction.Commit(); err != nil {
		t.Fatalf("retry canceled transaction: %v", err)
	}
	assertServiceRuntimePair(t, registry, "new")
}

func serviceRuntimePair(provider ServiceProvider, suffix, extensionVersion string) []ServiceRuntimePublication {
	aInstance := "a-" + suffix
	bInstance := "b-" + suffix
	return []ServiceRuntimePublication{
		testServiceRuntime("set.a", extensionVersion, aInstance, nil, nil, []ServiceRegistration{
			serviceRegistration("set.a", aInstance, "set.a.lookup", "1.0.0", 0, provider),
		}),
		testServiceRuntime("set.b", extensionVersion, bInstance, nil, nil, []ServiceRegistration{
			serviceRegistration("set.b", bInstance, "set.b.lookup", "1.0.0", 0, provider),
		}),
	}
}

func assertServiceRuntimePair(t *testing.T, registry *ServiceRegistry, suffix string) {
	t.Helper()
	services, err := registry.List("", "")
	if err != nil {
		t.Fatalf("list runtime set: %v", err)
	}
	if len(services) != 2 || services[0].Winner.InstanceID != "a-"+suffix || services[1].Winner.InstanceID != "b-"+suffix {
		t.Fatalf("runtime set = %#v", services)
	}
}
