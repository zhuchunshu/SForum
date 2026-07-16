package mediaregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

func (r *Registry) Plan(ctx context.Context, request PlanRequest, authorizer Authorizer) (Plan, error) {
	if r == nil || ctx == nil || authorizer == nil {
		return Plan{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	request.Kind = strings.ToLower(strings.TrimSpace(request.Kind))
	request.Purpose = normalizePurpose(request.Purpose)
	request.Permission = strings.TrimSpace(request.Permission)
	request.Actor.ID = strings.TrimSpace(request.Actor.ID)
	request.Actor.PermissionFingerprint = strings.TrimSpace(request.Actor.PermissionFingerprint)
	if !validPlanKind(request.Kind) || request.Purpose == "" || request.Purpose == "*" || !validPermission(request.Permission) ||
		request.Actor.ID == "" || !validPlainString(request.Actor.ID, maxActorBytes) || request.Actor.PermissionFingerprint == "" ||
		!validPlainString(request.Actor.PermissionFingerprint, maxStringBytes) {
		return Plan{}, ErrInvalid
	}
	ingestionPlan := request.Kind == PlanUpload || request.Kind == PlanProcess
	source, err := normalizeSource(request.Source, ingestionPlan)
	if err != nil {
		return Plan{}, err
	}
	state := r.load()
	policy := lifecyclePolicy(request.Purpose, request.Permission)
	if ingestionPlan {
		var found bool
		policy, found = selectPolicy(state, request.Purpose)
		if !found {
			return Plan{}, ErrPolicyUnavailable
		}
	}
	upload, err := validateUploadFacts(request.Kind, request.Upload, source, policy)
	if err != nil {
		return Plan{}, err
	}
	baseAuthorization := AuthorizationRequest{Actor: request.Actor, Permission: request.Permission, PlanKind: request.Kind, Purpose: request.Purpose, SourceID: source.ID}
	if !authorizer.Authorize(ctx, baseAuthorization) {
		return Plan{}, ErrPermissionDenied
	}
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	checked := map[string]struct{}{request.Permission: {}}
	if ingestionPlan {
		if policy.RequiredPermission != request.Permission && !authorizer.Authorize(ctx, AuthorizationRequest{
			Actor: request.Actor, Permission: policy.RequiredPermission, PlanKind: request.Kind, Purpose: request.Purpose, SourceID: source.ID,
		}) {
			return Plan{}, ErrPermissionDenied
		}
		if err := ctx.Err(); err != nil {
			return Plan{}, err
		}
		checked[policy.RequiredPermission] = struct{}{}
	}

	processors := selectProcessors(state, request.Kind, request.Purpose, source.MIME)
	variants := selectVariants(state, request.Purpose, processors)
	if len(variants) > policy.Budget.MaxVariants {
		return Plan{}, ErrBudgetExceeded
	}
	steps := buildPlanSteps(processors, variants)
	for _, step := range steps {
		permission := step.Processor.RequiredPermission
		if _, exists := checked[permission]; exists {
			continue
		}
		if !authorizer.Authorize(ctx, AuthorizationRequest{Actor: request.Actor, Permission: permission, PlanKind: request.Kind, Purpose: request.Purpose, SourceID: source.ID}) {
			return Plan{}, ErrPermissionDenied
		}
		if err := ctx.Err(); err != nil {
			return Plan{}, err
		}
		checked[permission] = struct{}{}
	}
	plan := Plan{SchemaVersion: SchemaVersion, Revision: state.revision, RegistryDigest: state.digest, SafeMode: state.safeMode,
		Kind: request.Kind, Purpose: request.Purpose, Permission: request.Permission, Actor: request.Actor, Source: source, Upload: upload,
		Policy: clonePolicyContribution(policy), Steps: steps, Conflicts: relevantConflicts(state.conflicts, policy.Purpose, steps), OriginalFallback: source}
	if r.load() != state {
		return Plan{}, ErrPlanStale
	}
	plan.Digest = computePlanDigest(plan)
	return clonePlan(plan), nil
}

// ValidatePlan is required immediately before and after provider execution. It
// rejects caller mutation, Registry publication drift, and revoked permission.
func (r *Registry) ValidatePlan(ctx context.Context, plan Plan, authorizer Authorizer) error {
	if r == nil || ctx == nil || authorizer == nil || plan.SchemaVersion != SchemaVersion {
		return ErrPlanStale
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	state := r.load()
	if plan.Revision != state.revision || plan.RegistryDigest != state.digest || plan.SafeMode != state.safeMode ||
		plan.Digest == "" || plan.Digest != computePlanDigest(plan) || plan.Source != plan.OriginalFallback {
		return ErrPlanStale
	}
	expected, err := r.Plan(ctx, PlanRequest{Kind: plan.Kind, Purpose: plan.Purpose, Permission: plan.Permission,
		Actor: plan.Actor, Source: plan.Source, Upload: plan.Upload}, authorizer)
	if err != nil {
		return err
	}
	if expected.Digest != plan.Digest {
		return ErrPlanStale
	}
	return nil
}

func (r *Registry) canonicalPlanStep(step PlanStep) (PlanStep, bool) {
	if r == nil {
		return PlanStep{}, false
	}
	state := r.load()
	processor, found := state.processors[step.Processor.ID]
	if !found || !equalProcessorContribution(step.Processor, processor) {
		return PlanStep{}, false
	}
	result := PlanStep{ID: step.ID, Processor: cloneProcessorContribution(processor)}
	for _, value := range step.Variants {
		variant, found := state.variants[value.ID]
		if !found || !equalVariantContribution(value, variant) || variant.ProcessorID != processor.ID {
			return PlanStep{}, false
		}
		result.Variants = append(result.Variants, variant)
	}
	if len(step.Variants) == 0 && step.Variants != nil {
		result.Variants = []VariantContribution{}
	}
	return result, true
}

func equalPolicyContribution(left, right MIMEPolicyContribution) bool {
	leftDeclaration, leftErr := normalizePolicy(left.MIMEPolicyDeclaration)
	rightDeclaration, rightErr := normalizePolicy(right.MIMEPolicyDeclaration)
	return leftErr == nil && rightErr == nil && artifactIdentityEqual(left.Artifact, right.Artifact) && reflect.DeepEqual(leftDeclaration, rightDeclaration)
}

func equalProcessorContribution(left, right ProcessorContribution) bool {
	leftDeclaration, leftErr := normalizeProcessor(left.ProcessorDeclaration)
	rightDeclaration, rightErr := normalizeProcessor(right.ProcessorDeclaration)
	return leftErr == nil && rightErr == nil && artifactIdentityEqual(left.Artifact, right.Artifact) && reflect.DeepEqual(leftDeclaration, rightDeclaration)
}

func equalVariantContribution(left, right VariantContribution) bool {
	leftDeclaration, leftErr := normalizeVariant(left.VariantDeclaration)
	rightDeclaration, rightErr := normalizeVariant(right.VariantDeclaration)
	return leftErr == nil && rightErr == nil && artifactIdentityEqual(left.Artifact, right.Artifact) && leftDeclaration == rightDeclaration
}

func selectPolicy(state *registryState, purpose string) (MIMEPolicyContribution, bool) {
	values := state.policyGroups[policyConflictKey(purpose)]
	key := purpose
	if len(values) == 0 {
		values = state.policyGroups[policyConflictKey("*")]
		key = "*"
	}
	if len(values) == 0 {
		return MIMEPolicyContribution{}, false
	}
	ref, _ := selectedRef(state, ConflictMIMEPolicy, key, policyRefs(values))
	for _, value := range values {
		if policyRef(value) == ref {
			return clonePolicyContribution(value), true
		}
	}
	return MIMEPolicyContribution{}, false
}

func selectProcessors(state *registryState, planKind, purpose, mediaType string) []ProcessorContribution {
	stages := stageSetForPlan(planKind)
	composed := []ProcessorContribution{}
	exclusive := map[string][]ProcessorContribution{}
	for _, value := range state.processors {
		if !stages[value.Stage] || !purposeCompatible(value.Purpose, purpose) || !matchesAnyMIME(value.MIMEs, mediaType) {
			continue
		}
		if value.Mode == ProcessorCompose {
			composed = append(composed, cloneProcessorContribution(value))
			continue
		}
		key := value.Stage + "/" + value.Slot
		exclusive[key] = append(exclusive[key], value)
	}
	for _, candidates := range exclusive {
		exact := candidates[:0]
		for _, candidate := range candidates {
			if candidate.Purpose == purpose {
				exact = append(exact, candidate)
			}
		}
		if len(exact) > 0 {
			candidates = exact
		}
		sort.Slice(candidates, func(i, j int) bool { return processorBefore(candidates[i], candidates[j]) })
		declaredKey := processorConflictKey(candidates[0])
		refs := processorRefs(candidates)
		winner, _ := selectedRef(state, ConflictProcessor, declaredKey, refs)
		for _, candidate := range candidates {
			if processorRef(candidate) == winner {
				composed = append(composed, cloneProcessorContribution(candidate))
				break
			}
		}
	}
	sort.Slice(composed, func(i, j int) bool {
		rankI, rankJ := stageRank(composed[i].Stage), stageRank(composed[j].Stage)
		if rankI != rankJ {
			return rankI < rankJ
		}
		return processorBefore(composed[i], composed[j])
	})
	return composed
}

func selectVariants(state *registryState, purpose string, processors []ProcessorContribution) []VariantContribution {
	selectedProcessors := map[string]struct{}{}
	for _, processor := range processors {
		if processor.Stage == StageTransform {
			selectedProcessors[processor.ID] = struct{}{}
		}
	}
	byName := map[string][]VariantContribution{}
	for _, value := range state.variants {
		if _, ok := selectedProcessors[value.ProcessorID]; ok && purposeCompatible(value.Purpose, purpose) {
			byName[value.Name] = append(byName[value.Name], value)
		}
	}
	result := []VariantContribution{}
	for _, candidates := range byName {
		exact := candidates[:0]
		for _, candidate := range candidates {
			if candidate.Purpose == purpose {
				exact = append(exact, candidate)
			}
		}
		if len(exact) > 0 {
			candidates = exact
		}
		sort.Slice(candidates, func(i, j int) bool { return variantBefore(candidates[i], candidates[j]) })
		key := variantConflictKey(candidates[0].Purpose, candidates[0].Name)
		refs := variantRefs(candidates)
		winner, _ := selectedRef(state, ConflictVariant, key, refs)
		for _, candidate := range candidates {
			if variantRef(candidate) == winner {
				result = append(result, candidate)
				break
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return variantBefore(result[i], result[j])
	})
	return result
}

func buildPlanSteps(processors []ProcessorContribution, variants []VariantContribution) []PlanStep {
	byProcessor := map[string][]VariantContribution{}
	for _, variant := range variants {
		byProcessor[variant.ProcessorID] = append(byProcessor[variant.ProcessorID], variant)
	}
	result := make([]PlanStep, 0, len(processors))
	for _, processor := range processors {
		attached := byProcessor[processor.ID]
		if processor.Stage == StageTransform && len(attached) == 0 {
			continue
		}
		result = append(result, PlanStep{ID: processor.Stage + ":" + processor.ID, Processor: cloneProcessorContribution(processor), Variants: slicesCloneVariants(attached)})
	}
	return result
}

func validateUploadFacts(planKind string, input UploadFacts, source SourceAsset, policy MIMEPolicyContribution) (UploadFacts, error) {
	// 历史资产的交付、保留和删除不重新走当前上传准入。站点收紧 MIME、
	// 文件名或大小策略后，既有 immutable source 仍必须能完成生命周期操作。
	if planKind != PlanUpload && planKind != PlanProcess {
		return UploadFacts{
			BatchFileCount: 1, DeclaredMIME: source.MIME, DetectedMIMEs: []string{source.MIME},
		}, nil
	}
	if input.BatchFileCount <= 0 || input.BatchFileCount > policy.Budget.MaxFiles || source.SizeBytes > policy.Budget.MaxFileBytes {
		return UploadFacts{}, ErrBudgetExceeded
	}
	if len(source.Filename) > policy.Budget.MaxFilenameBytes {
		return UploadFacts{}, ErrBudgetExceeded
	}
	if len(input.DetectedMIMEs) == 0 || len(input.DetectedMIMEs) > policy.Budget.MaxMIMECandidates {
		return UploadFacts{}, ErrBudgetExceeded
	}
	seen := map[string]struct{}{}
	detected := make([]string, 0, len(input.DetectedMIMEs))
	containsSource := false
	for _, raw := range input.DetectedMIMEs {
		value, err := normalizeExactMIME(raw)
		if err != nil {
			return UploadFacts{}, ErrInvalid
		}
		if value == source.MIME {
			containsSource = true
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		detected = append(detected, value)
	}
	if !containsSource {
		return UploadFacts{}, ErrMIMEConfusion
	}
	sort.Strings(detected)
	input.DetectedMIMEs = detected
	if input.DeclaredMIME != "" {
		value, err := normalizeExactMIME(input.DeclaredMIME)
		if err != nil {
			return UploadFacts{}, ErrInvalid
		}
		input.DeclaredMIME = value
	}
	// StrictDeclaredMIME 要求调用方显式给出 declared MIME；省略与混淆同等拒绝。
	if policy.StrictDeclaredMIME {
		if input.DeclaredMIME == "" {
			return UploadFacts{}, ErrMIMEConfusion
		}
		if input.DeclaredMIME != source.MIME && !allowedMIMEAlias(policy.MIMEAliases, input.DeclaredMIME, source.MIME) {
			return UploadFacts{}, ErrMIMEConfusion
		}
	}
	if matchesAnyMIME(policy.DeniedMIMEs, source.MIME) || !matchesAnyMIME(policy.AllowedMIMEs, source.MIME) {
		return UploadFacts{}, ErrMediaRejected
	}
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(source.Filename)), ".")
	if len(policy.AllowedExtensions) > 0 && !containsString(policy.AllowedExtensions, extension) {
		return UploadFacts{}, ErrMediaRejected
	}
	if input.DecompressedBytes < 0 {
		return UploadFacts{}, ErrInvalid
	}
	if input.Archive && input.DecompressedBytes <= 0 || policy.RequireExpandedSize && input.DecompressedBytes <= 0 {
		return UploadFacts{}, ErrBudgetExceeded
	}
	if input.DecompressedBytes > 0 {
		if input.DecompressedBytes > policy.Budget.MaxDecompressedBytes || source.SizeBytes <= 0 ||
			input.DecompressedBytes > source.SizeBytes*policy.Budget.MaxDecompressionRatio {
			return UploadFacts{}, ErrBudgetExceeded
		}
	}
	return input, nil
}

func normalizeSource(input SourceAsset, enforceUploadFilename bool) (SourceAsset, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Digest = strings.ToLower(strings.TrimSpace(input.Digest))
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.Filename = strings.TrimSpace(input.Filename)
	mimeType, err := normalizeExactMIME(input.MIME)
	if err != nil {
		return SourceAsset{}, err
	}
	input.MIME = mimeType
	validHistoricalFilename := input.Filename == "" || validPlainString(input.Filename, hostMaximumBudget.MaxFilenameBytes)
	if enforceUploadFilename {
		validHistoricalFilename = validFilename(input.Filename)
	}
	if input.ID == "" || !validPlainString(input.ID, maxActorBytes) || !digestPattern.MatchString(input.Digest) || !input.Immutable || input.SizeBytes <= 0 ||
		(input.Kind != SourceOriginal && input.Kind != SourceSourceOfTruth) || !validHistoricalFilename {
		return SourceAsset{}, ErrInvalid
	}
	return input, nil
}

func lifecyclePolicy(purpose, permission string) MIMEPolicyContribution {
	packageDigest := sha256.Sum256([]byte(SchemaVersion + "\x00host-lifecycle-policy"))
	impactDigest := sha256.Sum256([]byte(SchemaVersion + "\x00host-lifecycle-policy-impact"))
	artifact, _ := NewCoreArtifact("core.media.lifecycle", "1.0.0", hex.EncodeToString(packageDigest[:]), hex.EncodeToString(impactDigest[:]))
	return MIMEPolicyContribution{MIMEPolicyDeclaration: MIMEPolicyDeclaration{
		ID: "core.media.lifecycle", ContractVersion: "core.media.lifecycle@1", Purpose: purpose,
		RequiredPermission: permission, AllowedMIMEs: []string{"*/*"}, Budget: MaximumBudget(),
	}, Artifact: artifact}
}

func validFilename(value string) bool {
	if value == "" || !utf8.ValidString(value) || len(value) > hostMaximumBudget.MaxFilenameBytes || value == "." || value == ".." || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") || strings.ContainsAny(value, "/\\:") {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func computePlanDigest(plan Plan) string {
	plan.Digest = ""
	// Plan deliberately rejects public JSON serialization. Actor and original
	// filename also carry json:"-", so bind them explicitly into the private
	// digest without making them loggable.
	type digestPlan Plan
	payload := struct {
		Plan     digestPlan `json:"plan"`
		Actor    Actor      `json:"actor"`
		Filename string     `json:"filename"`
	}{Plan: digestPlan(plan), Actor: plan.Actor, Filename: plan.Source.Filename}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
func clonePlan(value Plan) Plan {
	value.Upload.DetectedMIMEs = append([]string(nil), value.Upload.DetectedMIMEs...)
	value.Policy = clonePolicyContribution(value.Policy)
	steps := make([]PlanStep, len(value.Steps))
	copy(steps, value.Steps)
	value.Steps = steps
	for i := range value.Steps {
		value.Steps[i] = clonePlanStep(value.Steps[i])
	}
	value.Conflicts = cloneConflicts(value.Conflicts)
	return value
}
func clonePlanStep(value PlanStep) PlanStep {
	value.Processor = cloneProcessorContribution(value.Processor)
	value.Variants = slicesCloneVariants(value.Variants)
	return value
}
func slicesCloneVariants(values []VariantContribution) []VariantContribution {
	return append([]VariantContribution(nil), values...)
}

// relevantConflicts 仅保留与实际选中 policy / plan steps 相关的冲突，
// 保持 buildConflicts 的排序并按 plan 规模有界。
func relevantConflicts(values []ProviderConflict, policyPurpose string, steps []PlanStep) []ProviderConflict {
	if len(values) == 0 {
		return nil
	}
	processorKeys := make(map[string]struct{}, len(steps))
	variantKeys := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		if step.Processor.Mode == ProcessorExclusive {
			processorKeys[processorConflictKey(step.Processor)] = struct{}{}
		}
		for _, variant := range step.Variants {
			variantKeys[variantConflictKey(variant.Purpose, variant.Name)] = struct{}{}
		}
	}
	result := make([]ProviderConflict, 0, len(steps)+1)
	for _, conflict := range values {
		switch conflict.Family {
		case ConflictMIMEPolicy:
			if conflict.Key == policyPurpose {
				result = append(result, cloneConflict(conflict))
			}
		case ConflictProcessor:
			if _, ok := processorKeys[conflict.Key]; ok {
				result = append(result, cloneConflict(conflict))
			}
		case ConflictVariant:
			if _, ok := variantKeys[conflict.Key]; ok {
				result = append(result, cloneConflict(conflict))
			}
		}
	}
	return result
}
func policyRefs(values []MIMEPolicyContribution) []ProviderRef {
	result := make([]ProviderRef, len(values))
	for i, v := range values {
		result[i] = policyRef(v)
	}
	return result
}
func processorRefs(values []ProcessorContribution) []ProviderRef {
	result := make([]ProviderRef, len(values))
	for i, v := range values {
		result[i] = processorRef(v)
	}
	return result
}
func variantRefs(values []VariantContribution) []ProviderRef {
	result := make([]ProviderRef, len(values))
	for i, v := range values {
		result[i] = variantRef(v)
	}
	return result
}

func validPlanKind(value string) bool {
	switch value {
	case PlanUpload, PlanProcess, PlanDelivery, PlanRetention, PlanDelete:
		return true
	default:
		return false
	}
}
func stageSetForPlan(kind string) map[string]bool {
	result := map[string]bool{}
	switch kind {
	case PlanUpload, PlanProcess:
		for _, stage := range []string{StageValidate, StageScan, StageMetadata, StageTransform, StageCDN, StageRetention} {
			result[stage] = true
		}
	case PlanDelivery:
		result[StageCDN] = true
	case PlanRetention:
		result[StageRetention] = true
	case PlanDelete:
		result[StageRetention] = true
		result[StageBeforeDelete] = true
		result[StageAfterDelete] = true
	}
	return result
}
func stageRank(value string) int {
	switch value {
	case StageValidate:
		return 10
	case StageScan:
		return 20
	case StageMetadata:
		return 30
	case StageTransform:
		return 40
	case StageCDN:
		return 50
	case StageRetention:
		return 60
	case StageBeforeDelete:
		return 70
	case StageAfterDelete:
		return 80
	default:
		return 100
	}
}
func matchesAnyMIME(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if pattern == "*/*" || pattern == value || strings.HasSuffix(pattern, "/*") && strings.HasPrefix(value, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}
func allowedMIMEAlias(values []MIMEAlias, declared, detected string) bool {
	for _, value := range values {
		if value.Declared == declared && value.Detected == detected {
			return true
		}
	}
	return false
}
func containsString(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}
