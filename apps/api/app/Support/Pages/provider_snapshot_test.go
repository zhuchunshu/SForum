package pages

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestProviderSnapshotRestoresOnceAndHotPathsAvoidBindingReads(t *testing.T) {
	ctx := t.Context()
	store := newCountingBindingStore(ProviderBinding{
		PageID: "forum.home", ExtensionID: "snapshot.theme", ContributionID: "snapshot.home",
		Version: "1.0.0", PackageDigest: "snapshot-digest", ContractVersion: "sforum.page.home@1", ApprovedBy: 1,
	})
	registry := NewRegistry(store)
	if err := registry.RegisterContributions("snapshot.theme", []PageContribution{
		{
			ID: "snapshot.home", Action: ActionReplace, Target: "forum.home", Template: "templates/home.html",
			Contract: "sforum.page.home@1", Version: "1.0.0", PackageDigest: "snapshot-digest",
		},
		{
			ID: "snapshot.docs", Action: ActionAdd, Path: "/snapshot/:slug",
			Version: "1.0.0", PackageDigest: "snapshot-digest",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if resolved, err := registry.Resolve(ctx, "forum.home"); err != nil || resolved.Provider != ProviderCore {
		t.Fatalf("binding became visible before boot restore: %#v, %v", resolved, err)
	}
	if err := registry.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	if err := registry.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}

	for range 100 {
		resolved, err := registry.Resolve(ctx, "forum.home")
		if err != nil || resolved.Provider != "snapshot.theme" {
			t.Fatalf("resolve = %#v, %v", resolved, err)
		}
		if match, ok := registry.ResolveAddedPathMatch("/snapshot/one"); !ok || match.Contribution.ID != "snapshot.docs" {
			t.Fatalf("add match = %#v, %t", match, ok)
		}
	}
	if _, err := registry.ListProviders(ctx); err != nil {
		t.Fatal(err)
	}
	if store.listCalls.Load() != 1 || store.getCalls.Load() != 0 {
		t.Fatalf("store reads: list=%d get=%d", store.listCalls.Load(), store.getCalls.Load())
	}

	// The published map owns cloned strings and is unaffected by later Store state.
	store.replaceBinding(ProviderBinding{
		PageID: "forum.home", ExtensionID: "snapshot.theme", ContributionID: "snapshot.home",
		Version: "1.0.0", PackageDigest: "stale-digest", ContractVersion: "sforum.page.home@1",
	})
	resolved, err := registry.Resolve(ctx, "forum.home")
	if err != nil || resolved.Provider != "snapshot.theme" {
		t.Fatalf("hot path observed unpublished store mutation: %#v, %v", resolved, err)
	}
}

func TestProviderSnapshotPublishesOnlyAfterDurableApproveAndRestore(t *testing.T) {
	ctx := t.Context()
	store := newCountingBindingStore()
	registry := NewRegistry(store)
	if err := registry.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterContributions("snapshot.theme", []PageContribution{{
		ID: "snapshot.home", Action: ActionReplace, Target: "forum.home", Template: "templates/home.html",
		Contract: "sforum.page.home@1", Version: "1.0.0", PackageDigest: "snapshot-digest",
	}}); err != nil {
		t.Fatal(err)
	}
	binding := ProviderBinding{
		PageID: "forum.home", ExtensionID: "snapshot.theme", ContributionID: "snapshot.home",
		Version: "1.0.0", PackageDigest: "snapshot-digest", ContractVersion: "sforum.page.home@1", ApprovedBy: 7,
	}

	store.setWriteErrors(errors.New("upsert failed"), nil)
	revision := registry.Revision()
	if err := registry.ApproveReplace(ctx, binding); err == nil {
		t.Fatal("expected durable upsert failure")
	}
	if registry.Revision() != revision || mustResolveProvider(t, registry, ctx) != ProviderCore {
		t.Fatal("failed approval changed published snapshot")
	}

	store.setWriteErrors(nil, nil)
	if err := registry.ApproveReplace(ctx, binding); err != nil {
		t.Fatal(err)
	}
	if mustResolveProvider(t, registry, ctx) != binding.ExtensionID {
		t.Fatal("approved provider was not published")
	}

	store.setWriteErrors(nil, errors.New("delete failed"))
	revision = registry.Revision()
	if err := registry.RestoreCore(ctx, binding.PageID); err == nil {
		t.Fatal("expected durable delete failure")
	}
	if registry.Revision() != revision || mustResolveProvider(t, registry, ctx) != binding.ExtensionID {
		t.Fatal("failed restore changed published snapshot")
	}

	store.setWriteErrors(nil, nil)
	if err := registry.RestoreCore(ctx, binding.PageID); err != nil {
		t.Fatal(err)
	}
	if mustResolveProvider(t, registry, ctx) != ProviderCore {
		t.Fatal("core restore was not published")
	}
	if store.getCalls.Load() != 0 || store.upsertCalls.Load() != 2 || store.deleteCalls.Load() != 2 {
		t.Fatalf("store calls: get=%d upsert=%d delete=%d", store.getCalls.Load(), store.upsertCalls.Load(), store.deleteCalls.Load())
	}
}

func TestProviderSnapshotBootRestoreFailureIsStickyAndUnpublished(t *testing.T) {
	store := newCountingBindingStore()
	store.listErr = errors.New("list failed")
	registry := NewRegistry(store)
	for range 2 {
		if err := registry.RestoreBindings(t.Context()); !errors.Is(err, store.listErr) {
			t.Fatalf("restore error = %v", err)
		}
	}
	if store.listCalls.Load() != 1 || registry.Revision() != 0 {
		t.Fatalf("failed boot restore list=%d revision=%d", store.listCalls.Load(), registry.Revision())
	}
}

func TestProviderSnapshotConcurrentApprovalAndLifecyclePublication(t *testing.T) {
	ctx := context.Background()
	store := newCountingBindingStore()
	registry := NewRegistry(store)
	if err := registry.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	contribution := PageContribution{
		ID: "snapshot.home", Action: ActionReplace, Target: "forum.home", Contract: "sforum.page.home@1",
		Version: "1.0.0", PackageDigest: "snapshot-digest",
	}
	if err := registry.RegisterContributions("snapshot.theme", []PageContribution{contribution}); err != nil {
		t.Fatal(err)
	}
	binding := ProviderBinding{
		PageID: "forum.home", ExtensionID: "snapshot.theme", ContributionID: "snapshot.home",
		Version: "1.0.0", PackageDigest: "snapshot-digest", ContractVersion: "sforum.page.home@1", ApprovedBy: 9,
	}

	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for range 500 {
				resolved, err := registry.Resolve(ctx, "forum.home")
				if err != nil || resolved.Provider != ProviderCore && resolved.Provider != binding.ExtensionID {
					t.Errorf("incoherent resolve = %#v, %v", resolved, err)
					return
				}
			}
		}()
	}
	for range 50 {
		if err := registry.ApproveReplace(ctx, binding); err != nil {
			t.Fatal(err)
		}
		registry.ClearExtension(binding.ExtensionID)
		if err := registry.RegisterContributions(binding.ExtensionID, []PageContribution{contribution}); err != nil {
			t.Fatal(err)
		}
		if err := registry.RestoreCore(ctx, binding.PageID); err != nil {
			t.Fatal(err)
		}
	}
	readers.Wait()
	if store.getCalls.Load() != 0 {
		t.Fatalf("concurrent hot path called GetBinding %d times", store.getCalls.Load())
	}
}

func TestThemeActivationReplacesBindingsAndCleansStaleOwnersAcrossRestart(t *testing.T) {
	ctx := t.Context()
	store := newCountingBindingStore(
		ProviderBinding{PageID: "forum.home", ExtensionID: "old.home-theme", ContributionID: "old.home", ContractVersion: "sforum.page.home@1"},
		ProviderBinding{PageID: "forum.topic.show", ExtensionID: "old.topic-theme", ContributionID: "old.topic", ContractVersion: "sforum.page.topic_show@1"},
	)
	registry := NewRegistry(store)
	if err := registry.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	contribution := PageContribution{
		ID: "new.theme.home", Action: ActionReplace, Target: "forum.home", Template: "templates/home.html",
		Contract: "sforum.page.home@1", Version: "2.0.0", PackageDigest: "new-digest",
	}
	if err := registry.ActivateThemeContributionsReplacing(
		ctx, "new.theme", []PageContribution{contribution}, 42, "old.home-theme", "old.topic-theme",
	); err != nil {
		t.Fatal(err)
	}
	resolved, err := registry.Resolve(ctx, "forum.home")
	if err != nil || resolved.Provider != "new.theme" {
		t.Fatalf("new provider=%#v err=%v", resolved, err)
	}
	if topic, err := registry.Resolve(ctx, "forum.topic.show"); err != nil || topic.Provider != ProviderCore {
		t.Fatalf("stale topic binding survived: %#v err=%v", topic, err)
	}

	restarted := NewRegistry(store)
	if err := restarted.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	if err := restarted.RestoreThemeContributions(ctx, "new.theme", []PageContribution{contribution}, []string{"old.home-theme", "old.topic-theme"}); err != nil {
		t.Fatal(err)
	}
	resolved, err = restarted.Resolve(ctx, "forum.home")
	if err != nil || resolved.Provider != "new.theme" {
		t.Fatalf("restart provider=%#v err=%v", resolved, err)
	}
	stored, ok, err := store.GetBinding(ctx, "forum.home")
	if err != nil || !ok || stored.ApprovedBy != 42 || stored.PackageDigest != "new-digest" {
		t.Fatalf("stored binding=%#v ok=%v err=%v", stored, ok, err)
	}
	if _, ok, _ := store.GetBinding(ctx, "forum.topic.show"); ok {
		t.Fatal("stale binding remained durable")
	}
}

func TestThemeActivationRequiresActorApproval(t *testing.T) {
	registry := NewRegistry(NewMemoryStore())
	err := registry.ActivateThemeContributions(t.Context(), "theme.demo", []PageContribution{{
		ID: "theme.demo.home", Action: ActionReplace, Target: "forum.home",
		Contract: "sforum.page.home@1", Version: "1.0.0", PackageDigest: "digest",
	}}, "", 0)
	if !errors.Is(err, ErrApprovalRequired) || mustResolveProvider(t, registry, t.Context()) != ProviderCore {
		t.Fatalf("err=%v provider=%s", err, mustResolveProvider(t, registry, t.Context()))
	}
}

func TestUnapprovedThemeSwitchCleansOldBindingImmediatelyAndOnRestart(t *testing.T) {
	ctx := t.Context()
	store := NewMemoryStore()
	oldBinding := ProviderBinding{
		PageID: "forum.home", ExtensionID: "old.theme", ContributionID: "old.home",
		Version: "1.0.0", PackageDigest: "old-digest", ContractVersion: "sforum.page.home@1", ApprovedBy: 9,
	}
	if err := store.UpsertBinding(ctx, oldBinding); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry(store)
	if err := registry.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterContributions("old.theme", []PageContribution{{
		ID: "old.home", Action: ActionReplace, Target: "forum.home", Contract: "sforum.page.home@1",
		Version: "1.0.0", PackageDigest: "old-digest",
	}}); err != nil {
		t.Fatal(err)
	}
	newContribution := PageContribution{
		ID: "default.home", Action: ActionReplace, Target: "forum.home", Contract: "sforum.page.home@1",
		Version: "2.0.0", PackageDigest: "default-digest",
	}
	if err := registry.SwitchThemeContributions(ctx, "default.theme", []PageContribution{newContribution}, "old.theme"); err != nil {
		t.Fatal(err)
	}
	if provider := mustResolveProvider(t, registry, ctx); provider != ProviderCore {
		t.Fatalf("unapproved replacement became active: %s", provider)
	}
	if _, ok, _ := store.GetBinding(ctx, "forum.home"); ok {
		t.Fatal("old approval remained durable after theme switch")
	}
	restarted := NewRegistry(store)
	if err := restarted.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	if err := restarted.RegisterContributions("default.theme", []PageContribution{newContribution}); err != nil {
		t.Fatal(err)
	}
	if provider := mustResolveProvider(t, restarted, ctx); provider != ProviderCore {
		t.Fatalf("stale approval returned after restart: %s", provider)
	}
}

func TestThemeActivationStoreFailureLeavesPublishedRevisionUntouched(t *testing.T) {
	ctx := t.Context()
	store := newCountingBindingStore(ProviderBinding{
		PageID: "forum.home", ExtensionID: "old.theme", ContributionID: "old.home",
		Version: "1.0.0", PackageDigest: "old-digest", ContractVersion: "sforum.page.home@1",
	})
	registry := NewRegistry(store)
	if err := registry.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	old := PageContribution{
		ID: "old.home", Action: ActionReplace, Target: "forum.home", Contract: "sforum.page.home@1",
		Version: "1.0.0", PackageDigest: "old-digest",
	}
	if err := registry.RegisterContributions("old.theme", []PageContribution{old}); err != nil {
		t.Fatal(err)
	}
	revision := registry.Revision()
	store.setWriteErrors(errors.New("replace failed"), nil)
	err := registry.ActivateThemeContributions(ctx, "new.theme", []PageContribution{{
		ID: "new.home", Action: ActionReplace, Target: "forum.home", Contract: "sforum.page.home@1",
		Version: "2.0.0", PackageDigest: "new-digest",
	}}, "old.theme", 7)
	if err == nil || registry.Revision() != revision || mustResolveProvider(t, registry, ctx) != "old.theme" {
		t.Fatalf("err=%v revision=%d provider=%s", err, registry.Revision(), mustResolveProvider(t, registry, ctx))
	}
}

func mustResolveProvider(t *testing.T, registry *Registry, ctx context.Context) string {
	t.Helper()
	resolved, err := registry.Resolve(ctx, "forum.home")
	if err != nil {
		t.Fatal(err)
	}
	return resolved.Provider
}

type countingBindingStore struct {
	mu        sync.RWMutex
	bindings  map[string]ProviderBinding
	listErr   error
	upsertErr error
	deleteErr error

	listCalls   atomic.Int64
	getCalls    atomic.Int64
	upsertCalls atomic.Int64
	deleteCalls atomic.Int64
}

func newCountingBindingStore(bindings ...ProviderBinding) *countingBindingStore {
	store := &countingBindingStore{bindings: make(map[string]ProviderBinding, len(bindings))}
	for _, binding := range bindings {
		store.bindings[binding.PageID] = cloneBinding(binding)
	}
	return store
}

func (s *countingBindingStore) ListBindings(context.Context) ([]ProviderBinding, error) {
	s.listCalls.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	result := make([]ProviderBinding, 0, len(s.bindings))
	for _, binding := range s.bindings {
		result = append(result, cloneBinding(binding))
	}
	return result, nil
}

func (s *countingBindingStore) GetBinding(_ context.Context, pageID string) (ProviderBinding, bool, error) {
	s.getCalls.Add(1)
	s.mu.RLock()
	defer s.mu.RUnlock()
	binding, ok := s.bindings[pageID]
	return cloneBinding(binding), ok, nil
}

func (s *countingBindingStore) UpsertBinding(_ context.Context, binding ProviderBinding) error {
	s.upsertCalls.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.bindings[binding.PageID] = cloneBinding(binding)
	return nil
}

func (s *countingBindingStore) DeleteBinding(_ context.Context, pageID string) error {
	s.deleteCalls.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.bindings, pageID)
	return nil
}

func (s *countingBindingStore) ReplaceExtensionBindings(_ context.Context, extensionIDs []string, bindings []ProviderBinding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.upsertErr != nil {
		return s.upsertErr
	}
	owners := map[string]struct{}{}
	for _, id := range extensionIDs {
		owners[id] = struct{}{}
	}
	for pageID, binding := range s.bindings {
		if _, ok := owners[binding.ExtensionID]; ok {
			delete(s.bindings, pageID)
		}
	}
	for _, binding := range bindings {
		s.bindings[binding.PageID] = cloneBinding(binding)
	}
	return nil
}

func (s *countingBindingStore) ReconcileExtensionBindings(_ context.Context, activeExtensionID string, staleExtensionIDs []string, allowed []ProviderBinding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.upsertErr != nil {
		return s.upsertErr
	}
	stale := map[string]struct{}{}
	for _, id := range staleExtensionIDs {
		stale[id] = struct{}{}
	}
	allowedByPage := map[string]ProviderBinding{}
	for _, binding := range allowed {
		allowedByPage[binding.PageID] = binding
	}
	for pageID, binding := range s.bindings {
		if _, ok := stale[binding.ExtensionID]; ok && binding.ExtensionID != activeExtensionID {
			delete(s.bindings, pageID)
			continue
		}
		if binding.ExtensionID == activeExtensionID {
			exact, ok := allowedByPage[pageID]
			if !ok || !sameProviderArtifact(binding, exact) {
				delete(s.bindings, pageID)
			}
		}
	}
	return nil
}

func (s *countingBindingStore) replaceBinding(binding ProviderBinding) {
	s.mu.Lock()
	s.bindings[binding.PageID] = cloneBinding(binding)
	s.mu.Unlock()
}

func (s *countingBindingStore) setWriteErrors(upsertErr, deleteErr error) {
	s.mu.Lock()
	s.upsertErr = upsertErr
	s.deleteErr = deleteErr
	s.mu.Unlock()
}
