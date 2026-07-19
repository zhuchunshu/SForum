package identityregistry

import (
	"errors"
	"strings"
	"testing"
)

func TestProviderResolutionIsImmutableAndIgnoresUnrelatedRevisionDrift(t *testing.T) {
	registry := New()
	publication := testPublication(1)
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	resolution, err := registry.ResolveProviderSnapshot("fixture.identity.provider.risk")
	if err != nil {
		t.Fatal(err)
	}

	artifact, err := NewCoreArtifact("core.identity.unrelated", "1.0.0", strings.Repeat("f", 64))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Publish(Publication{Artifact: artifact}); err != nil {
		t.Fatal(err)
	}
	if err := registry.ValidateProviderResolution(resolution); err != nil {
		t.Fatalf("unrelated publication invalidated exact provider: %v", err)
	}

	resolution.Provider.ID = "mutated"
	current, err := registry.ResolveProvider("fixture.identity.provider.risk")
	if err != nil || current.ID != "fixture.identity.provider.risk" {
		t.Fatalf("resolution aliased Registry state: provider=%#v err=%v", current, err)
	}
}

func TestProviderResolutionRejectsSafeModeAndSameIDArtifactReplacement(t *testing.T) {
	t.Run("safe mode", func(t *testing.T) {
		registry := New()
		publication := testPublication(1)
		if _, err := registry.Publish(publication); err != nil {
			t.Fatal(err)
		}
		resolution, err := registry.ResolveProviderSnapshot("fixture.identity.provider.risk")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := registry.ReplaceAllIfRevision(
			registry.Revision(), []Publication{publication}, registry.Snapshot().Tombstones, true,
		); err != nil {
			t.Fatal(err)
		}
		if err := registry.ValidateProviderResolution(resolution); !errors.Is(err, ErrSafeMode) {
			t.Fatalf("safe mode validation err=%v", err)
		}
		if _, err := registry.ResolveProviderSnapshot(resolution.Provider.ID); !errors.Is(err, ErrSafeMode) {
			t.Fatalf("safe mode resolve err=%v", err)
		}
	})

	t.Run("same id replacement", func(t *testing.T) {
		registry := New()
		source := testPublication(1)
		if _, err := registry.Publish(source); err != nil {
			t.Fatal(err)
		}
		resolution, err := registry.ResolveProviderSnapshot("fixture.identity.provider.risk")
		if err != nil {
			t.Fatal(err)
		}
		if _, removed, err := registry.Remove(source.Artifact); err != nil || !removed {
			t.Fatalf("remove source removed=%t err=%v", removed, err)
		}
		replacement := testPublication(2)
		if _, err := registry.Publish(replacement); err != nil {
			t.Fatal(err)
		}
		if err := registry.ValidateProviderResolution(resolution); !errors.Is(err, ErrArtifactConflict) {
			t.Fatalf("same-id replacement validation err=%v", err)
		}
	})
}
