package identityregistry

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
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

func TestRegistrySessionPolicyLeaseSerializesAuthorityWriters(t *testing.T) {
	registry := New()
	publication := testPublication(1)
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseLease := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseLease)
	leaseResult := make(chan error, 1)
	go func() {
		leaseResult <- registry.RunWithSessionPolicyLease("core.session.default", func(claim SessionPolicyLeaseClaim) error {
			if claim.Revision != 1 || claim.SafeMode || claim.Provider != nil {
				return fmt.Errorf("unexpected lease claim: %#v", claim)
			}
			close(entered)
			<-release
			return nil
		})
	}()
	awaitRegistryLeaseSignal(t, entered, "session policy lease")

	writerStarted := make(chan struct{})
	writerResult := make(chan error, 1)
	go func() {
		close(writerStarted)
		_, err := registry.ReplaceAllIfRevision(
			1,
			[]Publication{publication},
			registry.Snapshot().Tombstones,
			true,
		)
		writerResult <- err
	}()
	awaitRegistryLeaseSignal(t, writerStarted, "Registry writer")
	deadline := time.Now().Add(5 * time.Second)
	for registry.mu.TryRLock() {
		registry.mu.RUnlock()
		if time.Now().After(deadline) {
			t.Fatal("Registry writer did not queue before deadline")
		}
		runtime.Gosched()
	}
	select {
	case err := <-writerResult:
		t.Fatalf("Registry writer crossed active read lease: %v", err)
	default:
	}

	releaseLease()
	if err := awaitRegistryLeaseResult(t, leaseResult, "session policy lease"); err != nil {
		t.Fatal(err)
	}
	if err := awaitRegistryLeaseResult(t, writerResult, "Registry writer"); err != nil {
		t.Fatal(err)
	}
	if !registry.Snapshot().SafeMode {
		t.Fatal("Safe Mode writer did not publish after read lease returned")
	}
}

func TestRegistrySessionPolicyLeaseReleasesAfterErrorAndPanic(t *testing.T) {
	registry := New()
	publication := testPublication(1)
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("Host effect failed")
	if err := registry.RunWithSessionPolicyLease("core.session.default", func(SessionPolicyLeaseClaim) error { return wantErr }); !errors.Is(err, wantErr) {
		t.Fatalf("lease error = %v", err)
	}
	if !registry.mu.TryLock() {
		t.Fatal("lease read lock remained held after callback error")
	}
	registry.mu.Unlock()

	panicValue := "Host effect panic"
	func() {
		defer func() {
			if recovered := recover(); recovered != panicValue {
				t.Fatalf("recovered panic = %#v", recovered)
			}
		}()
		_ = registry.RunWithSessionPolicyLease("core.session.default", func(SessionPolicyLeaseClaim) error { panic(panicValue) })
	}()
	if !registry.mu.TryLock() {
		t.Fatal("lease read lock remained held after callback panic")
	}
	registry.mu.Unlock()

	if _, removed, err := registry.Remove(publication.Artifact); err != nil || !removed {
		t.Fatalf("writer after lease exits removed=%t err=%v", removed, err)
	}
	if err := (*Registry)(nil).RunWithSessionPolicyLease("core.session.default", func(SessionPolicyLeaseClaim) error { return nil }); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil Registry error = %v", err)
	}
	if err := registry.RunWithSessionPolicyLease("core.session.default", nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil callback error = %v", err)
	}
}

func awaitRegistryLeaseSignal(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}

func awaitRegistryLeaseResult(t *testing.T, result <-chan error, label string) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s result", label)
		return nil
	}
}
