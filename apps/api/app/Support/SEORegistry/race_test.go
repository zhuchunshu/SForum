package seoregistry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentSnapshotsAndExecutionObserveOnlyExactCompleteGraphs(t *testing.T) {
	first := testPublication("plugin.race", 'a')
	firstDeclaration := testDeclaration(first, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0)
	first.Contributions = []Declaration{firstDeclaration}
	second := testPublication("plugin.race", 'b')
	second.Artifact.ExtensionVersion = "2.0.0"
	second.Artifact.VersionID = 2
	second.Artifact.RuntimeInstanceID = "runtime-plugin-race-v2"
	secondDeclaration := testDeclaration(second, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0)
	second.Contributions = []Declaration{secondDeclaration}
	registry := New()
	if _, err := registry.Publish(first); err != nil {
		t.Fatal(err)
	}
	firstContribution := Contribution{Declaration: firstDeclaration, Artifact: first.Artifact}
	secondContribution := Contribution{Declaration: secondDeclaration, Artifact: second.Artifact}
	provider := func(title string) ProviderFunc {
		return func(_ context.Context, request ProviderRequest) (ProviderResult, error) {
			request.Current.Title = title
			return ProviderResult{Document: request.Current}, nil
		}
	}
	runtime := mustRuntime(t, registry, newTestAdmission(), []ProviderBinding{
		testBinding(first, firstContribution.Declaration, provider("first")),
		testBinding(second, secondContribution.Declaration, provider("second")),
	}, nil)

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
			publication := first
			if index%2 == 0 {
				publication = second
			}
			if _, err := registry.ReplaceAll([]Publication{publication}, false); err != nil {
				report(fmt.Errorf("replace %d: %w", index, err))
				return
			}
		}
	}()
	for reader := 0; reader < 2; reader++ {
		go func(reader int) {
			defer group.Done()
			<-start
			for index := 0; index < 500; index++ {
				snapshot := registry.Snapshot()
				if len(snapshot.Publications) != 1 || len(snapshot.Contributions) != 1 ||
					snapshot.Digest == "" || snapshot.Revision == 0 ||
					snapshot.Publications[0].Artifact != snapshot.Contributions[0].Artifact {
					report(fmt.Errorf("reader %d partial snapshot: %#v", reader, snapshot))
					return
				}
			}
		}(reader)
	}
	for executor := 0; executor < 2; executor++ {
		go func(executor int) {
			defer group.Done()
			<-start
			for index := 0; index < 300; index++ {
				result, err := runtime.Execute(context.Background(), ExecuteRequest{Scope: "core.page.topic"})
				if err != nil {
					if errors.Is(err, ErrSnapshotStale) || errors.Is(err, ErrArtifactUnavailable) {
						continue
					}
					report(fmt.Errorf("executor %d: %w", executor, err))
					return
				}
				if result.Document.Title != "first" && result.Document.Title != "second" {
					report(fmt.Errorf("executor %d partial result: %#v", executor, result))
					return
				}
			}
		}(executor)
	}
	close(start)
	group.Wait()
	select {
	case err := <-errorsCh:
		t.Fatal(err)
	default:
	}
}

func TestTraceRingConcurrentAppendAndDetachedRead(t *testing.T) {
	ring := NewExecutionTraceRing(32)
	var group sync.WaitGroup
	for writer := 0; writer < 8; writer++ {
		group.Add(1)
		go func(writer int) {
			defer group.Done()
			for index := 0; index < 100; index++ {
				ring.AppendSEOExecutionTrace(ExecutionTrace{
					Scope: "core.page.topic", Stage: "release_fence", Outcome: TraceOutcomeApplied,
					Calls: []ProviderCallTrace{{ID: fmt.Sprintf("plugin.trace.%d.%d", writer, index)}},
				})
			}
		}(writer)
	}
	group.Wait()
	traces := ring.SEOExecutionTraces(32)
	if len(traces) != 32 {
		t.Fatalf("trace count=%d", len(traces))
	}
	traces[0].Calls[0].ID = "forged"
	if ring.SEOExecutionTraces(1)[0].Calls[0].ID == "forged" {
		t.Fatal("trace read aliases ring storage")
	}
}
