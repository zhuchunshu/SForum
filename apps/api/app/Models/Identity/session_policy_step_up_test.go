package identity

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSessionPolicyStepUpEvidenceIssueConsumeAndReplay(t *testing.T) {
	t.Parallel()

	stepUpStore := NewMemorySessionPolicyStepUpStore()
	policyStore := &fakeSessionPolicyStore{resolve: sessionEvaluationTestResolution()}
	invoker := &fakeSessionPolicyInvoker{
		output: map[string]any{"disposition": SessionPolicyDispositionStepUp},
	}
	evaluator, err := NewSessionPolicyEvaluator(policyStore, invoker)
	if err != nil {
		t.Fatal(err)
	}
	evaluator.WithStepUpStore(stepUpStore)

	var effectCalls atomic.Int32
	result, err := evaluator.RequireAllowAndRun(
		t.Context(),
		SessionEvaluationInput{UserID: 7, TokenVersion: 1, Purpose: SessionEvaluationPurposeIssue},
		func(context.Context) error {
			effectCalls.Add(1)
			return nil
		},
	)
	if !errors.Is(err, ErrSessionPolicyEvaluationStepUp) {
		t.Fatalf("step_up err=%v", err)
	}
	if result.StepUpToken == "" || result.Disposition != SessionPolicyDispositionStepUp {
		t.Fatalf("result=%#v", result)
	}
	if effectCalls.Load() != 0 || invoker.calls.Load() != 1 {
		t.Fatalf("effect=%d invoker=%d", effectCalls.Load(), invoker.calls.Load())
	}

	// Completed path: consume evidence without re-invoking plugin.
	invoker.output = map[string]any{"disposition": SessionPolicyDispositionDeny}
	completed, err := evaluator.RequireAllowAndRun(
		t.Context(),
		SessionEvaluationInput{
			UserID: 7, TokenVersion: 1, Purpose: SessionEvaluationPurposeIssue,
			StepUpEvidenceToken: result.StepUpToken,
		},
		func(context.Context) error {
			effectCalls.Add(1)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("complete step_up: %v", err)
	}
	if completed.Disposition != SessionPolicyDispositionAllow || effectCalls.Load() != 1 {
		t.Fatalf("completed result=%#v effect=%d", completed, effectCalls.Load())
	}
	if invoker.calls.Load() != 1 {
		t.Fatalf("plugin re-invoked: %d", invoker.calls.Load())
	}

	// Replay must fail closed.
	_, err = evaluator.RequireAllowAndRun(
		t.Context(),
		SessionEvaluationInput{
			UserID: 7, TokenVersion: 1, Purpose: SessionEvaluationPurposeIssue,
			StepUpEvidenceToken: result.StepUpToken,
		},
		func(context.Context) error {
			t.Fatal("replay must not run effect")
			return nil
		},
	)
	if !errors.Is(err, ErrSessionPolicyStepUpReplayed) {
		t.Fatalf("replay err=%v", err)
	}
	if effectCalls.Load() != 1 {
		t.Fatalf("replay effect calls=%d", effectCalls.Load())
	}
}

func TestMemorySessionPolicyStepUpStoreStaleAndExpired(t *testing.T) {
	t.Parallel()

	store := NewMemorySessionPolicyStepUpStore()
	now := time.Now().UTC()
	store.nowFn = func() time.Time { return now }
	claim := SessionPolicyStepUpClaim{
		UserID: 3, TokenVersion: 2, Purpose: SessionEvaluationPurposeRenew,
		PolicyID: "plugin.session.policy", SelectionRevision: 3,
		RegistryRevision: 7, RegistryDigest: "digest",
		PackageDigest:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OwnerExtensionID: "plugin.membership",
	}
	tokenHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := store.Issue(t.Context(), claim, now.Add(time.Minute), tokenHash); err != nil {
		t.Fatal(err)
	}
	stale := claim
	stale.SelectionRevision = 5
	if err := store.ConsumeForEffect(t.Context(), tokenHash, stale); !errors.Is(err, ErrSessionPolicyStepUpStale) {
		t.Fatalf("stale=%v", err)
	}
	store.nowFn = func() time.Time { return now.Add(2 * time.Minute) }
	if err := store.ConsumeForEffect(t.Context(), tokenHash, claim); !errors.Is(err, ErrSessionPolicyStepUpExpired) {
		t.Fatalf("expired=%v", err)
	}
}

func TestSessionPolicyStepUpDoesNotMintOnInspectEvaluate(t *testing.T) {
	t.Parallel()

	stepUpStore := NewMemorySessionPolicyStepUpStore()
	policyStore := &fakeSessionPolicyStore{resolve: sessionEvaluationTestResolution()}
	invoker := &fakeSessionPolicyInvoker{
		output: map[string]any{"disposition": SessionPolicyDispositionStepUp},
	}
	evaluator, err := NewSessionPolicyEvaluator(policyStore, invoker)
	if err != nil {
		t.Fatal(err)
	}
	evaluator.WithStepUpStore(stepUpStore)

	result, err := evaluator.Evaluate(
		t.Context(),
		SessionEvaluationInput{UserID: 7, TokenVersion: 1, Purpose: SessionEvaluationPurposeIssue},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != SessionPolicyDispositionStepUp || result.StepUpToken != "" {
		t.Fatalf("inspect evaluate must not mint evidence: %#v", result)
	}
}
