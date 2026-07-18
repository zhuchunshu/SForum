package queryregistry

import (
	"context"
	"errors"
	"sort"
	"sync"
)

type executionAdmissionRequirement struct {
	artifact   Artifact
	queryOwner bool
}

type executionAdmissionSet struct {
	parent  context.Context
	ctx     context.Context
	cancel  context.CancelCauseFunc
	leases  []ExecutionAdmissionLease
	stops   []func() bool
	release sync.Once
}

type executionAdmissionContextKey struct{}

func newExecutionAdmissionSet(parent context.Context, capacity int) *executionAdmissionSet {
	base, cancel := context.WithCancelCause(parent)
	set := &executionAdmissionSet{
		parent: parent,
		cancel: cancel,
		leases: make([]ExecutionAdmissionLease, 0, capacity),
		stops:  make([]func() bool, 0, capacity),
	}
	set.ctx = context.WithValue(base, executionAdmissionContextKey{}, set)
	return set
}

func (s *executionAdmissionSet) add(lease ExecutionAdmissionLease) error {
	if s == nil || lease.Context == nil || lease.Release == nil {
		if lease.Release != nil {
			lease.Release()
		}
		return ErrArtifactUnavailable
	}
	s.leases = append(s.leases, lease)
	s.stops = append(s.stops, context.AfterFunc(lease.Context, func() {
		cause := context.Cause(lease.Context)
		if cause == nil {
			cause = lease.Context.Err()
		}
		if cause == nil {
			cause = context.Canceled
		}
		s.cancel(cause)
	}))
	if err := executionContextError(lease.Context); err != nil {
		s.cancel(err)
		return err
	}
	return nil
}

func (s *executionAdmissionSet) Release() {
	if s == nil {
		return
	}
	s.release.Do(func() {
		for index := len(s.stops) - 1; index >= 0; index-- {
			s.stops[index]()
		}
		for index := len(s.leases) - 1; index >= 0; index-- {
			s.leases[index].Release()
		}
		s.cancel(nil)
	})
}

func (s *executionAdmissionSet) executionError() error {
	if s == nil {
		return ErrExecutionInvalid
	}
	parentCause := context.Cause(s.parent)
	groupCause := context.Cause(s.ctx)
	independent := make([]error, 0, len(s.leases))
	for _, lease := range s.leases {
		if lease.Context == nil {
			return ErrArtifactUnavailable
		}
		cause := context.Cause(lease.Context)
		if cause == nil {
			cause = lease.Context.Err()
		}
		if cause != nil && (parentCause == nil || !errorMatchesContextCancellation(s.parent, cause)) {
			independent = append(independent, cause)
		}
	}
	if groupCause != nil && (parentCause == nil || !errorMatchesContextCancellation(s.parent, groupCause)) {
		return errors.Join(ErrArtifactUnavailable, groupCause)
	}
	if len(independent) > 0 {
		cause := errors.Join(independent...)
		s.cancel(cause)
		return errors.Join(ErrArtifactUnavailable, cause)
	}
	if groupCause != nil {
		return contextCancellationError(s.parent)
	}
	if parentCause != nil {
		return contextCancellationError(s.parent)
	}
	return nil
}

func sameCancellationCause(left, right error) bool {
	return left != nil && right != nil && errors.Is(left, right) && errors.Is(right, left)
}

func errorMatchesContextCancellation(ctx context.Context, err error) bool {
	if ctx == nil || err == nil {
		return false
	}
	cause := context.Cause(ctx)
	if cause == nil {
		return false
	}
	if sameCancellationCause(err, cause) {
		return true
	}
	return ctx.Err() != nil && !errors.Is(err, ErrArtifactUnavailable) &&
		errors.Is(err, cause) && errors.Is(err, ctx.Err())
}

func contextCancellationError(ctx context.Context) error {
	if ctx == nil {
		return ErrExecutionInvalid
	}
	cause := context.Cause(ctx)
	state := ctx.Err()
	if cause == nil {
		return state
	}
	if state != nil && !errors.Is(cause, state) {
		return errors.Join(state, cause)
	}
	return cause
}

// acquireExecutionSet holds every exact artifact that contributes executable
// behavior until the result crosses the final Host release fence. This also
// protects cache hits, where no provider/filter call would otherwise acquire a
// runtime lease.
func (r *ExecutionRuntime) acquireExecutionSet(
	ctx context.Context,
	plan QueryPlan,
	filters []preparedResultFilter,
) ([]preparedResultFilter, []ResultFilterExecutionTrace, context.Context, func(), error) {
	requirements := map[Artifact]executionAdmissionRequirement{
		plan.Query.Artifact: {artifact: plan.Query.Artifact, queryOwner: true},
	}
	for _, filter := range filters {
		artifact := filter.registration.Artifact
		requirement := requirements[artifact]
		requirement.artifact = artifact
		requirements[artifact] = requirement
	}

	ordered := make([]executionAdmissionRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		ordered = append(ordered, requirement)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return artifactBefore(ordered[i].artifact, ordered[j].artifact)
	})

	admissions := newExecutionAdmissionSet(ctx, len(ordered))
	evidence := make([]ResultFilterExecutionTrace, 0)
	for _, requirement := range ordered {
		lease, err := r.acquireExactExecution(ctx, requirement.artifact)
		if err != nil {
			for _, filter := range filters {
				if filter.registration.Artifact == requirement.artifact {
					evidence = append(evidence, resultFilterExecutionTrace(
						filter.registration, ResultFilterTraceUnavailable, 0,
					))
				}
			}
			admissions.Release()
			if contextErr := executionContextError(ctx); contextErr != nil && errorMatchesContextCancellation(ctx, err) {
				return nil, evidence, ctx, func() {}, contextErr
			}
			if requirement.queryOwner {
				return nil, evidence, ctx, func() {}, errors.Join(ErrArtifactUnavailable, err)
			}
			// fail_open only controls ordinary plugin callback failures. A selected
			// filter whose exact runtime admission cannot be held is a Host fence and
			// must fail closed before provider code executes.
			return nil, evidence, ctx, func() {}, errors.Join(ErrDependencyDenied, err)
		}
		if err := admissions.add(lease); err != nil {
			admissions.Release()
			if contextErr := executionContextError(ctx); contextErr != nil && errorMatchesContextCancellation(ctx, err) {
				return nil, evidence, ctx, func() {}, contextErr
			}
			if requirement.queryOwner {
				return nil, evidence, ctx, func() {}, errors.Join(ErrArtifactUnavailable, err)
			}
			return nil, evidence, ctx, func() {}, errors.Join(ErrDependencyDenied, err)
		}
	}
	if err := executionContextError(admissions.ctx); err != nil {
		admissions.Release()
		return nil, evidence, ctx, func() {}, err
	}
	return filters, evidence, admissions.ctx, admissions.Release, nil
}

func (r *ExecutionRuntime) acquireExactExecution(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
	if err := r.registry.requireArtifactAdmitted(artifact); err != nil {
		return ExecutionAdmissionLease{}, err
	}
	lease, err := r.acquireExecutionAdmission(ctx, artifact)
	if err != nil {
		return ExecutionAdmissionLease{}, err
	}
	if err := executionContextError(lease.Context); err != nil {
		lease.Release()
		return ExecutionAdmissionLease{}, err
	}
	// Admission can close while a lease is being acquired. Do not execute or
	// release cached material unless both Host gates agree on the exact tuple.
	if err := r.registry.requireArtifactAdmitted(artifact); err != nil {
		lease.Release()
		return ExecutionAdmissionLease{}, err
	}
	return lease, nil
}

func (r *ExecutionRuntime) requireFilterArtifactsAdmitted(filters []preparedResultFilter) error {
	seen := make(map[Artifact]struct{}, len(filters))
	for _, filter := range filters {
		artifact := filter.registration.Artifact
		if _, checked := seen[artifact]; checked {
			continue
		}
		seen[artifact] = struct{}{}
		if err := r.registry.requireArtifactAdmitted(artifact); err != nil {
			return errors.Join(ErrDependencyDenied, err)
		}
	}
	return nil
}
