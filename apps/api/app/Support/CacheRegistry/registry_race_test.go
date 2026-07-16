package cacheregistry

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestRegistryConcurrentReadersObserveCompleteSnapshots(t *testing.T) {
	first := raceCorePublication('a')
	second := raceCorePublication('b')
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{first}, false); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errorsCh := make(chan error, 1)
	report := func(err error) {
		select {
		case errorsCh <- err:
		default:
		}
	}
	var group sync.WaitGroup
	group.Add(5)
	go func() {
		defer group.Done()
		<-start
		for index := 0; index < 300; index++ {
			publication := first
			if index%2 == 1 {
				publication = second
			}
			if _, err := registry.ReplaceAll([]Publication{publication}, false); err != nil && !errors.Is(err, ErrRevisionConflict) {
				report(fmt.Errorf("replace %d: %w", index, err))
				return
			}
		}
	}()
	for reader := 0; reader < 4; reader++ {
		go func(reader int) {
			defer group.Done()
			<-start
			for index := 0; index < 600; index++ {
				snapshot := registry.Snapshot()
				if snapshot.Revision == 0 || snapshot.Digest == "" || len(snapshot.Publications) != 1 || len(snapshot.Caches) != 1 ||
					snapshot.Caches[0].Artifact.PackageDigest == "" || snapshot.Caches[0].Namespace == "" {
					report(fmt.Errorf("reader %d partial snapshot: %#v", reader, snapshot))
					return
				}
				plan, err := registry.Plan(PlanRequest{CacheID: "core.race.items"})
				if errors.Is(err, ErrArtifactConflict) || errors.Is(err, ErrPlanStale) {
					continue
				}
				if err != nil || plan.Revision == 0 || plan.Digest == "" || plan.Cache.ID == "" || plan.Isolation.SegmentDigest == "" {
					report(fmt.Errorf("reader %d partial plan: %#v err=%v", reader, plan, err))
					return
				}
			}
		}(reader)
	}
	close(start)
	group.Wait()
	select {
	case err := <-errorsCh:
		t.Fatal(err)
	default:
	}
}

func TestRegistryConcurrentPublishAndRemove(t *testing.T) {
	registry := New()
	core := raceCorePublication('a')
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}

	const workers = 32
	start := make(chan struct{})
	errorsCh := make(chan error, workers)
	var group sync.WaitGroup
	for index := 0; index < workers; index++ {
		publication := testPublication(fmt.Sprintf("race.cache.%02d", index), false, byte('a'+index%6))
		publication.Caches = []Declaration{testDeclaration(publication.Artifact.ExtensionID+".items", PolicyPrivate)}
		group.Add(1)
		go func(publication Publication) {
			defer group.Done()
			<-start
			if _, err := registry.Publish(publication); err != nil {
				errorsCh <- err
				return
			}
			if _, removed, err := registry.Remove(publication.Artifact); err != nil || !removed {
				if err == nil {
					err = fmt.Errorf("publication %s was not removed", publication.Artifact.ExtensionID)
				}
				errorsCh <- err
			}
		}(publication)
	}
	close(start)
	group.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Error(err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Publications) != 1 || len(snapshot.Caches) != 1 || snapshot.Publications[0].Artifact != core.Artifact {
		t.Fatalf("concurrent publish/remove left partial graph = %#v", snapshot)
	}
}

func TestRegistryConcurrentStaleMutationsAreFenced(t *testing.T) {
	registry := New()
	initial := testPublication("stale.cache", false, 'a')
	initial.Caches = []Declaration{testDeclaration("stale.cache.items", PolicyPublic)}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	replacement := testPublication("stale.cache", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-stale-cache-v2"
	replacement.Caches = []Declaration{testDeclaration("stale.cache.items", PolicyPublic)}
	if _, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()

	start := make(chan struct{})
	results := make(chan error, 3)
	go func() {
		<-start
		_, _, err := registry.Remove(initial.Artifact)
		results <- err
	}()
	go func() {
		<-start
		_, err := registry.Publish(initial)
		results <- err
	}()
	go func() {
		<-start
		_, err := registry.PublishIfArtifact(initial.Artifact, replacement)
		results <- err
	}()
	close(start)
	for index := 0; index < 3; index++ {
		if err := <-results; !errors.Is(err, ErrArtifactConflict) {
			t.Fatalf("stale concurrent mutation = %v", err)
		}
	}
	after := registry.Snapshot()
	if after.Revision != before.Revision || after.Digest != before.Digest || after.Publications[0].Artifact != replacement.Artifact {
		t.Fatalf("stale mutation changed graph: before=%#v after=%#v", before, after)
	}
}

func raceCorePublication(digest byte) Publication {
	publication := testPublication("core.race", true, digest)
	declaration := testDeclaration("core.race.items", PolicyPublic)
	declaration.Tags = []string{fmt.Sprintf("core.race.tag.%c", digest)}
	publication.Caches = []Declaration{declaration}
	return publication
}
