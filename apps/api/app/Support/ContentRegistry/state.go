package contentregistry

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
)

type registryState struct {
	revision     uint64
	digest       string
	safeMode     bool
	publications map[string]Publication
	content      map[string]Contribution
	tombstones   map[string]Tombstone
	history      map[string]PublicationRecord
}

func emptyState() *registryState {
	state := &registryState{
		publications: map[string]Publication{},
		content:      map[string]Contribution{},
		tombstones:   map[string]Tombstone{},
		history:      map[string]PublicationRecord{},
	}
	state.digest = computeGraphDigest(nil, nil, nil, false)
	return state
}

func buildState(
	revision uint64,
	input []Publication,
	tombstones []Tombstone,
	history []PublicationRecord,
	safeMode bool,
	previous *registryState,
) (*registryState, error) {
	if len(tombstones) > maxTombstonesTotal || len(history) > maxPublicationHistoryTotal {
		return nil, ErrInvalid
	}
	publications, err := normalizePublications(filterSafeModeInput(input, safeMode))
	if err != nil {
		return nil, err
	}
	state := &registryState{
		revision:     revision,
		safeMode:     safeMode,
		publications: make(map[string]Publication, len(publications)),
		content:      map[string]Contribution{},
		tombstones:   map[string]Tombstone{},
		history:      map[string]PublicationRecord{},
	}
	owners := map[string]string{}
	// 进程内已经观察到的 ownership/package history 只能追加。启动恢复可
	// 补充 durable rows，但缺失的恢复载荷不能擦除当前进程已经看到的事实。
	if previous != nil {
		for _, value := range previous.tombstoneValues() {
			if err := addTombstone(state, owners, value); err != nil {
				return nil, err
			}
		}
		for _, value := range previous.historyValues() {
			if err := addPublicationRecord(state, value); err != nil {
				return nil, err
			}
		}
	}
	for _, value := range tombstones {
		if err := addTombstone(state, owners, value); err != nil {
			return nil, err
		}
	}
	for _, value := range history {
		if err := addPublicationRecord(state, value); err != nil {
			return nil, err
		}
	}
	for _, publication := range publications {
		if err := addPublicationRecord(state, publicationRecord(publication)); err != nil {
			return nil, err
		}
		state.publications[publication.Artifact.ExtensionID] = clonePublication(publication)
		for _, declaration := range publication.Content {
			if err := addTombstone(state, owners, declarationTombstone(publication.Artifact, declaration)); err != nil {
				return nil, err
			}
			// Content IDs are globally unique across the active graph. Manifest
			// already namespaces them under the extension id; composition and
			// multi-plugin merge semantics are deliberately not invented here.
			if existing, duplicate := state.content[declaration.ID]; duplicate {
				return nil, fmt.Errorf("%w: duplicate content id %s owned by %s and %s",
					ErrConflict, declaration.ID, existing.Artifact.ExtensionID, publication.Artifact.ExtensionID)
			}
			state.content[declaration.ID] = Contribution{
				Declaration: cloneDeclaration(declaration),
				Artifact:    publication.Artifact,
			}
		}
	}
	if len(state.tombstones) > maxTombstonesTotal || len(state.history) > maxPublicationHistoryTotal {
		return nil, ErrInvalid
	}
	state.digest = computeGraphDigest(
		sortedPublications(state.publications), state.tombstoneValues(), state.historyValues(), safeMode,
	)
	return state, nil
}

func addTombstone(state *registryState, owners map[string]string, raw Tombstone) error {
	tombstone, err := normalizeTombstone(raw)
	if err != nil {
		return err
	}
	if owner := owners[tombstone.ID]; owner != "" && owner != tombstone.OwnerExtensionID {
		return fmt.Errorf("%w: content id %s remains owned by %s", ErrConflict, tombstone.ID, owner)
	}
	owners[tombstone.ID] = tombstone.OwnerExtensionID
	key := tombstoneKey(tombstone)
	if existing, found := state.tombstones[key]; found && existing != tombstone {
		return fmt.Errorf("%w: content contract %s %s changed definition", ErrConflict, tombstone.ID, tombstone.ContractVersion)
	}
	state.tombstones[key] = tombstone
	return nil
}

func addPublicationRecord(state *registryState, raw PublicationRecord) error {
	record, err := normalizePublicationRecord(raw)
	if err != nil {
		return err
	}
	key := publicationHistoryKey(record)
	if existing, found := state.history[key]; found && existing != record {
		return fmt.Errorf("%w: exact package %s changed content declarations", ErrArtifactConflict, record.ExtensionID)
	}
	state.history[key] = record
	return nil
}

func declarationTombstone(artifact Artifact, declaration Declaration) Tombstone {
	return Tombstone{
		ID: declaration.ID, ContractVersion: declaration.ContractVersion,
		OwnerExtensionID: artifact.ExtensionID, DefinitionDigest: declarationDigest(declaration),
	}
}

func publicationRecord(publication Publication) PublicationRecord {
	return PublicationRecord{
		ExtensionID: publication.Artifact.ExtensionID, ExtensionVersion: publication.Artifact.ExtensionVersion,
		PackageDigest: publication.Artifact.PackageDigest, ContentDigest: publicationContentDigest(publication.Content),
	}
}

func cloneDeclaration(value Declaration) Declaration {
	return value
}

func cloneContribution(value Contribution) Contribution {
	value.Declaration = cloneDeclaration(value.Declaration)
	return value
}

func clonePublication(value Publication) Publication {
	value.Content = slices.Clone(value.Content)
	for index := range value.Content {
		value.Content[index] = cloneDeclaration(value.Content[index])
	}
	return value
}

func clonePublicationMap(values map[string]Publication) map[string]Publication {
	result := make(map[string]Publication, len(values))
	for extensionID, publication := range values {
		result[extensionID] = clonePublication(publication)
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

func sortedContributions(values map[string]Contribution) []Contribution {
	result := make([]Contribution, 0, len(values))
	for _, contribution := range values {
		result = append(result, cloneContribution(contribution))
	}
	sort.Slice(result, func(i, j int) bool {
		return contributionBefore(result[i], result[j])
	})
	return result
}

func (s *registryState) tombstoneValues() []Tombstone {
	result := make([]Tombstone, 0, len(s.tombstones))
	for _, value := range s.tombstones {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		if result[i].ContractVersion != result[j].ContractVersion {
			return result[i].ContractVersion < result[j].ContractVersion
		}
		return result[i].OwnerExtensionID < result[j].OwnerExtensionID
	})
	return result
}

func (s *registryState) historyValues() []PublicationRecord {
	result := make([]PublicationRecord, 0, len(s.history))
	for _, value := range s.history {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ExtensionID != result[j].ExtensionID {
			return result[i].ExtensionID < result[j].ExtensionID
		}
		if result[i].ExtensionVersion != result[j].ExtensionVersion {
			return result[i].ExtensionVersion < result[j].ExtensionVersion
		}
		return result[i].PackageDigest < result[j].PackageDigest
	})
	return result
}

func contributionBefore(left, right Contribution) bool {
	if left.Kind != right.Kind {
		return left.Kind < right.Kind
	}
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	return left.Artifact.ExtensionID < right.Artifact.ExtensionID
}

func snapshotFromState(state *registryState) Snapshot {
	return Snapshot{
		SchemaVersion: SchemaVersion,
		Revision:      state.revision,
		Digest:        state.digest,
		SafeMode:      state.safeMode,
		Publications:  sortedPublications(state.publications),
		Content:       sortedContributions(state.content),
		Tombstones:    state.tombstoneValues(),
		History:       state.historyValues(),
	}
}

func cloneSnapshot(value Snapshot) Snapshot {
	value.Publications = slices.Clone(value.Publications)
	for index := range value.Publications {
		value.Publications[index] = clonePublication(value.Publications[index])
	}
	value.Content = slices.Clone(value.Content)
	for index := range value.Content {
		value.Content[index] = cloneContribution(value.Content[index])
	}
	value.Tombstones = slices.Clone(value.Tombstones)
	value.History = slices.Clone(value.History)
	return value
}

func equalPublicationMaps(left, right map[string]Publication) bool {
	return reflect.DeepEqual(left, right)
}

func equalStates(left, right *registryState) bool {
	return left.safeMode == right.safeMode &&
		reflect.DeepEqual(left.publications, right.publications) &&
		reflect.DeepEqual(left.tombstones, right.tombstones) &&
		reflect.DeepEqual(left.history, right.history)
}

func equalPublications(left, right Publication) bool {
	return reflect.DeepEqual(left, right)
}

func validateExactPublicationReplay(current, next map[string]Publication) error {
	for extensionID, active := range current {
		candidate, found := next[extensionID]
		if !found || active.Artifact != candidate.Artifact {
			continue
		}
		if !equalPublications(active, candidate) {
			return fmt.Errorf("%w: exact artifact %s changed declarations", ErrArtifactConflict, extensionID)
		}
	}
	return nil
}

// Full-graph reconciliation may retire third-party publications, but Core
// catalogs require their sealed exact-artifact mutation path. This prevents a
// stale or partial restore payload from silently deleting Host declarations.
func validateCorePublicationTransition(current, next map[string]Publication) error {
	for extensionID, active := range current {
		if !active.Artifact.Core {
			continue
		}
		candidate, found := next[extensionID]
		if !found || candidate.Artifact != active.Artifact {
			return fmt.Errorf("%w: core publication %s requires exact artifact mutation", ErrArtifactConflict, extensionID)
		}
	}
	return nil
}
