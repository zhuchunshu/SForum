package routes

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestBindRouteExecutionPoliciesFreezesExactMultiMethodPlans(t *testing.T) {
	artifact := routeArtifact("policy.freeze", "1.0.0", 'a')
	declaration := pluginRoute("policy.freeze.route", "/policy-freeze", 0, "GET", "POST")
	want := map[string]RouteExecutionPolicy{
		"GET":  {RateLimit: "disabled", Idempotency: "disabled"},
		"POST": {RateLimit: "host.ip_write@1", Idempotency: "required.24h@1", IdempotencyRequired: true},
	}
	resolver := &policySnapshotResolver{artifact: artifact, routeID: declaration.ID, policies: want}
	bound, err := BindRouteExecutionPolicies(Publication{Plugins: []PluginRouteSet{{
		Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration},
	}}}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if len(bound.Policies) != 2 || resolver.calls != 2 {
		t.Fatalf("bindings=%#v resolver calls=%d", bound.Policies, resolver.calls)
	}

	registry := NewRegistry()
	if _, err := registry.Publish(bound); err != nil {
		t.Fatal(err)
	}
	bound.Policies[0].Policy.Idempotency = "mutated-after-publication"
	for method, expected := range want {
		plan, err := registry.BuildExecutionPlan(method, declaration.Path)
		if err != nil {
			t.Fatalf("build %s plan: %v", method, err)
		}
		policy, ok := plan.ExecutionPolicy()
		if !ok || policy != expected {
			t.Fatalf("%s policy=%#v bound=%t want=%#v", method, policy, ok, expected)
		}
	}

	public := registry.Snapshot()
	publication := registry.PublicationSnapshot()
	public.Policies[0].Policy.RateLimit = "mutated-public-view"
	publication.Publication.Policies[0].Policy.RateLimit = "mutated-publication-view"
	if current := registry.Snapshot().Policies[0].Policy.RateLimit; current == "mutated-public-view" {
		t.Fatal("public snapshot mutation reached immutable policy state")
	}
	if current := registry.PublicationSnapshot().Publication.Policies[0].Policy.RateLimit; current == "mutated-publication-view" {
		t.Fatal("publication snapshot mutation reached immutable policy state")
	}
}

func TestBindRouteExecutionPoliciesSynthesizesDisabledTerminalPolicy(t *testing.T) {
	artifact := routeArtifact("policy.disabled", "1.0.0", 'd')
	declaration := pluginRoute("policy.disabled.route", "/policy-disabled", 0, "GET")
	bound, err := BindRouteExecutionPolicies(Publication{Plugins: []PluginRouteSet{{
		Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration},
	}}}, missingPolicySnapshotResolver{})
	if err != nil {
		t.Fatal(err)
	}
	if bound.Policies == nil || len(bound.Policies) != 1 ||
		bound.Policies[0].Policy != (RouteExecutionPolicy{RateLimit: "disabled", Idempotency: "disabled"}) {
		t.Fatalf("bound policies=%#v", bound.Policies)
	}
	registry := NewRegistry()
	if _, err := registry.Publish(bound); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.BuildExecutionPlan("GET", declaration.Path)
	if err != nil {
		t.Fatal(err)
	}
	policy, ok := plan.ExecutionPolicy()
	if !ok || policy != bound.Policies[0].Policy {
		t.Fatalf("policy=%#v bound=%t", policy, ok)
	}
}

func TestBindRouteExecutionPoliciesNeedsNoResolverWithoutPluginTerminals(t *testing.T) {
	bound, err := BindRouteExecutionPolicies(Publication{
		Core: []CoreRoute{coreRoute("core.route.policy.host", "GET", "/policy-host")},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bound.Policies == nil || len(bound.Policies) != 0 {
		t.Fatalf("host-only policies=%#v", bound.Policies)
	}

	safe, err := BindRouteExecutionPolicies(Publication{
		SafeMode: true,
		Plugins: []PluginRouteSet{{
			Artifact: PluginArtifact{ExtensionID: "invalid"},
			Routes:   []extensionmanifest.ManifestRoute{{Action: "invalid"}},
		}},
	}, nil)
	if err != nil || safe.Policies != nil {
		t.Fatalf("safe-mode bound=%#v error=%v", safe, err)
	}
}

func TestRegistryRejectsUnboundDuplicateAndMalformedPolicies(t *testing.T) {
	base := policySnapshotPublication("1.0.0", 'a', "policy.invalid.route@1", RouteExecutionPolicy{
		RateLimit: "host.ip_write@1", Idempotency: "required.24h@1", IdempotencyRequired: true,
	})
	if _, err := NewRegistry().Publish(base); err != nil {
		t.Fatalf("valid baseline rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*Publication)
	}{
		{name: "unknown route", mutate: func(value *Publication) { value.Policies[0].RouteID = "policy.invalid.unknown" }},
		{name: "runtime drift", mutate: func(value *Publication) { value.Policies[0].Artifact.RuntimeInstanceID = "runtime-drift" }},
		{name: "duplicate", mutate: func(value *Publication) { value.Policies = append(value.Policies, value.Policies[0]) }},
		{name: "required contract missing", mutate: func(value *Publication) { value.Policies[0].Policy.Idempotency = "" }},
		{name: "policy whitespace", mutate: func(value *Publication) { value.Policies[0].Policy.RateLimit = " host.ip_write@1" }},
		{name: "unknown rate policy", mutate: func(value *Publication) { value.Policies[0].Policy.RateLimit = "host.custom@1" }},
		{name: "required flag drift", mutate: func(value *Publication) { value.Policies[0].Policy.IdempotencyRequired = false }},
		{name: "authoritative set missing", mutate: func(value *Publication) { value.Policies = []RoutePolicyBinding{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := clonePublication(base)
			test.mutate(&value)
			if _, err := NewRegistry().Publish(value); !errors.Is(err, ErrInvalidRoute) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestRegistryPreservesNilAndEmptyPolicyPublicationMeaning(t *testing.T) {
	legacy := policySnapshotPublication("1.0.0", 'a', "policy.atomic.route@1", RouteExecutionPolicy{})
	legacy.Policies = nil
	registry := NewRegistry()
	if _, err := registry.Publish(legacy); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.BuildExecutionPlan("POST", "/policy-atomic")
	if err != nil {
		t.Fatal(err)
	}
	if _, bound := plan.ExecutionPolicy(); bound {
		t.Fatal("legacy nil policy publication became authoritative")
	}
	if registry.PublicationSnapshot().Publication.Policies != nil {
		t.Fatal("legacy nil policy publication was not preserved")
	}

	authoritative := clonePublication(legacy)
	authoritative.Policies = []RoutePolicyBinding{}
	if authoritative.Policies == nil {
		t.Fatal("empty authoritative policy set became nil while cloning")
	}
	if _, err := NewRegistry().Publish(authoritative); !errors.Is(err, ErrInvalidRoute) {
		t.Fatalf("incomplete authoritative publication error=%v", err)
	}
	emptyRegistry := NewRegistry()
	emptySnapshot, err := emptyRegistry.Publish(Publication{Policies: []RoutePolicyBinding{}})
	if err != nil {
		t.Fatal(err)
	}
	if emptySnapshot.Policies == nil || emptyRegistry.PublicationSnapshot().Publication.Policies == nil {
		t.Fatal("valid empty authoritative policy set became nil")
	}
}

func TestRegistrySafeModeDropsPluginPoliciesBeforeValidation(t *testing.T) {
	snapshot, err := NewRegistry().Publish(Publication{
		SafeMode: true,
		Core:     []CoreRoute{coreRoute("core.route.policy.safe", "GET", "/policy-safe")},
		Plugins: []PluginRouteSet{{
			Artifact: PluginArtifact{ExtensionID: "invalid"},
			Routes:   []extensionmanifest.ManifestRoute{{Action: "invalid"}},
		}},
		Policies: []RoutePolicyBinding{{RouteID: "invalid", Policy: RouteExecutionPolicy{
			IdempotencyRequired: true,
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.SafeMode || len(snapshot.Routes) != 1 || len(snapshot.Policies) != 0 {
		t.Fatalf("snapshot=%#v", snapshot)
	}
}

func TestRegistryWildcardTerminalAllowsOnlyBoundDisabledPolicy(t *testing.T) {
	artifact := routeArtifact("policy.wildcard", "1.0.0", 'e')
	declaration := pluginRoute("policy.wildcard.route", "/policy-wildcard", 0, "*")
	publication := Publication{
		Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration}}},
		Policies: []RoutePolicyBinding{{
			Artifact: artifact, RouteID: declaration.ID, ContractVersion: declaration.ContractVersion,
			Method: "*", Policy: RouteExecutionPolicy{RateLimit: "disabled", Idempotency: "disabled"},
		}},
	}
	registry := NewRegistry()
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "POST"} {
		plan, err := registry.BuildExecutionPlan(method, declaration.Path)
		if err != nil {
			t.Fatalf("build %s wildcard plan: %v", method, err)
		}
		policy, ok := plan.ExecutionPolicy()
		if !ok || policy != publication.Policies[0].Policy {
			t.Fatalf("%s policy=%#v bound=%t", method, policy, ok)
		}
	}

	publication.Policies[0].Policy = RouteExecutionPolicy{
		RateLimit: "host.ip_write@1", Idempotency: "required.24h@1", IdempotencyRequired: true,
	}
	if _, err := NewRegistry().Publish(publication); !errors.Is(err, ErrInvalidRoute) {
		t.Fatalf("required wildcard policy error=%v", err)
	}
}

func TestRegistryRejectsRequiredReplayCredentialMutatingContributions(t *testing.T) {
	for _, action := range []string{
		extensionmanifest.RouteActionBefore,
		extensionmanifest.RouteActionFilter,
		extensionmanifest.RouteActionWrap,
		extensionmanifest.RouteActionGlobalMiddleware,
	} {
		t.Run(action, func(t *testing.T) {
			publication := requiredReplayCredentialMutationPublication(action, "/headers/authorization")
			if _, err := NewRegistry().Publish(publication); !errors.Is(err, ErrInvalidRoute) {
				t.Fatalf("credential mutation error=%v", err)
			}
		})
	}

	publication := requiredReplayCredentialMutationPublication(
		extensionmanifest.RouteActionFilter, "/headers/x-trace",
	)
	if _, err := NewRegistry().Publish(publication); err != nil {
		t.Fatalf("ordinary request metadata mutation rejected: %v", err)
	}
}

func TestRoutePolicyAndExecutionPlanPublishAtomicallyTo64Readers(t *testing.T) {
	registry := NewRegistry()
	providers := NewProviderSelectionAPI(registry, newMemoryProviderSelectionStore())
	first := policySnapshotPublication("1.0.0", 'a', "policy.atomic.route@1", RouteExecutionPolicy{
		RateLimit: "host.ip_write@1", Idempotency: "disabled",
	})
	second := policySnapshotPublication("2.0.0", 'b', "policy.atomic.route@2", RouteExecutionPolicy{
		RateLimit: "host.ip_write@1", Idempotency: "required.24h@1", IdempotencyRequired: true,
	})
	if _, err := registry.Publish(first); err != nil {
		t.Fatal(err)
	}

	const readers = 64
	start := make(chan struct{})
	var workers sync.WaitGroup
	var failed atomic.Bool
	var firstSeen atomic.Int64
	var secondSeen atomic.Int64
	errorsSeen := make(chan error, 1)
	report := func(err error) {
		failed.Store(true)
		select {
		case errorsSeen <- err:
		default:
		}
	}
	verify := func(plan RouteExecutionPlan) error {
		policy, ok := plan.ExecutionPolicy()
		if !ok {
			return fmt.Errorf("revision %d has no bound policy", plan.Revision())
		}
		switch plan.Terminal().ContractVersion {
		case "policy.atomic.route@1":
			firstSeen.Add(1)
			if policy != first.Policies[0].Policy {
				return fmt.Errorf("v1 plan observed policy %#v", policy)
			}
		case "policy.atomic.route@2":
			secondSeen.Add(1)
			if policy != second.Policies[0].Policy {
				return fmt.Errorf("v2 plan observed policy %#v", policy)
			}
		default:
			return fmt.Errorf("unknown terminal %#v", plan.Terminal())
		}
		return nil
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
				if err := verify(plan); err != nil {
					report(err)
					return
				}
				selected, err := providers.BuildExecutionPlan(context.Background(), "POST", "/policy-atomic")
				if err != nil {
					report(err)
					return
				}
				if err := verify(selected); err != nil {
					report(err)
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

func TestRoutePolicyCASConflictAndRollbackPreserveExactPlan(t *testing.T) {
	registry := NewRegistry()
	first := policySnapshotPublication("1.0.0", 'a', "policy.atomic.route@1", RouteExecutionPolicy{
		RateLimit: "host.ip_write@1", Idempotency: "disabled",
	})
	second := policySnapshotPublication("2.0.0", 'b', "policy.atomic.route@2", RouteExecutionPolicy{
		RateLimit: "host.ip_write@1", Idempotency: "required.24h@1", IdempotencyRequired: true,
	})
	initial, err := registry.Publish(first)
	if err != nil {
		t.Fatal(err)
	}
	rollback := registry.PublicationSnapshot().Publication
	if _, err := registry.PublishIfRevision(second, 0); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale publication error=%v", err)
	}
	assertPolicySnapshotPlan(t, registry, initial.Revision, first.Policies[0].Policy)

	upgraded, err := registry.PublishIfRevision(second, initial.Revision)
	if err != nil {
		t.Fatal(err)
	}
	assertPolicySnapshotPlan(t, registry, upgraded.Revision, second.Policies[0].Policy)
	restored, err := registry.PublishIfRevision(rollback, upgraded.Revision)
	if err != nil {
		t.Fatal(err)
	}
	assertPolicySnapshotPlan(t, registry, restored.Revision, first.Policies[0].Policy)
}

func assertPolicySnapshotPlan(t *testing.T, registry *Registry, revision uint64, want RouteExecutionPolicy) {
	t.Helper()
	plan, err := registry.BuildExecutionPlan("POST", "/policy-atomic")
	if err != nil {
		t.Fatal(err)
	}
	policy, ok := plan.ExecutionPolicy()
	if plan.Revision() != revision || !ok || policy != want {
		t.Fatalf("plan revision=%d policy=%#v bound=%t want revision=%d policy=%#v", plan.Revision(), policy, ok, revision, want)
	}
}

func policySnapshotPublication(
	version string,
	digest rune,
	contractVersion string,
	policy RouteExecutionPolicy,
) Publication {
	artifact := routeArtifact("policy.atomic", version, digest)
	declaration := pluginRoute("policy.atomic.route", "/policy-atomic", 0, "POST")
	declaration.ContractVersion = contractVersion
	return Publication{
		Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{declaration}}},
		Policies: []RoutePolicyBinding{{
			Artifact: artifact, RouteID: declaration.ID, ContractVersion: contractVersion,
			Method: "POST", Policy: policy,
		}},
	}
}

func requiredReplayCredentialMutationPublication(action, field string) Publication {
	artifact := routeArtifact("policy.credentials", "1.0.0", 'c')
	terminal := pluginRoute("policy.credentials.route", "/policy-credentials", 0, "POST")
	modifier := modifierRoute(
		"policy.credentials."+action, terminal.ID, terminal.Path, action, "POST", 10,
	)
	modifier.Guard = extensionmanifest.GuardCoreRaw
	modifier.MutableRequestFields = []string{field}
	if action == extensionmanifest.RouteActionGlobalMiddleware {
		modifier.TargetID = ""
		modifier.Path = ""
		modifier.Methods = nil
	}
	return Publication{
		Plugins: []PluginRouteSet{{
			Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{terminal, modifier},
		}},
		Policies: []RoutePolicyBinding{{
			Artifact: artifact, RouteID: terminal.ID, ContractVersion: terminal.ContractVersion,
			Method: "POST", Policy: RouteExecutionPolicy{
				RateLimit: "host.ip_write@1", Idempotency: "required.24h@1", IdempotencyRequired: true,
			},
		}},
	}
}

type policySnapshotResolver struct {
	artifact PluginArtifact
	routeID  string
	policies map[string]RouteExecutionPolicy
	calls    int
}

type missingPolicySnapshotResolver struct{}

func (missingPolicySnapshotResolver) ResolveRouteExecutionPolicy(RouteExecutionStep) (RouteExecutionPolicy, error) {
	return RouteExecutionPolicy{}, ErrRoutePolicyNotFound
}

func (r *policySnapshotResolver) ResolveRouteExecutionPolicy(step RouteExecutionStep) (RouteExecutionPolicy, error) {
	if step.Provider.Artifact != r.artifact || step.RouteID != r.routeID {
		return RouteExecutionPolicy{}, errors.New("policy resolver received a different exact route")
	}
	r.calls++
	policy, ok := r.policies[step.Method]
	if !ok {
		return RouteExecutionPolicy{}, ErrRoutePolicyNotFound
	}
	return policy, nil
}
