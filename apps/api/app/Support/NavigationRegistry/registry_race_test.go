package navigationregistry

import (
	"fmt"
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
				if len(snapshot.Navigation) != 2 || snapshot.Revision == 0 || snapshot.Digest == "" {
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
