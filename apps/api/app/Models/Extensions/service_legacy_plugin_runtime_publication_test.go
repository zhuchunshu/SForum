package extensions

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestLegacyServiceUsesAtomicPluginRuntimePublicationStore(t *testing.T) {
	item := legacyPluginRuntimeServiceFixture(t, StatusInstalled)
	events := []string{}
	base := &orderedQueryStore{
		fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
		events:             &events,
	}
	store := &recordingLegacyPluginRuntimeStore{orderedQueryStore: base, events: &events}
	runtime := &orderedQueryRuntime{events: &events}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime)

	enabled, err := service.Enable(
		t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if enabled.Status != StatusEnabled || !slices.Equal(events, []string{
		"desired.enable", "store.enable", "runtime.start",
	}) {
		t.Fatalf("legacy desired enable order=%v enabled=%+v", events, enabled)
	}

	events = events[:0]
	disabled, err := service.Disable(t.Context(), extensionManager(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != StatusDisabled || !slices.Equal(events, []string{
		"runtime.stop", "desired.disable", "store.disable",
	}) {
		t.Fatalf("legacy desired disable order=%v disabled=%+v", events, disabled)
	}
	if !slices.Equal(store.reasons, []PluginRuntimePublicationReason{
		PluginRuntimePublicationEnable, PluginRuntimePublicationDisable,
	}) || !slices.Equal(store.actors, []int64{extensionManager().ID, extensionManager().ID}) {
		t.Fatalf("legacy desired evidence reasons=%v actors=%v", store.reasons, store.actors)
	}
}

func TestLegacyServiceRuntimeFailurePublishesCompensatingDisable(t *testing.T) {
	item := legacyPluginRuntimeServiceFixture(t, StatusInstalled)
	events := []string{}
	base := &orderedQueryStore{
		fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
		events:             &events,
	}
	store := &recordingLegacyPluginRuntimeStore{orderedQueryStore: base, events: &events}
	runtime := &orderedQueryRuntime{events: &events}
	runtime.startErr = errors.New("start failed")
	service := NewServiceWithRuntime(store, t.TempDir(), runtime)

	_, err := service.Enable(
		t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true},
	)
	if !errors.Is(err, ErrRuntimeFailed) || !slices.Equal(events, []string{
		"desired.enable", "store.enable", "runtime.start", "desired.disable", "store.disable",
	}) {
		t.Fatalf("legacy runtime compensation err=%v order=%v", err, events)
	}
	if got := store.items[item.ID].Status; got != StatusDisabled {
		t.Fatalf("legacy runtime compensation status=%q", got)
	}
	if !slices.Equal(store.reasons, []PluginRuntimePublicationReason{
		PluginRuntimePublicationEnable, PluginRuntimePublicationDisable,
	}) {
		t.Fatalf("legacy runtime compensation reasons=%v", store.reasons)
	}
}

func TestLegacyServiceReportsCompensatingDesiredDisableFailure(t *testing.T) {
	item := legacyPluginRuntimeServiceFixture(t, StatusInstalled)
	events := []string{}
	base := &orderedQueryStore{
		fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
		events:             &events,
	}
	disableErr := errors.New("desired compensation failed")
	store := &recordingLegacyPluginRuntimeStore{
		orderedQueryStore: base, events: &events, disableErr: disableErr,
	}
	runtime := &orderedQueryRuntime{events: &events}
	runtime.startErr = errors.New("start failed")
	service := NewServiceWithRuntime(store, t.TempDir(), runtime)

	_, err := service.Enable(
		t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true},
	)
	if !errors.Is(err, ErrRuntimeFailed) || !errors.Is(err, disableErr) || !slices.Equal(events, []string{
		"desired.enable", "store.enable", "runtime.start", "desired.disable",
	}) {
		t.Fatalf("legacy desired compensation failure err=%v order=%v", err, events)
	}
	if got := store.items[item.ID].Status; got != StatusEnabled {
		t.Fatalf("failed desired compensation guessed disabled state=%q", got)
	}
}

func TestLegacyServiceDesiredDisableFailureRestoresQuarantinedQuery(t *testing.T) {
	item := legacyQueryServiceExtension(t, StatusEnabled)
	events := []string{}
	base := &orderedQueryStore{
		fakeExtensionStore: newFakeExtensionStore(map[string]Extension{item.ID: item}),
		events:             &events,
	}
	store := &recordingLegacyPluginRuntimeStore{
		orderedQueryStore: base,
		events:            &events,
		disableErr:        errors.New("desired disable failed"),
	}
	runtime := &orderedQueryRuntime{events: &events}
	queries := &recordingQueryPublicationBoundary{events: &events}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime).BindRuntimeQueryPublications(queries)

	_, err := service.Disable(t.Context(), extensionManager(), item.ID)
	if !errors.Is(err, store.disableErr) || !slices.Equal(events, []string{
		"query.quarantine", "desired.disable", "query.rollback",
	}) {
		t.Fatalf("legacy desired disable failure err=%v order=%v", err, events)
	}
	if store.items[item.ID].Status != StatusEnabled || len(runtime.stopped) != 0 || queries.rollbackCalls != 1 {
		t.Fatalf("legacy desired disable compensation store=%+v runtime=%+v queries=%+v",
			store.items[item.ID], runtime, queries)
	}
}

func TestLegacyServiceStoreWithoutPublicationBoundaryPreservesV1(t *testing.T) {
	item := legacyPluginRuntimeServiceFixture(t, StatusInstalled)
	store := newFakeExtensionStore(map[string]Extension{item.ID: item})
	service := NewServiceWithRuntime(store, t.TempDir(), &fakeRuntimeManager{})

	if _, err := service.Enable(
		t.Context(), extensionManager(), item.ID, EnableInput{ConfirmCapabilities: true},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Disable(t.Context(), extensionManager(), item.ID); err != nil {
		t.Fatal(err)
	}
	if store.enabledID != item.ID || store.disabledID != item.ID {
		t.Fatalf("V1 store calls enabled=%q disabled=%q", store.enabledID, store.disabledID)
	}
}

func legacyPluginRuntimeServiceFixture(t *testing.T, status string) Extension {
	t.Helper()
	item := legacyQueryServiceExtension(t, status)
	item.Manifest.Queries = nil
	item.Manifest.Cache = nil
	refreshTrustPackageIdentity(t, &item)
	return item
}

type recordingLegacyPluginRuntimeStore struct {
	*orderedQueryStore
	events     *[]string
	reasons    []PluginRuntimePublicationReason
	actors     []int64
	enableErr  error
	disableErr error
}

func (s *recordingLegacyPluginRuntimeStore) EnableLegacyPluginRuntime(
	ctx context.Context,
	target Extension,
	actorUserID int64,
) (Extension, PluginRuntimePublication, error) {
	*s.events = append(*s.events, "desired.enable")
	if s.enableErr != nil {
		return Extension{}, PluginRuntimePublication{}, s.enableErr
	}
	enabled, err := s.orderedQueryStore.Enable(ctx, target.ID, target.Type)
	if err != nil {
		return Extension{}, PluginRuntimePublication{}, err
	}
	s.reasons = append(s.reasons, PluginRuntimePublicationEnable)
	s.actors = append(s.actors, actorUserID)
	return enabled, PluginRuntimePublication{Reason: PluginRuntimePublicationEnable}, nil
}

func (s *recordingLegacyPluginRuntimeStore) DisableLegacyPluginRuntime(
	ctx context.Context,
	target Extension,
	actorUserID int64,
) (Extension, PluginRuntimePublication, error) {
	*s.events = append(*s.events, "desired.disable")
	if s.disableErr != nil {
		return Extension{}, PluginRuntimePublication{}, s.disableErr
	}
	disabled, err := s.orderedQueryStore.Disable(ctx, target.ID)
	if err != nil {
		return Extension{}, PluginRuntimePublication{}, err
	}
	s.reasons = append(s.reasons, PluginRuntimePublicationDisable)
	s.actors = append(s.actors, actorUserID)
	return disabled, PluginRuntimePublication{Reason: PluginRuntimePublicationDisable}, nil
}

var _ LegacyPluginRuntimePublicationStore = (*recordingLegacyPluginRuntimeStore)(nil)
