package queryregistry

import (
	"fmt"
	"slices"
	"sort"

	semver "github.com/Masterminds/semver/v3"
)

type registryState struct {
	revision     uint64
	digest       string
	safeMode     bool
	publications map[string]Publication
	queries      map[string]QueryContribution
	schemas      map[string]compiledResultSchema
}

func emptyState() *registryState {
	return &registryState{
		digest:       computeGraphDigest(nil, false),
		publications: map[string]Publication{},
		queries:      map[string]QueryContribution{},
		schemas:      map[string]compiledResultSchema{},
	}
}

func buildState(revision uint64, input []Publication, safeMode bool) (*registryState, error) {
	filtered := filterSafeModeInput(input, safeMode)
	publications, err := normalizePublications(filtered)
	if err != nil {
		return nil, err
	}
	state := emptyState()
	state.revision = revision
	state.safeMode = safeMode

	// ManifestQuery is a complete plan declaration, not a field-level action
	// contribution. Composition is therefore:
	// 1) unique query IDs
	// 2) unique provider slots within each declaration (validated at normalize)
	// Multi-plugin field/relation/filter/sort merge requires later contract fields.
	// Executable providers, result filters, and Schemas must join this same
	// immutable revision; private material is retained only in publications.
	for _, publication := range publications {
		for _, declaration := range publication.Queries {
			if existing, duplicate := state.queries[declaration.ID]; duplicate {
				return nil, fmt.Errorf("%w: duplicate query id %s owned by %s and %s",
					ErrConflict, declaration.ID, existing.Artifact.ExtensionID, publication.Artifact.ExtensionID)
			}
			contribution := QueryContribution{
				QueryDeclaration: cloneQueryDeclaration(declaration),
				Artifact:         publication.Artifact,
			}
			if err := detectSlotProviderConflicts(contribution); err != nil {
				return nil, err
			}
			compiled, bound, err := publicationResultSchema(publication.Artifact, declaration)
			if err != nil {
				return nil, fmt.Errorf("%w: result schema for %s", ErrInvalid, declaration.ID)
			}
			if bound {
				state.schemas[declaration.ID] = compiled
			}
			if _, _, err := publicationExecutableProvider(publication.Artifact, declaration); err != nil {
				return nil, fmt.Errorf("%w: executable provider for %s", ErrInvalid, declaration.ID)
			}
			// Plans and inspection carry only public digests. Raw Schema bytes,
			// validators, and callables stay in the publication private material.
			contribution.boundResultSchema = nil
			contribution.boundProvider = nil
			state.queries[contribution.ID] = contribution
		}
		for _, filter := range publication.ResultFilters {
			if _, _, err := publicationExecutableFilter(publication.Artifact, filter); err != nil {
				return nil, fmt.Errorf("%w: executable result filter for %s", ErrInvalid, filter.ID)
			}
		}
	}
	publications = bindGraphResultFilterIdentities(publications, state.queries)
	state.digest = computeGraphDigest(publications, safeMode)
	for _, publication := range publications {
		state.publications[publication.Artifact.ExtensionID] = clonePublication(publication)
	}
	return state, nil
}

// bindGraphResultFilterIdentities derives filter identity only after every
// query owner is known. Missing or drifted optional dependencies stay dormant;
// matchingFiltersWithEvidence applies their fail-open/fail-closed policy.
func bindGraphResultFilterIdentities(
	input []Publication,
	queries map[string]QueryContribution,
) []Publication {
	result := make([]Publication, len(input))
	for publicationIndex, publication := range input {
		publication = clonePublication(publication)
		for filterIndex := range publication.ResultFilters {
			filter := &publication.ResultFilters[filterIndex]
			filter.IdentityFields = nil
			target, found := queries[filter.QueryID]
			if !found || target.Handler == "" || len(target.IdentityFields) == 0 ||
				target.ContractVersion != filter.QueryContractVersion ||
				target.PlanVersion != filter.QueryPlanVersion ||
				!resultFilterOwnerMatches(publication.Artifact, *filter, target.Artifact) {
				continue
			}
			filter.IdentityFields = slices.Clone(target.IdentityFields)
		}
		result[publicationIndex] = publication
	}
	return result
}

func resultFilterOwnerMatches(filterOwner Artifact, filter ResultFilterDeclaration, queryOwner Artifact) bool {
	if filterOwner.ExtensionID == queryOwner.ExtensionID {
		return filter.Dependency == nil
	}
	if filter.Dependency == nil || filter.Dependency.ExtensionID != queryOwner.ExtensionID {
		return false
	}
	constraint, err := semver.NewConstraint(filter.Dependency.VersionConstraint)
	if err != nil {
		return false
	}
	version, err := semver.StrictNewVersion(queryOwner.ExtensionVersion)
	return err == nil && constraint.Check(version)
}

func detectSlotProviderConflicts(contribution QueryContribution) error {
	// 每个槽位必须有且仅有一个 provider：当前声明自身。重复名已在 normalize 拒绝。
	providers := map[string]string{}
	record := func(kind, name string) error {
		key := kind + "\x00" + name
		if owner, found := providers[key]; found {
			return fmt.Errorf("%w: duplicate %s provider %s on %s (also %s)",
				ErrConflict, kind, name, contribution.ID, owner)
		}
		providers[key] = contribution.ID
		return nil
	}
	for _, field := range contribution.Fields {
		if err := record(ProviderKindField, field); err != nil {
			return err
		}
	}
	for _, relation := range contribution.Relations {
		if err := record(ProviderKindRelation, relation); err != nil {
			return err
		}
	}
	for _, filter := range contribution.Filters {
		if err := record(ProviderKindFilter, filter); err != nil {
			return err
		}
	}
	for _, sortField := range contribution.Sort {
		if err := record(ProviderKindSort, sortField); err != nil {
			return err
		}
	}
	return nil
}

func sortedQueryValues(values map[string]QueryContribution) []QueryContribution {
	result := make([]QueryContribution, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.Entity != right.Entity {
			return left.Entity < right.Entity
		}
		if left.PlanVersion != right.PlanVersion {
			return left.PlanVersion < right.PlanVersion
		}
		if left.Artifact.ExtensionID != right.Artifact.ExtensionID {
			return left.Artifact.ExtensionID < right.Artifact.ExtensionID
		}
		return left.ID < right.ID
	})
	return result
}
