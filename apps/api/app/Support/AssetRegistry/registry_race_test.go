package assetregistry

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestRegistryConcurrentFullReplaceCASAndQuarantinePublishOneWinner(t *testing.T) {
	for attempt := 0; attempt < 50; attempt++ {
		t.Run(fmt.Sprintf("attempt-%02d", attempt), func(t *testing.T) {
			registry := New()
			initial := fixturePublication("cas.quarantine", digestA, []Declaration{
				assetDeclaration("cas.quarantine.initial", nil),
			})
			if _, err := registry.Publish(initial); err != nil {
				t.Fatal(err)
			}
			expectedRevision := registry.Revision()
			replacement := fixturePublication("cas.quarantine", digestB, []Declaration{
				assetDeclaration("cas.quarantine.replacement", nil),
			})
			replacement.Artifact.ExtensionVersion = "2.0.0"
			replacement.Artifact.PackageDigest = digestB

			type outcome struct {
				kind        string
				revision    uint64
				quarantined []Artifact
				err         error
			}
			start := make(chan struct{})
			results := make(chan outcome, 2)
			go func() {
				<-start
				revision, err := registry.ReplaceAllIfRevision(expectedRevision, []Publication{replacement})
				results <- outcome{kind: "replace", revision: revision, err: err}
			}()
			go func() {
				<-start
				revision, quarantined, err := registry.QuarantineExact(initial.Artifact)
				results <- outcome{kind: "quarantine", revision: revision, quarantined: quarantined, err: err}
			}()
			close(start)

			winner := ""
			for range 2 {
				result := <-results
				if result.err == nil {
					if winner != "" || result.revision != expectedRevision+1 {
						t.Fatalf("unexpected winner: previous=%s result=%#v", winner, result)
					}
					winner = result.kind
					if result.kind == "quarantine" && len(result.quarantined) != 1 {
						t.Fatalf("quarantine winner=%#v", result)
					}
					continue
				}
				if result.revision != expectedRevision+1 ||
					(result.kind == "replace" && !errors.Is(result.err, ErrRevisionConflict)) ||
					(result.kind == "quarantine" && !errors.Is(result.err, ErrArtifactConflict)) {
					t.Fatalf("unexpected loser: %#v", result)
				}
			}
			if winner == "" {
				t.Fatal("both concurrent writers failed")
			}
			snapshot := registry.Snapshot()
			if snapshot.Revision != expectedRevision+1 {
				t.Fatalf("final revision=%d snapshot=%#v", snapshot.Revision, snapshot)
			}
			if winner == "replace" && (len(snapshot.Assets) != 1 || snapshot.Assets[0].Handle != "cas.quarantine.replacement") {
				t.Fatalf("replace winner published wrong snapshot: %#v", snapshot)
			}
			if winner == "quarantine" && (len(snapshot.Publications) != 0 || len(snapshot.Assets) != 0) {
				t.Fatalf("quarantine winner published wrong snapshot: %#v", snapshot)
			}
		})
	}
}

func TestRegistryConcurrentFullReplaceCASPublishesOneOfFifty(t *testing.T) {
	const writers = 50
	registry := New()
	initial := fixturePublication("cas.fifty", digestA, []Declaration{
		assetDeclaration("cas.fifty.initial", nil),
	})
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	expectedRevision := registry.Revision()
	start := make(chan struct{})
	results := make(chan error, writers)
	for writer := range writers {
		writer := writer
		go func() {
			<-start
			handle := fmt.Sprintf("cas.fifty.candidate.%02d", writer)
			candidate := fixturePublication("cas.fifty", digestB, []Declaration{
				assetDeclaration(handle, nil),
			})
			candidate.Artifact.ExtensionVersion = "2.0.0"
			candidate.Artifact.PackageDigest = fmt.Sprintf("%064x", writer+1)
			_, err := registry.ReplaceAllIfRevision(expectedRevision, []Publication{candidate})
			results <- err
		}()
	}
	close(start)
	succeeded, conflicted := 0, 0
	for range writers {
		switch err := <-results; {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrRevisionConflict):
			conflicted++
		default:
			t.Fatalf("unexpected CAS result: %v", err)
		}
	}
	snapshot := registry.Snapshot()
	if succeeded != 1 || conflicted != writers-1 || snapshot.Revision != expectedRevision+1 ||
		len(snapshot.Publications) != 1 || len(snapshot.Assets) != 1 {
		t.Fatalf("50-way CAS: success=%d conflict=%d snapshot=%#v", succeeded, conflicted, snapshot)
	}
}

func TestRegistryConcurrentQuarantineExactSwapsOnce(t *testing.T) {
	const writers = 50
	registry := New()
	publication := fixturePublication("single.quarantine", digestA, []Declaration{
		assetDeclaration("single.quarantine.entry", nil),
	})
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		quarantined []Artifact
		err         error
	}
	start := make(chan struct{})
	results := make(chan outcome, writers)
	for range writers {
		go func() {
			<-start
			_, quarantined, err := registry.QuarantineExact(publication.Artifact)
			results <- outcome{quarantined: quarantined, err: err}
		}()
	}
	close(start)
	removed := 0
	for range writers {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		if len(result.quarantined) == 1 {
			removed++
		} else if result.quarantined == nil || len(result.quarantined) != 0 {
			t.Fatalf("quarantined=%#v", result.quarantined)
		}
	}
	if snapshot := registry.Snapshot(); removed != 1 || snapshot.Revision != 2 || len(snapshot.Publications) != 0 {
		t.Fatalf("removed=%d snapshot=%#v", removed, snapshot)
	}
}

func TestRegistryConcurrentReadersObserveWholeQuarantineGraphs(t *testing.T) {
	owner := fixturePublication("reader.owner", digestA, []Declaration{
		assetDeclaration("reader.owner.shared", nil),
	})
	consumer := fixturePublication("reader.consumer", digestB, []Declaration{
		assetDeclaration("reader.consumer.entry", []string{"reader.owner.shared"}),
	})
	unrelated := fixturePublication("reader.unrelated", digestC, []Declaration{
		assetDeclaration("reader.unrelated.entry", nil),
	})
	full := []Publication{consumer, unrelated, owner}
	registry := New()
	if _, err := registry.ReplaceAll(full); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	problems := make(chan error, 8)
	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-start
			for iteration := 0; iteration < 400; iteration++ {
				snapshot := registry.Snapshot()
				if !wholeQuarantineSnapshot(snapshot) {
					problems <- fmt.Errorf("partial snapshot: %#v", snapshot)
					return
				}
				plan, err := registry.Plan(PlanRequest{IncludeGlobal: true})
				if err != nil || !wholeQuarantinePlan(plan) {
					problems <- fmt.Errorf("partial plan: %#v: %w", plan, err)
					return
				}
				publication, ok := registry.SnapshotPublication("reader.unrelated")
				if !ok || len(publication.Assets) != 1 {
					problems <- fmt.Errorf("unrelated publication disappeared: %#v", publication)
					return
				}
				publication.Assets[0].Handle = "forged.reader.handle"
			}
		}()
	}
	close(start)
	for iteration := 0; iteration < 200; iteration++ {
		if _, quarantined, err := registry.QuarantineExact(owner.Artifact); err != nil || len(quarantined) != 2 {
			t.Fatalf("quarantine %d: artifacts=%#v err=%v", iteration, quarantined, err)
		}
		revision := registry.Revision()
		if _, err := registry.ReplaceAllIfRevision(revision, full); err != nil {
			t.Fatalf("restore %d: %v", iteration, err)
		}
	}
	readers.Wait()
	close(problems)
	for problem := range problems {
		t.Fatal(problem)
	}
}

func wholeQuarantineSnapshot(snapshot Snapshot) bool {
	if len(snapshot.Publications) == 1 && len(snapshot.Assets) == 1 {
		return snapshot.Publications[0].Artifact.ExtensionID == "reader.unrelated" &&
			snapshot.Assets[0].Handle == "reader.unrelated.entry"
	}
	if len(snapshot.Publications) != 3 || len(snapshot.Assets) != 3 {
		return false
	}
	return snapshot.Assets[0].Handle == "reader.consumer.entry" &&
		snapshot.Assets[1].Handle == "reader.owner.shared" &&
		snapshot.Assets[2].Handle == "reader.unrelated.entry"
}

func wholeQuarantinePlan(plan []Asset) bool {
	if len(plan) == 1 {
		return plan[0].Handle == "reader.unrelated.entry"
	}
	if len(plan) != 3 {
		return false
	}
	return plan[0].Handle == "reader.owner.shared" &&
		plan[1].Handle == "reader.consumer.entry" &&
		plan[2].Handle == "reader.unrelated.entry"
}
