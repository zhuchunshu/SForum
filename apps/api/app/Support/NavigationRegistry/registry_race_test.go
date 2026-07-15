package navigationregistry

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestRegistryConcurrentReadersObserveOnlyCompleteSnapshots(t *testing.T) {
	first := concurrentPublicationSet('a', "First")
	second := concurrentPublicationSet('b', "Second")
	registry := New()
	if _, err := registry.ReplaceAll(first); err != nil {
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
		for index := 0; index < 200; index++ {
			set := first
			if index%2 == 1 {
				set = second
			}
			if _, err := registry.ReplaceAll(set); err != nil {
				report(fmt.Errorf("replace %d: %w", index, err))
				return
			}
		}
	}()
	for reader := 0; reader < 4; reader++ {
		go func(reader int) {
			defer group.Done()
			<-start
			for index := 0; index < 500; index++ {
				snapshot := registry.Snapshot()
				if len(snapshot.Navigation) != 2 || len(snapshot.Publications) != 1 ||
					snapshot.Revision == 0 || snapshot.Digest == "" {
					report(fmt.Errorf("reader %d partial snapshot: %#v", reader, snapshot))
					return
				}
				resolution, err := registry.ResolveNavigation(NavigationResolveRequest{})
				if err != nil || len(resolution.Targets) != 2 || resolution.CacheKey == "" {
					report(fmt.Errorf("reader %d partial resolution: %#v err=%v", reader, resolution, err))
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

func TestRegistryConcurrentStaleRemoveAndPublishAreFenced(t *testing.T) {
	registry := New()
	initial := publication("race.navigation", false, 'a')
	initial.Navigation = []NavigationDeclaration{
		navigation("race.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	replacement := publication("race.navigation", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.PackageDigest = strings.Repeat("b", 64)
	replacement.Navigation = []NavigationDeclaration{
		navigation("race.navigation.item.next", NavigationKindItem, ActionAdd, "", 0),
	}
	if _, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()

	start := make(chan struct{})
	results := make(chan error, 3)
	// 只并发陈旧路径；成功路径单独验证，避免 remove 清空后 Publish 被误计为成功。
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
		_, err := registry.PublishIfArtifact(initial.Artifact, initial)
		results <- err
	}()
	close(start)

	for range 3 {
		err := <-results
		if !errors.Is(err, ErrArtifactConflict) {
			t.Fatalf("stale path should conflict, got %v", err)
		}
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("stale concurrent paths changed snapshot: before=%#v after=%#v", before, after)
	}
	if revision, removed, err := registry.Remove(replacement.Artifact); err != nil || !removed || revision != before.Revision+1 {
		t.Fatalf("exact active remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
}

func concurrentPublicationSet(digest rune, label string) []Publication {
	core := publication("core.concurrent", true, digest)
	core.Navigation = []NavigationDeclaration{
		navigation("core.concurrent.item.one", NavigationKindItem, ActionAdd, "", 10),
		navigation("core.concurrent.item.two", NavigationKindItem, ActionAdd, "", 20),
	}
	core.Navigation[0].Label = label + " one"
	core.Navigation[1].Label = label + " two"
	return []Publication{core}
}
