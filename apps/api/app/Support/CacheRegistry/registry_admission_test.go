package cacheregistry

import (
	"errors"
	"testing"
)

func TestThirdPartyResolveAndPlanRequireExactRuntimeAdmission(t *testing.T) {
	plugin := testPublication("admit.cache", false, 'a')
	plugin.Caches = []Declaration{testDeclaration("admit.cache.items", PolicyPublic)}
	registry := New()
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve(plugin.Caches[0].ID); !errors.Is(err, ErrArtifactUnavailable) {
		t.Fatalf("resolve without Host admission = %v", err)
	}
	if _, err := registry.Plan(PlanRequest{CacheID: plugin.Caches[0].ID}); !errors.Is(err, ErrArtifactUnavailable) {
		t.Fatalf("plan without Host admission = %v", err)
	}

	seen := Artifact{}
	registry.WithPluginAdmission(func(artifact Artifact) bool {
		seen = artifact
		return artifact == plugin.Artifact
	})
	resolved, err := registry.Resolve(plugin.Caches[0].ID)
	if err != nil || seen != plugin.Artifact || resolved.Artifact != plugin.Artifact {
		t.Fatalf("exact admission: resolved=%#v seen=%#v err=%v", resolved, seen, err)
	}
	plan, err := registry.Plan(PlanRequest{CacheID: plugin.Caches[0].ID})
	if err != nil || plan.Cache.Artifact != plugin.Artifact {
		t.Fatalf("exact admitted plan=%#v err=%v", plan, err)
	}

	registry.WithPluginAdmission(func(artifact Artifact) bool {
		return artifact.ExtensionID == plugin.Artifact.ExtensionID && artifact.RuntimeInstanceID == "wrong-runtime"
	})
	if _, err := registry.Resolve(plugin.Caches[0].ID); !errors.Is(err, ErrArtifactUnavailable) {
		t.Fatalf("non-exact admission = %v", err)
	}
}

func TestAdmissionIsRecheckedAroundExternalCallback(t *testing.T) {
	plugin := testPublication("drain.cache", false, 'a')
	plugin.Caches = []Declaration{testDeclaration("drain.cache.items", PolicyPublic)}
	registry := New()
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	calls := 0
	registry.WithPluginAdmission(func(artifact Artifact) bool {
		calls++
		return artifact == plugin.Artifact && calls == 1
	})
	if _, err := registry.Resolve(plugin.Caches[0].ID); !errors.Is(err, ErrArtifactUnavailable) {
		t.Fatalf("resolve survived drain during admission callback = %v", err)
	}
	if calls < 2 {
		t.Fatalf("admission was not rechecked, calls=%d", calls)
	}

	calls = 0
	if _, err := registry.Plan(PlanRequest{CacheID: plugin.Caches[0].ID}); !errors.Is(err, ErrArtifactUnavailable) {
		t.Fatalf("plan survived drain during admission callback = %v", err)
	}
}

func TestCoreResolveAndPlanUseSealedHostAuthority(t *testing.T) {
	core := testPublication("core.cache", true, 'a')
	core.Caches = []Declaration{testDeclaration("core.cache.items", PolicyPublic)}
	registry := New().WithPluginAdmission(func(Artifact) bool {
		t.Fatal("Core must not call plugin admission")
		return false
	})
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve(core.Caches[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Plan(PlanRequest{CacheID: core.Caches[0].ID}); err != nil {
		t.Fatal(err)
	}
}

func TestActorAndPermissionPoliciesProduceIsolatedSegments(t *testing.T) {
	plugin := testPublication("policy.cache", false, 'a')
	actor := testDeclaration("policy.cache.actor", PolicyActor)
	permission := testDeclaration("policy.cache.permission", PolicyPermission)
	private := testDeclaration("policy.cache.private", PolicyPrivate)
	public := testDeclaration("policy.cache.public", PolicyPublic)
	plugin.Caches = []Declaration{actor, permission, private, public}
	registry := New().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}

	actorA := mustPlan(t, registry, PlanRequest{CacheID: actor.ID, ActorFingerprint: "actor-a", PermissionFingerprint: "ignored-a"})
	actorB := mustPlan(t, registry, PlanRequest{CacheID: actor.ID, ActorFingerprint: "actor-b", PermissionFingerprint: "ignored-b"})
	actorAReplay := mustPlan(t, registry, PlanRequest{CacheID: actor.ID, ActorFingerprint: "actor-a", PermissionFingerprint: "ignored-c"})
	if actorA.Isolation.SegmentDigest == actorB.Isolation.SegmentDigest ||
		actorA.Isolation.SegmentDigest != actorAReplay.Isolation.SegmentDigest ||
		actorA.Isolation.PermissionFingerprint != "" {
		t.Fatalf("actor isolation: a=%#v b=%#v replay=%#v", actorA.Isolation, actorB.Isolation, actorAReplay.Isolation)
	}
	if _, err := registry.Plan(PlanRequest{CacheID: actor.ID}); !errors.Is(err, ErrIsolationRequired) {
		t.Fatalf("actor policy without actor projection = %v", err)
	}

	permissionA := mustPlan(t, registry, PlanRequest{
		CacheID: permission.ID, ActorFingerprint: "ignored-a", PermissionFingerprint: "roles:a",
	})
	permissionB := mustPlan(t, registry, PlanRequest{
		CacheID: permission.ID, ActorFingerprint: "ignored-b", PermissionFingerprint: "roles:b",
	})
	permissionAReplay := mustPlan(t, registry, PlanRequest{
		CacheID: permission.ID, ActorFingerprint: "ignored-c", PermissionFingerprint: "roles:a",
	})
	if permissionA.Isolation.SegmentDigest == permissionB.Isolation.SegmentDigest ||
		permissionA.Isolation.SegmentDigest != permissionAReplay.Isolation.SegmentDigest ||
		permissionA.Isolation.ActorFingerprint != "" {
		t.Fatalf("permission isolation: a=%#v b=%#v replay=%#v", permissionA.Isolation, permissionB.Isolation, permissionAReplay.Isolation)
	}
	if _, err := registry.Plan(PlanRequest{CacheID: permission.ID}); !errors.Is(err, ErrIsolationRequired) {
		t.Fatalf("permission policy without permission projection = %v", err)
	}

	privateA := mustPlan(t, registry, PlanRequest{CacheID: private.ID, ActorFingerprint: "ignored-a"})
	privateB := mustPlan(t, registry, PlanRequest{CacheID: private.ID, ActorFingerprint: "ignored-b"})
	publicA := mustPlan(t, registry, PlanRequest{CacheID: public.ID, PermissionFingerprint: "ignored-a"})
	publicB := mustPlan(t, registry, PlanRequest{CacheID: public.ID, PermissionFingerprint: "ignored-b"})
	if privateA.Isolation.SegmentDigest != privateB.Isolation.SegmentDigest ||
		publicA.Isolation.SegmentDigest != publicB.Isolation.SegmentDigest {
		t.Fatal("irrelevant projections changed private/public isolation")
	}
}

func TestPlanSupportsDeclaredNamespaceCanonicalLocaleAndPostOperationFence(t *testing.T) {
	plugin := testPublication("locale.cache", false, 'a')
	declaration := testDeclaration("locale.cache.items", PolicyPublic)
	plugin.Caches = []Declaration{declaration}
	registry := New().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	byID := mustPlan(t, registry, PlanRequest{CacheID: declaration.ID, LocaleFingerprint: "zh-cn"})
	byNamespace := mustPlan(t, registry, PlanRequest{Namespace: declaration.Namespace, LocaleFingerprint: "zh-CN"})
	english := mustPlan(t, registry, PlanRequest{CacheID: declaration.ID, LocaleFingerprint: "en-US"})
	if byID.Isolation.LocaleFingerprint != "zh-CN" || byID.Isolation.SegmentDigest != byNamespace.Isolation.SegmentDigest ||
		byID.Isolation.SegmentDigest == english.Isolation.SegmentDigest {
		t.Fatalf("locale/namespace plans: byID=%#v byNamespace=%#v english=%#v", byID, byNamespace, english)
	}
	if err := registry.ValidatePlan(byID); err != nil {
		t.Fatal(err)
	}
	replacement := testPublication("locale.cache", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-locale-cache-v2"
	replacement.Caches = []Declaration{declaration}
	if _, err := registry.PublishIfArtifact(plugin.Artifact, replacement); err != nil {
		t.Fatal(err)
	}
	if err := registry.ValidatePlan(byID); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("old plan after replacement = %v", err)
	}
}

func mustPlan(t *testing.T, registry *Registry, request PlanRequest) Plan {
	t.Helper()
	plan, err := registry.Plan(request)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}
