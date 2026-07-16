package seoregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ExecutionRuntime struct {
	registry     *Registry
	admission    ExecutionAdmission
	finalPolicy  FinalPolicy
	providers    map[providerBindingIdentity]ProviderBinding
	timeout      time.Duration
	maximumBytes int
	trace        ExecutionTraceSink
}

type providerBindingIdentity struct {
	contributionID string
	artifact       Artifact
}

func NewExecutionRuntime(config ExecutionConfig) (*ExecutionRuntime, error) {
	if config.Registry == nil || config.Admission == nil || config.FinalPolicy == nil {
		return nil, ErrExecutionInvalid
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = defaultExecutionTimeout
	}
	maximumBytes := config.MaximumBytes
	if maximumBytes == 0 {
		maximumBytes = defaultMaximumBytes
	}
	if timeout < time.Millisecond || timeout > maximumExecutionTimeout || maximumBytes < 1024 || maximumBytes > maximumOutputBytes {
		return nil, ErrExecutionInvalid
	}
	providers := make(map[providerBindingIdentity]ProviderBinding, len(config.Providers))
	for _, raw := range config.Providers {
		binding, err := normalizeProviderBinding(raw)
		if err != nil {
			return nil, err
		}
		key := providerBindingKey(binding.ContributionID, binding.Artifact)
		if _, duplicate := providers[key]; duplicate {
			return nil, fmt.Errorf("%w: duplicate provider binding %s", ErrExecutionInvalid, binding.ContributionID)
		}
		providers[key] = binding
	}
	return &ExecutionRuntime{
		registry: config.Registry, admission: config.Admission, finalPolicy: config.FinalPolicy, providers: providers,
		timeout: timeout, maximumBytes: maximumBytes, trace: config.Trace,
	}, nil
}

func (r *ExecutionRuntime) Execute(ctx context.Context, request ExecuteRequest) (result ExecuteResult, err error) {
	started := time.Now()
	trace := ExecutionTrace{Scope: request.Scope, Stage: "validate", Outcome: TraceOutcomeInternal}
	defer func() {
		trace.Duration = time.Since(started)
		trace.Outcome = traceOutcome(err)
		if err == nil {
			trace.Outcome = TraceOutcomeApplied
			trace.Applied = len(result.Applied)
			trace.Fallbacks = len(result.Fallbacks)
		}
		if r != nil && r.trace != nil {
			r.trace.AppendSEOExecutionTrace(boundExecutionTrace(trace))
		}
	}()
	if r == nil || r.registry == nil || r.admission == nil || r.finalPolicy == nil || ctx == nil {
		return ExecuteResult{}, ErrExecutionInvalid
	}
	scope := strings.ToLower(strings.TrimSpace(request.Scope))
	trace.Scope = scope
	if scope != GlobalScope && !idPattern.MatchString(scope) {
		return ExecuteResult{}, ErrInvalid
	}
	base := cloneDocument(request.Base)
	if err := r.validateBoundedDocument(base); err != nil {
		return ExecuteResult{}, err
	}
	executionCtx, cancelExecution := context.WithTimeout(ctx, r.timeout)
	defer cancelExecution()
	state := r.registry.load()
	trace.Revision, trace.SnapshotDigest = state.revision, state.digest
	contributions := cloneContributions(contributionsForScope(state, scope))
	if len(contributions) == 0 {
		return ExecuteResult{
			SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest, Document: cloneDocument(base),
		}, nil
	}
	trace.Stage = "admission"
	bindings, leases, admissionErrors, err := r.acquireExecutionSet(executionCtx, state, contributions)
	if err != nil {
		return ExecuteResult{}, err
	}
	defer releaseLeases(leases)

	current := base
	applied := make([]ContributionRef, 0, len(contributions))
	fallbacks := make([]FallbackRecord, 0)
	usedArtifacts := make(map[Artifact]struct{})
	for _, kind := range executionKindOrder {
		byAction := contributionsByKindAndAction(contributions, kind)
		trace.Stage = "add"
		if scalarKind(kind) {
			if documentKindEmpty(current, kind) {
				for _, contribution := range byAction[ActionAdd] {
					next, callTrace, callErr := r.callProvider(
						executionCtx, state, scope, current, contribution, bindings, leases, admissionErrors, usedArtifacts,
					)
					trace.Calls = append(trace.Calls, callTrace)
					if callErr != nil {
						if shouldAlwaysFailClosed(callErr) || contribution.FailurePolicy == FailurePolicyFailClosed {
							return ExecuteResult{}, callErr
						}
						fallbacks = append(fallbacks, fallbackRecord(contribution, callErr))
						continue
					}
					current = next
					applied = append(applied, contributionRef(contribution))
					break
				}
			}
		} else {
			for _, contribution := range byAction[ActionAdd] {
				next, callTrace, callErr := r.callProvider(
					executionCtx, state, scope, current, contribution, bindings, leases, admissionErrors, usedArtifacts,
				)
				trace.Calls = append(trace.Calls, callTrace)
				if callErr != nil {
					if shouldAlwaysFailClosed(callErr) || contribution.FailurePolicy == FailurePolicyFailClosed {
						return ExecuteResult{}, callErr
					}
					fallbacks = append(fallbacks, fallbackRecord(contribution, callErr))
					continue
				}
				current = next
				applied = append(applied, contributionRef(contribution))
			}
		}

		trace.Stage = "replace"
		for _, contribution := range byAction[ActionReplace] {
			next, callTrace, callErr := r.callProvider(
				executionCtx, state, scope, current, contribution, bindings, leases, admissionErrors, usedArtifacts,
			)
			trace.Calls = append(trace.Calls, callTrace)
			if callErr != nil {
				if shouldAlwaysFailClosed(callErr) || contribution.FailurePolicy == FailurePolicyFailClosed {
					return ExecuteResult{}, callErr
				}
				fallbacks = append(fallbacks, fallbackRecord(contribution, callErr))
				continue
			}
			current = next
			applied = append(applied, contributionRef(contribution))
			break
		}

		trace.Stage = "filter"
		for _, contribution := range byAction[ActionFilter] {
			next, callTrace, callErr := r.callProvider(
				executionCtx, state, scope, current, contribution, bindings, leases, admissionErrors, usedArtifacts,
			)
			trace.Calls = append(trace.Calls, callTrace)
			if callErr != nil {
				if shouldAlwaysFailClosed(callErr) || contribution.FailurePolicy == FailurePolicyFailClosed {
					return ExecuteResult{}, callErr
				}
				fallbacks = append(fallbacks, fallbackRecord(contribution, callErr))
				continue
			}
			current = next
			applied = append(applied, contributionRef(contribution))
		}
	}

	trace.Stage = "release_fence"
	if err := r.finalFence(executionCtx, state, scope, base, current, leases, usedArtifacts); err != nil {
		return ExecuteResult{}, err
	}
	return cloneExecuteResult(ExecuteResult{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest,
		Document: current, Applied: applied, Fallbacks: fallbacks,
	}), nil
}

func (r *ExecutionRuntime) acquireExecutionSet(
	ctx context.Context,
	state *registryState,
	contributions []Contribution,
) (map[string]ProviderBinding, map[Artifact]AdmissionLease, map[Artifact]error, error) {
	bindings := make(map[string]ProviderBinding, len(contributions))
	artifacts := make(map[Artifact]struct{})
	for _, contribution := range contributions {
		binding, found := r.providers[providerBindingKey(contribution.ID, contribution.Artifact)]
		if !found || !bindingMatchesContribution(binding, contribution) {
			continue
		}
		bindings[contribution.ID] = binding
		artifacts[contribution.Artifact] = struct{}{}
	}
	ordered := make([]Artifact, 0, len(artifacts))
	for artifact := range artifacts {
		ordered = append(ordered, artifact)
	}
	sort.Slice(ordered, func(i, j int) bool { return artifactBefore(ordered[i], ordered[j]) })
	leases := make(map[Artifact]AdmissionLease, len(ordered))
	admissionErrors := make(map[Artifact]error)
	for _, artifact := range ordered {
		if r.registry.load() != state {
			releaseLeases(leases)
			return nil, nil, nil, ErrSnapshotStale
		}
		lease, err := r.admission.AcquireSEOExecution(ctx, artifact)
		if err != nil || lease == nil || lease.Context() == nil {
			if lease != nil {
				lease.Release()
			}
			if err == nil {
				err = ErrArtifactUnavailable
			}
			admissionErrors[artifact] = errors.Join(ErrArtifactUnavailable, err)
			continue
		}
		if lease.Context().Err() != nil {
			lease.Release()
			admissionErrors[artifact] = errors.Join(ErrArtifactUnavailable, context.Cause(lease.Context()))
			continue
		}
		leases[artifact] = lease
		if r.registry.load() != state {
			releaseLeases(leases)
			return nil, nil, nil, ErrSnapshotStale
		}
		if err := ctx.Err(); err != nil {
			releaseLeases(leases)
			return nil, nil, nil, err
		}
	}
	return bindings, leases, admissionErrors, nil
}

func (r *ExecutionRuntime) callProvider(
	ctx context.Context,
	state *registryState,
	scope string,
	current Document,
	contribution Contribution,
	bindings map[string]ProviderBinding,
	leases map[Artifact]AdmissionLease,
	admissionErrors map[Artifact]error,
	usedArtifacts map[Artifact]struct{},
) (Document, ProviderCallTrace, error) {
	started := time.Now()
	callTrace := providerCallTrace(contribution)
	finish := func(outcome string) {
		callTrace.Duration = time.Since(started)
		callTrace.Outcome = outcome
	}
	if r.registry.load() != state {
		finish(TraceCallSnapshotStale)
		return Document{}, callTrace, ErrSnapshotStale
	}
	binding, found := bindings[contribution.ID]
	if !found {
		finish(TraceCallUnavailable)
		return Document{}, callTrace, ErrProviderUnavailable
	}
	callTrace.ProviderDigest = binding.ProviderDigest
	if admissionErr := admissionErrors[contribution.Artifact]; admissionErr != nil {
		finish(TraceCallRuntimeStale)
		return Document{}, callTrace, admissionErr
	}
	lease := leases[contribution.Artifact]
	if lease == nil || lease.Context() == nil {
		finish(TraceCallRuntimeStale)
		return Document{}, callTrace, ErrArtifactUnavailable
	}
	if lease.Context().Err() != nil {
		finish(TraceCallRuntimeStale)
		return Document{}, callTrace, errors.Join(ErrArtifactUnavailable, context.Cause(lease.Context()))
	}
	usedArtifacts[contribution.Artifact] = struct{}{}
	callBase, cancelCall := context.WithCancelCause(lease.Context())
	stopOuter := context.AfterFunc(ctx, func() { cancelCall(context.Cause(ctx)) })
	callCtx, cancelTimeout := context.WithTimeout(callBase, contribution.Timeout)
	providerResult, providerErr := invokeProvider(binding.Provider, callCtx, ProviderRequest{
		Scope: scope, Contribution: cloneContribution(contribution), Current: cloneDocument(current),
	})
	callContextErr := callCtx.Err()
	stopOuter()
	cancelTimeout()
	cancelCall(nil)
	if callContextErr != nil {
		finish(TraceCallDeadline)
		if ctx.Err() != nil {
			return Document{}, callTrace, ctx.Err()
		}
		return Document{}, callTrace, errors.Join(ErrProviderFailed, ErrProviderDeadline)
	}
	if providerErr != nil {
		finish(TraceCallFailed)
		return Document{}, callTrace, errors.Join(ErrProviderFailed, providerErr)
	}
	if r.registry.load() != state {
		finish(TraceCallSnapshotStale)
		return Document{}, callTrace, ErrSnapshotStale
	}
	if lease.Context().Err() != nil {
		finish(TraceCallRuntimeStale)
		return Document{}, callTrace, errors.Join(ErrArtifactUnavailable, context.Cause(lease.Context()))
	}
	next := cloneDocument(providerResult.Document)
	if err := r.validateBoundedDocument(next); err != nil {
		finish(TraceCallOutputInvalid)
		return Document{}, callTrace, err
	}
	if err := validateMutation(current, next, contribution); err != nil {
		finish(TraceCallMutationDenied)
		return Document{}, callTrace, fmt.Errorf("%w: contribution %s", err, contribution.ID)
	}
	finish(TraceCallApplied)
	return next, callTrace, nil
}

func invokeProvider(provider Provider, ctx context.Context, request ProviderRequest) (result ProviderResult, err error) {
	defer func() {
		if recover() != nil {
			result = ProviderResult{}
			err = ErrProviderFailed
		}
	}()
	return provider.ApplySEO(ctx, request)
}

func (r *ExecutionRuntime) finalFence(
	ctx context.Context,
	state *registryState,
	scope string,
	base Document,
	document Document,
	leases map[Artifact]AdmissionLease,
	used map[Artifact]struct{},
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.validateBoundedDocument(document); err != nil {
		return err
	}
	if err := r.finalPolicy.ValidateSEO(ctx, FinalPolicyRequest{
		Scope: scope, Base: cloneDocument(base), Document: cloneDocument(document),
	}); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.Join(ErrPolicyDenied, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.registry.load() != state {
		return ErrSnapshotStale
	}
	for artifact := range used {
		lease := leases[artifact]
		if lease == nil || lease.Context() == nil {
			return ErrArtifactUnavailable
		}
		if lease.Context().Err() != nil {
			return errors.Join(ErrArtifactUnavailable, context.Cause(lease.Context()))
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.registry.load() != state {
		return ErrSnapshotStale
	}
	return nil
}

func (r *ExecutionRuntime) validateBoundedDocument(document Document) error {
	if err := validateDocument(document); err != nil {
		return err
	}
	body, err := json.Marshal(document)
	if err != nil {
		return ErrOutputInvalid
	}
	if len(body) > r.maximumBytes {
		return ErrOutputTooLarge
	}
	return nil
}

func contributionsByKindAndAction(values []Contribution, kind string) map[string][]Contribution {
	result := map[string][]Contribution{ActionAdd: {}, ActionReplace: {}, ActionFilter: {}}
	for _, contribution := range values {
		if contribution.Kind == kind {
			result[contribution.Action] = append(result[contribution.Action], contribution)
		}
	}
	for action := range result {
		sort.Slice(result[action], func(i, j int) bool { return contributionBefore(result[action][i], result[action][j]) })
	}
	return result
}

func shouldAlwaysFailClosed(err error) bool {
	return errors.Is(err, ErrSnapshotStale) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func contributionRef(contribution Contribution) ContributionRef {
	return ContributionRef{ID: contribution.ID, Action: contribution.Action, Kind: contribution.Kind, Artifact: contribution.Artifact}
}

func fallbackRecord(contribution Contribution, err error) FallbackRecord {
	return FallbackRecord{Contribution: contributionRef(contribution), Reason: fallbackReason(err)}
}

func fallbackReason(err error) string {
	switch {
	case errors.Is(err, ErrProviderUnavailable):
		return "provider_unavailable"
	case errors.Is(err, ErrArtifactUnavailable):
		return "runtime_unavailable"
	case errors.Is(err, ErrMutationDenied):
		return "mutation_denied"
	case errors.Is(err, ErrOutputTooLarge):
		return "output_too_large"
	case errors.Is(err, ErrOutputInvalid):
		return "output_invalid"
	case errors.Is(err, ErrProviderDeadline):
		return "deadline"
	default:
		return "provider_failed"
	}
}

func releaseLeases(leases map[Artifact]AdmissionLease) {
	artifacts := make([]Artifact, 0, len(leases))
	for artifact := range leases {
		artifacts = append(artifacts, artifact)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifactBefore(artifacts[j], artifacts[i]) })
	for _, artifact := range artifacts {
		leases[artifact].Release()
	}
}

func (r *ExecutionRuntime) Inspect(scope string) (RuntimeInspection, error) {
	if r == nil || r.registry == nil {
		return RuntimeInspection{}, ErrExecutionInvalid
	}
	inspection, err := r.registry.Inspect(scope)
	if err != nil {
		return RuntimeInspection{}, err
	}
	result := RuntimeInspection{Scope: inspection, Providers: make([]ProviderInspection, 0, len(inspection.Contributions))}
	for _, contribution := range inspection.Contributions {
		binding, found := r.providers[providerBindingKey(contribution.ID, contribution.Artifact)]
		bound := found && bindingMatchesContribution(binding, contribution)
		provider := ProviderInspection{Contribution: cloneContribution(contribution), Bound: bound}
		if bound {
			provider.ProviderDigest = binding.ProviderDigest
		}
		result.Providers = append(result.Providers, provider)
	}
	return result, nil
}
