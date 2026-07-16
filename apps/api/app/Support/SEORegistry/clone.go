package seoregistry

import (
	"slices"
	"sort"
)

func clonePublication(value Publication) Publication {
	value.Contributions = slices.Clone(value.Contributions)
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
	sort.Slice(result, func(i, j int) bool { return artifactBefore(result[i].Artifact, result[j].Artifact) })
	return result
}

func cloneContribution(value Contribution) Contribution { return value }

func cloneContributions(values []Contribution) []Contribution {
	return slices.Clone(values)
}

func cloneConflict(value Conflict) Conflict {
	value.Candidates = cloneContributions(value.Candidates)
	return value
}

func cloneConflicts(values []Conflict) []Conflict {
	result := slices.Clone(values)
	for index := range result {
		result[index] = cloneConflict(result[index])
	}
	return result
}

func cloneSnapshot(value Snapshot) Snapshot {
	value.Publications = slices.Clone(value.Publications)
	for index := range value.Publications {
		value.Publications[index] = clonePublication(value.Publications[index])
	}
	value.Contributions = cloneContributions(value.Contributions)
	value.Conflicts = cloneConflicts(value.Conflicts)
	return value
}

func cloneDocument(value Document) Document {
	value.Meta = slices.Clone(value.Meta)
	value.Hreflang = slices.Clone(value.Hreflang)
	value.Sitemap = slices.Clone(value.Sitemap)
	for index := range value.Sitemap {
		if value.Sitemap[index].Priority != nil {
			priority := *value.Sitemap[index].Priority
			value.Sitemap[index].Priority = &priority
		}
	}
	value.JSONLD = slices.Clone(value.JSONLD)
	for index := range value.JSONLD {
		value.JSONLD[index] = cloneJSONLDDocument(value.JSONLD[index])
	}
	return value
}

func cloneJSONLDDocument(value JSONLDDocument) JSONLDDocument {
	value.ImageURLs = slices.Clone(value.ImageURLs)
	value.Author = slices.Clone(value.Author)
	value.Breadcrumbs = slices.Clone(value.Breadcrumbs)
	if value.Publisher != nil {
		publisher := *value.Publisher
		value.Publisher = &publisher
	}
	return value
}

func cloneExecuteResult(value ExecuteResult) ExecuteResult {
	value.Document = cloneDocument(value.Document)
	value.Applied = slices.Clone(value.Applied)
	value.Fallbacks = slices.Clone(value.Fallbacks)
	return value
}
