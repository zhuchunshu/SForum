package identity

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

func TestSessionPolicyEvaluatorCoreProviderResolutionAndInvokeExact(t *testing.T) {
	store := &fakeSessionPolicyStore{
		resolve: IdentitySessionPolicyResolution{
			PolicyID: IdentitySessionPolicyCoreDefault,
			Source:   IdentitySessionPolicySourceCore,
			Selection: &IdentitySessionPolicySelection{
				IdentitySessionPolicyEvidence: IdentitySessionPolicyEvidence{
					PolicyID: IdentitySessionPolicyCoreDefault,
				},
				Revision: 0,
				Implicit: true,
			},
			RegistryRevision: 3,
			RegistryDigest:   "core-digest",
		},
	}
	evaluator, err := NewSessionPolicyEvaluator(store, nil)
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}

	resolution, err := evaluator.ProviderResolution(t.Context())
	if err != nil || resolution.Source != IdentitySessionPolicySourceCore ||
		resolution.Operation != "" || resolution.Provider != nil {
		t.Fatalf("core resolution=%#v err=%v", resolution, err)
	}

	result, err := evaluator.InvokeExact(t.Context(), resolution, SessionEvaluationInput{
		UserID: 42, Purpose: SessionEvaluationPurposeIssue,
	})
	if err != nil || result.Disposition != SessionPolicyDispositionAllow ||
		result.Source != IdentitySessionPolicySourceCore || result.SelectionRevision != 0 {
		t.Fatalf("core invoke=%#v err=%v", result, err)
	}

	allowed, err := evaluator.RequireAllow(t.Context(), SessionEvaluationInput{
		UserID: 42, Purpose: SessionEvaluationPurposeRenew,
	})
	if err != nil || allowed.Disposition != SessionPolicyDispositionAllow {
		t.Fatalf("core require allow=%#v err=%v", allowed, err)
	}
	if store.resolveCalls.Load() < 3 {
		t.Fatalf("expected resolve+recheck calls, got %d", store.resolveCalls.Load())
	}
}

func TestSessionPolicyEvaluatorSafeModeNeverInvokesPlugin(t *testing.T) {
	invoker := &fakeSessionPolicyInvoker{}
	store := &fakeSessionPolicyStore{
		resolve: IdentitySessionPolicyResolution{
			PolicyID:         IdentitySessionPolicyCoreDefault,
			Source:           IdentitySessionPolicySourceSafeMode,
			RegistryRevision: 9,
			RegistryDigest:   "safe-digest",
		},
	}
	evaluator, err := NewSessionPolicyEvaluator(store, invoker)
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}
	result, err := evaluator.Evaluate(t.Context(), SessionEvaluationInput{
		UserID: 7, Purpose: SessionEvaluationPurposeIssue,
	})
	if err != nil || result.Disposition != SessionPolicyDispositionAllow ||
		result.Source != IdentitySessionPolicySourceSafeMode || invoker.calls.Load() != 0 {
		t.Fatalf("safe mode result=%#v invokerCalls=%d err=%v", result, invoker.calls.Load(), err)
	}
}

func TestSessionPolicyEvaluatorPluginInvokeExactDispositions(t *testing.T) {
	provider := sessionEvaluationTestProvider()
	selection := &IdentitySessionPolicySelection{
		IdentitySessionPolicyEvidence: IdentitySessionPolicyEvidence{
			PolicyID:                "plugin.session.policy",
			ProviderContractVersion: "1",
			OwnerExtensionID:        provider.Artifact.ExtensionID,
			OwnerExtensionVersionID: provider.Artifact.VersionID,
			OwnerExtensionVersion:   provider.Artifact.ExtensionVersion,
			OwnerPackageDigest:      provider.Artifact.PackageDigest,
			DeclarationRevision:     4,
		},
		Revision: 4,
	}
	base := IdentitySessionPolicyResolution{
		PolicyID:         "plugin.session.policy",
		Source:           IdentitySessionPolicySourcePlugin,
		Selection:        selection,
		Provider:         &provider,
		RegistryRevision: 11,
		RegistryDigest:   "plugin-digest",
	}

	for _, test := range []struct {
		name        string
		disposition string
		wantErr     error
	}{
		{name: "allow", disposition: SessionPolicyDispositionAllow},
		{name: "deny", disposition: SessionPolicyDispositionDeny, wantErr: ErrSessionPolicyEvaluationDenied},
		{name: "step_up", disposition: SessionPolicyDispositionStepUp, wantErr: ErrSessionPolicyEvaluationStepUp},
	} {
		t.Run(test.name, func(t *testing.T) {
			invoker := &fakeSessionPolicyInvoker{
				output: map[string]any{"disposition": test.disposition},
			}
			store := &fakeSessionPolicyStore{resolve: base}
			evaluator, err := NewSessionPolicyEvaluator(store, invoker)
			if err != nil {
				t.Fatalf("new evaluator: %v", err)
			}
			result, err := evaluator.RequireAllow(t.Context(), SessionEvaluationInput{
				UserID: 99, Purpose: SessionEvaluationPurposeIssue,
				CorrelationID: "corr-1", DeviceFingerprint: "device-a",
			})
			if test.wantErr == nil {
				if err != nil || result.Disposition != SessionPolicyDispositionAllow ||
					invoker.calls.Load() != 1 || invoker.lastOperation != sessionEvaluateOperation ||
					invoker.lastActor != 99 || invoker.fenceCalls.Load() != 1 {
					t.Fatalf("allow result=%#v invoker=%#v err=%v", result, invoker, err)
				}
				if invoker.lastInput["purpose"] != SessionEvaluationPurposeIssue ||
					invoker.lastInput["userId"] != int64(99) ||
					invoker.lastInput["correlationId"] != "corr-1" {
					t.Fatalf("plugin input = %#v", invoker.lastInput)
				}
				return
			}
			if !errors.Is(err, test.wantErr) || invoker.calls.Load() != 1 {
				t.Fatalf("want %v, got result=%#v err=%v calls=%d", test.wantErr, result, err, invoker.calls.Load())
			}
		})
	}
}

func TestSessionPolicyEvaluatorPluginFailsClosedWithoutInvokerOrMalformedOutput(t *testing.T) {
	provider := sessionEvaluationTestProvider()
	selection := &IdentitySessionPolicySelection{
		IdentitySessionPolicyEvidence: IdentitySessionPolicyEvidence{
			PolicyID:                "plugin.session.policy",
			ProviderContractVersion: "1",
			OwnerExtensionID:        provider.Artifact.ExtensionID,
			OwnerExtensionVersionID: provider.Artifact.VersionID,
			OwnerExtensionVersion:   provider.Artifact.ExtensionVersion,
			OwnerPackageDigest:      provider.Artifact.PackageDigest,
			DeclarationRevision:     2,
		},
		Revision: 2,
	}
	resolution := IdentitySessionPolicyResolution{
		PolicyID: "plugin.session.policy", Source: IdentitySessionPolicySourcePlugin,
		Selection: selection, Provider: &provider,
		RegistryRevision: 1, RegistryDigest: "d1",
	}

	t.Run("missing invoker", func(t *testing.T) {
		evaluator, err := NewSessionPolicyEvaluator(&fakeSessionPolicyStore{resolve: resolution}, nil)
		if err != nil {
			t.Fatalf("new evaluator: %v", err)
		}
		if _, err := evaluator.Evaluate(t.Context(), SessionEvaluationInput{
			UserID: 1, Purpose: SessionEvaluationPurposeIssue,
		}); !errors.Is(err, ErrSessionPolicyEvaluationUnavailable) {
			t.Fatalf("missing invoker err=%v", err)
		}
	})

	t.Run("malformed disposition", func(t *testing.T) {
		invoker := &fakeSessionPolicyInvoker{output: map[string]any{"disposition": "maybe"}}
		evaluator, err := NewSessionPolicyEvaluator(&fakeSessionPolicyStore{resolve: resolution}, invoker)
		if err != nil {
			t.Fatalf("new evaluator: %v", err)
		}
		if _, err := evaluator.Evaluate(t.Context(), SessionEvaluationInput{
			UserID: 1, Purpose: SessionEvaluationPurposeIssue,
		}); !errors.Is(err, ErrSessionPolicyEvaluationUnavailable) {
			t.Fatalf("malformed disposition err=%v", err)
		}
	})

	t.Run("invoker transport failure", func(t *testing.T) {
		invoker := &fakeSessionPolicyInvoker{err: errors.New("transport down")}
		evaluator, err := NewSessionPolicyEvaluator(&fakeSessionPolicyStore{resolve: resolution}, invoker)
		if err != nil {
			t.Fatalf("new evaluator: %v", err)
		}
		if _, err := evaluator.Evaluate(t.Context(), SessionEvaluationInput{
			UserID: 1, Purpose: SessionEvaluationPurposeRenew,
		}); !errors.Is(err, ErrSessionPolicyEvaluationUnavailable) {
			t.Fatalf("transport failure err=%v", err)
		}
	})
}

func TestSessionPolicyEvaluatorRecheckDetectsSelectionDrift(t *testing.T) {
	provider := sessionEvaluationTestProvider()
	selection := &IdentitySessionPolicySelection{
		IdentitySessionPolicyEvidence: IdentitySessionPolicyEvidence{
			PolicyID:                "plugin.session.policy",
			ProviderContractVersion: "1",
			OwnerExtensionID:        provider.Artifact.ExtensionID,
			OwnerExtensionVersionID: provider.Artifact.VersionID,
			OwnerExtensionVersion:   provider.Artifact.ExtensionVersion,
			OwnerPackageDigest:      provider.Artifact.PackageDigest,
			DeclarationRevision:     5,
		},
		Revision: 5,
	}
	store := &fakeSessionPolicyStore{
		resolve: IdentitySessionPolicyResolution{
			PolicyID: "plugin.session.policy", Source: IdentitySessionPolicySourcePlugin,
			Selection: selection, Provider: &provider,
			RegistryRevision: 2, RegistryDigest: "before",
		},
	}
	invoker := &fakeSessionPolicyInvoker{
		output: map[string]any{"disposition": SessionPolicyDispositionAllow},
		afterInvoke: func() {
			// Simulate concurrent lifecycle reset to Core between invoke and effect.
			store.resolve = IdentitySessionPolicyResolution{
				PolicyID: IdentitySessionPolicyCoreDefault,
				Source:   IdentitySessionPolicySourceCore,
				Selection: &IdentitySessionPolicySelection{
					IdentitySessionPolicyEvidence: IdentitySessionPolicyEvidence{
						PolicyID: IdentitySessionPolicyCoreDefault,
					},
					Revision: 6,
				},
				RegistryRevision: 3,
				RegistryDigest:   "after",
			}
		},
	}
	evaluator, err := NewSessionPolicyEvaluator(store, invoker)
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}
	if _, err := evaluator.Evaluate(t.Context(), SessionEvaluationInput{
		UserID: 3, Purpose: SessionEvaluationPurposeIssue,
	}); !errors.Is(err, ErrSessionPolicyEvaluationStale) {
		t.Fatalf("selection drift err=%v", err)
	}
}

func TestSessionPolicyEvaluatorRejectsInvalidInput(t *testing.T) {
	evaluator, err := NewSessionPolicyEvaluator(&fakeSessionPolicyStore{
		resolve: IdentitySessionPolicyResolution{
			PolicyID: IdentitySessionPolicyCoreDefault,
			Source:   IdentitySessionPolicySourceCore,
		},
	}, nil)
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}
	for _, input := range []SessionEvaluationInput{
		{UserID: 0, Purpose: SessionEvaluationPurposeIssue},
		{UserID: 1, Purpose: "logout"},
		{UserID: 1, Purpose: SessionEvaluationPurposeIssue, CorrelationID: string(make([]byte, 129))},
	} {
		if _, err := evaluator.Evaluate(t.Context(), input); !errors.Is(err, ErrSessionPolicyEvaluationInvalid) {
			t.Fatalf("input %#v err=%v", input, err)
		}
	}
	if _, err := NewSessionPolicyEvaluator(nil, nil); !errors.Is(err, ErrIdentitySessionPolicyStoreUnavailable) {
		t.Fatalf("nil store err=%v", err)
	}
}

func TestSessionPolicyEvaluatorFenceMustRunBeforeEffect(t *testing.T) {
	provider := sessionEvaluationTestProvider()
	selection := &IdentitySessionPolicySelection{
		IdentitySessionPolicyEvidence: IdentitySessionPolicyEvidence{
			PolicyID:                "plugin.session.policy",
			ProviderContractVersion: "1",
			OwnerExtensionID:        provider.Artifact.ExtensionID,
			OwnerExtensionVersionID: provider.Artifact.VersionID,
			OwnerExtensionVersion:   provider.Artifact.ExtensionVersion,
			OwnerPackageDigest:      provider.Artifact.PackageDigest,
			DeclarationRevision:     1,
		},
		Revision: 1,
	}
	invoker := &fakeSessionPolicyInvoker{
		output:     map[string]any{"disposition": SessionPolicyDispositionAllow},
		skipFence:  true,
		fenceError: errors.New("admission lost"),
	}
	// When skipFence is true the invoker still requires Accept to call fence;
	// configure fenceError path instead: Accept calls fence, fence fails.
	invoker.skipFence = false
	evaluator, err := NewSessionPolicyEvaluator(&fakeSessionPolicyStore{
		resolve: IdentitySessionPolicyResolution{
			PolicyID: "plugin.session.policy", Source: IdentitySessionPolicySourcePlugin,
			Selection: selection, Provider: &provider,
			RegistryRevision: 1, RegistryDigest: "d",
		},
	}, invoker)
	if err != nil {
		t.Fatalf("new evaluator: %v", err)
	}
	if _, err := evaluator.Evaluate(t.Context(), SessionEvaluationInput{
		UserID: 1, Purpose: SessionEvaluationPurposeIssue,
	}); !errors.Is(err, ErrSessionPolicyEvaluationUnavailable) {
		t.Fatalf("fence failure err=%v", err)
	}
}

func sessionEvaluationTestProvider() identityregistry.ProviderContribution {
	return identityregistry.ProviderContribution{
		Provider: identityregistry.Provider{
			ID:              "plugin.session.provider",
			ContractVersion: "1",
			Kind:            identityregistry.ProviderKindSession,
			Handler:         "session",
			Priority:        10,
			Operations: []identityregistry.ProviderOperation{{
				Name: sessionEvaluateOperation, InputSchema: "schemas/session-input.json",
				OutputSchema: "schemas/session-output.json", TimeoutMS: 1_000,
				FailurePolicy: identityregistry.ProviderFailureFailClosed,
			}},
		},
		Artifact: identityregistry.Artifact{
			ExtensionID: "plugin.membership", ExtensionVersion: "1.0.0",
			VersionID: 8, PackageDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			RuntimeInstanceID: "instance-1",
		},
	}
}

type fakeSessionPolicyStore struct {
	resolve      IdentitySessionPolicyResolution
	resolveErr   error
	resolveCalls atomic.Int32
}

func (s *fakeSessionPolicyStore) Current(context.Context) (IdentitySessionPolicySelection, error) {
	return IdentitySessionPolicySelection{}, errors.New("unused")
}
func (s *fakeSessionPolicyStore) Candidate(context.Context, string) (IdentitySessionPolicyEvidence, error) {
	return IdentitySessionPolicyEvidence{}, errors.New("unused")
}
func (s *fakeSessionPolicyStore) Resolve(context.Context) (IdentitySessionPolicyResolution, error) {
	s.resolveCalls.Add(1)
	if s.resolveErr != nil {
		return IdentitySessionPolicyResolution{}, s.resolveErr
	}
	return s.resolve, nil
}
func (s *fakeSessionPolicyStore) Select(context.Context, SelectIdentitySessionPolicyInput) (IdentitySessionPolicyMutation, error) {
	return IdentitySessionPolicyMutation{}, errors.New("unused")
}
func (s *fakeSessionPolicyStore) Reset(context.Context, ResetIdentitySessionPolicyInput) (IdentitySessionPolicyMutation, error) {
	return IdentitySessionPolicyMutation{}, errors.New("unused")
}
func (s *fakeSessionPolicyStore) ListEvents(context.Context, int) ([]IdentitySessionPolicyEvent, error) {
	return nil, errors.New("unused")
}

type fakeSessionPolicyInvoker struct {
	output        map[string]any
	err           error
	fenceError    error
	skipFence     bool
	afterInvoke   func()
	calls         atomic.Int32
	fenceCalls    atomic.Int32
	lastOperation string
	lastActor     int64
	lastInput     map[string]any
}

func (i *fakeSessionPolicyInvoker) InvokeExact(
	_ context.Context,
	_ identityregistry.ProviderContribution,
	operation string,
	actorUserID int64,
	input map[string]any,
	accept func(context.Context, map[string]any, func() error) error,
) error {
	i.calls.Add(1)
	i.lastOperation = operation
	i.lastActor = actorUserID
	i.lastInput = input
	if i.err != nil {
		return i.err
	}
	if i.afterInvoke != nil {
		i.afterInvoke()
	}
	fence := func() error {
		i.fenceCalls.Add(1)
		return i.fenceError
	}
	if i.skipFence {
		return accept(context.Background(), i.output, nil)
	}
	return accept(context.Background(), i.output, fence)
}
