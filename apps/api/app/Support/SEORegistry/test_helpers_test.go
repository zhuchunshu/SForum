package seoregistry

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testArtifact(extensionID string, digest byte) Artifact {
	return Artifact{
		ExtensionID: extensionID, ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat(string(digest), 64), ImpactDigest: strings.Repeat("f", 64),
		VersionID: 1, RuntimeInstanceID: "runtime-" + strings.ReplaceAll(extensionID, ".", "-"),
	}
}

func testPublication(extensionID string, digest byte) Publication {
	return Publication{Artifact: testArtifact(extensionID, digest)}
}

func testDeclaration(publication Publication, suffix, scope, kind, action, failure string, priority int) Declaration {
	id := publication.Artifact.ExtensionID + ".seo." + suffix
	return Declaration{
		ID: id, ContractVersion: id + "@1", Scope: scope, Kind: kind, Action: action,
		Handler:  publication.Artifact.ExtensionID + ".handler." + suffix,
		Priority: priority, FailurePolicy: failure, Timeout: 100 * time.Millisecond,
	}
}

func testBinding(publication Publication, declaration Declaration, provider Provider) ProviderBinding {
	return ProviderBinding{
		ContributionID: declaration.ID, ContractVersion: declaration.ContractVersion,
		Handler: declaration.Handler, Artifact: publication.Artifact, Provider: provider,
	}
}

func publishForExecution(t *testing.T, publication Publication) (*Registry, []Contribution) {
	t.Helper()
	registry := New()
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	inspection, err := registry.Inspect("core.page.topic")
	if err != nil {
		t.Fatal(err)
	}
	return registry, inspection.Contributions
}

func contributionByID(t *testing.T, values []Contribution, id string) Contribution {
	t.Helper()
	for _, contribution := range values {
		if contribution.ID == id {
			return contribution
		}
	}
	t.Fatalf("missing contribution %s", id)
	return Contribution{}
}

type testLease struct {
	ctx      context.Context
	released atomic.Bool
}

func (l *testLease) Context() context.Context { return l.ctx }

func (l *testLease) Release() { l.released.Store(true) }

type testAdmission struct {
	mu      sync.Mutex
	leases  map[Artifact]*testLease
	acquire func(context.Context, Artifact) (AdmissionLease, error)
	calls   atomic.Int32
}

func newTestAdmission() *testAdmission {
	return &testAdmission{leases: make(map[Artifact]*testLease)}
}

func (a *testAdmission) AcquireSEOExecution(ctx context.Context, artifact Artifact) (AdmissionLease, error) {
	a.calls.Add(1)
	if a.acquire != nil {
		return a.acquire(ctx, artifact)
	}
	lease := &testLease{ctx: ctx}
	a.mu.Lock()
	a.leases[artifact] = lease
	a.mu.Unlock()
	return lease, nil
}

func (a *testAdmission) active(artifact Artifact) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	lease := a.leases[artifact]
	return lease != nil && !lease.released.Load()
}

func (a *testAdmission) released(artifact Artifact) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	lease := a.leases[artifact]
	return lease != nil && lease.released.Load()
}

func mustRuntime(t *testing.T, registry *Registry, admission ExecutionAdmission, bindings []ProviderBinding, trace ExecutionTraceSink) *ExecutionRuntime {
	t.Helper()
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Admission: admission, FinalPolicy: allowTestSEOFinalPolicy(),
		Providers: bindings, Trace: trace,
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func allowTestSEOFinalPolicy() FinalPolicy {
	return FinalPolicyFunc(func(context.Context, FinalPolicyRequest) error { return nil })
}

func zeroExecuteResult(value ExecuteResult) bool { return reflect.DeepEqual(value, ExecuteResult{}) }
