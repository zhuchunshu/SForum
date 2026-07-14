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

func (s *countingBindingStore) GetBinding(context.Context, string) (ProviderBinding, bool, error) {
	s.getCalls.Add(1)
	return ProviderBinding{}, false, nil
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
