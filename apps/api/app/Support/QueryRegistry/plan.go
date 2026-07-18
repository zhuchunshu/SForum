package queryregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Plan validates a typed query request against the active immutable snapshot,
// runs the Host permission recheck, and returns an execution-ready plan with
// cache isolation metadata. This package never executes the plan.
func (r *Registry) Plan(ctx context.Context, request PlanRequest) (QueryPlan, error) {
	if r == nil {
		return QueryPlan{}, ErrInvalid
	}
	if len(request.ResultFilters) > 0 {
		// 冻结 ManifestQuery 没有 resultFilters 字段；不得 silently 当 filters 执行。
		return QueryPlan{}, fmt.Errorf("%w: ManifestQuery lacks resultFilters; add a dedicated resultFilters declaration before planning post-result filters", ErrContractInsufficient)
	}
	if request.MaxCost < 0 || request.MaxCost > maxCostValue {
		return QueryPlan{}, ErrInvalid
	}
	queryID := strings.ToLower(strings.TrimSpace(request.QueryID))
	if !idPattern.MatchString(queryID) {
		return QueryPlan{}, ErrInvalid
	}
	state := r.load()
	contribution, ok := state.queries[queryID]
	if !ok {
		return QueryPlan{}, ErrNotFound
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return QueryPlan{}, err
	}
	if request.PlanVersion != "" && strings.TrimSpace(request.PlanVersion) != contribution.PlanVersion {
		return QueryPlan{}, fmt.Errorf("%w: plan version mismatch", ErrConflict)
	}
	if request.ResultSchema != "" && strings.TrimSpace(request.ResultSchema) != contribution.ResultSchema {
		return QueryPlan{}, fmt.Errorf("%w: result schema mismatch", ErrConflict)
	}

	fields, err := selectNames(contribution.Fields, request.Fields, maxPlanFields, true)
	if err != nil {
		return QueryPlan{}, err
	}
	relations, err := selectNames(contribution.Relations, request.Relations, maxPlanRelations, false)
	if err != nil {
		return QueryPlan{}, err
	}
	// 第三方 executable relation 仍不是本切片 join/SQL 面；Host-only relations 保持。
	if contribution.Handler != "" && len(relations) > 0 && !validCoreArtifactSeal(contribution.Artifact) {
		return QueryPlan{}, fmt.Errorf("%w: third-party executable queries cannot select relations", ErrContractInsufficient)
	}
	filters, err := selectFilters(contribution.Filters, request.Filters)
	if err != nil {
		return QueryPlan{}, err
	}
	sorts, err := selectExecutableSorts(contribution, request.Sorts)
	if err != nil {
		return QueryPlan{}, err
	}
	pagination, err := selectPagination(contribution.Pagination, request.Pagination)
	if err != nil {
		return QueryPlan{}, err
	}
	if contribution.Handler != "" && pagination.Mode != PaginationNone {
		// offset/cursor 续页依赖稳定行身份；强制把 identity fields 纳入选择集。
		fields, err = ensureIdentityFieldsSelected(fields, contribution.IdentityFields)
		if err != nil {
			return QueryPlan{}, err
		}
	}
	locale, err := normalizeLocale(request.Locale)
	if err != nil {
		return QueryPlan{}, err
	}
	scope, err := normalizeScope(request.Scope)
	if err != nil {
		return QueryPlan{}, err
	}
	actorFingerprint, policyFingerprint, err := normalizePermissionFingerprints(request.Permission)
	if err != nil {
		return QueryPlan{}, err
	}
	if contribution.PermissionPolicy != PermissionPolicyPublic &&
		(actorFingerprint == "" || policyFingerprint == "") {
		return QueryPlan{}, fmt.Errorf("%w: non-public queries require Host actor and policy fingerprints", ErrDenied)
	}
	var cursor CursorClaims
	if pagination.Mode == PaginationCursor && pagination.Cursor != "" {
		cursor, err = r.decodeCursorForPlan(
			state, contribution, pagination, actorFingerprint, policyFingerprint, locale, scope,
		)
		if err != nil {
			return QueryPlan{}, err
		}
		pagination.Offset = cursor.Offset
		pagination.Limit = cursor.Limit
	}

	// Shape digest excludes cursor offset so the same query shape keeps a stable
	// identity across pages; cursor validation still binds query/plan/shape.
	shapeDigest := planShapeDigest(contribution, fields, relations, filters, sorts, PaginationPlan{
		Mode: pagination.Mode, Limit: pagination.Limit,
	})
	if cursor.ShapeDigest != "" && cursor.ShapeDigest != shapeDigest {
		return QueryPlan{}, ErrCursorInvalid
	}
	providers := buildProviders(contribution, fields, relations, filters, sorts)
	cost, err := r.estimateCost(contribution, fields, relations, filters, sorts, pagination, request.MaxCost)
	if err != nil {
		return QueryPlan{}, err
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return QueryPlan{}, err
	}
	if cost.Units > cost.Maximum {
		return QueryPlan{}, fmt.Errorf("%w: cost %d exceeds maximum %d", ErrCostExceeded, cost.Units, cost.Maximum)
	}

	claim := PermissionClaim{
		QueryID:          contribution.ID,
		ContractVersion:  contribution.ContractVersion,
		PlanVersion:      contribution.PlanVersion,
		Entity:           contribution.Entity,
		PermissionPolicy: contribution.PermissionPolicy,
		ResultSchema:     contribution.ResultSchema,
		ShapeDigest:      shapeDigest,
		Artifact:         contribution.Artifact,
		Locale:           locale,
		Scope:            scope,
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return QueryPlan{}, err
	}
	if err := authorizePlan(ctx, request.Permission, claim); err != nil {
		return QueryPlan{}, err
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return QueryPlan{}, err
	}
	// Host callbacks may inspect external policy state and must not be able to
	// swap the registry behind a plan that is about to be returned.
	if r.load() != state {
		return QueryPlan{}, ErrArtifactConflict
	}

	plan := QueryPlan{
		SchemaVersion:     SchemaVersion,
		Revision:          state.revision,
		Digest:            state.digest,
		ShapeDigest:       shapeDigest,
		CacheTags:         append([]string(nil), contribution.CacheTags...),
		Query:             cloneContribution(contribution),
		Fields:            fields,
		Relations:         relations,
		Filters:           filters,
		Sorts:             sorts,
		Pagination:        pagination,
		Cost:              cost,
		RequestedMaximum:  request.MaxCost,
		Recheck:           claim,
		Providers:         providers,
		Locale:            locale,
		Scope:             scope,
		ActorFingerprint:  actorFingerprint,
		PolicyFingerprint: policyFingerprint,
	}
	plan.CacheKey = planCacheKey(state, plan)
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return QueryPlan{}, err
	}
	return cloneQueryPlan(plan), nil
}

// RecheckBeforeRelease performs the second Host-owned permission recheck using
// the exact claim frozen into the plan. It also verifies the active snapshot
// still matches the plan digest so a concurrent ReplaceAll cannot release
// results against a stale authorization decision.
//
// Host must supply a fresh PermissionInput; fingerprints alone never authorize.
// PermissionPolicy always comes from plan.Recheck, never from the caller.
func (r *Registry) RecheckBeforeRelease(ctx context.Context, plan QueryPlan, input PermissionInput) error {
	if r == nil {
		return ErrInvalid
	}
	actorFingerprint, policyFingerprint, err := normalizePermissionFingerprints(input)
	if err != nil {
		return err
	}
	if plan.Recheck.PermissionPolicy != PermissionPolicyPublic &&
		(actorFingerprint == "" || policyFingerprint == "") {
		return fmt.Errorf("%w: non-public query release requires Host actor and policy fingerprints", ErrDenied)
	}
	if actorFingerprint != plan.ActorFingerprint || policyFingerprint != plan.PolicyFingerprint {
		return fmt.Errorf("%w: permission projection changed before result release", ErrArtifactConflict)
	}
	state := r.load()
	if plan.Digest != state.digest || plan.Revision != state.revision {
		return ErrArtifactConflict
	}
	contribution, ok := state.queries[plan.Recheck.QueryID]
	if !ok || contribution.Artifact != plan.Recheck.Artifact ||
		contribution.ContractVersion != plan.Recheck.ContractVersion ||
		contribution.PlanVersion != plan.Recheck.PlanVersion ||
		contribution.PermissionPolicy != plan.Recheck.PermissionPolicy ||
		contribution.ResultSchema != plan.Recheck.ResultSchema {
		return ErrArtifactConflict
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return err
	}
	if err := r.validatePlanForRelease(state, contribution, plan); err != nil {
		return err
	}
	// 二次放行只信任计划中冻结的声明策略与 Host callback。回调期间若
	// registry 被替换，旧结果不能越过最终响应边界。
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return err
	}
	if err := authorizePlan(ctx, input, plan.Recheck); err != nil {
		return err
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return err
	}
	if r.load() != state {
		return ErrArtifactConflict
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return err
	}
	return nil
}

func authorizePlan(ctx context.Context, input PermissionInput, claim PermissionClaim) error {
	switch claim.PermissionPolicy {
	case PermissionPolicyPublic:
		// 公开查询仍允许 Host 注入 recheck；若提供则必须通过。
		if input.Recheck != nil {
			if err := input.Recheck.AuthorizeQuery(ctx, claim); err != nil {
				if err == ErrDenied {
					return ErrDenied
				}
				return fmt.Errorf("%w: %v", ErrDenied, err)
			}
		}
		return nil
	case PermissionPolicyLogin:
		if !input.Authenticated {
			return ErrDenied
		}
	default:
		// 权限键策略：仅 Host callback 可授权，PolicyFingerprint 不能单独放行。
	}
	if input.Recheck == nil {
		return fmt.Errorf("%w: host permission recheck callback is required", ErrDenied)
	}
	if err := input.Recheck.AuthorizeQuery(ctx, claim); err != nil {
		if err == ErrDenied {
			return ErrDenied
		}
		return fmt.Errorf("%w: %v", ErrDenied, err)
	}
	return nil
}

func selectNames(allowlist, requested []string, limit int, defaultAll bool) ([]string, error) {
	if len(requested) > limit {
		return nil, ErrInvalid
	}
	allowed := map[string]bool{}
	for _, name := range allowlist {
		allowed[name] = true
	}
	if len(requested) == 0 {
		if defaultAll {
			return append([]string(nil), allowlist...), nil
		}
		return nil, nil
	}
	result := make([]string, 0, len(requested))
	seen := map[string]bool{}
	for _, name := range requested {
		name = strings.TrimSpace(name)
		if !allowed[name] || seen[name] {
			return nil, ErrInvalid
		}
		seen[name] = true
		result = append(result, name)
	}
	return result, nil
}

func selectFilters(allowlist []string, requested []FilterValue) ([]FilterValue, error) {
	if len(requested) > maxPlanFilters {
		return nil, ErrInvalid
	}
	allowed := map[string]bool{}
	for _, name := range allowlist {
		allowed[name] = true
	}
	result := make([]FilterValue, 0, len(requested))
	seen := map[string]bool{}
	for _, item := range requested {
		field := strings.TrimSpace(item.Field)
		value := strings.TrimSpace(item.Value)
		if !allowed[field] || seen[field] || value == "" || len(value) > maxFilterValueLength {
			return nil, ErrInvalid
		}
		seen[field] = true
		result = append(result, FilterValue{Field: field, Value: value})
	}
	return result, nil
}

func selectSorts(allowlist []string, requested []SortValue) ([]SortValue, error) {
	if len(requested) > maxPlanSorts {
		return nil, ErrInvalid
	}
	allowed := map[string]bool{}
	for _, name := range allowlist {
		allowed[name] = true
	}
	if len(requested) == 0 {
		return nil, nil
	}
	result := make([]SortValue, 0, len(requested))
	seen := map[string]bool{}
	for _, item := range requested {
		field := strings.TrimSpace(item.Field)
		if !allowed[field] || seen[field] {
			return nil, ErrInvalid
		}
		seen[field] = true
		result = append(result, SortValue{Field: field, Descending: item.Descending})
	}
	return result, nil
}

func selectExecutableSorts(contribution QueryContribution, requested []SortValue) ([]SortValue, error) {
	if contribution.Handler == "" {
		return selectSorts(contribution.Sort, requested)
	}
	if len(requested) == 0 {
		// 空 sort 使用声明 DefaultSort，保证 executable 分页顺序确定性。
		return append([]SortValue(nil), contribution.DefaultSort...), nil
	}
	sorts, err := selectSorts(contribution.Sort, requested)
	if err != nil {
		return nil, err
	}
	return appendIdentitySortTieBreakers(sorts, contribution.DefaultSort, contribution.IdentityFields)
}

func appendIdentitySortTieBreakers(
	sorts []SortValue,
	defaultSort []SortValue,
	identityFields []string,
) ([]SortValue, error) {
	selected := map[string]bool{}
	for _, item := range sorts {
		selected[item.Field] = true
	}
	defaultByField := map[string]SortValue{}
	for _, item := range defaultSort {
		defaultByField[item.Field] = item
	}
	result := append([]SortValue(nil), sorts...)
	for _, identity := range identityFields {
		if selected[identity] {
			continue
		}
		if len(result) >= maxPlanSorts {
			return nil, ErrInvalid
		}
		if item, ok := defaultByField[identity]; ok {
			result = append(result, item)
		} else {
			result = append(result, SortValue{Field: identity})
		}
		selected[identity] = true
	}
	return result, nil
}

func ensureIdentityFieldsSelected(fields, identityFields []string) ([]string, error) {
	if len(identityFields) == 0 {
		return nil, ErrInvalid
	}
	selected := map[string]bool{}
	for _, field := range fields {
		selected[field] = true
	}
	result := append([]string(nil), fields...)
	for _, identity := range identityFields {
		if selected[identity] {
			continue
		}
		if len(result) >= maxPlanFields {
			return nil, ErrInvalid
		}
		result = append(result, identity)
		selected[identity] = true
	}
	return result, nil
}

func selectPagination(mode string, request PaginationRequest) (PaginationPlan, error) {
	limit := request.Limit
	offset := request.Offset
	cursor := strings.TrimSpace(request.Cursor)
	switch mode {
	case PaginationNone:
		if offset != 0 || cursor != "" || (limit != 0 && limit != 1) {
			return PaginationPlan{}, ErrInvalid
		}
		if limit == 0 {
			limit = 1
		}
		return PaginationPlan{Mode: PaginationNone, Limit: limit}, nil
	case PaginationOffset:
		if cursor != "" {
			return PaginationPlan{}, ErrInvalid
		}
		if limit == 0 {
			limit = defaultPageLimit
		}
		if limit < 1 || limit > maximumPageLimit || offset < 0 || offset > maximumOffset {
			return PaginationPlan{}, ErrInvalid
		}
		return PaginationPlan{Mode: PaginationOffset, Offset: offset, Limit: limit}, nil
	case PaginationCursor:
		if offset != 0 {
			return PaginationPlan{}, ErrInvalid
		}
		if limit == 0 && cursor == "" {
			limit = defaultPageLimit
		}
		if limit < 0 || limit > maximumPageLimit || (cursor == "" && limit < 1) {
			return PaginationPlan{}, ErrInvalid
		}
		return PaginationPlan{Mode: PaginationCursor, Limit: limit, Cursor: cursor}, nil
	default:
		return PaginationPlan{}, ErrInvalid
	}
}

func (r *Registry) estimateCost(
	contribution QueryContribution,
	fields, relations []string,
	filters []FilterValue,
	sorts []SortValue,
	pagination PaginationPlan,
	requestedMaximum int,
) (QueryCost, error) {
	if r == nil || r.costPolicy == nil {
		return QueryCost{}, fmt.Errorf("%w: Host query cost policy is not configured", ErrContractInsufficient)
	}
	if requestedMaximum < 0 || requestedMaximum > maxCostValue {
		return QueryCost{}, ErrInvalid
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return QueryCost{}, err
	}
	cost, err := r.costPolicy.EstimateQueryCost(QueryCostInput{
		Query: cloneContribution(contribution), Fields: slices.Clone(fields), Relations: slices.Clone(relations),
		Filters: slices.Clone(filters), Sorts: slices.Clone(sorts), Pagination: pagination,
		RequestedMaximum: requestedMaximum,
	})
	if err != nil {
		return QueryCost{}, err
	}
	if err := r.requireArtifactAdmitted(contribution.Artifact); err != nil {
		return QueryCost{}, err
	}
	if cost.Units < 0 || cost.Units > maxCostValue || cost.Maximum <= 0 || cost.Maximum > maxCostValue {
		return QueryCost{}, ErrInvalid
	}
	return cost, nil
}

func (r *Registry) validatePlanForRelease(
	state *registryState,
	contribution QueryContribution,
	plan QueryPlan,
) error {
	if plan.SchemaVersion != SchemaVersion || plan.Revision != state.revision || plan.Digest != state.digest ||
		!reflect.DeepEqual(plan.Query, contribution) || plan.ShapeDigest == "" || plan.CacheKey == "" {
		return ErrArtifactConflict
	}
	// plan.Fields 已含 Host 强制补齐的 identity；不能用默认 selectNames 回放。
	fields, err := selectNames(contribution.Fields, plan.Fields, maxPlanFields, false)
	if err != nil || len(fields) == 0 || !slices.Equal(fields, plan.Fields) {
		return ErrArtifactConflict
	}
	if contribution.Handler != "" && plan.Pagination.Mode != PaginationNone {
		forced, forceErr := ensureIdentityFieldsSelected(fields, contribution.IdentityFields)
		if forceErr != nil || !slices.Equal(forced, plan.Fields) {
			return ErrArtifactConflict
		}
	}
	relations, err := selectNames(contribution.Relations, plan.Relations, maxPlanRelations, false)
	if err != nil || !slices.Equal(relations, plan.Relations) {
		return ErrArtifactConflict
	}
	if contribution.Handler != "" && len(relations) > 0 && !validCoreArtifactSeal(contribution.Artifact) {
		return ErrArtifactConflict
	}
	filters, err := selectFilters(contribution.Filters, plan.Filters)
	if err != nil || !slices.Equal(filters, plan.Filters) {
		return ErrArtifactConflict
	}
	// 回放时 plan.Sorts 已是 DefaultSort 或 caller+identity；按 allowlist 校验即可。
	sorts, err := selectSorts(contribution.Sort, plan.Sorts)
	if err != nil || !slices.Equal(sorts, plan.Sorts) {
		return ErrArtifactConflict
	}
	if contribution.Handler != "" {
		expectedSorts, sortErr := selectExecutableSorts(contribution, plan.Sorts)
		// plan.Sorts 可能已含 tie-breaker；与从空/caller 重建的结果不一定相同。
		// 这里只要求每个 identity 字段都出现在最终 sort 中。
		if sortErr != nil {
			return ErrArtifactConflict
		}
		_ = expectedSorts
		for _, identity := range contribution.IdentityFields {
			found := false
			for _, item := range plan.Sorts {
				if item.Field == identity {
					found = true
					break
				}
			}
			if !found {
				return ErrArtifactConflict
			}
		}
	}
	if plan.Pagination.Mode != contribution.Pagination {
		return ErrArtifactConflict
	}
	locale, err := normalizeLocale(plan.Locale)
	if err != nil || locale != plan.Locale {
		return ErrArtifactConflict
	}
	scope, err := normalizeScope(plan.Scope)
	if err != nil || scope != plan.Scope {
		return ErrArtifactConflict
	}
	paginationRequest := PaginationRequest{Limit: plan.Pagination.Limit, Cursor: plan.Pagination.Cursor}
	if plan.Pagination.Mode != PaginationCursor {
		paginationRequest.Offset = plan.Pagination.Offset
	}
	normalizedPagination, err := selectPagination(contribution.Pagination, paginationRequest)
	if err != nil {
		return ErrArtifactConflict
	}
	if normalizedPagination.Mode == PaginationCursor && normalizedPagination.Cursor != "" {
		claims, cursorErr := r.decodeCursorForPlan(
			state, contribution, normalizedPagination, plan.ActorFingerprint, plan.PolicyFingerprint,
			plan.Locale, plan.Scope,
		)
		if cursorErr != nil {
			return ErrArtifactConflict
		}
		if claims.ShapeDigest != plan.ShapeDigest {
			return ErrArtifactConflict
		}
		normalizedPagination.Offset = claims.Offset
		normalizedPagination.Limit = claims.Limit
	}
	shapeDigest := planShapeDigest(contribution, fields, relations, filters, sorts, PaginationPlan{
		Mode: normalizedPagination.Mode, Limit: normalizedPagination.Limit,
	})
	if normalizedPagination != plan.Pagination || shapeDigest != plan.ShapeDigest {
		return ErrArtifactConflict
	}
	expectedClaim := PermissionClaim{
		QueryID: contribution.ID, ContractVersion: contribution.ContractVersion,
		PlanVersion: contribution.PlanVersion, Entity: contribution.Entity,
		PermissionPolicy: contribution.PermissionPolicy, ResultSchema: contribution.ResultSchema,
		ShapeDigest: shapeDigest, Artifact: contribution.Artifact, Locale: locale, Scope: scope,
	}
	if plan.Recheck != expectedClaim || !slices.Equal(plan.CacheTags, contribution.CacheTags) {
		return ErrArtifactConflict
	}
	providers := buildProviders(contribution, fields, relations, filters, sorts)
	if !slices.Equal(plan.Providers, providers) {
		return ErrArtifactConflict
	}
	cost, err := r.estimateCost(contribution, fields, relations, filters, sorts, plan.Pagination, plan.RequestedMaximum)
	if errors.Is(err, ErrArtifactUnavailable) {
		return err
	}
	if err != nil || cost != plan.Cost || cost.Units > cost.Maximum || plan.CacheKey != planCacheKey(state, plan) {
		return ErrArtifactConflict
	}
	return nil
}

func buildProviders(contribution QueryContribution, fields, relations []string, filters []FilterValue, sorts []SortValue) []ProviderRef {
	result := []ProviderRef{{
		Kind: ProviderKindQuery, Name: contribution.ID,
		ContributionID: contribution.ID, Artifact: contribution.Artifact,
	}}
	for _, field := range fields {
		result = append(result, ProviderRef{
			Kind: ProviderKindField, Name: field,
			ContributionID: contribution.ID, Artifact: contribution.Artifact,
		})
	}
	for _, relation := range relations {
		result = append(result, ProviderRef{
			Kind: ProviderKindRelation, Name: relation,
			ContributionID: contribution.ID, Artifact: contribution.Artifact,
		})
	}
	for _, filter := range filters {
		result = append(result, ProviderRef{
			Kind: ProviderKindFilter, Name: filter.Field,
			ContributionID: contribution.ID, Artifact: contribution.Artifact,
		})
	}
	for _, sortValue := range sorts {
		result = append(result, ProviderRef{
			Kind: ProviderKindSort, Name: sortValue.Field,
			ContributionID: contribution.ID, Artifact: contribution.Artifact,
		})
	}
	return result
}

func planShapeDigest(
	contribution QueryContribution,
	fields, relations []string,
	filters []FilterValue,
	sorts []SortValue,
	pagination PaginationPlan,
) string {
	document := struct {
		ID           string         `json:"id"`
		PlanVersion  string         `json:"planVersion"`
		ResultSchema string         `json:"resultSchema"`
		Fields       []string       `json:"fields"`
		Relations    []string       `json:"relations"`
		Filters      []FilterValue  `json:"filters"`
		Sorts        []SortValue    `json:"sorts"`
		Pagination   PaginationPlan `json:"pagination"`
		Artifact     Artifact       `json:"artifact"`
	}{
		ID: contribution.ID, PlanVersion: contribution.PlanVersion, ResultSchema: contribution.ResultSchema,
		Fields: fields, Relations: relations, Filters: filters, Sorts: sorts, Pagination: pagination,
		Artifact: contribution.Artifact,
	}
	body, _ := json.Marshal(document)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func planCacheKey(state *registryState, plan QueryPlan) string {
	var value strings.Builder
	value.WriteString(SchemaVersion)
	value.WriteByte(0)
	value.WriteString(strconv.FormatUint(state.revision, 10))
	value.WriteByte(0)
	value.WriteString(state.digest)
	value.WriteByte(0)
	value.WriteString(strconv.FormatBool(state.safeMode))
	value.WriteByte(0)
	value.WriteString(plan.ShapeDigest)
	value.WriteByte(0)
	// 分页 offset/cursor 不进 shape，但必须隔离缓存，避免跨页污染。
	value.WriteString(plan.Pagination.Mode)
	value.WriteByte(0)
	value.WriteString(strconv.Itoa(plan.Pagination.Offset))
	value.WriteByte(0)
	value.WriteString(strconv.Itoa(plan.Pagination.Limit))
	value.WriteByte(0)
	value.WriteString(plan.Pagination.Cursor)
	value.WriteByte(0)
	value.WriteString(plan.ActorFingerprint)
	value.WriteByte(0)
	value.WriteString(plan.PolicyFingerprint)
	value.WriteByte(0)
	value.WriteString(plan.Recheck.PermissionPolicy)
	value.WriteByte(0)
	value.WriteString(plan.Locale)
	value.WriteByte(0)
	value.WriteString(plan.Scope)
	value.WriteByte(0)
	value.WriteString(strings.Join(plan.CacheTags, "\x1f"))
	value.WriteByte(0)
	// Exact providers bind cache entries to owning artifacts so a denied private
	// plan cannot be served from a public entry for the same shape.
	providers := append([]ProviderRef(nil), plan.Providers...)
	sort.Slice(providers, func(i, j int) bool {
		return providerKey(providers[i]) < providerKey(providers[j])
	})
	for _, provider := range providers {
		value.WriteString(providerKey(provider))
		value.WriteByte(0x1f)
	}
	sum := sha256.Sum256([]byte(value.String()))
	return hex.EncodeToString(sum[:])
}

func providerKey(provider ProviderRef) string {
	return provider.Kind + "\x00" + provider.Name + "\x00" + provider.ContributionID + "\x00" +
		provider.Artifact.ExtensionID + "\x00" + provider.Artifact.ExtensionVersion + "\x00" +
		provider.Artifact.PackageDigest + "\x00" + strconv.FormatInt(provider.Artifact.VersionID, 10) + "\x00" +
		provider.Artifact.RuntimeInstanceID + "\x00" +
		strconv.FormatBool(provider.Artifact.Core)
}

func normalizeLocale(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > maxLocaleLength || !localePattern.MatchString(value) {
		return "", ErrInvalid
	}
	return value, nil
}

func normalizeScope(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if len(value) > maxScopeLength || !idPattern.MatchString(value) {
		return "", ErrInvalid
	}
	return value, nil
}

func normalizePermissionFingerprints(input PermissionInput) (actor, policy string, err error) {
	actor = strings.TrimSpace(input.ActorFingerprint)
	policy = strings.TrimSpace(input.PolicyFingerprint)
	if len(actor) > maxFingerprintLength || len(policy) > maxFingerprintLength {
		return "", "", ErrInvalid
	}
	// Fingerprints are opaque Host material; reject control characters only.
	for _, value := range []string{actor, policy} {
		for _, r := range value {
			if r < 0x20 || r == 0x7f {
				return "", "", ErrInvalid
			}
		}
	}
	return actor, policy, nil
}
