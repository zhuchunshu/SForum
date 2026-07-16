package mediaregistry

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
)

type registryState struct {
	revision        uint64
	digest          string
	safeMode        bool
	publications    map[string]Publication
	policies        map[string]MIMEPolicyContribution
	processors      map[string]ProcessorContribution
	variants        map[string]VariantContribution
	policyGroups    map[string][]MIMEPolicyContribution
	processorGroups map[string][]ProcessorContribution
	variantGroups   map[string][]VariantContribution
	variantBindings []VariantBinding
	selections      map[string]ProviderSelection
	conflicts       []ProviderConflict
}

// RuntimeInstanceID is intentionally excluded. Restarting the same immutable
// package/version row cannot reinterpret its media declarations.
type artifactDeclarationIdentity struct {
	extensionID      string
	extensionVersion string
	packageDigest    string
	versionID        int64
	core             bool
}

func declarationIdentity(artifact Artifact) artifactDeclarationIdentity {
	return artifactDeclarationIdentity{
		extensionID: artifact.ExtensionID, extensionVersion: artifact.ExtensionVersion,
		packageDigest: artifact.PackageDigest, versionID: artifact.VersionID, core: artifact.Core,
	}
}

func emptyState() *registryState {
	return &registryState{digest: graphDigest(nil, nil, false), publications: map[string]Publication{}, policies: map[string]MIMEPolicyContribution{},
		processors: map[string]ProcessorContribution{}, variants: map[string]VariantContribution{}, policyGroups: map[string][]MIMEPolicyContribution{},
		processorGroups: map[string][]ProcessorContribution{}, variantGroups: map[string][]VariantContribution{}, selections: map[string]ProviderSelection{}}
}

func buildState(revision uint64, raw []Publication, safeMode bool, requested []ProviderSelection) (*registryState, error) {
	publications, err := normalizePublications(raw, safeMode)
	if err != nil {
		return nil, err
	}
	state := &registryState{revision: revision, safeMode: safeMode, publications: map[string]Publication{}, policies: map[string]MIMEPolicyContribution{},
		processors: map[string]ProcessorContribution{}, variants: map[string]VariantContribution{}, policyGroups: map[string][]MIMEPolicyContribution{},
		processorGroups: map[string][]ProcessorContribution{}, variantGroups: map[string][]VariantContribution{}, selections: map[string]ProviderSelection{}}
	for _, publication := range publications {
		state.publications[publication.Artifact.ExtensionID] = clonePublication(publication)
		for _, declaration := range publication.Policies {
			if existing, found := state.policies[declaration.ID]; found {
				return nil, duplicateContribution("policy", declaration.ID, existing.Artifact, publication.Artifact)
			}
			value := MIMEPolicyContribution{MIMEPolicyDeclaration: clonePolicy(declaration), Artifact: publication.Artifact}
			state.policies[value.ID] = value
			state.policyGroups[policyConflictKey(value.Purpose)] = append(state.policyGroups[policyConflictKey(value.Purpose)], value)
		}
		for _, declaration := range publication.Processors {
			if existing, found := state.processors[declaration.ID]; found {
				return nil, duplicateContribution("processor", declaration.ID, existing.Artifact, publication.Artifact)
			}
			value := ProcessorContribution{ProcessorDeclaration: cloneProcessor(declaration), Artifact: publication.Artifact}
			state.processors[value.ID] = value
			if value.Mode == ProcessorExclusive {
				key := processorConflictKey(value)
				state.processorGroups[key] = append(state.processorGroups[key], value)
			}
		}
		for _, declaration := range publication.Variants {
			if existing, found := state.variants[declaration.ID]; found {
				return nil, duplicateContribution("variant", declaration.ID, existing.Artifact, publication.Artifact)
			}
			value := VariantContribution{VariantDeclaration: declaration, Artifact: publication.Artifact}
			state.variants[value.ID] = value
			key := variantConflictKey(value.Purpose, value.Name)
			state.variantGroups[key] = append(state.variantGroups[key], value)
		}
	}
	// 跨包 variant→processor 依赖不得阻塞 exact-artifact Remove：无法解析的
	// variant 仅退出 executable projection，声明与 pending 状态仍可检查。同包内
	// 错误 stage/purpose 绑定仍 fail-closed。
	state.variants, state.variantGroups, state.variantBindings, err = resolveVariantBindings(state.variants, state.processors)
	if err != nil {
		return nil, err
	}
	sortContributionGroups(state)
	for _, selection := range requested {
		if canonical, valid := canonicalSelectionForState(state, selection); valid {
			state.selections[selectionMapKey(canonical.Family, canonical.Key)] = canonical
		}
	}
	state.conflicts = buildConflicts(state)
	selectionValues := sortedSelections(state.selections)
	// 摘要同时绑定 active graph 与 pending 声明；否则孤儿依赖变化会在同一
	// digest 下悄然改变未来可激活的执行图。
	state.digest = graphDigest(sortedActivePublications(state), selectionValues, safeMode)
	return state, nil
}

// resolveVariantBindings 激活身份完整匹配的 variant；processor 已卸载或身份
// 不匹配时保留 pending 状态，不把跨包依赖升级成 Remove 失败。
func resolveVariantBindings(variants map[string]VariantContribution, processors map[string]ProcessorContribution) (map[string]VariantContribution, map[string][]VariantContribution, []VariantBinding, error) {
	kept := make(map[string]VariantContribution, len(variants))
	groups := map[string][]VariantContribution{}
	bindings := make([]VariantBinding, 0, len(variants))
	for _, variant := range variants {
		processor, found := processors[variant.ProcessorID]
		if !found {
			bindings = append(bindings, VariantBinding{Variant: variant, Status: VariantBindingPending, Reason: VariantPendingProcessorMissing})
			continue
		}
		if processor.Artifact.ExtensionID != variant.ProcessorOwnerExtensionID ||
			processor.ContractVersion != variant.ProcessorContractVersion ||
			processor.Artifact.PackageDigest != variant.ProcessorPackageDigest {
			bindings = append(bindings, VariantBinding{Variant: variant, Status: VariantBindingPending, Reason: VariantPendingProcessorIdentityMismatch})
			continue
		}
		if processor.Stage != StageTransform || !purposeCompatible(variant.Purpose, processor.Purpose) {
			return nil, nil, nil, fmt.Errorf("%w: variant %s references incompatible transform %s", ErrConflict, variant.ID, variant.ProcessorID)
		}
		kept[variant.ID] = variant
		key := variantConflictKey(variant.Purpose, variant.Name)
		groups[key] = append(groups[key], variant)
		ref := processorRef(processor)
		bindings = append(bindings, VariantBinding{Variant: variant, Status: VariantBindingActive, Processor: &ref})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Variant.ID < bindings[j].Variant.ID })
	return kept, groups, bindings, nil
}

func sortContributionGroups(state *registryState) {
	for key := range state.policyGroups {
		sort.Slice(state.policyGroups[key], func(i, j int) bool { return policyBefore(state.policyGroups[key][i], state.policyGroups[key][j]) })
	}
	for key := range state.processorGroups {
		sort.Slice(state.processorGroups[key], func(i, j int) bool {
			return processorBefore(state.processorGroups[key][i], state.processorGroups[key][j])
		})
	}
	for key := range state.variantGroups {
		sort.Slice(state.variantGroups[key], func(i, j int) bool { return variantBefore(state.variantGroups[key][i], state.variantGroups[key][j]) })
	}
}

func buildConflicts(state *registryState) []ProviderConflict {
	result := []ProviderConflict{}
	for key, values := range state.policyGroups {
		if len(values) > 1 {
			result = append(result, conflictFromPolicy(state, key, values))
		}
	}
	for key, values := range state.processorGroups {
		if len(values) > 1 {
			result = append(result, conflictFromProcessors(state, key, values))
		}
	}
	for key, values := range state.variantGroups {
		if len(values) > 1 {
			result = append(result, conflictFromVariants(state, key, values))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Family != result[j].Family {
			return result[i].Family < result[j].Family
		}
		return result[i].Key < result[j].Key
	})
	return result
}

func conflictFromPolicy(state *registryState, key string, values []MIMEPolicyContribution) ProviderConflict {
	refs := make([]ProviderRef, len(values))
	for i, value := range values {
		refs[i] = policyRef(value)
	}
	winner, selected := selectedRef(state, ConflictMIMEPolicy, key, refs)
	return ProviderConflict{Family: ConflictMIMEPolicy, Key: key, Candidates: refs, Winner: winner, SelectionConfigured: selected}
}
func conflictFromProcessors(state *registryState, key string, values []ProcessorContribution) ProviderConflict {
	refs := make([]ProviderRef, len(values))
	for i, value := range values {
		refs[i] = processorRef(value)
	}
	winner, selected := selectedRef(state, ConflictProcessor, key, refs)
	return ProviderConflict{Family: ConflictProcessor, Key: key, Candidates: refs, Winner: winner, SelectionConfigured: selected}
}
func conflictFromVariants(state *registryState, key string, values []VariantContribution) ProviderConflict {
	refs := make([]ProviderRef, len(values))
	for i, value := range values {
		refs[i] = variantRef(value)
	}
	winner, selected := selectedRef(state, ConflictVariant, key, refs)
	return ProviderConflict{Family: ConflictVariant, Key: key, Candidates: refs, Winner: winner, SelectionConfigured: selected}
}

func selectedRef(state *registryState, family, key string, refs []ProviderRef) (ProviderRef, bool) {
	selection, ok := state.selections[selectionMapKey(family, key)]
	if !ok {
		return refs[0], false
	}
	for _, ref := range refs {
		if ref == selection.Provider {
			return ref, true
		}
	}
	return refs[0], false
}

func selectionValidForState(state *registryState, selection ProviderSelection) bool {
	_, valid := canonicalSelectionForState(state, selection)
	return valid
}

func canonicalSelectionForState(state *registryState, selection ProviderSelection) (ProviderSelection, bool) {
	selection.Family = stringsLower(selection.Family)
	selection.Key = stringsLower(selection.Key)
	selection.Provider.ContributionID = stringsLower(selection.Provider.ContributionID)
	if !validConflictFamily(selection.Family) || selection.Key == "" || !idPattern.MatchString(selection.Provider.ContributionID) {
		return ProviderSelection{}, false
	}
	refs := state.candidateRefs(selection.Family, selection.Key)
	for _, ref := range refs {
		if ref.ContributionID == selection.Provider.ContributionID && artifactIdentityEqual(ref.Artifact, selection.Provider.Artifact) {
			selection.Provider = ref
			return selection, true
		}
	}
	return ProviderSelection{}, false
}

func (state *registryState) candidateRefs(family, key string) []ProviderRef {
	switch family {
	case ConflictMIMEPolicy:
		values := state.policyGroups[key]
		result := make([]ProviderRef, len(values))
		for i, v := range values {
			result[i] = policyRef(v)
		}
		return result
	case ConflictProcessor:
		values := state.processorGroups[key]
		result := make([]ProviderRef, len(values))
		for i, v := range values {
			result[i] = processorRef(v)
		}
		return result
	case ConflictVariant:
		values := state.variantGroups[key]
		result := make([]ProviderRef, len(values))
		for i, v := range values {
			result[i] = variantRef(v)
		}
		return result
	default:
		return nil
	}
}

func snapshotFromState(state *registryState) Snapshot {
	return Snapshot{SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest, SafeMode: state.safeMode,
		Publications: sortedActivePublications(state), Policies: sortedPolicies(state.policies), Processors: sortedProcessors(state.processors),
		Variants: sortedVariants(state.variants), VariantBindings: cloneVariantBindings(state.variantBindings),
		Selections: sortedSelections(state.selections), Conflicts: cloneConflicts(state.conflicts)}
}

func activePublication(state *registryState, value Publication) Publication {
	// Publication inspection is declaration inspection, so pending variants must
	// remain visible. Snapshot.Variants remains the active executable projection.
	return clonePublication(value)
}

func sortedActivePublications(state *registryState) []Publication {
	result := make([]Publication, 0, len(state.publications))
	for _, publication := range state.publications {
		result = append(result, activePublication(state, publication))
	}
	sort.Slice(result, func(i, j int) bool { return artifactBefore(result[i].Artifact, result[j].Artifact) })
	return result
}

func clonePolicy(value MIMEPolicyDeclaration) MIMEPolicyDeclaration {
	value.AllowedMIMEs = slices.Clone(value.AllowedMIMEs)
	value.DeniedMIMEs = slices.Clone(value.DeniedMIMEs)
	value.AllowedExtensions = slices.Clone(value.AllowedExtensions)
	value.MIMEAliases = slices.Clone(value.MIMEAliases)
	return value
}
func cloneProcessor(value ProcessorDeclaration) ProcessorDeclaration {
	value.MIMEs = slices.Clone(value.MIMEs)
	return value
}
func clonePublication(value Publication) Publication {
	value.Policies = slices.Clone(value.Policies)
	for i := range value.Policies {
		value.Policies[i] = clonePolicy(value.Policies[i])
	}
	value.Processors = slices.Clone(value.Processors)
	for i := range value.Processors {
		value.Processors[i] = cloneProcessor(value.Processors[i])
	}
	value.Variants = slices.Clone(value.Variants)
	return value
}
func clonePolicyContribution(value MIMEPolicyContribution) MIMEPolicyContribution {
	value.MIMEPolicyDeclaration = clonePolicy(value.MIMEPolicyDeclaration)
	return value
}
func cloneProcessorContribution(value ProcessorContribution) ProcessorContribution {
	value.ProcessorDeclaration = cloneProcessor(value.ProcessorDeclaration)
	return value
}
func cloneVariantContribution(value VariantContribution) VariantContribution { return value }
func cloneSelection(value ProviderSelection) ProviderSelection               { return value }
func cloneVariantBinding(value VariantBinding) VariantBinding {
	if value.Processor != nil {
		processor := *value.Processor
		value.Processor = &processor
	}
	return value
}
func cloneVariantBindings(values []VariantBinding) []VariantBinding {
	result := slices.Clone(values)
	for i := range result {
		result[i] = cloneVariantBinding(result[i])
	}
	return result
}
func cloneConflict(value ProviderConflict) ProviderConflict {
	value.Candidates = slices.Clone(value.Candidates)
	return value
}
func cloneConflicts(values []ProviderConflict) []ProviderConflict {
	result := slices.Clone(values)
	for i := range result {
		result[i] = cloneConflict(result[i])
	}
	return result
}
func cloneSnapshot(value Snapshot) Snapshot {
	value.Publications = slices.Clone(value.Publications)
	for i := range value.Publications {
		value.Publications[i] = clonePublication(value.Publications[i])
	}
	value.Policies = slices.Clone(value.Policies)
	for i := range value.Policies {
		value.Policies[i] = clonePolicyContribution(value.Policies[i])
	}
	value.Processors = slices.Clone(value.Processors)
	for i := range value.Processors {
		value.Processors[i] = cloneProcessorContribution(value.Processors[i])
	}
	value.Variants = slices.Clone(value.Variants)
	value.VariantBindings = cloneVariantBindings(value.VariantBindings)
	value.Selections = slices.Clone(value.Selections)
	value.Conflicts = cloneConflicts(value.Conflicts)
	return value
}

func clonePublicationMap(values map[string]Publication) map[string]Publication {
	result := make(map[string]Publication, len(values))
	for k, v := range values {
		result[k] = clonePublication(v)
	}
	return result
}
func publicationValues(values map[string]Publication) []Publication {
	result := make([]Publication, 0, len(values))
	for _, v := range values {
		result = append(result, clonePublication(v))
	}
	return result
}
func sortedPublications(values map[string]Publication) []Publication {
	result := publicationValues(values)
	sort.Slice(result, func(i, j int) bool { return artifactBefore(result[i].Artifact, result[j].Artifact) })
	return result
}
func sortedPolicies(values map[string]MIMEPolicyContribution) []MIMEPolicyContribution {
	result := make([]MIMEPolicyContribution, 0, len(values))
	for _, v := range values {
		result = append(result, clonePolicyContribution(v))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
func sortedProcessors(values map[string]ProcessorContribution) []ProcessorContribution {
	result := make([]ProcessorContribution, 0, len(values))
	for _, v := range values {
		result = append(result, cloneProcessorContribution(v))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
func sortedVariants(values map[string]VariantContribution) []VariantContribution {
	result := make([]VariantContribution, 0, len(values))
	for _, v := range values {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
func sortedSelections(values map[string]ProviderSelection) []ProviderSelection {
	result := make([]ProviderSelection, 0, len(values))
	for _, v := range values {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Family != result[j].Family {
			return result[i].Family < result[j].Family
		}
		return result[i].Key < result[j].Key
	})
	return result
}

func equalPublicationMaps(left, right map[string]Publication) bool {
	return reflect.DeepEqual(left, right)
}
func equalPublications(left, right Publication) bool { return reflect.DeepEqual(left, right) }
func equalPublicationDeclarations(left, right Publication) bool {
	return reflect.DeepEqual(left.Policies, right.Policies) &&
		reflect.DeepEqual(left.Processors, right.Processors) &&
		reflect.DeepEqual(left.Variants, right.Variants)
}
func validateExactReplay(current, next map[string]Publication) error {
	for id, active := range current {
		candidate, found := next[id]
		if found && active.Artifact == candidate.Artifact && !equalPublications(active, candidate) {
			return fmt.Errorf("%w: exact artifact %s changed declarations", ErrArtifactConflict, id)
		}
	}
	return nil
}

func validatePublicationHistory(history map[artifactDeclarationIdentity]Publication, publications map[string]Publication) error {
	for extensionID, candidate := range publications {
		sealed, found := history[declarationIdentity(candidate.Artifact)]
		if found && !equalPublicationDeclarations(sealed, candidate) {
			return fmt.Errorf("%w: immutable artifact %s changed declarations", ErrArtifactConflict, extensionID)
		}
	}
	return nil
}

func recordPublicationHistory(history map[artifactDeclarationIdentity]Publication, publications map[string]Publication) {
	for _, publication := range publications {
		history[declarationIdentity(publication.Artifact)] = clonePublication(publication)
	}
}

func duplicateContribution(kind, id string, left, right Artifact) error {
	return fmt.Errorf("%w: duplicate %s %s owned by %s and %s", ErrConflict, kind, id, left.ExtensionID, right.ExtensionID)
}
func policyConflictKey(purpose string) string { return purpose }
func processorConflictKey(value ProcessorContribution) string {
	return value.Stage + "/" + value.Purpose + "/" + value.Slot
}
func variantConflictKey(purpose, name string) string { return purpose + "/" + name }
func selectionMapKey(family, key string) string      { return family + "\x00" + key }
func policyRef(value MIMEPolicyContribution) ProviderRef {
	return ProviderRef{ContributionID: value.ID, Artifact: value.Artifact}
}
func processorRef(value ProcessorContribution) ProviderRef {
	return ProviderRef{ContributionID: value.ID, Artifact: value.Artifact}
}
func variantRef(value VariantContribution) ProviderRef {
	return ProviderRef{ContributionID: value.ID, Artifact: value.Artifact}
}
func stringsLower(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func validConflictFamily(value string) bool {
	return value == ConflictMIMEPolicy || value == ConflictProcessor || value == ConflictVariant
}
func purposeCompatible(left, right string) bool { return left == right || left == "*" || right == "*" }
func artifactIdentityEqual(left, right Artifact) bool {
	return left.ExtensionID == right.ExtensionID && left.ExtensionVersion == right.ExtensionVersion &&
		left.PackageDigest == right.PackageDigest && left.ImpactDigest == right.ImpactDigest &&
		left.VersionID == right.VersionID && left.RuntimeInstanceID == right.RuntimeInstanceID && left.Core == right.Core
}
func policyBefore(left, right MIMEPolicyContribution) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.Artifact.ExtensionID != right.Artifact.ExtensionID {
		return left.Artifact.ExtensionID < right.Artifact.ExtensionID
	}
	return left.ID < right.ID
}
func processorBefore(left, right ProcessorContribution) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.Artifact.ExtensionID != right.Artifact.ExtensionID {
		return left.Artifact.ExtensionID < right.Artifact.ExtensionID
	}
	return left.ID < right.ID
}
func variantBefore(left, right VariantContribution) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.Artifact.ExtensionID != right.Artifact.ExtensionID {
		return left.Artifact.ExtensionID < right.Artifact.ExtensionID
	}
	return left.ID < right.ID
}
