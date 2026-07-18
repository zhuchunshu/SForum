package queryregistry

import (
	"slices"
	"sort"
)

func clonePublication(value Publication) Publication {
	// slices.Clone 保留 nil/非 nil 语义，避免 DeepEqual 把空声明误判为漂移。
	value.Queries = slices.Clone(value.Queries)
	for index := range value.Queries {
		value.Queries[index] = cloneQueryDeclaration(value.Queries[index])
	}
	value.ResultFilters = slices.Clone(value.ResultFilters)
	for index := range value.ResultFilters {
		value.ResultFilters[index] = cloneResultFilterDeclaration(value.ResultFilters[index])
	}
	return value
}

func cloneQueryDeclaration(value QueryDeclaration) QueryDeclaration {
	value.Fields = slices.Clone(value.Fields)
	value.Relations = slices.Clone(value.Relations)
	value.Filters = slices.Clone(value.Filters)
	value.Sort = slices.Clone(value.Sort)
	value.CacheTags = slices.Clone(value.CacheTags)
	value.IdentityFields = slices.Clone(value.IdentityFields)
	value.DefaultSort = slices.Clone(value.DefaultSort)
	if value.boundResultSchema != nil {
		material := cloneCompiledResultSchema(*value.boundResultSchema)
		value.boundResultSchema = &material
	}
	if value.boundProvider != nil {
		material := *value.boundProvider
		value.boundProvider = &material
	}
	return value
}

func cloneResultFilterDeclaration(value ResultFilterDeclaration) ResultFilterDeclaration {
	value.IdentityFields = slices.Clone(value.IdentityFields)
	if value.Dependency != nil {
		dependency := *value.Dependency
		value.Dependency = &dependency
	}
	if value.boundFilter != nil {
		material := *value.boundFilter
		value.boundFilter = &material
	}
	return value
}

func cloneContribution(value QueryContribution) QueryContribution {
	value.QueryDeclaration = cloneQueryDeclaration(value.QueryDeclaration)
	// 计划与 Resolve 仅暴露公开 metadata；callable 留在 publication private material。
	value.boundResultSchema = nil
	value.boundProvider = nil
	return value
}

func clonePublicationMap(values map[string]Publication) map[string]Publication {
	result := make(map[string]Publication, len(values))
	for id, publication := range values {
		result[id] = clonePublication(publication)
	}
	return result
}

func publicationValues(values map[string]Publication) []Publication {
	result := make([]Publication, 0, len(values))
	for _, publication := range values {
		result = append(result, clonePublication(publication))
	}
	return result
}

func sortedPublications(values map[string]Publication) []Publication {
	result := publicationValues(values)
	sort.Slice(result, func(i, j int) bool {
		return artifactBefore(result[i].Artifact, result[j].Artifact)
	})
	return result
}

func snapshotFromState(state *registryState) Snapshot {
	return Snapshot{
		SchemaVersion: SchemaVersion,
		Revision:      state.revision,
		Digest:        state.digest,
		SafeMode:      state.safeMode,
		Publications:  sortedPublications(state.publications),
		Queries:       sortedQueryValues(state.queries),
	}
}

func cloneSnapshot(value Snapshot) Snapshot {
	value.Publications = append([]Publication(nil), value.Publications...)
	for index := range value.Publications {
		value.Publications[index] = clonePublication(value.Publications[index])
	}
	value.Queries = append([]QueryContribution(nil), value.Queries...)
	for index := range value.Queries {
		value.Queries[index] = cloneContribution(value.Queries[index])
	}
	return value
}

func cloneQueryPlan(value QueryPlan) QueryPlan {
	value.CacheTags = slices.Clone(value.CacheTags)
	value.Fields = slices.Clone(value.Fields)
	value.Relations = slices.Clone(value.Relations)
	value.Filters = slices.Clone(value.Filters)
	value.Sorts = slices.Clone(value.Sorts)
	value.Providers = slices.Clone(value.Providers)
	value.Query = cloneContribution(value.Query)
	return value
}
