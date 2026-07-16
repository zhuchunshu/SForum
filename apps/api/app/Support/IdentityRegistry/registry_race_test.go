package identityregistry

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestRegistrySnapshotsRemainAtomicDuringExactUpgrades(t *testing.T) {
	registry := New()
	active := testPublication(1)
	if _, err := registry.Publish(active); err != nil {
		t.Fatal(err)
	}

	var readers sync.WaitGroup
	stop := make(chan struct{})
	for index := 0; index < 8; index++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				snapshot := registry.Snapshot()
				if snapshot.Digest == "" || len(snapshot.Publications) != 1 || len(snapshot.Permissions) != 1 || len(snapshot.Providers) != 1 {
					t.Errorf("partial snapshot: %#v", snapshot)
					return
				}
				if snapshot.Publications[0].Artifact != snapshot.Permissions[0].Artifact ||
					snapshot.Publications[0].Artifact != snapshot.Providers[0].Artifact {
					t.Errorf("mixed artifact snapshot: %#v", snapshot)
					return
				}
			}
		}()
	}
	for version := 2; version <= 20; version++ {
		target := testPublication(version % 9)
		target.Artifact.ExtensionVersion = fmt.Sprintf("%d.0.0", version)
		target.Artifact.VersionID = int64(version)
		target.Artifact.RuntimeInstanceID = fmt.Sprintf("runtime-%d", version)
		target.Artifact.PackageDigest = fmt.Sprintf("%064x", version)
		if _, err := registry.PublishIfArtifact(active.Artifact, target); err != nil {
			close(stop)
			readers.Wait()
			t.Fatalf("upgrade %d: %v", version, err)
		}
		active = target
	}
	close(stop)
	readers.Wait()
}

func TestRegistryConcurrentExactWritersHaveOneWinner(t *testing.T) {
	registry := New()
	source := testPublication(1)
	if _, err := registry.Publish(source); err != nil {
		t.Fatal(err)
	}
	left := testPublication(2)
	right := testPublication(3)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, target := range []Publication{left, right} {
		target := target
		go func() {
			<-start
			_, err := registry.PublishIfArtifact(source.Artifact, target)
			results <- err
		}()
	}
	close(start)
	var succeeded, conflicted int
	for index := 0; index < 2; index++ {
		switch err := <-results; {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrArtifactConflict):
			conflicted++
		default:
			t.Fatalf("writer error = %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 || registry.Revision() != 2 {
		t.Fatalf("succeeded=%d conflicted=%d revision=%d", succeeded, conflicted, registry.Revision())
	}
}
