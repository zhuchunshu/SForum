package queryregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// ExecutableProviderMaterial is one exact query-owner executable binding. The
// Provider callable is private publication material and never enters JSON.
type ExecutableProviderMaterial struct {
	QueryID         string
	ContractVersion string
	PlanVersion     string
	ResultSchema    string
	Handler         string
	// ProviderDigest optionally overrides the Host-derived default digest. Empty
	// input derives a stable digest from public executable metadata.
	ProviderDigest string
	Provider       ExecutableProvider
}

// ExecutableResultFilterMaterial is one exact independent result-filter binding.
// The Filter callable is private publication material and never enters JSON.
type ExecutableResultFilterMaterial struct {
	ID                   string
	ContractVersion      string
	QueryID              string
	QueryContractVersion string
	QueryPlanVersion     string
	Handler              string
	Priority             int
	FailurePolicy        string
	TimeoutMS            int
	Dependency           *ResultFilterDependency
	// FilterDigest is Host-derived after normalization; callers leave it empty.
	FilterDigest string
	Filter       ResultFilter
}

type boundExecutableProvider struct {
	handler        string
	providerDigest string
	provider       ExecutableProvider
}

type boundExecutableFilter struct {
	handler      string
	filterDigest string
	timeout      time.Duration
	filter       ResultFilter
}

// BindExecutableRuntime attaches immutable private provider and result-filter
// material to an exact publication. Provider/filter/schema bindings must later
// publish under one Registry revision; this helper rejects missing, duplicate,
// unowned, and contract mismatches before publication.
func BindExecutableRuntime(
	publication Publication,
	providers []ExecutableProviderMaterial,
	filters []ExecutableResultFilterMaterial,
) (Publication, error) {
	artifact, err := normalizeArtifact(publication.Artifact)
	if err != nil {
		return Publication{}, ErrExecutionInvalid
	}
	result := clonePublication(publication)
	result.Artifact = artifact

	queryIndex := make(map[string]int, len(result.Queries))
	for index, raw := range result.Queries {
		declaration, declarationErr := normalizeQueryDeclaration(artifact, raw)
		if declarationErr != nil {
			return Publication{}, errorsJoinExecution(declarationErr)
		}
		if _, duplicate := queryIndex[declaration.ID]; duplicate {
			return Publication{}, fmt.Errorf("%w: duplicate query %s", ErrExecutionInvalid, declaration.ID)
		}
		// 绑定前清空可伪造的公开 digest 与 private material，仅接受本调用写入的材料。
		declaration.ProviderDigest = ""
		declaration.boundProvider = nil
		result.Queries[index] = declaration
		queryIndex[declaration.ID] = index
	}

	filterIndex := make(map[string]int, len(result.ResultFilters))
	for index, raw := range result.ResultFilters {
		declaration, declarationErr := normalizeResultFilterDeclaration(artifact, raw, result.Queries)
		if declarationErr != nil {
			return Publication{}, errorsJoinExecution(declarationErr)
		}
		if _, duplicate := filterIndex[declaration.ID]; duplicate {
			return Publication{}, fmt.Errorf("%w: duplicate result filter %s", ErrExecutionInvalid, declaration.ID)
		}
		declaration.FilterDigest = ""
		declaration.boundFilter = nil
		result.ResultFilters[index] = declaration
		filterIndex[declaration.ID] = index
	}

	boundQueries := make(map[string]struct{}, len(providers))
	for _, raw := range providers {
		material, materialErr := normalizeExecutableProviderMaterial(artifact, raw)
		if materialErr != nil {
			return Publication{}, materialErr
		}
		index, found := queryIndex[material.QueryID]
		if !found {
			return Publication{}, fmt.Errorf("%w: provider has no exact query declaration", ErrExecutionInvalid)
		}
		if _, duplicate := boundQueries[material.QueryID]; duplicate {
			return Publication{}, fmt.Errorf("%w: duplicate executable provider %s", ErrExecutionInvalid, material.QueryID)
		}
		declaration := &result.Queries[index]
		if declaration.Handler == "" ||
			declaration.Handler != material.Handler ||
			declaration.ContractVersion != material.ContractVersion ||
			declaration.PlanVersion != material.PlanVersion ||
			declaration.ResultSchema != material.ResultSchema {
			return Publication{}, fmt.Errorf("%w: provider does not match query %s", ErrExecutionInvalid, material.QueryID)
		}
		declaration.ProviderDigest = material.ProviderDigest
		declaration.boundProvider = &boundExecutableProvider{
			handler: material.Handler, providerDigest: material.ProviderDigest, provider: material.Provider,
		}
		boundQueries[material.QueryID] = struct{}{}
	}

	boundFilters := make(map[string]struct{}, len(filters))
	for _, raw := range filters {
		material, materialErr := normalizeExecutableResultFilterMaterial(artifact, raw)
		if materialErr != nil {
			return Publication{}, materialErr
		}
		index, found := filterIndex[material.ID]
		if !found {
			return Publication{}, fmt.Errorf("%w: filter has no exact result-filter declaration", ErrExecutionInvalid)
		}
		if _, duplicate := boundFilters[material.ID]; duplicate {
			return Publication{}, fmt.Errorf("%w: duplicate result filter %s", ErrExecutionInvalid, material.ID)
		}
		declaration := &result.ResultFilters[index]
		if declaration.Handler != material.Handler ||
			declaration.ContractVersion != material.ContractVersion ||
			declaration.QueryID != material.QueryID ||
			declaration.QueryContractVersion != material.QueryContractVersion ||
			declaration.QueryPlanVersion != material.QueryPlanVersion ||
			declaration.Priority != material.Priority ||
			declaration.FailurePolicy != material.FailurePolicy ||
			declaration.TimeoutMS != material.TimeoutMS ||
			!resultFilterDependencyEqual(declaration.Dependency, material.Dependency) {
			return Publication{}, fmt.Errorf("%w: filter material does not match declaration %s", ErrExecutionInvalid, material.ID)
		}
		declaration.FilterDigest = material.FilterDigest
		declaration.boundFilter = &boundExecutableFilter{
			handler: material.Handler, filterDigest: material.FilterDigest,
			timeout: time.Duration(material.TimeoutMS) * time.Millisecond, filter: material.Filter,
		}
		boundFilters[material.ID] = struct{}{}
	}

	// 完整绑定：每个声明了 handler 的 query 与每条 result filter 都必须有 private material。
	for _, declaration := range result.Queries {
		if declaration.Handler == "" {
			continue
		}
		if declaration.boundProvider == nil || declaration.ProviderDigest == "" {
			return Publication{}, fmt.Errorf("%w: missing executable provider for %s", ErrExecutionInvalid, declaration.ID)
		}
	}
	for _, declaration := range result.ResultFilters {
		if declaration.boundFilter == nil || declaration.FilterDigest == "" {
			return Publication{}, fmt.Errorf("%w: missing executable result filter for %s", ErrExecutionInvalid, declaration.ID)
		}
	}
	return result, nil
}

func normalizeExecutableProviderMaterial(
	artifact Artifact,
	input ExecutableProviderMaterial,
) (ExecutableProviderMaterial, error) {
	input.QueryID = strings.ToLower(strings.TrimSpace(input.QueryID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.PlanVersion = strings.TrimSpace(input.PlanVersion)
	input.ResultSchema = strings.TrimSpace(input.ResultSchema)
	input.Handler = strings.TrimSpace(input.Handler)
	input.ProviderDigest = normalizeDigest(input.ProviderDigest)
	if !validContributionIdentity(artifact, input.QueryID, input.ContractVersion) ||
		!contractPattern.MatchString(input.PlanVersion) || !validSchemaRef(input.ResultSchema) ||
		!validExecutableHandler(artifact.ExtensionID, input.Handler) || input.Provider == nil {
		return ExecutableProviderMaterial{}, ErrExecutionInvalid
	}
	if input.ProviderDigest == "" {
		input.ProviderDigest = executableProviderDigest(artifact, input.QueryID, input.ContractVersion,
			input.PlanVersion, input.ResultSchema, input.Handler)
	}
	if !digestPattern.MatchString(input.ProviderDigest) {
		return ExecutableProviderMaterial{}, ErrExecutionInvalid
	}
	return input, nil
}

func normalizeExecutableResultFilterMaterial(
	artifact Artifact,
	input ExecutableResultFilterMaterial,
) (ExecutableResultFilterMaterial, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.QueryID = strings.ToLower(strings.TrimSpace(input.QueryID))
	input.QueryContractVersion = strings.TrimSpace(input.QueryContractVersion)
	input.QueryPlanVersion = strings.TrimSpace(input.QueryPlanVersion)
	input.Handler = strings.TrimSpace(input.Handler)
	input.FailurePolicy = strings.ToLower(strings.TrimSpace(input.FailurePolicy))
	if input.FailurePolicy == "" {
		input.FailurePolicy = ResultFilterFailClosed
	}
	if input.TimeoutMS == 0 {
		input.TimeoutMS = int(defaultFilterTimeout / time.Millisecond)
	}
	if input.Dependency != nil {
		dependency := *input.Dependency
		dependency.ExtensionID = strings.ToLower(strings.TrimSpace(dependency.ExtensionID))
		dependency.VersionConstraint = strings.TrimSpace(dependency.VersionConstraint)
		input.Dependency = &dependency
	}
	if !validContributionIdentity(artifact, input.ID, input.ContractVersion) ||
		!idPattern.MatchString(input.QueryID) || !contractPattern.MatchString(input.QueryContractVersion) ||
		!contractPattern.MatchString(input.QueryPlanVersion) ||
		!validExecutableHandler(artifact.ExtensionID, input.Handler) ||
		(input.FailurePolicy != ResultFilterFailClosed && input.FailurePolicy != ResultFilterFailOpen) ||
		input.TimeoutMS < 1 || input.TimeoutMS > int(maximumFilterTimeout/time.Millisecond) ||
		input.Filter == nil {
		return ExecutableResultFilterMaterial{}, ErrExecutionInvalid
	}
	if input.Dependency != nil {
		if !idPattern.MatchString(input.Dependency.ExtensionID) || input.Dependency.VersionConstraint == "" {
			return ExecutableResultFilterMaterial{}, ErrExecutionInvalid
		}
	}
	input.FilterDigest = executableResultFilterDigest(artifact, input)
	return input, nil
}

func executableProviderDigest(
	artifact Artifact,
	queryID, contractVersion, planVersion, resultSchema, handler string,
) string {
	material := SchemaVersion + "\x00executable-provider\x00" + artifact.ExtensionID + "\x00" +
		artifact.ExtensionVersion + "\x00" + artifact.PackageDigest + "\x00" +
		queryID + "\x00" + contractVersion + "\x00" + planVersion + "\x00" +
		resultSchema + "\x00" + handler
	digest := sha256.Sum256([]byte(material))
	return hex.EncodeToString(digest[:])
}

func executableResultFilterDigest(artifact Artifact, input ExecutableResultFilterMaterial) string {
	dependencyID, dependencyConstraint := "", ""
	if input.Dependency != nil {
		dependencyID = input.Dependency.ExtensionID
		dependencyConstraint = input.Dependency.VersionConstraint
	}
	material := SchemaVersion + "\x00executable-result-filter\x00" + artifact.ExtensionID + "\x00" +
		artifact.ExtensionVersion + "\x00" + artifact.PackageDigest + "\x00" +
		input.ID + "\x00" + input.ContractVersion + "\x00" + input.QueryID + "\x00" +
		input.QueryContractVersion + "\x00" + input.QueryPlanVersion + "\x00" +
		input.Handler + "\x00" + fmt.Sprintf("%d", input.Priority) + "\x00" +
		input.FailurePolicy + "\x00" + fmt.Sprintf("%d", input.TimeoutMS) + "\x00" +
		dependencyID + "\x00" + dependencyConstraint
	digest := sha256.Sum256([]byte(material))
	return hex.EncodeToString(digest[:])
}

func validExecutableHandler(extensionID, handler string) bool {
	handler = strings.TrimSpace(handler)
	if handler == "" || len(handler) > 256 || strings.Contains(handler, "://") ||
		strings.Contains(handler, "..") || !strings.HasPrefix(handler, extensionID+".") {
		return false
	}
	for _, char := range handler {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func resultFilterDependencyEqual(left, right *ResultFilterDependency) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return left.ExtensionID == right.ExtensionID && left.VersionConstraint == right.VersionConstraint
}

func errorsJoinExecution(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrExecutionInvalid, err)
}

// publicationExecutableProvider validates Host-bound private provider material.
func publicationExecutableProvider(
	artifact Artifact,
	declaration QueryDeclaration,
) (boundExecutableProvider, bool, error) {
	if declaration.ProviderDigest == "" && declaration.boundProvider == nil {
		return boundExecutableProvider{}, false, nil
	}
	if !digestPattern.MatchString(declaration.ProviderDigest) || declaration.boundProvider == nil ||
		declaration.Handler == "" {
		return boundExecutableProvider{}, false, ErrInvalid
	}
	material := *declaration.boundProvider
	if material.provider == nil || material.handler != declaration.Handler ||
		material.providerDigest != declaration.ProviderDigest {
		return boundExecutableProvider{}, false, ErrInvalid
	}
	// 允许适配器覆盖 digest，但必须是稳定 64 位 hex 且与 public 字段一致。
	if material.providerDigest != declaration.ProviderDigest {
		return boundExecutableProvider{}, false, ErrInvalid
	}
	return material, true, nil
}

func publicationExecutableFilter(
	artifact Artifact,
	declaration ResultFilterDeclaration,
) (boundExecutableFilter, bool, error) {
	if declaration.FilterDigest == "" && declaration.boundFilter == nil {
		return boundExecutableFilter{}, false, nil
	}
	if !digestPattern.MatchString(declaration.FilterDigest) || declaration.boundFilter == nil ||
		declaration.Handler == "" {
		return boundExecutableFilter{}, false, ErrInvalid
	}
	material := *declaration.boundFilter
	if material.filter == nil || material.handler != declaration.Handler ||
		material.filterDigest != declaration.FilterDigest ||
		material.timeout != time.Duration(declaration.TimeoutMS)*time.Millisecond {
		return boundExecutableFilter{}, false, ErrInvalid
	}
	return material, true, nil
}
