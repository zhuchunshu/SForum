package queryregistry

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

func (r *ExecutionRuntime) Execute(ctx context.Context, request PlanRequest) (result QueryResult, err error) {
	started := time.Now()
	traceQueryID := unplannedExecutionTraceQueryID(request.QueryID)
	if r != nil {
		traceQueryID = knownExecutionTraceQueryID(r.registry, request.QueryID)
	}
	trace := ExecutionTrace{QueryID: traceQueryID, Stage: "plan", Outcome: TraceOutcomeInternal}
	defer func() {
		trace.Duration = time.Since(started)
		trace.Outcome = executionTraceOutcome(err)
		if err == nil {
			trace.Rows = len(result.Rows)
			trace.Outcome = TraceOutcomeAllowed
		}
		if r != nil && r.trace != nil {
			r.trace.AppendExecutionTrace(boundExecutionTrace(trace))
		}
	}()
	if r == nil || r.registry == nil || r.providers == nil || r.schemas == nil || ctx == nil {
		return QueryResult{}, ErrExecutionInvalid
	}
	ctx, cancelExecution := context.WithTimeout(ctx, r.timeout)
	defer cancelExecution()

	plan, err := r.registry.Plan(ctx, request)
	if err != nil {
		return QueryResult{}, err
	}
	populateExecutionTrace(&trace, plan)
	filters, matchEvidence, err := r.matchingFiltersWithEvidence(plan.Query)
	trace.ResultFilters = append(trace.ResultFilters, matchEvidence...)
	if err != nil {
		trace.Stage = "dependency"
		return QueryResult{}, err
	}
	trace.Stage = "admission"
	filters, admissionEvidence, releaseExecutionSet, err := r.acquireExecutionSet(ctx, plan, filters)
	trace.ResultFilters = append(trace.ResultFilters, admissionEvidence...)
	if err != nil {
		return QueryResult{}, err
	}
	defer releaseExecutionSet()
	filterPlan := resultFilterPlanDigest(filters)
	if plan.Pagination.Mode != PaginationNone {
		for _, filter := range filters {
			if len(filter.registration.IdentityFields) == 0 ||
				!selectedIdentityFields(plan.Fields, filter.registration.IdentityFields) {
				trace.Stage = "filter_identity"
				return QueryResult{}, fmt.Errorf("%w: paginated result filter %s requires selected identity fields",
					ErrContractInsufficient, filter.registration.ID)
			}
		}
	}
	trace.Stage = "provider_resolve"
	binding, err := r.providers.resolveQueryProvider(ctx, cloneQueryPlan(plan))
	if err != nil {
		return QueryResult{}, errors.Join(ErrProviderUnavailable, err)
	}
	binding, err = normalizeProviderBinding(binding)
	if err != nil || !providerBindingMatchesPlan(binding, plan) {
		return QueryResult{}, ErrProviderUnavailable
	}
	trace.ProviderDigest = binding.ProviderDigest
	trace.FilterPlan = filterPlan
	trace.FilterCount = len(filters)
	cacheKey := executionCacheKey(plan, filterPlan, binding.ProviderDigest)
	cacheTags := executionCacheTags(plan, filterPlan, binding.ProviderDigest)
	if err := r.registry.validateCursorExecution(plan, binding.ProviderDigest, filterPlan); err != nil {
		trace.Stage = "cursor_execution"
		return QueryResult{}, err
	}
	// Frozen ManifestQuery cannot declare result-filter cache tags or purity.
	// Caching already-filtered rows would therefore retain external/filter-owned
	// state with no authoritative invalidation path. Keep such executions
	// uncached until that contract exists.
	cacheEligible := r.cache != nil && len(cacheTags) > 0 && len(filters) == 0

	if cacheEligible {
		trace.Stage = "cache_load"
		cached, found, cacheErr := r.cache.LoadQueryResult(ctx, cacheKey)
		switch {
		case cacheErr != nil:
			trace.CacheStatus = "load_error"
		case found:
			trace.CacheStatus = "hit"
			trace.Stage = "cache_permission"
			if err := r.registry.RecheckBeforeRelease(ctx, plan, request.Permission); err != nil {
				return QueryResult{}, err
			}
			trace.Stage = "cache_validate"
			cachedResult, err := r.validateCachedResult(
				ctx, plan, filterPlan, binding.ProviderDigest, cacheKey, cacheTags, cached,
			)
			if err != nil {
				return QueryResult{}, err
			}
			if err := validatePaginatedFilterIdentities(plan, cachedResult.Rows, filters); err != nil {
				return QueryResult{}, ErrCachePoisoned
			}
			releasedResult := cloneQueryResult(cachedResult)
			trace.Stage = "release"
			if err := ctx.Err(); err != nil {
				return QueryResult{}, err
			}
			if err := r.requireFilterArtifactsAdmitted(filters); err != nil {
				return QueryResult{}, err
			}
			if err := r.registry.RecheckBeforeRelease(ctx, plan, request.Permission); err != nil {
				return QueryResult{}, err
			}
			return releasedResult, nil
		default:
			trace.CacheStatus = "miss"
		}
	} else if r.cache == nil {
		trace.CacheStatus = "disabled"
	} else if len(filters) > 0 {
		trace.CacheStatus = "filter_bypass"
	} else {
		trace.CacheStatus = "no_tags"
	}

	fetchLimit := plan.Pagination.Limit
	if plan.Pagination.Mode != PaginationNone {
		fetchLimit++
	}
	if fetchLimit < 1 || fetchLimit > maximumExecutionRows {
		return QueryResult{}, ErrExecutionInvalid
	}
	providerRequest := ProviderExecutionRequest{Plan: cloneQueryPlan(plan), FetchLimit: fetchLimit}
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	// This is the final Host authority check immediately before provider code can
	// observe the already-detached plan. The process admission lease remains held
	// through the final result release.
	trace.Stage = "provider_permission"
	if err := r.registry.RecheckBeforeRelease(ctx, plan, request.Permission); err != nil {
		cancel()
		return QueryResult{}, err
	}
	trace.Stage = "provider_execute"
	providerResult, callErr := binding.Provider.ExecuteQuery(callCtx, providerRequest)
	callContextErr := callCtx.Err()
	cancel()
	if callErr != nil {
		if callContextErr != nil {
			return QueryResult{}, callContextErr
		}
		return QueryResult{}, errors.Join(ErrProviderFailed, callErr)
	}
	if callContextErr != nil {
		return QueryResult{}, callContextErr
	}
	if len(providerResult.Rows) > fetchLimit {
		return QueryResult{}, ErrResultTooLarge
	}
	if err := r.registry.requireArtifactAdmitted(plan.Query.Artifact); err != nil {
		return QueryResult{}, err
	}

	trace.Stage = "provider_schema"
	rows, _, err := cloneRowsBounded(providerResult.Rows, r.maxResultBytes)
	if err != nil {
		return QueryResult{}, err
	}
	if err := r.validateRows(ctx, plan, rows); err != nil {
		return QueryResult{}, err
	}
	if err := validatePaginatedFilterIdentities(plan, rows, filters); err != nil {
		return QueryResult{}, err
	}
	rows, err = r.applyResultFilters(ctx, plan, request.Permission, rows, filters, &trace)
	if err != nil {
		return QueryResult{}, err
	}

	hasMore := plan.Pagination.Mode != PaginationNone && len(rows) > plan.Pagination.Limit
	if hasMore {
		rows = rows[:plan.Pagination.Limit]
	}
	rows, _, err = cloneRowsBounded(rows, r.maxResultBytes)
	if err != nil {
		return QueryResult{}, err
	}
	trace.Stage = "release_schema"
	if err := r.validateRows(ctx, plan, rows); err != nil {
		return QueryResult{}, err
	}
	page, err := r.buildResultPage(plan, hasMore, binding.ProviderDigest, filterPlan)
	if err != nil {
		return QueryResult{}, err
	}
	trace.Stage = "release_permission"
	if err := r.registry.RecheckBeforeRelease(ctx, plan, request.Permission); err != nil {
		return QueryResult{}, err
	}
	result = QueryResult{
		Rows: rows, Page: page, CacheKey: cacheKey, CacheTags: slices.Clone(cacheTags),
		Revision: plan.Revision, Digest: plan.Digest, FilterPlan: filterPlan,
		ProviderDigest: binding.ProviderDigest,
	}

	if cacheEligible {
		trace.Stage = "cache_store"
		entry := cachedResultFromRelease(plan, filterPlan, binding.ProviderDigest, cacheKey, cacheTags, result)
		entry.Rows, _, _ = cloneRowsBounded(entry.Rows, r.maxResultBytes)
		if cacheErr := r.cache.StoreQueryResult(ctx, cacheKey, entry, slices.Clone(cacheTags)); cacheErr != nil {
			trace.CacheStatus = "store_error"
		} else {
			trace.CacheStatus = "stored"
		}
	}
	releasedResult := cloneQueryResult(result)
	// Cache writes are not release authority. Recheck again so a permission or
	// snapshot change during storage cannot release stale material.
	trace.Stage = "release"
	if err := ctx.Err(); err != nil {
		return QueryResult{}, err
	}
	if err := r.requireFilterArtifactsAdmitted(filters); err != nil {
		return QueryResult{}, err
	}
	if err := r.registry.RecheckBeforeRelease(ctx, plan, request.Permission); err != nil {
		return QueryResult{}, err
	}
	return releasedResult, nil
}

func (r *ExecutionRuntime) applyResultFilters(
	ctx context.Context,
	plan QueryPlan,
	permission PermissionInput,
	rows []QueryRow,
	filters []preparedResultFilter,
	trace *ExecutionTrace,
) ([]QueryRow, error) {
	// Frozen QueryDeclaration has no result-filter cost units. Do not synthesize
	// plugin pricing: bound the representable JSON work here and re-estimate the
	// immutable query-plan cost at every callback fence below.
	budget, err := newResultFilterJSONBudget(r.maxResultBytes)
	if err != nil {
		return nil, err
	}
	current := rows
	for _, prepared := range filters {
		registration := prepared.registration
		started := time.Now()
		candidate, filterErr := r.invokeResultFilter(ctx, plan, permission, current, registration, budget, trace)
		if filterErr != nil {
			hostFailure := ctx.Err() != nil || errors.Is(filterErr, ErrDenied) ||
				errors.Is(filterErr, ErrArtifactConflict) || errors.Is(filterErr, ErrRevisionConflict) ||
				errors.Is(filterErr, ErrArtifactUnavailable) || errors.Is(filterErr, ErrContractInsufficient) ||
				errors.Is(filterErr, ErrResultInvalid) || errors.Is(filterErr, ErrResultTooLarge) || errors.Is(filterErr, ErrCostExceeded) ||
				errors.Is(filterErr, ErrInvalid) || errors.Is(filterErr, ErrExecutionInvalid)
			outcome := ResultFilterTraceFailed
			if registration.FailurePolicy == ResultFilterFailOpen && !hostFailure {
				outcome = ResultFilterTraceSkipped
			}
			trace.ResultFilters = append(trace.ResultFilters, resultFilterExecutionTrace(
				registration, outcome, time.Since(started),
			))
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			// Host permission, snapshot, and exact-runtime failures are never a
			// plugin-controlled fail-open decision.
			if hostFailure {
				return nil, filterErr
			}
			if registration.FailurePolicy == ResultFilterFailOpen {
				continue
			}
			if errors.Is(filterErr, context.Canceled) || errors.Is(filterErr, context.DeadlineExceeded) ||
				errors.Is(filterErr, ErrArtifactUnavailable) || errors.Is(filterErr, ErrArtifactConflict) ||
				errors.Is(filterErr, ErrDenied) || errors.Is(filterErr, ErrResultInvalid) ||
				errors.Is(filterErr, ErrResultTooLarge) {
				return nil, filterErr
			}
			return nil, errors.Join(ErrProviderFailed, filterErr)
		}
		trace.ResultFilters = append(trace.ResultFilters, resultFilterExecutionTrace(
			registration, ResultFilterTraceApplied, time.Since(started),
		))
		current = candidate
	}
	return current, nil
}

func (r *ExecutionRuntime) invokeResultFilter(
	ctx context.Context,
	plan QueryPlan,
	permission PermissionInput,
	current []QueryRow,
	registration ResultFilterRegistration,
	budget *resultJSONBudget,
	trace *ExecutionTrace,
) ([]QueryRow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	trace.Stage = "filter_admission"
	if err := r.registry.requireArtifactAdmitted(registration.Artifact); err != nil {
		return nil, errors.Join(ErrDependencyDenied, err)
	}
	inputRows, _, err := cloneRowsBoundedWithBudget(current, r.maxResultBytes, budget)
	if err != nil {
		return nil, err
	}
	trace.Stage = "filter_execute"
	filterCtx, cancel := context.WithTimeout(ctx, registration.Timeout)
	// Filters receive result data, so Host authority is refreshed after preparing
	// detached input and immediately before invocation. The exact filter lease
	// was acquired before cache lookup and remains held.
	if err := r.registry.RecheckBeforeRelease(ctx, plan, permission); err != nil {
		cancel()
		return nil, err
	}
	if err := r.registry.requireArtifactAdmitted(registration.Artifact); err != nil {
		cancel()
		return nil, errors.Join(ErrDependencyDenied, err)
	}
	candidateResult, filterErr := registration.Filter.FilterQueryResult(filterCtx, ResultFilterRequest{
		Plan: cloneQueryPlan(plan), Rows: inputRows,
	})
	filterContextErr := filterCtx.Err()
	cancel()
	if filterErr == nil && filterContextErr != nil {
		filterErr = filterContextErr
	}
	// Callback failure policy never controls Host authority. Recheck the exact
	// filter artifact, immutable plan/cost, and live permission even when the
	// plugin returned an ordinary error that would otherwise be fail-open.
	if fenceErr := r.recheckAfterResultFilterCallback(ctx, plan, permission, registration.Artifact); fenceErr != nil {
		return nil, fenceErr
	}
	var candidate []QueryRow
	if filterErr == nil {
		candidate, _, filterErr = cloneRowsBoundedWithBudget(candidateResult.Rows, r.maxResultBytes, budget)
	}
	if filterErr == nil {
		filterErr = preserveFilterCardinalityAndOrder(current, candidate, registration.IdentityFields)
	}
	if filterErr == nil {
		trace.Stage = "filter_schema"
		filterErr = r.validateRowsWithBudget(ctx, plan, candidate, budget)
	}
	// Output cloning and schema validation are Host callbacks/work too. A filter's
	// fail-open policy cannot hide authority drift that occurs during those steps.
	if fenceErr := r.recheckAfterResultFilterCallback(ctx, plan, permission, registration.Artifact); fenceErr != nil {
		return nil, fenceErr
	}
	return candidate, filterErr
}

func (r *ExecutionRuntime) recheckAfterResultFilterCallback(
	ctx context.Context,
	plan QueryPlan,
	permission PermissionInput,
	artifact Artifact,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.registry.requireArtifactAdmitted(artifact); err != nil {
		return errors.Join(ErrDependencyDenied, err)
	}
	if err := r.registry.RecheckBeforeRelease(ctx, plan, permission); err != nil {
		return err
	}
	// The permission/cost callback above may itself race a filter drain.
	if err := r.registry.requireArtifactAdmitted(artifact); err != nil {
		return errors.Join(ErrDependencyDenied, err)
	}
	return ctx.Err()
}

func selectedIdentityFields(fields, identities []string) bool {
	selected := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		selected[field] = struct{}{}
	}
	for _, identity := range identities {
		if _, ok := selected[identity]; !ok {
			return false
		}
	}
	return true
}

func (r *ExecutionRuntime) acquireExecutionAdmission(ctx context.Context, artifact Artifact) (func(), error) {
	if validCoreArtifactSeal(artifact) {
		return func() {}, nil
	}
	if r == nil || r.admission == nil {
		return nil, ErrArtifactUnavailable
	}
	release, err := r.admission.AcquireQueryExecution(ctx, artifact)
	if err != nil || release == nil {
		if err == nil {
			err = ErrArtifactUnavailable
		}
		return nil, errors.Join(ErrArtifactUnavailable, err)
	}
	return release, nil
}

func cloneQueryResult(input QueryResult) QueryResult {
	input.Rows, _, _ = cloneRowsBounded(input.Rows, maximumResultBytes)
	input.CacheTags = slices.Clone(input.CacheTags)
	return input
}

func populateExecutionTrace(trace *ExecutionTrace, plan QueryPlan) {
	trace.QueryID = plan.Query.ID
	trace.ContractVersion = plan.Query.ContractVersion
	trace.PlanVersion = plan.Query.PlanVersion
	trace.ResultSchema = plan.Query.ResultSchema
	trace.ExtensionID = plan.Query.Artifact.ExtensionID
	trace.ExtensionVersion = plan.Query.Artifact.ExtensionVersion
	trace.ArtifactDigest = plan.Query.Artifact.PackageDigest
	trace.Revision = plan.Revision
	trace.SnapshotDigest = plan.Digest
	trace.ShapeDigest = plan.ShapeDigest
	trace.CostUnits = plan.Cost.Units
	trace.CostMaximum = plan.Cost.Maximum
	trace.PageMode = plan.Pagination.Mode
	trace.PageLimit = plan.Pagination.Limit
}

func executionTraceOutcome(err error) string {
	switch {
	case err == nil:
		return TraceOutcomeAllowed
	case errors.Is(err, context.DeadlineExceeded):
		return TraceOutcomeDeadline
	case errors.Is(err, context.Canceled):
		return TraceOutcomeCancelled
	case errors.Is(err, ErrDenied):
		return TraceOutcomeDenied
	case errors.Is(err, ErrCostExceeded):
		return TraceOutcomeCostExceeded
	case errors.Is(err, ErrCachePoisoned):
		return TraceOutcomeCachePoisoned
	case errors.Is(err, ErrDependencyDenied):
		return TraceOutcomeDependencyDenied
	case errors.Is(err, ErrProviderUnavailable):
		return TraceOutcomeProviderMissing
	case errors.Is(err, ErrProviderFailed):
		return TraceOutcomeProviderFailure
	case errors.Is(err, ErrResultTooLarge):
		return TraceOutcomeResultTooLarge
	case errors.Is(err, ErrResultInvalid):
		return TraceOutcomeSchemaInvalid
	case errors.Is(err, ErrArtifactUnavailable):
		return TraceOutcomeRuntimeStale
	case errors.Is(err, ErrArtifactConflict), errors.Is(err, ErrRevisionConflict):
		return TraceOutcomeSnapshotStale
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrExecutionInvalid), errors.Is(err, ErrCursorInvalid),
		errors.Is(err, ErrContractInsufficient):
		return TraceOutcomeInvalid
	default:
		return TraceOutcomeInternal
	}
}
