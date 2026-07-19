package identity

import (
	"errors"
	"sync"
	"testing"
)

func TestIdentitySessionPolicyPostgresConcurrentCASHasOneWinner(t *testing.T) {
	fixture := newIdentityPersistencePGFixture(t)
	candidate, err := fixture.sessionPolicy.Candidate(fixture.ctx, fixture.sessionProvider.ID)
	if err != nil {
		t.Fatal(err)
	}

	selectResults := runIdentitySessionPolicyRace(2, func() error {
		_, err := fixture.sessionPolicy.Select(fixture.ctx, SelectIdentitySessionPolicyInput{
			Candidate: candidate, ExpectedRevision: 0, ActorUserID: fixture.adminUserID,
		})
		return err
	})
	assertIdentitySessionPolicyRaceResult(t, selectResults)
	fixture.assertSessionPolicyCounts(1, 1, 1)
	current, err := fixture.sessionPolicy.Current(fixture.ctx)
	if err != nil || current.Revision != 1 || current.IdentitySessionPolicyEvidence != candidate {
		t.Fatalf("selection after race = %#v, %v", current, err)
	}

	resetResults := runIdentitySessionPolicyRace(2, func() error {
		_, err := fixture.sessionPolicy.Reset(fixture.ctx, ResetIdentitySessionPolicyInput{
			ExpectedRevision: 1, ActorUserID: fixture.adminUserID, ReasonCode: "operator_reset",
		})
		return err
	})
	assertIdentitySessionPolicyRaceResult(t, resetResults)
	fixture.assertSessionPolicyCounts(1, 2, 2)
	current, err = fixture.sessionPolicy.Current(fixture.ctx)
	if err != nil || current.Revision != 2 || current.PolicyID != IdentitySessionPolicyCoreDefault || current.Implicit {
		t.Fatalf("Core selection after reset race = %#v, %v", current, err)
	}
}

func runIdentitySessionPolicyRace(count int, operation func() error) []error {
	start := make(chan struct{})
	results := make(chan error, count)
	var group sync.WaitGroup
	group.Add(count)
	for range count {
		go func() {
			defer group.Done()
			<-start
			results <- operation()
		}()
	}
	close(start)
	group.Wait()
	close(results)
	errorsFound := make([]error, 0, count)
	for err := range results {
		errorsFound = append(errorsFound, err)
	}
	return errorsFound
}

func assertIdentitySessionPolicyRaceResult(t *testing.T, results []error) {
	t.Helper()
	winners, conflicts := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, ErrIdentitySessionPolicyRevisionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected race error: %v", err)
		}
	}
	if winners != 1 || conflicts != len(results)-1 {
		t.Fatalf("race results winners=%d conflicts=%d all=%v", winners, conflicts, results)
	}
}
