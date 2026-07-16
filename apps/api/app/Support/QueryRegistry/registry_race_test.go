package queryregistry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestRegistryConcurrentReadersObserveOnlyCompleteSnapshots(t *testing.T) {
	first := concurrentPublicationSet('a')
	second := concurrentPublicationSet('b')
	registry := newPlanningRegistry()
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
			if _, err := registry.ReplaceAll(set, false); err != nil && !errors.Is(err, ErrRevisionConflict) {
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
				if len(snapshot.Queries) != 1 || len(snapshot.Publications) != 1 ||
					snapshot.Revision == 0 || snapshot.Digest == "" ||
					snapshot.Queries[0].ID == "" || snapshot.Publications[0].Artifact.PackageDigest == "" {
					report(fmt.Errorf("reader %d partial snapshot: %#v", reader, snapshot))
					return
				}
				// Plan 可能观察到更新的完整快照；只要求计划自身自洽。
				plan, err := registry.Plan(context.Background(), PlanRequest{
					QueryID: "core.race.items",
					Permission: PermissionInput{
						Authenticated: true, Recheck: allowAll(),
					},
				})
				if errors.Is(err, ErrArtifactConflict) {
					// Writer swapped the immutable graph during plan authorization.
					continue
				}
				if err != nil || plan.CacheKey == "" || plan.Digest == "" ||
					plan.Query.ID != "core.race.items" || plan.Revision == 0 ||
					len(plan.Fields) == 0 || len(plan.Providers) == 0 {
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

func TestRegistryConcurrentStaleRemoveAndPublishAreFenced(t *testing.T) {
	registry := New()
	initial := publication("race.query", false, 'a')
	initial.Queries = []QueryDeclaration{query("race.query.items", "race.item", PaginationNone, "public")}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	replacement := publication("race.query", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.PackageDigest = strings.Repeat("b", 64)
	replacement.Queries = []QueryDeclaration{query("race.query.items", "race.item", PaginationNone, "public")}
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

func concurrentPublicationSet(digest byte) []Publication {
	item := publication("core.race", true, digest)
	item.Queries = []QueryDeclaration{query("core.race.items", "core.race.item", PaginationOffset, "public")}
	item.Queries[0].CacheTags = []string{fmt.Sprintf("core.race.tag.%c", digest)}
	return []Publication{item}
}
