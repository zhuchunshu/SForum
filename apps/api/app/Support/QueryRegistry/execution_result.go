package queryregistry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

const resultCacheSchemaVersion = "sforum.query-result-cache@1"

func cloneRowsBounded(rows []QueryRow, maximumBytes int) ([]QueryRow, int, error) {
	return cloneRowsBoundedWithBudget(rows, maximumBytes, nil)
}

func cloneRowsBoundedWithBudget(
	rows []QueryRow,
	maximumBytes int,
	cumulative *resultJSONBudget,
) ([]QueryRow, int, error) {
	if len(rows) > maximumExecutionRows {
		return nil, 0, ErrResultTooLarge
	}
	measure, err := measureRowsBounded(rows, maximumBytes)
	if err != nil {
		return nil, 0, err
	}
	if cumulative != nil {
		if err := cumulative.consumeMeasure(measure); err != nil {
			return nil, 0, err
		}
	}
	body, err := json.Marshal(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: result is not JSON-compatible", ErrResultInvalid)
	}
	if len(body) > maximumBytes {
		return nil, len(body), ErrResultTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var cloned []QueryRow
	if err := decoder.Decode(&cloned); err != nil {
		return nil, len(body), fmt.Errorf("%w: clone result", ErrResultInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, len(body), fmt.Errorf("%w: trailing result data", ErrResultInvalid)
	}
	if cloned == nil {
		cloned = []QueryRow{}
	}
	return cloned, len(body), nil
}

func (r *ExecutionRuntime) validateRows(ctx context.Context, plan QueryPlan, rows []QueryRow) error {
	return r.validateRowsWithBudget(ctx, plan, rows, nil)
}

func (r *ExecutionRuntime) validateRowsWithBudget(
	ctx context.Context,
	plan QueryPlan,
	rows []QueryRow,
	cumulative *resultJSONBudget,
) error {
	if r == nil || r.schemas == nil || len(rows) > maximumExecutionRows {
		return ErrResultInvalid
	}
	allowed := make(map[string]struct{}, len(plan.Fields)+len(plan.Relations))
	for _, field := range plan.Fields {
		allowed[field] = struct{}{}
	}
	for _, relation := range plan.Relations {
		allowed[relation] = struct{}{}
	}
	for index, row := range rows {
		if row == nil {
			return fmt.Errorf("%w: row %d is null", ErrResultInvalid, index)
		}
		for key := range row {
			if _, ok := allowed[key]; !ok {
				return fmt.Errorf("%w: row %d contains undeclared field %q", ErrResultInvalid, index, key)
			}
		}
		for _, field := range plan.Fields {
			if _, ok := row[field]; !ok {
				return fmt.Errorf("%w: row %d omits selected field %q", ErrResultInvalid, index, field)
			}
		}
		for _, relation := range plan.Relations {
			if _, ok := row[relation]; !ok {
				return fmt.Errorf("%w: row %d omits selected relation %q", ErrResultInvalid, index, relation)
			}
		}
		claim := ResultSchemaClaim{
			QueryID: plan.Query.ID, ContractVersion: plan.Query.ContractVersion,
			PlanVersion: plan.Query.PlanVersion, ResultSchema: plan.Query.ResultSchema,
			ShapeDigest: plan.ShapeDigest, Artifact: plan.Query.Artifact,
			Fields: slices.Clone(plan.Fields), Relations: slices.Clone(plan.Relations), RowIndex: index,
		}
		validatorRows, _, err := cloneRowsBoundedWithBudget([]QueryRow{row}, r.maxResultBytes, cumulative)
		if err != nil {
			return err
		}
		if err := r.schemas.ValidateQueryResult(ctx, claim, validatorRows[0]); err != nil {
			if contextErr := executionContextError(ctx); contextErr != nil {
				return contextErr
			}
			return fmt.Errorf("%w: %v", ErrResultInvalid, err)
		}
		if contextErr := executionContextError(ctx); contextErr != nil {
			return contextErr
		}
	}
	return nil
}

func preserveFilterCardinalityAndOrder(before, after []QueryRow, identityFields []string) error {
	if len(before) != len(after) {
		return fmt.Errorf("%w: result filters cannot change row cardinality", ErrResultInvalid)
	}
	for index := range before {
		for _, field := range identityFields {
			left, leftHasIdentity := before[index][field]
			right, rightHasIdentity := after[index][field]
			if !leftHasIdentity || !rightHasIdentity || !reflect.DeepEqual(left, right) {
				return fmt.Errorf("%w: result filters cannot reorder or replace row identity", ErrResultInvalid)
			}
		}
	}
	return nil
}

func validatePaginatedFilterIdentities(
	plan QueryPlan,
	rows []QueryRow,
	filters []preparedResultFilter,
) error {
	if plan.Pagination.Mode == PaginationNone {
		return nil
	}
	for _, filter := range filters {
		fields := filter.registration.IdentityFields
		seen := make(map[string]struct{}, len(rows))
		for index, row := range rows {
			identity := make([]any, len(fields))
			for fieldIndex, field := range fields {
				value, ok := row[field]
				if !ok {
					return fmt.Errorf("%w: row %d omits pagination identity %q", ErrResultInvalid, index, field)
				}
				identity[fieldIndex] = value
			}
			encoded, err := json.Marshal(identity)
			if err != nil {
				return fmt.Errorf("%w: pagination identity is not JSON-compatible", ErrResultInvalid)
			}
			key := string(encoded)
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("%w: result filter %s received duplicate pagination identity", ErrResultInvalid, filter.registration.ID)
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}

func (r *ExecutionRuntime) matchingFilters(query QueryContribution) ([]preparedResultFilter, error) {
	result, _, err := r.matchingFiltersWithEvidence(query)
	return result, err
}

func (r *ExecutionRuntime) matchingFiltersWithEvidence(
	query QueryContribution,
) ([]preparedResultFilter, []ResultFilterExecutionTrace, error) {
	result := make([]preparedResultFilter, 0)
	evidence := make([]ResultFilterExecutionTrace, 0)
	ownerVersion, err := semver.StrictNewVersion(query.Artifact.ExtensionVersion)
	if err != nil {
		return nil, evidence, ErrDependencyDenied
	}
	candidates, err := r.resultFilterCandidates(query)
	if err != nil {
		return nil, evidence, err
	}
	for _, filter := range candidates {
		registration := filter.registration
		if registration.QueryID != query.ID {
			continue
		}
		// Safe Mode is Host-owned and bypasses every third-party runtime even if a
		// stale admission callback still reports it healthy. A stale in-flight plan
		// is rejected later by the registry revision fence; a fresh Core plan must
		// not be blocked by a fail-closed plugin filter.
		if r.registry.load().safeMode && !validCoreArtifactSeal(registration.Artifact) {
			evidence = append(evidence, resultFilterExecutionTrace(
				registration, ResultFilterTraceUnavailable, 0,
			))
			continue
		}
		if registration.QueryContractVersion != query.ContractVersion || registration.QueryPlanVersion != query.PlanVersion {
			evidence = append(evidence, resultFilterExecutionTrace(
				registration, ResultFilterTraceContractMismatch, 0,
			))
			if registration.FailurePolicy == ResultFilterFailOpen {
				continue
			}
			return nil, evidence, fmt.Errorf("%w: result filter %s targets a stale query contract", ErrDependencyDenied, registration.ID)
		}
		crossPlugin := registration.Artifact.ExtensionID != query.Artifact.ExtensionID
		if crossPlugin {
			if filter.constraint == nil || registration.Dependency.ExtensionID != query.Artifact.ExtensionID ||
				!filter.constraint.Check(ownerVersion) {
				evidence = append(evidence, resultFilterExecutionTrace(
					registration, ResultFilterTraceDependencyMismatch, 0,
				))
				if registration.FailurePolicy == ResultFilterFailOpen {
					continue
				}
				return nil, evidence, fmt.Errorf("%w: result filter %s does not declare a matching query-owner dependency", ErrDependencyDenied, registration.ID)
			}
		} else if filter.constraint != nil &&
			(registration.Dependency.ExtensionID != query.Artifact.ExtensionID || !filter.constraint.Check(ownerVersion)) {
			evidence = append(evidence, resultFilterExecutionTrace(
				registration, ResultFilterTraceDependencyMismatch, 0,
			))
			if registration.FailurePolicy == ResultFilterFailOpen {
				continue
			}
			return nil, evidence, fmt.Errorf("%w: result filter %s self dependency is incompatible", ErrDependencyDenied, registration.ID)
		}
		if len(registration.IdentityFields) == 0 {
			evidence = append(evidence, resultFilterExecutionTrace(
				registration, ResultFilterTraceContractMismatch, 0,
			))
			if registration.FailurePolicy == ResultFilterFailOpen {
				continue
			}
			return nil, evidence, fmt.Errorf("%w: result filter %s has no Host-derived row identity",
				ErrContractInsufficient, registration.ID)
		}
		result = append(result, filter)
	}
	return result, evidence, nil
}

// resultFilterCandidates merges static bootstrap registrations with optional
// snapshot-backed sources. Dynamic filters resolve callables at match time.
func (r *ExecutionRuntime) resultFilterCandidates(query QueryContribution) ([]preparedResultFilter, error) {
	if r == nil {
		return nil, ErrExecutionInvalid
	}
	if r.filterSource == nil {
		return r.filters, nil
	}
	registrations, err := r.filterSource.ResultFiltersFor(query)
	if err != nil {
		return nil, errors.Join(ErrExecutionInvalid, err)
	}
	if len(registrations) > maximumResultFilters {
		return nil, ErrExecutionInvalid
	}
	if len(registrations) == 0 {
		return r.filters, nil
	}
	dynamic, err := prepareResultFilters(registrations)
	if err != nil {
		return nil, err
	}
	if len(r.filters) == 0 {
		return dynamic, nil
	}
	if len(r.filters)+len(dynamic) > maximumResultFilters {
		return nil, ErrExecutionInvalid
	}
	seen := make(map[string]struct{}, len(r.filters)+len(dynamic))
	merged := make([]preparedResultFilter, 0, len(r.filters)+len(dynamic))
	for _, filter := range r.filters {
		seen[filter.registration.ID] = struct{}{}
		merged = append(merged, filter)
	}
	for _, filter := range dynamic {
		if _, exists := seen[filter.registration.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate result filter %s", ErrExecutionInvalid, filter.registration.ID)
		}
		seen[filter.registration.ID] = struct{}{}
		merged = append(merged, filter)
	}
	// 保持与 prepareResultFilters 相同的优先级/artifact/ID 排序。
	sort.Slice(merged, func(i, j int) bool {
		left, right := merged[i].registration, merged[j].registration
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if left.Artifact != right.Artifact {
			return artifactBefore(left.Artifact, right.Artifact)
		}
		return left.ID < right.ID
	})
	return merged, nil
}

func resultFilterPlanDigest(filters []preparedResultFilter) string {
	var value strings.Builder
	value.WriteString(SchemaVersion)
	value.WriteString("\x00result-filters\x00")
	for _, filter := range filters {
		item := filter.registration
		value.WriteString(item.ID)
		value.WriteByte(0)
		value.WriteString(item.ContractVersion)
		value.WriteByte(0)
		value.WriteString(strconv.Itoa(item.Priority))
		value.WriteByte(0)
		value.WriteString(providerKey(ProviderRef{Kind: "result_filter", Name: item.ID, ContributionID: item.QueryID, Artifact: item.Artifact}))
		value.WriteByte(0)
		value.WriteString(item.Dependency.ExtensionID)
		value.WriteByte(0)
		value.WriteString(item.Dependency.VersionConstraint)
		value.WriteByte(0)
		value.WriteString(strings.Join(item.IdentityFields, "\x1e"))
		value.WriteByte(0)
		value.WriteString(item.FailurePolicy)
		value.WriteByte(0)
		value.WriteString(strconv.FormatInt(item.Timeout.Milliseconds(), 10))
		value.WriteByte(0x1f)
	}
	digest := sha256.Sum256([]byte(value.String()))
	return hex.EncodeToString(digest[:])
}

func executionCacheKey(plan QueryPlan, filterPlan, providerDigest string) string {
	digest := sha256.Sum256([]byte(
		resultCacheSchemaVersion + "\x00" + plan.CacheKey + "\x00" + filterPlan + "\x00" + providerDigest,
	))
	return hex.EncodeToString(digest[:])
}

func executionCacheTags(plan QueryPlan, filterPlan, providerDigest string) []string {
	if len(plan.CacheTags) == 0 {
		return []string{}
	}
	isolation := sha256.Sum256([]byte(
		resultCacheSchemaVersion + "\x00" + plan.CacheKey + "\x00" + strconv.FormatUint(plan.Revision, 10) + "\x00" + plan.Digest + "\x00" +
			cursorArtifactDigest(plan.Query.Artifact) + "\x00" + plan.ActorFingerprint + "\x00" +
			plan.PolicyFingerprint + "\x00" + plan.Locale + "\x00" + plan.Scope + "\x00" + filterPlan +
			"\x00" + providerDigest,
	))
	prefix := "query:" + hex.EncodeToString(isolation[:16]) + ":"
	result := make([]string, 0, len(plan.CacheTags)*2)
	for _, tag := range plan.CacheTags {
		sharedTag := sharedSemanticCacheTag(plan.Query.Artifact.ExtensionID, tag)
		opaqueTag := strings.TrimPrefix(sharedTag, "query:shared:")
		// The shared tag deliberately excludes actor, locale, request shape, page,
		// provider, and exact artifact identity. Stable owner identity prevents
		// cross-plugin collisions while one semantic mutation still evicts every
		// isolated variant across versions of the same owner.
		result = append(result, sharedTag, prefix+opaqueTag)
	}
	return result
}

func cachedResultFromRelease(
	plan QueryPlan,
	filterPlan, providerDigest, cacheKey string,
	tags []string,
	result QueryResult,
) CachedQueryResult {
	return CachedQueryResult{
		SchemaVersion: resultCacheSchemaVersion, CacheKey: cacheKey,
		RegistryRevision: plan.Revision, RegistryDigest: plan.Digest, ShapeDigest: plan.ShapeDigest,
		FilterPlan: filterPlan, QueryID: plan.Query.ID, ContractVersion: plan.Query.ContractVersion,
		PlanVersion: plan.Query.PlanVersion, ResultSchema: plan.Query.ResultSchema, Artifact: plan.Query.Artifact,
		ProviderDigest: providerDigest,
		Page:           result.Page, Rows: result.Rows, CacheTags: slices.Clone(tags),
	}
}

func (r *ExecutionRuntime) validateCachedResult(
	ctx context.Context,
	plan QueryPlan,
	filterPlan, providerDigest, cacheKey string,
	tags []string,
	cached CachedQueryResult,
) (QueryResult, error) {
	if cached.SchemaVersion != resultCacheSchemaVersion || cached.CacheKey != cacheKey ||
		cached.RegistryRevision != plan.Revision || cached.RegistryDigest != plan.Digest ||
		cached.ShapeDigest != plan.ShapeDigest || cached.FilterPlan != filterPlan ||
		cached.QueryID != plan.Query.ID || cached.ContractVersion != plan.Query.ContractVersion ||
		cached.PlanVersion != plan.Query.PlanVersion || cached.ResultSchema != plan.Query.ResultSchema ||
		cached.Artifact != plan.Query.Artifact || cached.ProviderDigest != providerDigest ||
		!slices.Equal(cached.CacheTags, tags) {
		return QueryResult{}, ErrCachePoisoned
	}
	expectedPage, err := r.buildResultPage(plan, cached.Page.HasMore, providerDigest, filterPlan)
	if err != nil || cached.Page != expectedPage {
		return QueryResult{}, ErrCachePoisoned
	}
	rows, _, err := cloneRowsBounded(cached.Rows, r.maxResultBytes)
	if err != nil || len(rows) > plan.Pagination.Limit {
		return QueryResult{}, ErrCachePoisoned
	}
	if err := r.validateRows(ctx, plan, rows); err != nil {
		if contextErr := executionContextError(ctx); contextErr != nil {
			return QueryResult{}, contextErr
		}
		return QueryResult{}, ErrCachePoisoned
	}
	if contextErr := executionContextError(ctx); contextErr != nil {
		return QueryResult{}, contextErr
	}
	return QueryResult{
		Rows: rows, Page: cached.Page, CacheKey: cacheKey, CacheTags: slices.Clone(tags), CacheHit: true,
		Revision: plan.Revision, Digest: plan.Digest, FilterPlan: filterPlan, ProviderDigest: providerDigest,
	}, nil
}

func (r *ExecutionRuntime) buildResultPage(
	plan QueryPlan,
	hasMore bool,
	providerDigest, filterPlan string,
) (QueryResultPage, error) {
	page := QueryResultPage{
		Mode: plan.Pagination.Mode, Offset: plan.Pagination.Offset,
		Limit: plan.Pagination.Limit, HasMore: hasMore,
	}
	if !hasMore {
		return page, nil
	}
	switch plan.Pagination.Mode {
	case PaginationOffset:
		if plan.Pagination.Offset > maximumOffset-plan.Pagination.Limit {
			return QueryResultPage{}, ErrResultTooLarge
		}
		page.NextOffset = plan.Pagination.Offset + plan.Pagination.Limit
	case PaginationCursor:
		cursor, err := r.registry.EncodeNextCursor(plan, providerDigest, filterPlan)
		if err != nil {
			return QueryResultPage{}, err
		}
		page.NextCursor = cursor
	case PaginationNone:
		return QueryResultPage{}, ErrResultInvalid
	default:
		return QueryResultPage{}, ErrResultInvalid
	}
	return page, nil
}
