package extensions

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestLegacyQueryEnableAndDisableUseHostPublicationBoundary(t *testing.T) {
	for _, test := range legacyQuerySurfaceCases() {
		t.Run(test.name, func(t *testing.T) {
			item := test.build(t, StatusInstalled)
			events := []string{}
			store := &orderedQueryStore{
				fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
				events:             &events,
			}
			runtime := &orderedQueryRuntime{events: &events}
			boundary := &recordingQueryPublicationBoundary{events: &events}
			service := NewServiceWithRuntime(store, t.TempDir(), runtime).BindRuntimeQueryPublications(boundary)

			enabled, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true})
			if err != nil {
				t.Fatal(err)
			}
			if enabled.Status != StatusEnabled || !slices.Equal(events, []string{
				"store.enable", "runtime.start", "query.publish",
			}) {
				t.Fatalf("legacy Query surface enable order = %v, enabled=%#v", events, enabled)
			}

			events = events[:0]
			disabled, err := service.Disable(t.Context(), extensionManager(), item.ID)
			if err != nil {
				t.Fatal(err)
			}
			if disabled.Status != StatusDisabled || !slices.Equal(events, []string{
				"query.quarantine", "store.disable", "runtime.stop",
			}) {
				t.Fatalf("legacy Query surface disable order = %v, disabled=%#v", events, disabled)
			}
			if boundary.publishCalls != 1 || boundary.quarantineCalls != 1 || boundary.rollbackCalls != 0 {
				t.Fatalf("Query boundary calls = %#v", boundary)
			}
		})
	}
}

func TestLegacyQueryEnablePublicationFailureStopsAndDisables(t *testing.T) {
	for _, test := range legacyQuerySurfaceCases() {
		t.Run(test.name, func(t *testing.T) {
			item := test.build(t, StatusInstalled)
			events := []string{}
			store := &orderedQueryStore{
				fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
				events:             &events,
			}
			runtime := &orderedQueryRuntime{events: &events}
			boundary := &recordingQueryPublicationBoundary{events: &events, publishErr: errors.New("publish failed")}
			service := NewServiceWithRuntime(store, t.TempDir(), runtime).BindRuntimeQueryPublications(boundary)

			_, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true})
			if !errors.Is(err, ErrRuntimeFailed) || !slices.Equal(events, []string{
				"store.enable", "runtime.start", "query.publish", "runtime.stop", "store.disable",
			}) {
				t.Fatalf("publication failure = %v, order=%v", err, events)
			}
			if got := store.items[item.ID].Status; got != StatusDisabled {
				t.Fatalf("publication failure left store status %q", got)
			}
		})
	}
}

func TestLegacyQueryDisableStoreFailureRestoresExactPublicationBeforeAdmission(t *testing.T) {
	for _, test := range legacyQuerySurfaceCases() {
		t.Run(test.name, func(t *testing.T) {
			item := test.build(t, StatusEnabled)
			events := []string{}
			storeErr := errors.New("disable failed")
			store := &orderedQueryStore{
				fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
				events:             &events,
				disableErr:         storeErr,
			}
			runtime := &orderedQueryRuntime{events: &events}
			boundary := &recordingQueryPublicationBoundary{events: &events}
			service := NewServiceWithRuntime(store, t.TempDir(), runtime).BindRuntimeQueryPublications(boundary)

			_, err := service.Disable(t.Context(), extensionManager(), item.ID)
			if !errors.Is(err, storeErr) || !slices.Equal(events, []string{
				"query.quarantine", "store.disable", "query.rollback",
			}) {
				t.Fatalf("disable compensation = %v, order=%v", err, events)
			}
			if len(runtime.stopped) != 0 || store.items[item.ID].Status != StatusEnabled || boundary.rollbackCalls != 1 {
				t.Fatalf("disable compensation state: runtime=%#v store=%#v boundary=%#v", runtime, store.items[item.ID], boundary)
			}
		})
	}
}

func TestLegacyQueryBoundaryIsClosedOrSkippedForOtherPaths(t *testing.T) {
	t.Run("missing boundary", func(t *testing.T) {
		for _, test := range legacyQuerySurfaceCases() {
			t.Run(test.name, func(t *testing.T) {
				item := test.build(t, StatusInstalled)
				store := newFakeExtensionStore(map[string]Extension{item.ID: item})
				runtime := &fakeRuntimeManager{}
				service := NewServiceWithRuntime(store, t.TempDir(), runtime)
				if _, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true}); !errors.Is(err, ErrRuntimeQueryPublicationUnavailable) {
					t.Fatalf("missing query publication boundary = %v", err)
				}
				if store.enabledID != "" || len(runtime.started) != 0 {
					t.Fatalf("missing boundary reached side effects: store=%q runtime=%v", store.enabledID, runtime.started)
				}
			})
		}
	})

	t.Run("safe mode", func(t *testing.T) {
		item := legacyQueryServiceExtension(t, StatusInstalled)
		boundary := &recordingQueryPublicationBoundary{}
		service := NewServiceWithOptions(
			newFakeExtensionStore(map[string]Extension{item.ID: item}), t.TempDir(), "", &fakeRuntimeManager{},
			WithSafeMode(true),
		).BindRuntimeQueryPublications(boundary)
		if _, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true}); !errors.Is(err, ErrSafeModeActive) {
			t.Fatalf("Safe Mode query enable = %v", err)
		}
		if boundary.publishCalls != 0 || boundary.quarantineCalls != 0 {
			t.Fatalf("Safe Mode called query boundary = %#v", boundary)
		}
	})

	t.Run("queryless", func(t *testing.T) {
		item := legacyQueryServiceExtension(t, StatusInstalled)
		item.Manifest.Queries = nil
		item.Manifest.QueryResultFilters = nil
		refreshTrustPackageIdentity(t, &item)
		boundary := &recordingQueryPublicationBoundary{}
		service := NewServiceWithRuntime(
			newFakeExtensionStore(map[string]Extension{item.ID: item}), t.TempDir(), &fakeRuntimeManager{},
		).BindRuntimeQueryPublications(boundary)
		if _, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Disable(t.Context(), extensionManager(), item.ID); err != nil {
			t.Fatal(err)
		}
		if boundary.publishCalls != 0 || boundary.quarantineCalls != 0 {
			t.Fatalf("queryless plugin called query boundary = %#v", boundary)
		}
	})

	t.Run("lifecycle v2", func(t *testing.T) {
		item := completeV3TrustExtension(t, "demo.query")
		item.Manifest.Dependencies = nil
		item.Status = StatusDisabled
		item.ActiveVersionID = 41
		refreshTrustPackageIdentity(t, &item)
		boundary := &recordingQueryPublicationBoundary{}
		service := NewServiceWithRuntime(
			newFakeExtensionStore(map[string]Extension{item.ID: item}), t.TempDir(), &fakeRuntimeManager{},
		).BindRuntimeQueryPublications(boundary)
		_, _ = service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true})
		if boundary.publishCalls != 0 || boundary.quarantineCalls != 0 {
			t.Fatalf("Lifecycle V2 called legacy query boundary = %#v", boundary)
		}
	})
}

func legacyQueryServiceExtension(t *testing.T, status string) Extension {
	t.Helper()
	item := completeV3TrustExtension(t, "demo.query")
	item.Status = status
	item.ActiveVersionID = 41
	item.Manifest.Lifecycle = nil
	item.Manifest.Dependencies = nil
	refreshTrustPackageIdentity(t, &item)
	return item
}

func legacyFilterOnlyQueryServiceExtension(t *testing.T, status string) Extension {
	t.Helper()
	item := legacyQueryServiceExtension(t, status)
	item.Manifest.Queries = nil
	item.Manifest.QueryResultFilters = []ManifestQueryResultFilter{{
		ID: item.ID + ".query.items.decorate", ContractVersion: item.ID + ".query.items.decorate@1",
		QueryID: "demo.owner.query.items", QueryContractVersion: "demo.owner.query.items@1",
		QueryPlanVersion: "demo.owner.query.items.plan@1", Handler: item.ID + ".query.items.decorate",
		FailurePolicy: "fail_open", TimeoutMS: 500,
		Dependency: &ManifestQueryResultFilterDependency{
			ExtensionID: "demo.owner", VersionConstraint: "^1.0.0",
		},
	}}
	item.Manifest.Dependencies = []ManifestDependency{{
		ID: "demo.owner", Version: "^1.0.0", Kind: "optional",
	}}
	refreshTrustPackageIdentity(t, &item)
	return item
}

func legacyQuerySurfaceCases() []struct {
	name  string
	build func(*testing.T, string) Extension
} {
	return []struct {
		name  string
		build func(*testing.T, string) Extension
	}{
		{name: "query owner", build: legacyQueryServiceExtension},
		{name: "filter only", build: legacyFilterOnlyQueryServiceExtension},
	}
}

type recordingQueryPublicationMutation struct {
	boundary *recordingQueryPublicationBoundary
}

func (m *recordingQueryPublicationMutation) Rollback() error {
	m.boundary.rollbackCalls++
	if m.boundary.events != nil {
		*m.boundary.events = append(*m.boundary.events, "query.rollback")
	}
	return m.boundary.rollbackErr
}

type recordingQueryPublicationBoundary struct {
	events          *[]string
	publishCalls    int
	quarantineCalls int
	rollbackCalls   int
	publishErr      error
	quarantineErr   error
	rollbackErr     error
}

func (b *recordingQueryPublicationBoundary) PublishRuntimeQueries(context.Context, Extension) (RuntimeQueryPublicationMutation, error) {
	b.publishCalls++
	if b.events != nil {
		*b.events = append(*b.events, "query.publish")
	}
	if b.publishErr != nil {
		return nil, b.publishErr
	}
	return &recordingQueryPublicationMutation{boundary: b}, nil
}

func (b *recordingQueryPublicationBoundary) QuarantineRuntimeQueries(context.Context, Extension) (RuntimeQueryPublicationMutation, error) {
	b.quarantineCalls++
	if b.events != nil {
		*b.events = append(*b.events, "query.quarantine")
	}
	if b.quarantineErr != nil {
		return nil, b.quarantineErr
	}
	return &recordingQueryPublicationMutation{boundary: b}, nil
}

type orderedQueryStore struct {
	*fakeExtensionStore
	events     *[]string
	disableErr error
}

func (s *orderedQueryStore) Enable(ctx context.Context, id, extensionType string) (Extension, error) {
	*s.events = append(*s.events, "store.enable")
	return s.fakeExtensionStore.Enable(ctx, id, extensionType)
}

func (s *orderedQueryStore) Disable(ctx context.Context, id string) (Extension, error) {
	*s.events = append(*s.events, "store.disable")
	if s.disableErr != nil {
		return Extension{}, s.disableErr
	}
	return s.fakeExtensionStore.Disable(ctx, id)
}

type orderedQueryRuntime struct {
	fakeRuntimeManager
	events *[]string
}

func (r *orderedQueryRuntime) Start(ctx context.Context, extension Extension) error {
	*r.events = append(*r.events, "runtime.start")
	return r.fakeRuntimeManager.Start(ctx, extension)
}

func (r *orderedQueryRuntime) Stop(ctx context.Context, extension Extension) error {
	*r.events = append(*r.events, "runtime.stop")
	return r.fakeRuntimeManager.Stop(ctx, extension)
}
