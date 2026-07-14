package extensions

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGuardPolicyCatalogPublishesExactArtifactsWithoutLookupIO(t *testing.T) {
	plugin := guardPolicyFixture("guard.plugin", TypePlugin, strings.Repeat("a", 64))
	plugin.Manifest.Backend.Entry = "bin/plugin"
	plugin.Manifest.Backend.ProtocolVersion = 2
	plugin.Manifest.Lifecycle = &ManifestLifecycle{ContractVersion: "sforum.lifecycle@2"}
	plugin.Source, plugin.IsSystem, plugin.IsDeletable = SourceUploaded, false, true
	plugin.StagedVersion = &ExtensionVersion{
		Version: "2.0.0", Manifest: plugin.Manifest, PackageDigest: strings.Repeat("b", 64),
	}
	plugin.StagedVersion.Manifest.Version = "2.0.0"
	source := &guardPolicySourceStub{items: []Extension{plugin}}
	trust := &guardPolicyTrustStub{trusted: map[string]bool{
		plugin.PackageDigest: false, plugin.StagedVersion.PackageDigest: true,
	}}
	catalog := NewGuardPolicyCatalog(source, trust, nil, GuardPolicyConfig{
		TrustChallengesEnabled: true, TTL: time.Minute,
	})
	if _, ok := catalog.Lookup(plugin.ID); ok {
		t.Fatal("unpublished catalog must be unavailable")
	}
	if err := catalog.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	lookup, ok := catalog.Lookup(plugin.ID)
	if !ok || !lookup.Found || lookup.Revision == 0 {
		t.Fatalf("lookup = %#v, ok=%v", lookup, ok)
	}
	entry := lookup.Entry
	if entry.ExtensionType != TypePlugin || entry.Version != "1.0.0" ||
		entry.PackageDigest != plugin.PackageDigest || entry.CurrentArtifactTrusted ||
		!entry.HasStagedArtifact || entry.StagedVersion != "2.0.0" ||
		entry.StagedPackageDigest != plugin.StagedVersion.PackageDigest || !entry.StagedArtifactTrusted {
		t.Fatalf("entry = %#v", entry)
	}
	for range 100 {
		if _, ok := catalog.Lookup(plugin.ID); !ok {
			t.Fatal("published catalog disappeared")
		}
	}
	if source.calls != 1 || trust.calls != 2 {
		t.Fatalf("lookup performed I/O: source=%d trust=%d", source.calls, trust.calls)
	}
}

func TestGuardPolicyCatalogRejectsDriftAndEventuallyExpires(t *testing.T) {
	plugin := guardPolicyFixture("guard.plugin", TypePlugin, strings.Repeat("a", 64))
	source := &guardPolicySourceStub{items: []Extension{plugin}}
	catalog := NewGuardPolicyCatalog(source, &guardPolicyTrustStub{}, nil, GuardPolicyConfig{TTL: 60 * time.Millisecond})
	if err := catalog.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	before, _ := catalog.Lookup(plugin.ID)

	drifted := plugin
	drifted.Manifest.Type = TypeTheme
	source.set([]Extension{drifted}, nil)
	if err := catalog.Refresh(context.Background()); err == nil {
		t.Fatal("manifest identity drift was accepted")
	}
	after, ok := catalog.Lookup(plugin.ID)
	if !ok || after.Revision != before.Revision {
		t.Fatalf("failed refresh replaced valid snapshot: before=%#v after=%#v", before, after)
	}

	time.Sleep(75 * time.Millisecond)
	if _, ok := catalog.Lookup(plugin.ID); ok {
		t.Fatal("failed refresh kept stale catalog indefinitely")
	}
}

func TestGuardPolicyCatalogRefreshFailurePreservesThenCloses(t *testing.T) {
	plugin := guardPolicyFixture("guard.plugin", TypePlugin, strings.Repeat("a", 64))
	source := &guardPolicySourceStub{items: []Extension{plugin}}
	catalog := NewGuardPolicyCatalog(source, &guardPolicyTrustStub{}, nil, GuardPolicyConfig{TTL: 80 * time.Millisecond})
	if err := catalog.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	source.set(nil, errors.New("database unavailable"))
	if err := catalog.Refresh(context.Background()); err == nil {
		t.Fatal("refresh failure was hidden")
	}
	if _, ok := catalog.Lookup(plugin.ID); !ok {
		t.Fatal("one failed refresh discarded a still-valid snapshot")
	}
	time.Sleep(75 * time.Millisecond)
	if _, ok := catalog.Lookup(plugin.ID); ok {
		t.Fatal("catalog remained available after refresh failures exceeded TTL")
	}
}

func TestGuardPolicyCatalogConcurrentRefreshAndLookup(t *testing.T) {
	plugin := guardPolicyFixture("guard.plugin", TypePlugin, strings.Repeat("a", 64))
	source := &guardPolicySourceStub{items: []Extension{plugin}}
	catalog := NewGuardPolicyCatalog(source, &guardPolicyTrustStub{}, nil, GuardPolicyConfig{TTL: time.Minute})
	if err := catalog.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 200 {
				lookup, ok := catalog.Lookup(plugin.ID)
				if !ok || !lookup.Found || lookup.Entry.PackageDigest != plugin.PackageDigest {
					t.Errorf("lookup = %#v, ok=%v", lookup, ok)
					return
				}
			}
		}()
	}
	for range 20 {
		if err := catalog.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	workers.Wait()
}

func guardPolicyFixture(id, extensionType, digest string) Extension {
	manifest := Manifest{ID: id, Name: id, Version: "1.0.0", Type: extensionType}
	return Extension{
		ID: id, Name: id, Version: "1.0.0", Type: extensionType,
		Status: StatusEnabled, Source: SourceBuiltin, IsSystem: true,
		Manifest: manifest, PackageDigest: digest,
	}
}

type guardPolicySourceStub struct {
	mu    sync.Mutex
	items []Extension
	err   error
	calls int
}

func (s *guardPolicySourceStub) List(context.Context) ([]Extension, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return append([]Extension(nil), s.items...), s.err
}

func (s *guardPolicySourceStub) set(items []Extension, err error) {
	s.mu.Lock()
	s.items, s.err = items, err
	s.mu.Unlock()
}

type guardPolicyTrustStub struct {
	mu      sync.Mutex
	trusted map[string]bool
	calls   int
}

func (s *guardPolicyTrustStub) TrustedArtifact(_ context.Context, extension Extension) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.trusted[extension.PackageDigest], nil
}
