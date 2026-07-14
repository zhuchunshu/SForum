package extensionsruntime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestManagerVersionedProviderFallsBackWithIsolatedInputAndHostRevalidation(t *testing.T) {
	starter := newProviderInvocationStarter()
	starter.invoke = func(_ context.Context, extension extensions.Extension, request VersionedProviderRequest) (VersionedProviderResponse, error) {
		starter.record(extension.ID)
		nested := request.Input["nested"].(map[string]any)
		if extension.ID == "providers.consumer" {
			nested["value"] = "mutated-by-failed-provider"
			return VersionedProviderResponse{}, errors.New("consumer crashed")
		}
		if nested["value"] != "safe" {
			return VersionedProviderResponse{}, errors.New("failed provider polluted fallback input")
		}
		return VersionedProviderResponse{Output: map[string]any{"status": "delivered"}}, nil
	}
	manager := NewManager(ManagerConfig{Starter: starter})
	owner, consumer := startProviderOwnerAndConsumer(t, manager, "next")
	callerInput := map[string]any{"nested": map[string]any{"value": "safe"}}
	validations := []string{}
	result, err := manager.InvokeVersionedProvider(context.Background(), VersionedProviderInvocation{
		Caller: providerCaller(t, manager, consumer), SlotID: providerSlotID,
		ContractVersion: providerContractVersion, Operation: VersionedProviderOperationInvoke,
		InputSchema: providerRequestSchema, Input: callerInput,
		Revalidate: func(_ context.Context, schema string, document map[string]any) error {
			validations = append(validations, schema)
			if schema == providerRequestSchema && document["nested"].(map[string]any)["value"] != "safe" {
				return errors.New("request input was mutated")
			}
			return nil
		},
	})
	if err != nil || result.ExtensionID != owner.ID || result.ProviderID != providerSlotID || result.Attempts != 2 || result.Output["status"] != "delivered" {
		t.Fatalf("fallback result = %#v, %v", result, err)
	}
	if got := starter.calls(); !reflect.DeepEqual(got, []string{consumer.ID, owner.ID}) {
		t.Fatalf("provider calls = %#v", got)
	}
	if !reflect.DeepEqual(validations, []string{providerRequestSchema, providerRequestSchema, providerResponseSchema}) {
		t.Fatalf("Host revalidations = %#v", validations)
	}
	if callerInput["nested"].(map[string]any)["value"] != "safe" {
		t.Fatalf("caller input was mutated = %#v", callerInput)
	}
}

func TestManagerVersionedProviderClosedAndInvalidOutputStopOrFallback(t *testing.T) {
	t.Run("closed stops after first failure", func(t *testing.T) {
		starter := newProviderInvocationStarter()
		starter.invoke = func(_ context.Context, extension extensions.Extension, _ VersionedProviderRequest) (VersionedProviderResponse, error) {
			starter.record(extension.ID)
			return VersionedProviderResponse{}, errors.New("provider failed")
		}
		manager := NewManager(ManagerConfig{Starter: starter})
		_, consumer := startProviderOwnerAndConsumer(t, manager, "closed")
		result, err := manager.InvokeVersionedProvider(context.Background(), VersionedProviderInvocation{
			Caller: providerCaller(t, manager, consumer), SlotID: providerSlotID,
			ContractVersion: providerContractVersion, Operation: VersionedProviderOperationInvoke,
			InputSchema: providerRequestSchema, Input: map[string]any{},
			Revalidate: func(context.Context, string, map[string]any) error { return nil },
		})
		if !errors.Is(err, ErrProviderSlotNoProvider) || result.Attempts != 1 || !reflect.DeepEqual(starter.calls(), []string{consumer.ID}) {
			t.Fatalf("closed fallback = %#v, calls=%#v, err=%v", result, starter.calls(), err)
		}
	})

	t.Run("next rejects invalid output and tries the next candidate", func(t *testing.T) {
		starter := newProviderInvocationStarter()
		starter.invoke = func(_ context.Context, extension extensions.Extension, _ VersionedProviderRequest) (VersionedProviderResponse, error) {
			starter.record(extension.ID)
			if extension.ID == "providers.consumer" {
				return VersionedProviderResponse{Output: map[string]any{"status": "invalid"}}, nil
			}
			return VersionedProviderResponse{Output: map[string]any{"status": "valid"}}, nil
		}
		manager := NewManager(ManagerConfig{Starter: starter})
		owner, consumer := startProviderOwnerAndConsumer(t, manager, "next")
		result, err := manager.InvokeVersionedProvider(context.Background(), VersionedProviderInvocation{
			Caller: providerCaller(t, manager, consumer), SlotID: providerSlotID,
			ContractVersion: providerContractVersion, Operation: VersionedProviderOperationInvoke,
			InputSchema: providerRequestSchema, Input: map[string]any{},
			Revalidate: func(_ context.Context, schema string, document map[string]any) error {
				if schema == providerResponseSchema && document["status"] == "invalid" {
					return errors.New("output schema rejected")
				}
				return nil
			},
		})
		if err != nil || result.ExtensionID != owner.ID || result.Attempts != 2 || result.Output["status"] != "valid" {
			t.Fatalf("output fallback = %#v, %v", result, err)
		}
	})
}

func TestManagerVersionedProviderEnforcesTimeoutWhenInvokerIgnoresContext(t *testing.T) {
	starter := newProviderInvocationStarter()
	starter.invoke = func(context.Context, extensions.Extension, VersionedProviderRequest) (VersionedProviderResponse, error) {
		time.Sleep(150 * time.Millisecond)
		return VersionedProviderResponse{Output: map[string]any{"status": "late"}}, nil
	}
	manager := NewManager(ManagerConfig{Starter: starter})
	owner := versionedProviderExtension("providers.owner", strings.Repeat("a", 64), providerSlotDefinition(10))
	owner.Manifest.Providers[0].Fallback = "closed"
	owner.Manifest.Providers[0].TimeoutMS = 20
	if err := manager.Start(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	result, err := manager.InvokeVersionedProvider(context.Background(), VersionedProviderInvocation{
		SlotID: providerSlotID, ContractVersion: providerContractVersion, Operation: VersionedProviderOperationInvoke,
		InputSchema: providerRequestSchema, Input: map[string]any{},
		Revalidate: func(context.Context, string, map[string]any) error { return nil },
	})
	if elapsed := time.Since(started); elapsed >= 100*time.Millisecond {
		t.Fatalf("Host deadline was not enforced: %v", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) || result.Attempts != 1 {
		t.Fatalf("timeout result = %#v, %v", result, err)
	}
}

func TestManagerVersionedProviderRequiresHostRevalidatorAndExactInstance(t *testing.T) {
	manager := NewManager(ManagerConfig{Starter: newProviderInvocationStarter()})
	owner := versionedProviderExtension("providers.owner", strings.Repeat("a", 64), providerSlotDefinition(10))
	if err := manager.Start(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.InvokeVersionedProvider(context.Background(), VersionedProviderInvocation{
		SlotID: providerSlotID, ContractVersion: providerContractVersion, Operation: VersionedProviderOperationInvoke,
		InputSchema: providerRequestSchema,
	}); !errors.Is(err, ErrProviderSlotInvalid) {
		t.Fatalf("missing Host revalidator = %v", err)
	}
	if _, err := manager.InvokeVersionedProvider(context.Background(), VersionedProviderInvocation{
		SlotID: providerSlotID, ContractVersion: providerContractVersion, Operation: "undeclared",
		InputSchema: providerRequestSchema,
		Revalidate:  func(context.Context, string, map[string]any) error { return nil },
	}); !errors.Is(err, ErrProviderSlotInvalid) {
		t.Fatalf("undeclared operation = %v", err)
	}

	emptyInstanceManager := NewManager(ManagerConfig{Starter: fakeStarter{}})
	if err := emptyInstanceManager.Start(context.Background(), owner); !errors.Is(err, ErrVersionedRuntimeContractInvalid) {
		t.Fatalf("provider-only empty instance = %v", err)
	}
	if snapshot := emptyInstanceManager.HookBus().ProviderSlots().Snapshot(); len(snapshot.Contracts) != 0 {
		t.Fatalf("invalid runtime published providers = %#v", snapshot)
	}
}

type providerInvocationStarter struct {
	*managerStagedStarter
	mu     sync.Mutex
	called []string
	invoke func(context.Context, extensions.Extension, VersionedProviderRequest) (VersionedProviderResponse, error)
}

func newProviderInvocationStarter() *providerInvocationStarter {
	return &providerInvocationStarter{managerStagedStarter: newManagerStagedStarter()}
}

func (s *providerInvocationStarter) InvokeVersionedProvider(ctx context.Context, extension extensions.Extension, request VersionedProviderRequest) (VersionedProviderResponse, error) {
	if s.invoke == nil {
		return VersionedProviderResponse{}, errors.New("provider invoker is not configured")
	}
	return s.invoke(ctx, extension, request)
}

func (s *providerInvocationStarter) record(extensionID string) {
	s.mu.Lock()
	s.called = append(s.called, extensionID)
	s.mu.Unlock()
}

func (s *providerInvocationStarter) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.called...)
}

func startProviderOwnerAndConsumer(t *testing.T, manager *Manager, fallback string) (extensions.Extension, extensions.Extension) {
	t.Helper()
	owner := versionedProviderExtension("providers.owner", strings.Repeat("a", 64), providerSlotDefinition(10))
	owner.Manifest.Providers[0].Fallback = fallback
	consumer := versionedProviderExtension("providers.consumer", strings.Repeat("b", 64), providerSlotConsumer("providers.consumer.delivery", 50))
	consumer.Manifest.Providers[0].Fallback = fallback
	consumer.Manifest.Dependencies = []extensions.ManifestDependency{{ID: owner.ID, Version: "^1.0.0", Kind: "required"}}
	if err := manager.Start(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background(), consumer); err != nil {
		t.Fatal(err)
	}
	return owner, consumer
}

func providerCaller(t *testing.T, manager *Manager, extension extensions.Extension) ProviderSlotCaller {
	t.Helper()
	active, err := manager.ActiveRuntimeInstance(extension.ID)
	if err != nil {
		t.Fatal(err)
	}
	return ProviderSlotCaller{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version, ArtifactDigest: extension.PackageDigest,
		RuntimeInstanceID: active.Identity.InstanceID, Attested: true,
	}
}
