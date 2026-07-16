package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	queryregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/QueryRegistry"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

type productionQueryActorStoreStub struct {
	actor identity.Actor
	err   error
}

func (s *productionQueryActorStoreStub) LoadActor(context.Context, int64) (identity.Actor, error) {
	return s.actor, s.err
}

func TestProductionQueryActorAuthorityUsesCanonicalLiveHostState(t *testing.T) {
	store := &productionQueryActorStoreStub{actor: identity.Actor{
		ID: 42, Status: identity.UserStatusActive,
		RoleKeys: []string{"moderator", "member", "moderator"},
		Permissions: map[string]bool{
			"query.test.read": true,
			"query.test.deny": false,
		},
		CreatedAt: time.Date(2026, 7, 16, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
	}}
	authority := &productionQueryActorAuthority{store: store}

	first, err := authority.ResolveProtocolV2QueryActor(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	store.actor.RoleKeys = []string{"member", "moderator"}
	store.actor.Permissions = map[string]bool{"query.test.deny": false, "query.test.read": true}
	second, err := authority.ResolveProtocolV2QueryActor(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !first.Authenticated || first.ActorUserID != 42 ||
		!strings.HasPrefix(first.ActorFingerprint, "actor:") ||
		!strings.HasPrefix(first.PolicyFingerprint, "policy:") {
		t.Fatalf("canonical projections differ: first=%#v second=%#v", first, second)
	}

	for _, policy := range []string{
		queryregistry.PermissionPolicyPublic,
		queryregistry.PermissionPolicyLogin,
		"query.test.read",
	} {
		projection, authorizeErr := authority.AuthorizeProtocolV2QueryActor(
			context.Background(), 42, queryregistry.PermissionClaim{PermissionPolicy: policy},
		)
		if authorizeErr != nil || projection != second {
			t.Fatalf("policy %q projection=%#v err=%v", policy, projection, authorizeErr)
		}
	}
	if _, err := authority.AuthorizeProtocolV2QueryActor(
		context.Background(), 42, queryregistry.PermissionClaim{PermissionPolicy: "query.test.denied"},
	); !errors.Is(err, hostapi.ErrProtocolV2QueryActorDenied) {
		t.Fatalf("denied permission error = %v", err)
	}

	store.actor.Permissions["query.test.changed"] = true
	changed, err := authority.ResolveProtocolV2QueryActor(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if changed.ActorFingerprint != second.ActorFingerprint || changed.PolicyFingerprint == second.PolicyFingerprint {
		t.Fatalf("policy mutation projection=%#v before=%#v", changed, second)
	}
}

func TestProductionQueryActorAuthorityRejectsInactiveMissingAndCancelledActors(t *testing.T) {
	store := &productionQueryActorStoreStub{actor: identity.Actor{ID: 42, Status: identity.UserStatusDisabled}}
	authority := &productionQueryActorAuthority{store: store}
	if _, err := authority.ResolveProtocolV2QueryActor(context.Background(), 42); !errors.Is(err, hostapi.ErrProtocolV2QueryActorDenied) {
		t.Fatalf("disabled actor error = %v", err)
	}
	store.actor = identity.Actor{ID: 7, Status: identity.UserStatusActive}
	if _, err := authority.ResolveProtocolV2QueryActor(context.Background(), 42); !errors.Is(err, hostapi.ErrProtocolV2QueryActorDenied) {
		t.Fatalf("mismatched actor error = %v", err)
	}
	store.err = identity.ErrUserNotFound
	if _, err := authority.ResolveProtocolV2QueryActor(context.Background(), 42); !errors.Is(err, hostapi.ErrProtocolV2QueryActorDenied) {
		t.Fatalf("missing actor error = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := authority.ResolveProtocolV2QueryActor(cancelled, 42); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled actor error = %v", err)
	}
}

type productionQueryAdmissionStub struct {
	gate        *extensionsruntime.RuntimeAdmissionGate
	snapshot    extensionsruntime.RuntimeInstanceSnapshot
	err         error
	calls       int
	extensionID string
	class       extensionsruntime.RuntimeCallClass
}

func (s *productionQueryAdmissionStub) acquire(
	ctx context.Context,
	extensionID string,
	class extensionsruntime.RuntimeCallClass,
) (extensionsruntime.RuntimeInstanceSnapshot, *extensionsruntime.RuntimeAdmissionLease, error) {
	s.calls++
	s.extensionID = extensionID
	s.class = class
	if s.err != nil {
		return extensionsruntime.RuntimeInstanceSnapshot{}, nil, s.err
	}
	lease, err := s.gate.Acquire(ctx, class)
	if err != nil {
		return extensionsruntime.RuntimeInstanceSnapshot{}, nil, err
	}
	snapshot := s.snapshot
	snapshot.Admission = s.gate.Snapshot()
	return snapshot, lease, nil
}

func newProductionQueryAdmissionHarness(t *testing.T) (*productionQueryRuntimeAdmission, *productionQueryAdmissionStub) {
	t.Helper()
	identity := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: "query.plugin", InstanceID: "runtime-7"}
	gate, err := extensionsruntime.NewRuntimeAdmissionGate(identity)
	if err != nil {
		t.Fatal(err)
	}
	stub := &productionQueryAdmissionStub{
		gate: gate,
		snapshot: extensionsruntime.RuntimeInstanceSnapshot{
			Identity: identity, ExtensionVersion: "1.2.3", ArtifactDigest: strings.Repeat("a", 64), Active: true,
		},
	}
	return &productionQueryRuntimeAdmission{acquire: stub.acquire}, stub
}

func productionQueryCallerIdentity() *protocolv2.ExtensionIdentity {
	return &protocolv2.ExtensionIdentity{
		ExtensionId: "query.plugin", ExtensionVersion: "1.2.3", ArtifactDigest: strings.Repeat("a", 64),
		TrustGrantId: "grant-7", RuntimeEpoch: 7, InstanceId: "runtime-7",
	}
}

func TestProductionQueryCallerAdmissionFencesExactRuntimeAndReleasesLease(t *testing.T) {
	admission, stub := newProductionQueryAdmissionHarness(t)
	if err := admission.AuthorizeProtocolV2QueryCaller(context.Background(), productionQueryCallerIdentity()); err != nil {
		t.Fatal(err)
	}
	if stub.calls != 1 || stub.extensionID != "query.plugin" || stub.class != extensionsruntime.RuntimeCallHost ||
		stub.gate.Snapshot().ActiveTotal != 0 {
		t.Fatalf("caller admission stub=%#v gate=%#v", stub, stub.gate.Snapshot())
	}

	stale := productionQueryCallerIdentity()
	stale.ArtifactDigest = strings.Repeat("b", 64)
	if err := admission.AuthorizeProtocolV2QueryCaller(context.Background(), stale); !errors.Is(err, errProductionQueryRegistryRuntimeStale) {
		t.Fatalf("stale caller error = %v", err)
	}
	if stub.gate.Snapshot().ActiveTotal != 0 {
		t.Fatalf("stale caller leaked lease: %#v", stub.gate.Snapshot())
	}

	stub.gate.BeginDrain()
	if err := admission.AuthorizeProtocolV2QueryCaller(context.Background(), productionQueryCallerIdentity()); !errors.Is(err, extensionsruntime.ErrRuntimeAdmissionDraining) {
		t.Fatalf("draining caller error = %v", err)
	}
}

func TestProductionQueryExecutionAdmissionHoldsAndReleasesExactPluginLease(t *testing.T) {
	admission, stub := newProductionQueryAdmissionHarness(t)
	artifact := queryregistry.Artifact{
		ExtensionID: "query.plugin", ExtensionVersion: "1.2.3", PackageDigest: strings.Repeat("a", 64),
		VersionID: 9, RuntimeInstanceID: "runtime-7",
	}
	release, err := admission.AcquireQueryExecution(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	if release == nil || stub.class != extensionsruntime.RuntimeCallProvider || stub.gate.Snapshot().ActiveTotal != 1 {
		t.Fatalf("active execution stub=%#v gate=%#v", stub, stub.gate.Snapshot())
	}
	release()
	release()
	if stub.gate.Snapshot().ActiveTotal != 0 {
		t.Fatalf("execution lease was not idempotently released: %#v", stub.gate.Snapshot())
	}

	artifact.RuntimeInstanceID = "runtime-old"
	if _, err := admission.AcquireQueryExecution(context.Background(), artifact); !errors.Is(err, errProductionQueryRegistryRuntimeStale) {
		t.Fatalf("stale artifact error = %v", err)
	}
	if stub.gate.Snapshot().ActiveTotal != 0 {
		t.Fatalf("stale artifact leaked lease: %#v", stub.gate.Snapshot())
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	artifact.RuntimeInstanceID = "runtime-7"
	if _, err := admission.AcquireQueryExecution(cancelled, artifact); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled execution error = %v", err)
	}
}

type productionQueryAuthorityResolverStub struct{}

func (productionQueryAuthorityResolverStub) ResolveProtocolV2QueryAuthority(
	context.Context,
	*protocolv2.ExtensionIdentity,
) (hostapi.ProtocolV2QueryAuthority, error) {
	return hostapi.ProtocolV2QueryAuthority{ExactArtifact: true, CoreViews: true}, nil
}

type productionQueryTraceSinkStub struct{}

func (productionQueryTraceSinkStub) RecordQueryTrace(hostapi.QueryTrace) {}

func TestBindProductionQueryRegistryUsesLifecycleSnapshotAndFreezesGateway(t *testing.T) {
	poolConfig, err := pgxpool.ParseConfig("postgres://postgres:postgres@127.0.0.1:1/sforum?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	stableRuntime, err := hostapi.NewPostgresProtocolV2QueryRuntime(pool, productionQueryAuthorityResolverStub{})
	if err != nil {
		t.Fatal(err)
	}
	registry, catalog, err := hostapi.NewQueryRegistryCoreRegistry(hostapi.QueryRegistryCoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	actors := &productionQueryActorStoreStub{actor: identity.Actor{ID: 42, Status: identity.UserStatusActive}}
	runtimeManager := extensionsruntime.NewManager(extensionsruntime.ManagerConfig{})
	gateway := hostapi.NewGateway(nil)
	t.Cleanup(func() { _ = gateway.Close() })

	bound, err := bindProductionQueryRegistry(
		registry, catalog, stableRuntime, actors, runtimeManager, gateway, productionQueryTraceSinkStub{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bound == nil || bound.Execution == nil || bound.Service == nil || !bound.Execution.BoundToRegistry(registry) {
		t.Fatalf("bound Query Registry = %#v", bound)
	}
	if _, err := bindProductionQueryRegistry(
		registry, catalog, stableRuntime, actors, runtimeManager, gateway, productionQueryTraceSinkStub{},
	); err == nil {
		t.Fatal("Gateway accepted a second production Query Registry binding")
	}
}
