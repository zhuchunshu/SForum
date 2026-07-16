package queryregistry

import (
	"context"
	"errors"
	"sort"
)

type executionAdmissionRequirement struct {
	artifact   Artifact
	queryOwner bool
	failClosed bool
}

// acquireExecutionSet holds every exact artifact that contributes executable
// behavior until the result crosses the final Host release fence. This also
// protects cache hits, where no provider/filter call would otherwise acquire a
// runtime lease.
func (r *ExecutionRuntime) acquireExecutionSet(
	ctx context.Context,
	plan QueryPlan,
	filters []preparedResultFilter,
) ([]preparedResultFilter, []ResultFilterExecutionTrace, func(), error) {
	requirements := map[Artifact]executionAdmissionRequirement{
		plan.Query.Artifact: {artifact: plan.Query.Artifact, queryOwner: true, failClosed: true},
	}
	for _, filter := range filters {
		artifact := filter.registration.Artifact
		requirement := requirements[artifact]
		requirement.artifact = artifact
		if filter.registration.FailurePolicy == ResultFilterFailClosed {
			requirement.failClosed = true
		}
		requirements[artifact] = requirement
	}

	ordered := make([]executionAdmissionRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		ordered = append(ordered, requirement)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return artifactBefore(ordered[i].artifact, ordered[j].artifact)
	})

	releases := make([]func(), 0, len(ordered))
	releaseAll := func() {
		for index := len(releases) - 1; index >= 0; index-- {
			releases[index]()
		}
	}
	excluded := make(map[Artifact]struct{})
	evidence := make([]ResultFilterExecutionTrace, 0)
	for _, requirement := range ordered {
		release, err := r.acquireExactExecution(ctx, requirement.artifact)
		if err != nil {
			for _, filter := range filters {
				if filter.registration.Artifact == requirement.artifact {
					evidence = append(evidence, resultFilterExecutionTrace(
						filter.registration, ResultFilterTraceUnavailable, 0,
					))
				}
			}
			if requirement.queryOwner || requirement.failClosed {
				releaseAll()
				if requirement.queryOwner {
					return nil, evidence, func() {}, errors.Join(ErrArtifactUnavailable, err)
				}
				return nil, evidence, func() {}, errors.Join(ErrDependencyDenied, err)
			}
			excluded[requirement.artifact] = struct{}{}
			continue
		}
		releases = append(releases, release)
	}

	active := make([]preparedResultFilter, 0, len(filters))
	for _, filter := range filters {
		if _, omitted := excluded[filter.registration.Artifact]; omitted {
			continue
		}
		active = append(active, filter)
	}
	return active, evidence, releaseAll, nil
}

func (r *ExecutionRuntime) acquireExactExecution(ctx context.Context, artifact Artifact) (func(), error) {
	if err := r.registry.requireArtifactAdmitted(artifact); err != nil {
		return nil, err
	}
	release, err := r.acquireExecutionAdmission(ctx, artifact)
	if err != nil {
		return nil, err
	}
	// Admission can close while a lease is being acquired. Do not execute or
	// release cached material unless both Host gates agree on the exact tuple.
	if err := r.registry.requireArtifactAdmitted(artifact); err != nil {
		release()
		return nil, err
	}
	return release, nil
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
