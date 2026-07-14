package extensions

import (
	"context"
	"errors"
	"testing"
	"time"
)

type lifecycleProviderSelectionStore struct {
	*fakeExtensionStore
	selectedMail string
	restoreCalls int
	selectErr    error
	restoreErr   error
}

func (s *lifecycleProviderSelectionStore) SelectedMailProvider(context.Context) (string, error) {
	return s.selectedMail, s.selectErr
}

func (s *lifecycleProviderSelectionStore) RestoreMailProvider(context.Context) error {
	s.restoreCalls++
	if s.restoreErr != nil {
		return s.restoreErr
	}
	s.selectedMail = ""
	return nil
}

type lifecycleStorageSelectionClearer struct {
	calls     []string
	failCalls int
	err       error
}

func (c *lifecycleStorageSelectionClearer) ClearStorageProviderSelectionIfMatch(_ context.Context, extensionID string) error {
	c.calls = append(c.calls, extensionID)
	if len(c.calls) <= c.failCalls {
		return c.err
	}
	return nil
}

func TestServiceLifecycleV2DisableClearsProviderSelectionsWithoutLegacyRuntimeStop(t *testing.T) {
	item := lifecycleV2ServiceArtifact(t, "service.provider-disable", "1.0.0", SourceBuiltin)
	item.Status = StatusEnabled
	base := newFakeExtensionStore(map[string]Extension{item.ID: item})
	store := &lifecycleProviderSelectionStore{fakeExtensionStore: base, selectedMail: item.ID}
	storage := &lifecycleStorageSelectionClearer{}
	runtime := &fakeRuntimeManager{}
	runner := &lifecycleV2RecordingRunner{operationID: 801, beforeRun: func(LifecycleCoordinatorRunInput) {
		published := store.items[item.ID]
		published.Status = StatusDisabled
		store.items[item.ID] = published
	}}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", runtime,
		WithAuditor(&lifecycleV2AuditWriter{nextID: 900}),
		WithStorageSelectionClearer(storage),
		WithLifecycleCoordinator(
			runner,
			func(context.Context, LifecycleMachineOperation, *Extension, Extension) error { return nil },
			lifecycleV2AuthorityStore{authority: lifecycleAuthorityTestGrant(t, item)},
		),
	)

	disabled, err := service.DisableWithInput(
		t.Context(), extensionManager(), item.ID,
		LifecycleRequestInput{IdempotencyKey: "provider-disable"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != StatusDisabled || store.selectedMail != "" || store.restoreCalls != 1 ||
		len(storage.calls) != 1 || storage.calls[0] != item.ID {
		t.Fatalf("disabled=%#v mail=%q restore=%d storage=%#v", disabled, store.selectedMail, store.restoreCalls, storage.calls)
	}
	if len(runtime.stopped) != 0 {
		t.Fatalf("V2 service used legacy runtime stop: %#v", runtime.stopped)
	}
}

func TestServiceLifecycleV2DisableReplayRetriesProviderCleanup(t *testing.T) {
	item := lifecycleV2ServiceArtifact(t, "service.provider-disable-replay", "1.0.0", SourceBuiltin)
	item.Status = StatusEnabled
	base := newFakeExtensionStore(map[string]Extension{item.ID: item})
	store := &lifecycleProviderSelectionStore{fakeExtensionStore: base, selectedMail: item.ID}
	cleanupFailure := errors.New("attachment option unavailable")
	storage := &lifecycleStorageSelectionClearer{failCalls: 1, err: cleanupFailure}
	runner := &lifecycleV2RecordingRunner{operationID: 802, beforeRun: func(LifecycleCoordinatorRunInput) {
		published := store.items[item.ID]
		published.Status = StatusDisabled
		store.items[item.ID] = published
	}}
	actor := extensionManager()
	service := NewServiceWithOptions(
		store, t.TempDir(), "", &fakeRuntimeManager{},
		WithAuditor(&lifecycleV2AuditWriter{nextID: 910}), WithStorageSelectionClearer(storage),
		WithLifecycleCoordinator(
			runner,
			func(context.Context, LifecycleMachineOperation, *Extension, Extension) error { return nil },
			lifecycleV2AuthorityStore{authority: lifecycleAuthorityTestGrant(t, item)},
		),
	)
	if _, err := service.DisableWithInput(
		t.Context(), actor, item.ID, LifecycleRequestInput{IdempotencyKey: "provider-disable-replay"},
	); !errors.Is(err, cleanupFailure) {
		t.Fatalf("first cleanup error = %v", err)
	}

	completedAt := time.Now().UTC()
	operation := LifecycleOperation{
		ID: 802, ExtensionID: item.ID, ExtensionVersion: item.Version, PackageDigest: item.PackageDigest,
		Operation: string(LifecycleMachineDisable), IdempotencyKey: "provider-disable-replay",
		RequestedByUserID: actor.ID, TerminalResult: LifecycleTerminalSucceeded, CompletedAt: &completedAt,
	}
	replayRunner := &lifecycleV2RecordingRunner{}
	replay := NewServiceWithOptions(
		store, t.TempDir(), "", &fakeRuntimeManager{},
		WithAuditor(&lifecycleV2AuditWriter{nextID: 920}), WithStorageSelectionClearer(storage),
		WithLifecycleCoordinator(
			replayRunner,
			func(context.Context, LifecycleMachineOperation, *Extension, Extension) error {
				t.Fatal("completed replay ran preflight")
				return nil
			},
			lifecycleV2AuthorityStore{authority: lifecycleAuthorityTestGrant(t, item), operation: &operation},
		),
	)
	disabled, err := replay.DisableWithInput(
		t.Context(), actor, item.ID, LifecycleRequestInput{IdempotencyKey: operation.IdempotencyKey},
	)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != StatusDisabled || len(storage.calls) != 2 || replayRunner.calls != 0 ||
		store.restoreCalls != 1 || store.selectedMail != "" {
		t.Fatalf("replay disabled=%#v storage=%#v runner=%d restore=%d mail=%q",
			disabled, storage.calls, replayRunner.calls, store.restoreCalls, store.selectedMail)
	}
}

func TestServiceLifecycleV2UninstallClearsProvidersBeforePhysicalFinalizer(t *testing.T) {
	item := lifecycleV2ServiceArtifact(t, "service.provider-uninstall", "1.0.0", SourceUploaded)
	item.Status = StatusDisabled
	base := newFakeExtensionStore(map[string]Extension{item.ID: item})
	store := &lifecycleProviderSelectionStore{fakeExtensionStore: base, selectedMail: item.ID}
	storage := &lifecycleStorageSelectionClearer{}
	runner := &lifecycleV2RecordingRunner{operationID: 803}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", &fakeRuntimeManager{},
		WithAuditor(&lifecycleV2AuditWriter{nextID: 930}), WithStorageSelectionClearer(storage),
		WithLifecycleCoordinator(
			runner,
			func(context.Context, LifecycleMachineOperation, *Extension, Extension) error { return nil },
			lifecycleV2AuthorityStore{authority: lifecycleAuthorityTestGrant(t, item)},
		),
		WithLifecycleCleanupFinalizer(func(_ context.Context, operationID int64) (LifecycleCleanupFinalization, error) {
			if store.selectedMail != "" || len(storage.calls) != 1 {
				t.Fatalf("physical finalizer ran before provider cleanup: mail=%q storage=%#v", store.selectedMail, storage.calls)
			}
			delete(store.items, item.ID)
			return LifecycleCleanupFinalization{OperationID: operationID, Status: "finalized", PhysicalPurgeComplete: true}, nil
		}),
	)

	result, err := service.UninstallWithResult(t.Context(), extensionManager(), item.ID, UninstallInput{
		IdempotencyKey: "provider-uninstall", RemovalMode: LifecycleRemovalPreserve,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Uninstalled || store.restoreCalls != 1 || len(storage.calls) != 1 {
		t.Fatalf("uninstall=%#v restore=%d storage=%#v", result, store.restoreCalls, storage.calls)
	}
}
