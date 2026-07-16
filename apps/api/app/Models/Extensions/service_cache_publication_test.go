package extensions

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestLegacyCacheEnableAndDisableUseHostPublicationBoundary(t *testing.T) {
	item := legacyCacheServiceExtension(t, StatusInstalled)
	events := []string{}
	store := &orderedQueryStore{
		fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
		events:             &events,
	}
	runtime := &orderedQueryRuntime{events: &events}
	boundary := &recordingCachePublicationBoundary{events: &events}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", runtime, WithRuntimeCachePublications(boundary),
	)

	enabled, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true})
	if err != nil {
		t.Fatal(err)
	}
	if enabled.Status != StatusEnabled || !slices.Equal(events, []string{
		"store.enable", "runtime.start", "cache.publish",
	}) {
		t.Fatalf("legacy cache enable order = %v, enabled=%#v", events, enabled)
	}

	events = events[:0]
	disabled, err := service.Disable(t.Context(), extensionManager(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != StatusDisabled || !slices.Equal(events, []string{
		"cache.quarantine", "store.disable", "runtime.stop",
	}) {
		t.Fatalf("legacy cache disable order = %v, disabled=%#v", events, disabled)
	}
	if boundary.publishCalls != 1 || boundary.quarantineCalls != 1 || boundary.rollbackCalls != 0 {
		t.Fatalf("cache boundary calls = %#v", boundary)
	}
}

func TestLegacyCacheEnablePublicationFailureCompensatesQueryFirstRuntime(t *testing.T) {
	item := legacyQueryAndCacheServiceExtension(t, StatusInstalled)
	events := []string{}
	store := &orderedQueryStore{
		fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
		events:             &events,
	}
	runtime := &orderedQueryRuntime{events: &events}
	queries := &recordingQueryPublicationBoundary{events: &events}
	caches := &recordingCachePublicationBoundary{events: &events, publishErr: errors.New("publish failed")}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", runtime, WithRuntimeCachePublications(caches),
	).BindRuntimeQueryPublications(queries)

	_, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true})
	if !errors.Is(err, ErrRuntimeFailed) || !slices.Equal(events, []string{
		"store.enable", "runtime.start", "query.publish", "cache.publish",
		"cache.rollback", "query.rollback", "runtime.stop", "store.disable",
	}) {
		t.Fatalf("cache publication failure = %v, order=%v", err, events)
	}
	if got := store.items[item.ID].Status; got != StatusDisabled {
		t.Fatalf("cache publication failure left store status %q", got)
	}
}

func TestLegacyCacheDisableStoreFailureRestoresBeforeQueryAdmission(t *testing.T) {
	item := legacyQueryAndCacheServiceExtension(t, StatusEnabled)
	events := []string{}
	storeErr := errors.New("disable failed")
	store := &orderedQueryStore{
		fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
		events:             &events,
		disableErr:         storeErr,
	}
	runtime := &orderedQueryRuntime{events: &events}
	queries := &recordingQueryPublicationBoundary{events: &events}
	caches := &recordingCachePublicationBoundary{events: &events}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", runtime, WithRuntimeCachePublications(caches),
	).BindRuntimeQueryPublications(queries)

	_, err := service.Disable(t.Context(), extensionManager(), item.ID)
	if !errors.Is(err, storeErr) || !slices.Equal(events, []string{
		"query.quarantine", "cache.quarantine", "store.disable", "cache.rollback", "query.rollback",
	}) {
		t.Fatalf("cache disable compensation = %v, order=%v", err, events)
	}
	if len(runtime.stopped) != 0 || store.items[item.ID].Status != StatusEnabled ||
		caches.rollbackCalls != 1 || queries.rollbackCalls != 1 {
		t.Fatalf("cache disable compensation state: runtime=%#v store=%#v caches=%#v queries=%#v",
			runtime, store.items[item.ID], caches, queries)
	}
}

func TestLegacyCacheQuarantineFailureKeepsQueryAdmissionClosed(t *testing.T) {
	item := legacyQueryAndCacheServiceExtension(t, StatusEnabled)
	events := []string{}
	store := &orderedQueryStore{
		fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
		events:             &events,
	}
	runtime := &orderedQueryRuntime{events: &events}
	queries := &recordingQueryPublicationBoundary{events: &events}
	caches := &recordingCachePublicationBoundary{events: &events, quarantineErr: errors.New("quarantine failed")}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", runtime, WithRuntimeCachePublications(caches),
	).BindRuntimeQueryPublications(queries)

	_, err := service.Disable(t.Context(), extensionManager(), item.ID)
	if !errors.Is(err, caches.quarantineErr) || !slices.Equal(events, []string{
		"query.quarantine", "cache.quarantine",
	}) {
		t.Fatalf("cache quarantine failure = %v, order=%v", err, events)
	}
	if queries.rollbackCalls != 0 || len(runtime.stopped) != 0 || store.items[item.ID].Status != StatusEnabled {
		t.Fatalf("cache quarantine failure reopened or stopped runtime: runtime=%#v store=%#v queries=%#v",
			runtime, store.items[item.ID], queries)
	}
}

func TestLegacyCacheBoundaryNilCompatibilityAndPathSelection(t *testing.T) {
	t.Run("missing boundary preserves legacy behavior", func(t *testing.T) {
		item := legacyCacheServiceExtension(t, StatusInstalled)
		store := newFakeExtensionStore(map[string]Extension{item.ID: item})
		runtime := &fakeRuntimeManager{}
		service := NewServiceWithRuntime(store, t.TempDir(), runtime)
		if _, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Disable(t.Context(), extensionManager(), item.ID); err != nil {
			t.Fatal(err)
		}
		if len(runtime.started) != 1 || len(runtime.stopped) != 1 {
			t.Fatalf("nil cache boundary changed legacy runtime behavior: %#v", runtime)
		}
	})

	t.Run("safe mode", func(t *testing.T) {
		item := legacyCacheServiceExtension(t, StatusInstalled)
		boundary := &recordingCachePublicationBoundary{}
		service := NewServiceWithOptions(
			newFakeExtensionStore(map[string]Extension{item.ID: item}), t.TempDir(), "", &fakeRuntimeManager{},
			WithSafeMode(true), WithRuntimeCachePublications(boundary),
		)
		if _, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true}); !errors.Is(err, ErrSafeModeActive) {
			t.Fatalf("Safe Mode cache enable = %v", err)
		}
		if boundary.publishCalls != 0 || boundary.quarantineCalls != 0 {
			t.Fatalf("Safe Mode called cache boundary = %#v", boundary)
		}
	})

	t.Run("cacheless", func(t *testing.T) {
		item := legacyCacheServiceExtension(t, StatusInstalled)
		item.Manifest.Cache = nil
		refreshTrustPackageIdentity(t, &item)
		boundary := &recordingCachePublicationBoundary{}
		service := NewServiceWithOptions(
			newFakeExtensionStore(map[string]Extension{item.ID: item}), t.TempDir(), "", &fakeRuntimeManager{},
			WithRuntimeCachePublications(boundary),
		)
		if _, err := service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Disable(t.Context(), extensionManager(), item.ID); err != nil {
			t.Fatal(err)
		}
		if boundary.publishCalls != 0 || boundary.quarantineCalls != 0 {
			t.Fatalf("cacheless plugin called cache boundary = %#v", boundary)
		}
	})

	t.Run("lifecycle v2", func(t *testing.T) {
		item := completeV3TrustExtension(t, "demo.cache.lifecycle")
		item.Manifest.Dependencies = nil
		item.Status = StatusDisabled
		item.ActiveVersionID = 41
		refreshTrustPackageIdentity(t, &item)
		boundary := &recordingCachePublicationBoundary{}
		service := NewServiceWithOptions(
			newFakeExtensionStore(map[string]Extension{item.ID: item}), t.TempDir(), "", &fakeRuntimeManager{},
			WithRuntimeCachePublications(boundary),
		)
		_, _ = service.Enable(t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true})
		if boundary.publishCalls != 0 || boundary.quarantineCalls != 0 {
			t.Fatalf("Lifecycle V2 called legacy cache boundary = %#v", boundary)
		}
	})
}

func legacyCacheServiceExtension(t *testing.T, status string) Extension {
	t.Helper()
	item := completeV3TrustExtension(t, "demo.cache")
	item.Status = status
	item.ActiveVersionID = 41
	item.Manifest.Lifecycle = nil
	item.Manifest.Dependencies = nil
	item.Manifest.Queries = nil
	item.Manifest.Cache = []ManifestCache{{
		ID: item.ID + ".cache.results", ContractVersion: item.ID + ".cache.results@1",
		Namespace: item.ID + ".results", Policy: "actor", Tags: []string{item.ID + ".cache.tag"},
		Provider: "core.cache.redis", Invalidators: []string{item.ID + ".cache.invalidate"},
	}}
	refreshTrustPackageIdentity(t, &item)
	return item
}

func legacyQueryAndCacheServiceExtension(t *testing.T, status string) Extension {
	t.Helper()
	item := completeV3TrustExtension(t, "demo.query-cache")
	item.Status = status
	item.ActiveVersionID = 41
	item.Manifest.Lifecycle = nil
	item.Manifest.Dependencies = nil
	refreshTrustPackageIdentity(t, &item)
	return item
}

type recordingCachePublicationMutation struct {
	boundary *recordingCachePublicationBoundary
}

func (m *recordingCachePublicationMutation) Rollback() error {
	m.boundary.rollbackCalls++
	if m.boundary.events != nil {
		*m.boundary.events = append(*m.boundary.events, "cache.rollback")
	}
	return m.boundary.rollbackErr
}

type recordingCachePublicationBoundary struct {
	events          *[]string
	publishCalls    int
	quarantineCalls int
	rollbackCalls   int
	publishErr      error
	quarantineErr   error
	rollbackErr     error
}

func (b *recordingCachePublicationBoundary) PublishRuntimeCaches(
	context.Context,
	Extension,
) (RuntimeCachePublicationMutation, error) {
	b.publishCalls++
	if b.events != nil {
		*b.events = append(*b.events, "cache.publish")
	}
	mutation := &recordingCachePublicationMutation{boundary: b}
	if b.publishErr != nil {
		return mutation, b.publishErr
	}
	return mutation, nil
}

func (b *recordingCachePublicationBoundary) QuarantineRuntimeCaches(
	context.Context,
	Extension,
) (RuntimeCachePublicationMutation, error) {
	b.quarantineCalls++
	if b.events != nil {
		*b.events = append(*b.events, "cache.quarantine")
	}
	if b.quarantineErr != nil {
		return nil, b.quarantineErr
	}
	return &recordingCachePublicationMutation{boundary: b}, nil
}
