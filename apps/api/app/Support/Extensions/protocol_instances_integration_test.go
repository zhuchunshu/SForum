package extensionsruntime_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

func TestProtocolV2StagesPublishesRollsBackAndStopsExactInstances(t *testing.T) {
	extension := protocolV2TestExtension(t, "v2")
	gateway, _ := newProtocolV2HostGateway()
	t.Cleanup(func() { _ = gateway.Close() })
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust:   staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
		HostAPI: gateway,
	})
	t.Cleanup(func() { _ = starter.Stop(context.Background(), extension) })

	firstTarget, err := starter.Start(context.Background(), extension)
	if err != nil {
		t.Fatal(err)
	}
	first := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: firstTarget.InstanceID}
	assertProtocolInstanceState(t, starter, first, extensionsruntime.ProtocolRuntimePublished)
	resolved, err := gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != first.InstanceID {
		t.Fatalf("first published service = %#v, %v", resolved, err)
	}
	firstProvider := resolved.Winner.Provider
	legacyTransition := extension
	legacyTransition.Manifest.Backend.ProtocolVersion = 1
	legacyTransition.Manifest.Backend.HostAPIVersion = ""
	if _, err := starter.Start(context.Background(), legacyTransition); !errors.Is(err, extensionsruntime.ErrProtocolInstanceTransitionBlocked) {
		t.Fatalf("v2 to v1 transition error = %v", err)
	}
	resolved, err = gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != first.InstanceID {
		t.Fatalf("blocked v1 transition changed active service = %#v, %v", resolved, err)
	}

	// Start 的请求 context 只约束启动；候选进程由 exact lifecycle 持有。
	stageCtx, cancelStage := context.WithCancel(context.Background())
	secondTarget, err := starter.StartInstance(stageCtx, extension)
	if err != nil {
		t.Fatal(err)
	}
	cancelStage()
	second := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: secondTarget.InstanceID}
	if second.InstanceID == "" || second.InstanceID == first.InstanceID {
		t.Fatalf("exact identities first=%#v second=%#v", first, second)
	}
	assertProtocolInstanceState(t, starter, second, extensionsruntime.ProtocolRuntimeStaged)
	if health, err := starter.HealthInstance(context.Background(), second); err != nil || !health.OK {
		t.Fatalf("staged health = %#v, %v", health, err)
	}
	stillFirst, err := gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || stillFirst.Winner.InstanceID != first.InstanceID {
		t.Fatalf("staged process changed active registry = %#v, %v", stillFirst, err)
	}

	published, err := starter.PublishInstance(context.Background(), second)
	if err != nil || published.State != extensionsruntime.ProtocolRuntimePublished {
		t.Fatalf("publish second = %#v, %v", published, err)
	}
	assertProtocolInstanceState(t, starter, first, extensionsruntime.ProtocolRuntimeRetained)
	if health, err := starter.HealthInstance(context.Background(), first); err != nil || !health.OK {
		t.Fatalf("retained old process health = %#v, %v", health, err)
	}
	resolved, err = gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != second.InstanceID {
		t.Fatalf("second published service = %#v, %v", resolved, err)
	}
	oldOutput, oldDetail, err := firstProvider.Invoke(
		context.Background(), nil, "runtime.v2.service.echo", "1.0.0", "echo",
		&protocolwire.TypedDocument{SchemaId: "runtime.v2.service.echo.request", SchemaVersion: "1"},
	)
	if err != nil || oldDetail != nil || oldOutput.GetSchemaId() != "runtime.v2.service.echo.response" {
		t.Fatalf("inflight old provider after publish = %#v, %#v, %v", oldOutput, oldDetail, err)
	}

	// Retained process can be republished as an exact rollback without restart.
	rolledBack, err := starter.PublishInstance(context.Background(), first)
	if err != nil || rolledBack.State != extensionsruntime.ProtocolRuntimePublished {
		t.Fatalf("rollback publish = %#v, %v", rolledBack, err)
	}
	assertProtocolInstanceState(t, starter, second, extensionsruntime.ProtocolRuntimeRetained)
	resolved, err = gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != first.InstanceID {
		t.Fatalf("rollback service = %#v, %v", resolved, err)
	}
	if _, err := starter.PublishInstance(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	// Stopping the stale instance must never unregister the replacement.
	if err := starter.StopInstance(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if _, err := starter.InspectInstance(first); !errors.Is(err, extensionsruntime.ErrRuntimeInstanceNotFound) {
		t.Fatalf("stopped inspect error = %v", err)
	}
	if err := starter.StopInstance(context.Background(), first); !errors.Is(err, extensionsruntime.ErrRuntimeInstanceNotFound) {
		t.Fatalf("stale stop error = %v", err)
	}
	resolved, err = gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != second.InstanceID {
		t.Fatalf("stale stop removed replacement = %#v, %v", resolved, err)
	}

	thirdTarget, err := starter.StartInstance(context.Background(), extension)
	if err != nil {
		t.Fatal(err)
	}
	third := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: thirdTarget.InstanceID}
	cancelledPublish, cancelPublish := context.WithCancel(context.Background())
	cancelPublish()
	if _, err := starter.PublishInstance(cancelledPublish, third); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled publish error = %v", err)
	}
	assertProtocolInstanceState(t, starter, third, extensionsruntime.ProtocolRuntimeStaged)
	resolved, err = gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != second.InstanceID {
		t.Fatalf("cancelled publish changed replacement = %#v, %v", resolved, err)
	}
	if err := starter.DiscardInstance(context.Background(), third); err != nil {
		t.Fatal(err)
	}
	if _, err := starter.InspectInstance(third); !errors.Is(err, extensionsruntime.ErrRuntimeInstanceNotFound) {
		t.Fatalf("discarded inspect error = %v", err)
	}
	if err := starter.DiscardInstance(context.Background(), second); !errors.Is(err, extensionsruntime.ErrProtocolInstancePublished) {
		t.Fatalf("published discard error = %v", err)
	}
}

func TestProtocolV1MustStopBeforeStagingProtocolV2(t *testing.T) {
	legacy := protocolV2TestExtension(t, "v1")
	legacy.Manifest.Backend.ProtocolVersion = 1
	legacy.Manifest.Backend.HostAPIVersion = ""
	candidate := protocolV2TestExtension(t, "v2")
	gateway, _ := newProtocolV2HostGateway()
	t.Cleanup(func() { _ = gateway.Close() })
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust:   staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
		HostAPI: gateway,
	})
	t.Cleanup(func() { _ = starter.Stop(context.Background(), candidate) })

	legacyTarget, err := starter.Start(context.Background(), legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := starter.StartInstance(context.Background(), candidate); !errors.Is(err, extensionsruntime.ErrProtocolInstanceTransitionBlocked) {
		t.Fatalf("v1 to staged v2 transition error = %v", err)
	}
	legacyIdentity := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: legacy.ID, InstanceID: legacyTarget.InstanceID}
	legacySnapshot, err := starter.InspectInstance(legacyIdentity)
	if err != nil || legacySnapshot.State != extensionsruntime.ProtocolRuntimePublished || legacySnapshot.ProtocolVersion != 1 {
		t.Fatalf("blocked transition changed legacy runtime = %#v, %v", legacySnapshot, err)
	}
	if err := starter.Stop(context.Background(), legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := starter.StartInstance(context.Background(), candidate); err != nil {
		t.Fatalf("stage v2 after explicit stop: %v", err)
	}
}

func TestProtocolV2ConcurrentPublishAndStaleStopKeepsReplacement(t *testing.T) {
	extension := protocolV2TestExtension(t, "v2")
	gateway, _ := newProtocolV2HostGateway()
	t.Cleanup(func() { _ = gateway.Close() })
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust:   staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
		HostAPI: gateway,
	})
	t.Cleanup(func() { _ = starter.Stop(context.Background(), extension) })
	firstTarget, err := starter.Start(context.Background(), extension)
	if err != nil {
		t.Fatal(err)
	}
	secondTarget, err := starter.StartInstance(context.Background(), extension)
	if err != nil {
		t.Fatal(err)
	}
	first := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: firstTarget.InstanceID}
	second := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: extension.ID, InstanceID: secondTarget.InstanceID}

	start := make(chan struct{})
	errorsCh := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		<-start
		_, err := starter.PublishInstance(context.Background(), second)
		errorsCh <- err
	}()
	go func() {
		defer group.Done()
		<-start
		errorsCh <- starter.StopInstance(context.Background(), first)
	}()
	close(start)
	group.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertProtocolInstanceState(t, starter, second, extensionsruntime.ProtocolRuntimePublished)
	resolved, err := gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != second.InstanceID {
		t.Fatalf("concurrent replacement service = %#v, %v", resolved, err)
	}
}

func TestProtocolV2UnpublishedStartDefersReadinessUntilHealthAndPublish(t *testing.T) {
	extension := protocolV2TestExtension(t, "v2")
	failing := protocolV2TestExtension(t, "v2-readiness-fail")
	gateway, _ := newProtocolV2HostGateway()
	t.Cleanup(func() { _ = gateway.Close() })
	starter := extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
		Trust:   staticRuntimeTrust{identity: extensions.RuntimeTrustIdentity{TrustGrantID: "41", ImpactDigest: "impact-41"}},
		HostAPI: gateway,
	})
	t.Cleanup(func() { _ = starter.Stop(context.Background(), extension) })
	activeTarget, err := starter.Start(context.Background(), extension)
	if err != nil {
		t.Fatal(err)
	}

	// 兼容 Start 仍是 launch+readiness+publish，readiness 失败不能替换活动实例。
	if _, err := starter.Start(context.Background(), failing); err == nil {
		t.Fatal("compatibility Start published a readiness-failing process")
	}
	resolved, err := gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != activeTarget.InstanceID {
		t.Fatalf("failed compatibility start changed registry = %#v, %v", resolved, err)
	}

	// Staged start intentionally stops before readiness so install hooks can run first.
	candidateTarget, err := starter.StartInstance(context.Background(), failing)
	if err != nil {
		t.Fatalf("stage readiness-failing candidate: %v", err)
	}
	candidate := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: failing.ID, InstanceID: candidateTarget.InstanceID}
	candidateSnapshot, err := starter.InspectInstance(candidate)
	if err != nil || candidateSnapshot.State != extensionsruntime.ProtocolRuntimeStaged || !candidateSnapshot.Healthy ||
		candidateSnapshot.Ready || candidateSnapshot.ReadinessChecked {
		t.Fatalf("staged readiness candidate = %#v, %v", candidateSnapshot, err)
	}
	if _, err := starter.HealthInstance(context.Background(), candidate); !errors.Is(err, extensionsruntime.ErrProtocolInstanceNotReady) {
		t.Fatalf("readiness health error = %v", err)
	}
	if _, err := starter.PublishInstance(context.Background(), candidate); !errors.Is(err, extensionsruntime.ErrProtocolInstanceNotReady) {
		t.Fatalf("readiness publish error = %v", err)
	}
	candidateSnapshot, err = starter.InspectInstance(candidate)
	if err != nil || candidateSnapshot.State != extensionsruntime.ProtocolRuntimeStaged || !candidateSnapshot.Healthy ||
		candidateSnapshot.Ready || !candidateSnapshot.ReadinessChecked {
		t.Fatalf("failed readiness candidate state = %#v, %v", candidateSnapshot, err)
	}
	resolved, err = gateway.ProtocolV2ServiceRegistry().ResolveExact("runtime.v2.service.echo", "1.0.0")
	if err != nil || resolved.Winner.InstanceID != activeTarget.InstanceID {
		t.Fatalf("readiness failure leaked registry = %#v, %v", resolved, err)
	}
	if err := starter.DiscardInstance(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
}

func assertProtocolInstanceState(
	t *testing.T,
	starter *extensionsruntime.ProtocolStarter,
	identity extensionsruntime.RuntimeInstanceIdentity,
	state extensionsruntime.ProtocolRuntimeInstanceState,
) {
	t.Helper()
	snapshot, err := starter.InspectInstance(identity)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Identity != identity || snapshot.State != state || !snapshot.Healthy || snapshot.ProtocolVersion != 2 ||
		snapshot.Target.InstanceID != identity.InstanceID || snapshot.StartedAt.IsZero() {
		t.Fatalf("instance snapshot = %#v, want state %s", snapshot, state)
	}
	if state != extensionsruntime.ProtocolRuntimeStaged && (!snapshot.Ready || !snapshot.ReadinessChecked) {
		t.Fatalf("published/retained instance is not ready: %#v", snapshot)
	}
}
