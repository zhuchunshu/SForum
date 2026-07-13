package hostapi

import (
	"errors"
	"sync"
	"testing"
)

func TestServiceRegistryResolveExactIncludesBuildMetadata(t *testing.T) {
	registry := NewServiceRegistry()
	alpha := &registryTestProvider{id: "alpha"}
	beta := &registryTestProvider{id: "beta"}
	if err := registry.ReplaceExtension("demo.plugin", []ServiceRegistration{
		serviceRegistration("demo.plugin", "runtime-1", "demo.lookup", "1.2.3+alpha", 0, alpha),
		serviceRegistration("demo.plugin", "runtime-1", "demo.lookup", "1.2.3+beta", 0, beta),
	}); err != nil {
		t.Fatal(err)
	}

	resolved, err := registry.ResolveExact("demo.lookup", "1.2.3+beta")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Winner.Provider != beta || resolved.Winner.Descriptor.GetVersion() != "1.2.3+beta" {
		t.Fatalf("exact build resolution = %#v", resolved.Winner)
	}
	if _, err := registry.ResolveExact("demo.lookup", "1.2.3"); !errors.Is(err, ErrServiceNotFound) {
		t.Fatalf("missing exact build error = %v", err)
	}
	if _, err := registry.ResolveExact("demo.lookup", "1.2"); !errors.Is(err, ErrInvalidServiceConstraint) {
		t.Fatalf("non-strict exact version error = %v", err)
	}
}

func TestServiceRegistryCompositionActionsFailClosedUntilChainExists(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{}
	baseline := serviceRegistration("demo.plugin", "runtime-1", "demo.lookup", "1.0.0", 0, provider)
	if err := registry.ReplaceExtension("demo.plugin", []ServiceRegistration{baseline}); err != nil {
		t.Fatal(err)
	}

	for _, action := range []string{ServiceActionBefore, ServiceActionAfter, ServiceActionWrap} {
		t.Run(action, func(t *testing.T) {
			contribution := serviceRegistration("demo.plugin", "runtime-2", "demo."+action, "1.0.0", 0, provider)
			contribution.Action = action
			contribution.TargetID = "demo.lookup"
			if err := registry.ReplaceExtension("demo.plugin", []ServiceRegistration{contribution}); !errors.Is(err, ErrInvalidServiceRegistration) {
				t.Fatalf("unsupported composition error = %v", err)
			}
			if registry.Revision() != 1 {
				t.Fatalf("failed composition advanced revision to %d", registry.Revision())
			}
			resolved, err := registry.ResolveExact("demo.lookup", "1.0.0")
			if err != nil || resolved.Winner.InstanceID != "runtime-1" {
				t.Fatalf("failed composition changed snapshot: %#v err=%v", resolved, err)
			}
		})
	}
}

func TestServiceRegistryInstanceBoundUnregisterCannotRemoveReplacement(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{}
	if err := registry.ReplaceExtension("demo.plugin", []ServiceRegistration{
		serviceRegistration("demo.plugin", "runtime-new", "demo.lookup", "1.0.0", 0, provider),
	}); err != nil {
		t.Fatal(err)
	}
	if registry.UnregisterProtocolV2ServiceInstance("demo.plugin", "runtime-old") {
		t.Fatal("stale runtime removed replacement registrations")
	}
	if registry.Revision() != 1 {
		t.Fatalf("stale unregister advanced revision to %d", registry.Revision())
	}
	if resolved, err := registry.ResolveExact("demo.lookup", "1.0.0"); err != nil || resolved.Winner.InstanceID != "runtime-new" {
		t.Fatalf("replacement disappeared: %#v err=%v", resolved, err)
	}
	if !registry.UnregisterProtocolV2ServiceInstance("demo.plugin", "runtime-new") {
		t.Fatal("current runtime was not unregistered")
	}
	if registry.Revision() != 2 {
		t.Fatalf("current unregister revision = %d", registry.Revision())
	}
}

func TestServiceRegistryReplacementWinsConcurrentStaleUnregister(t *testing.T) {
	registry := NewServiceRegistry()
	provider := &registryTestProvider{}
	for iteration := 0; iteration < 100; iteration++ {
		if err := registry.ReplaceExtension("demo.plugin", []ServiceRegistration{
			serviceRegistration("demo.plugin", "runtime-old", "demo.lookup", "1.0.0", 0, provider),
		}); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		var group sync.WaitGroup
		group.Add(2)
		go func() {
			defer group.Done()
			<-start
			if err := registry.ReplaceExtension("demo.plugin", []ServiceRegistration{
				serviceRegistration("demo.plugin", "runtime-new", "demo.lookup", "1.0.0", 0, provider),
			}); err != nil {
				t.Errorf("replace services: %v", err)
			}
		}()
		go func() {
			defer group.Done()
			<-start
			registry.UnregisterProtocolV2ServiceInstance("demo.plugin", "runtime-old")
		}()
		close(start)
		group.Wait()
		resolved, err := registry.ResolveExact("demo.lookup", "1.0.0")
		if err != nil || resolved.Winner.InstanceID != "runtime-new" {
			t.Fatalf("iteration %d replacement lost: %#v err=%v", iteration, resolved, err)
		}
	}
}
