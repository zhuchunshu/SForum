package contentregistry

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestRegistryConcurrentReadersObserveOnlyCompleteSnapshots(t *testing.T) {
	first := concurrentPublicationSet('a')
	second := concurrentPublicationSet('b')
	registry := New()
	if _, err := registry.ReplaceAll(first, false); err != nil {
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
			observedRevision := registry.Revision()
			if _, err := registry.ReplaceAllIfRevision(observedRevision, set, false); err != nil && !errors.Is(err, ErrRevisionConflict) {
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
				if len(snapshot.Content) != 2 || len(snapshot.Publications) != 1 ||
					snapshot.Revision == 0 || snapshot.Digest == "" ||
					snapshot.Content[0].ID == "" || snapshot.Publications[0].Artifact.PackageDigest == "" {
					report(fmt.Errorf("reader %d partial snapshot: %#v", reader, snapshot))
					return
				}
				// Readers may observe either complete graph; mixed kinds/ids mean partial swap.
				if snapshot.Content[0].Artifact.PackageDigest != snapshot.Content[1].Artifact.PackageDigest {
					report(fmt.Errorf("reader %d mixed artifact snapshot: %#v", reader, snapshot.Content))
					return
				}
				list := registry.List(KindBlock)
				if len(list) != 1 || list[0].Kind != KindBlock || list[0].ID == "" {
					report(fmt.Errorf("reader %d partial list: %#v", reader, list))
					return
				}
				resolved, err := registry.Resolve("race.graph.block.card")
				if err != nil || resolved.Kind != KindBlock || resolved.Artifact.PackageDigest == "" {
					report(fmt.Errorf("reader %d partial resolve: %#v err=%v", reader, resolved, err))
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
	initial := publication("race.content", false, 'a')
	initial.Content = []Declaration{content("race.content.block.a", KindBlock, "h", "race.content.block.a.schema@1")}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	replacement := publication("race.content", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.PackageDigest = strings.Repeat("b", 64)
	replacement.Artifact.RuntimeInstanceID = "runtime-race-v2"
	replacement.Content = []Declaration{content("race.content.block.a", KindBlock, "h", "race.content.block.a.schema@1")}
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
		// 陈旧 expected artifact：当前 active 已是 replacement。
		stale := replacement
		stale.Artifact.PackageDigest = strings.Repeat("f", 64)
		_, err := registry.PublishIfArtifact(initial.Artifact, stale)
		results <- err
	}()
	close(start)
	for index := 0; index < 3; index++ {
		if err := <-results; err == nil {
			t.Fatal("expected stale concurrent mutation to fail closed")
		}
	}
	after := registry.Snapshot()
	if after.Revision != before.Revision || after.Digest != before.Digest {
		t.Fatalf("stale mutations changed snapshot: before=%#v after=%#v", before, after)
	}
}

func TestRegistryConcurrentExactWritersHaveOneWinner(t *testing.T) {
	registry := New()
	source := publication("winner.content", false, 'a')
	source.Content = []Declaration{content("winner.content.block.a", KindBlock, "h", "winner.content.block.a.schema@1")}
	if _, err := registry.Publish(source); err != nil {
		t.Fatal(err)
	}
	left := publication("winner.content", false, 'b')
	left.Artifact.ExtensionVersion = "2.0.0"
	left.Artifact.VersionID = 2
	left.Artifact.RuntimeInstanceID = "runtime-left"
	left.Content = []Declaration{content("winner.content.block.a", KindBlock, "left", "winner.content.block.a.schema@1")}
	left.Content[0].ContractVersion = "winner.content.block.a@2"
	right := publication("winner.content", false, 'c')
	right.Artifact.ExtensionVersion = "3.0.0"
	right.Artifact.VersionID = 3
	right.Artifact.RuntimeInstanceID = "runtime-right"
	right.Content = []Declaration{content("winner.content.block.a", KindBlock, "right", "winner.content.block.a.schema@1")}
	right.Content[0].ContractVersion = "winner.content.block.a@3"

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

func concurrentPublicationSet(digest byte) []Publication {
	item := publication("race.graph", false, digest)
	item.Content = []Declaration{
		content("race.graph.block.card", KindBlock, "content.block", "race.graph.block.card.schema@1"),
		content("race.graph.mark.bold", KindMark, "content.mark", "race.graph.mark.bold.schema@1"),
	}
	item.Content[0].Renderer = fmt.Sprintf("race.graph.template.%c", digest)
	if digest == 'b' {
		item.Artifact.ExtensionVersion = "2.0.0"
		item.Artifact.VersionID = 2
		item.Artifact.RuntimeInstanceID = "runtime-race-graph-v2"
		item.Content[0].ContractVersion = "race.graph.block.card@2"
	}
	return []Publication{item}
}
