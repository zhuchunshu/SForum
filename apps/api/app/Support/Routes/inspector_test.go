package routes

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestInspectorCapturesExactSelectedChainConflictsAndDetachedValues(t *testing.T) {
	registry, providers, artifact, key, request := inspectorSelectedFixture(t)
	selected, err := providers.Select(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	ring := NewRouteTraceRing(8)
	ring.AppendRouteTrace(inspectorTraceEvent(artifact, request.ProviderRouteID, key.Method, key.PathSignature))
	ring.AppendRouteTrace(inspectorTraceEvent(
		routeArtifact("unrelated.trace", "1.0.0", 'f'), "unrelated.trace.route", "GET", "/s:unrelated",
	))
	inspector := NewInspector(registry, providers, ring)

	snapshot, err := inspector.Inspect(t.Context(), "post", "/inspect/topics?private=discarded")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Revision != registry.Revision() || snapshot.SafeMode || snapshot.Resolution != InspectionResolved {
		t.Fatalf("inspection header = %#v", snapshot)
	}
	if len(snapshot.Chain) != 2 || snapshot.Chain[0].Phase != RoutePhaseBefore ||
		snapshot.Chain[1].Phase != RoutePhaseHandler || snapshot.Chain[1].RouteID != request.ProviderRouteID {
		t.Fatalf("inspection chain = %#v", snapshot.Chain)
	}
	terminal := snapshot.Chain[1]
	if terminal.Provider.Artifact == nil || *terminal.Provider.Artifact != artifact || terminal.Guard == "" ||
		terminal.RequestSchema == "" || terminal.ResponseSchema == "" || terminal.Mode != "http" ||
		terminal.Fallback != "closed" || terminal.TimeoutMS != 2500 {
		t.Fatalf("terminal inspection = %#v", terminal)
	}
	if snapshot.Provider.Status != "selected" || snapshot.Provider.Desired == nil ||
		snapshot.Provider.Desired.Revision != selected.Revision || snapshot.Provider.Live == nil ||
		snapshot.Provider.Live.Artifact == nil || *snapshot.Provider.Live.Artifact != artifact {
		t.Fatalf("provider inspection = %#v", snapshot.Provider)
	}
	if len(snapshot.Conflicts) != 1 || snapshot.Conflicts[0].SelectionStatus != "selected" ||
		snapshot.Conflicts[0].Desired == nil || len(snapshot.Conflicts[0].Candidates) != 2 {
		t.Fatalf("conflict inspection = %#v", snapshot.Conflicts)
	}
	if len(snapshot.Traces) != 1 || snapshot.Traces[0].RouteID != request.ProviderRouteID {
		t.Fatalf("relevant traces = %#v", snapshot.Traces)
	}

	// Every exported pointer/slice is detached from Registry, selection store,
	// and trace ring state.
	snapshot.Chain[1].RouteID = "mutated"
	snapshot.Conflicts[0].Candidates[0].RouteID = "mutated"
	snapshot.Provider.Desired.Revision = 999
	snapshot.Traces[0].Provider.Artifact.ExtensionID = "mutated"
	again, err := inspector.Inspect(t.Context(), "POST", "/inspect/topics")
	if err != nil {
		t.Fatal(err)
	}
	if again.Chain[1].RouteID != request.ProviderRouteID || again.Provider.Desired.Revision != selected.Revision ||
		again.Traces[0].Provider.Artifact.ExtensionID != artifact.ExtensionID {
		t.Fatalf("caller mutated inspector state: %#v", again)
	}
}

func TestInspectorExposesStaleDesiredWithoutForgingLiveChain(t *testing.T) {
	registry, providers, artifact, key, request := inspectorSelectedFixture(t)
	selected, err := providers.Select(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	target := coreRoute(key.TargetRouteID, "POST", "/inspect/topics")
	target.ContractVersion = "sforum.route.inspect.create@2"
	target.Guard.ContractVersion = target.ContractVersion
	replacement := inspectorReplacement(request.ProviderRouteID, target.ID)
	before := inspectorBefore(artifact.ExtensionID+".before", target.ID)
	if _, err := registry.Publish(Publication{
		Core: []CoreRoute{target}, Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{replacement, before}}},
	}); err != nil {
		t.Fatal(err)
	}

	snapshot, err := NewInspector(registry, providers, nil).Inspect(t.Context(), "POST", "/inspect/topics")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Resolution != InspectionStale || len(snapshot.Chain) != 0 ||
		snapshot.Provider.Status != "stale" || snapshot.Provider.Live != nil ||
		snapshot.Provider.Desired == nil || snapshot.Provider.Desired.Revision != selected.Revision ||
		snapshot.Provider.Desired.Key.TargetContractVersion != key.TargetContractVersion {
		t.Fatalf("stale inspection = %#v", snapshot)
	}
}

func TestInspectorExposesUnselectedConflictWithoutImplicitPriorityWinner(t *testing.T) {
	registry, providers, _, _, _ := inspectorSelectedFixture(t)
	snapshot, err := NewInspector(registry, providers, nil).Inspect(t.Context(), "POST", "/inspect/topics")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Resolution != InspectionAmbiguous || len(snapshot.Chain) != 0 ||
		snapshot.Provider.Status != "unselected" || snapshot.Provider.Desired != nil || snapshot.Provider.Live != nil ||
		len(snapshot.Conflicts) != 1 || snapshot.Conflicts[0].SelectionStatus != "unselected" {
		t.Fatalf("unselected conflict inspection = %#v", snapshot)
	}
}

func TestInspectorExcludesUnrelatedConflictsAndTraces(t *testing.T) {
	registry, providers, artifact, _, _ := inspectorSelectedFixture(t)
	publication := registry.PublicationSnapshot().Publication
	unrelatedTarget := coreRoute("core.route.inspect.unrelated", "GET", "/inspect/unrelated")
	publication.Core = append(publication.Core, unrelatedTarget)
	publication.Plugins[0].Routes = append(
		publication.Plugins[0].Routes,
		inspectorReplacementForMethod("inspect.plugin.unrelated.one", unrelatedTarget.ID, "/inspect/unrelated", "GET"),
		inspectorReplacementForMethod("inspect.plugin.unrelated.two", unrelatedTarget.ID, "/inspect/unrelated", "GET"),
	)
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	ring := NewRouteTraceRing(8)
	ring.AppendRouteTrace(inspectorTraceEvent(artifact, "inspect.plugin.unrelated.one", "GET", "/s:inspect/s:unrelated"))

	snapshot, err := NewInspector(registry, providers, ring).Inspect(t.Context(), "POST", "/inspect/topics")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Resolution != InspectionAmbiguous || len(snapshot.Conflicts) != 1 ||
		snapshot.Conflicts[0].RouteID != "core.route.inspect.create" || len(snapshot.Traces) != 0 {
		t.Fatalf("inspection included unrelated route evidence: %#v", snapshot)
	}
}

func TestInspectorSafeModeContainsOnlyHostFacts(t *testing.T) {
	registry := NewRegistry()
	if _, err := registry.Publish(Publication{
		SafeMode: true,
		Core:     []CoreRoute{coreRoute("core.route.inspect.health", "GET", "/inspect/health")},
		Plugins:  []PluginRouteSet{{Artifact: routeArtifact("ignored.safe", "1.0.0", 'd')}},
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := NewInspector(registry, nil, nil).Inspect(t.Context(), "GET", "/inspect/health")
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.SafeMode || snapshot.Resolution != InspectionResolved || len(snapshot.Chain) != 1 ||
		snapshot.Chain[0].Provider.Kind != ProviderCore || snapshot.Chain[0].Provider.Artifact != nil ||
		snapshot.Provider.Status != "not_required" {
		t.Fatalf("safe-mode inspection = %#v", snapshot)
	}
}

func TestRouteTraceRingIsBoundedConcurrentDetachedAndPayloadFree(t *testing.T) {
	for _, forbidden := range []string{"request", "response", "header", "query", "body", "secret", "actor", "error"} {
		traceType := reflect.TypeOf(RouteTraceEvent{})
		for index := 0; index < traceType.NumField(); index++ {
			if strings.Contains(strings.ToLower(traceType.Field(index).Name), forbidden) {
				t.Fatalf("trace input exposes forbidden field %q", traceType.Field(index).Name)
			}
		}
	}

	ring := NewRouteTraceRing(2)
	artifact := routeArtifact("trace.exact", "1.0.0", 'e')
	for index := 0; index < 3; index++ {
		event := inspectorTraceEvent(artifact, "trace.exact.route", "GET", "/s:trace")
		event.StepIndex = index
		ring.AppendRouteTrace(event)
	}
	records := ring.RouteTraces(10)
	if len(records) != 2 || records[0].Sequence != 2 || records[1].Sequence != 3 ||
		records[0].StepIndex != 1 || records[1].Provider.Artifact == nil ||
		*records[1].Provider.Artifact != artifact {
		t.Fatalf("bounded trace records = %#v", records)
	}
	records[1].Provider.Artifact.ExtensionID = "mutated"
	if ring.RouteTraces(1)[0].Provider.Artifact.ExtensionID != artifact.ExtensionID {
		t.Fatal("caller mutated trace ring state")
	}
	invalid := inspectorTraceEvent(artifact, "trace.exact.route", "GET", "/s:trace")
	invalid.Provider.Artifact.PackageDigest = "forged"
	ring.AppendRouteTrace(invalid)
	if ring.RouteTraces(10)[1].Sequence != 3 {
		t.Fatal("invalid exact attribution entered trace ring")
	}

	concurrent := NewRouteTraceRing(64)
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := 0; index < 100; index++ {
				concurrent.AppendRouteTrace(inspectorTraceEvent(artifact, "trace.exact.route", "GET", "/s:trace"))
			}
		}()
	}
	group.Wait()
	records = concurrent.RouteTraces(0)
	if len(records) != 64 || records[0].Sequence != 737 || records[63].Sequence != 800 {
		t.Fatalf("concurrent bounded trace sequence = first=%d last=%d len=%d",
			records[0].Sequence, records[len(records)-1].Sequence, len(records))
	}
}

func TestInspectorDisclosesExactCustomGuardBinding(t *testing.T) {
	registry := NewRegistry()
	artifact := routeArtifact("inspect.guard", "1.0.0", 'a')
	route := pluginRoute("inspect.guard.route", "/inspect-guard", 0, "GET")
	route.Guard = "inspect.guard.owner"
	guard := pluginGuard(route.Guard, "custom")
	guard.Permissions = []string{"inspect.guard.read"}
	if _, err := registry.Publish(Publication{Plugins: []PluginRouteSet{{
		Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{route},
		Guards: []extensionmanifest.ManifestGuard{guard},
	}}}); err != nil {
		t.Fatal(err)
	}

	snapshot, err := NewInspector(registry, nil, nil).Inspect(t.Context(), "GET", route.Path)
	if err != nil || len(snapshot.Chain) != 1 || snapshot.Chain[0].PluginGuard == nil {
		t.Fatalf("inspection = %#v, %v", snapshot, err)
	}
	want := PluginGuardBinding{
		ID: guard.ID, ContractVersion: guard.ContractVersion, Kind: guard.Kind,
		Entry: guard.Entry, Digest: guard.Digest, Permissions: []string{"inspect.guard.read"},
	}
	if !equalPluginGuardBinding(*snapshot.Chain[0].PluginGuard, want) {
		t.Fatalf("inspected guard = %#v", snapshot.Chain[0].PluginGuard)
	}
	snapshot.Chain[0].PluginGuard.Permissions[0] = "forged"
	again, err := NewInspector(registry, nil, nil).Inspect(t.Context(), "GET", route.Path)
	if err != nil || !equalPluginGuardBinding(*again.Chain[0].PluginGuard, want) {
		t.Fatalf("detached inspected guard = %#v, %v", again.Chain[0].PluginGuard, err)
	}
}

func TestInspectorRejectsInvalidRequests(t *testing.T) {
	if _, err := (*Inspector)(nil).Inspect(t.Context(), "GET", "/"); !errors.Is(err, ErrInspectorInvalid) {
		t.Fatalf("nil inspector error = %v", err)
	}
	if _, err := NewInspector(NewRegistry(), nil, nil).Inspect(t.Context(), "*", "/"); !errors.Is(err, ErrInspectorInvalid) {
		t.Fatalf("wildcard inspection error = %v", err)
	}
}

func inspectorSelectedFixture(t *testing.T) (*Registry, *ProviderSelectionAPI, PluginArtifact, ProviderSelectionKey, SelectProviderRequest) {
	t.Helper()
	registry := NewRegistry()
	artifact := routeArtifact("inspect.plugin", "1.0.0", 'c')
	target := coreRoute("core.route.inspect.create", "POST", "/inspect/topics")
	providerRouteID := "inspect.plugin.writer"
	replacement := inspectorReplacement(providerRouteID, target.ID)
	before := inspectorBefore("inspect.plugin.before", target.ID)
	snapshot, err := registry.Publish(Publication{
		Core: []CoreRoute{target}, Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{replacement, before}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var signature string
	for _, route := range snapshot.Routes {
		if route.ID == target.ID {
			signature = route.PathSignature
		}
	}
	key := ProviderSelectionKey{
		TargetRouteID: target.ID, TargetContractVersion: target.ContractVersion,
		Method: "POST", PathSignature: signature,
	}
	request := SelectProviderRequest{
		Key: key, ProviderRouteID: providerRouteID, ProviderContractVersion: providerRouteID + "@1",
		ProviderArtifact: artifact, ActorUserID: 7, AuditEventID: 17,
	}
	return registry, NewProviderSelectionAPI(registry, newMemoryProviderSelectionStore()), artifact, key, request
}

func inspectorReplacement(id, targetID string) extensionmanifest.ManifestRoute {
	return inspectorReplacementForMethod(id, targetID, "/inspect/topics", "POST")
}

func inspectorReplacementForMethod(id, targetID, path, method string) extensionmanifest.ManifestRoute {
	return extensionmanifest.ManifestRoute{
		ID: id, ContractVersion: id + "@1", Action: extensionmanifest.RouteActionReplace,
		TargetID: targetID, Path: path, Methods: []string{method},
		Guard: extensionmanifest.GuardCoreInherit, Priority: 100, Fallback: "closed",
		Mode: extensionmanifest.RouteModeHTTP, Handler: "route.write",
		RequestSchema: id + ".request@1", ResponseSchema: id + ".response@1", TimeoutMS: 2500,
	}
}

func inspectorBefore(id, targetID string) extensionmanifest.ManifestRoute {
	return extensionmanifest.ManifestRoute{
		ID: id, ContractVersion: id + "@1", Action: extensionmanifest.RouteActionBefore,
		TargetID: targetID, Path: "/inspect/topics", Methods: []string{"POST"},
		Guard: extensionmanifest.GuardCorePermission, Permission: "topic.create",
		Priority: 50, Fallback: "closed", Mode: extensionmanifest.RouteModeHTTP,
		Handler: "route.before", RequestSchema: id + ".request@1", ResponseSchema: id + ".response@1",
	}
}

func inspectorTraceEvent(artifact PluginArtifact, routeID, method, signature string) RouteTraceEvent {
	return RouteTraceEvent{
		Revision: 1, StepIndex: 1, Phase: RoutePhaseHandler, Action: extensionmanifest.RouteActionReplace,
		RouteID: routeID, ContractVersion: routeID + "@1", Method: method,
		PathSignature: signature, Mode: extensionmanifest.RouteModeHTTP, Fallback: "closed",
		Outcome: RouteTraceSucceeded, Duration: time.Millisecond, CommitState: RouteCommitFinal,
		Provider: Provider{Kind: ProviderPlugin, Artifact: artifact},
	}
}
