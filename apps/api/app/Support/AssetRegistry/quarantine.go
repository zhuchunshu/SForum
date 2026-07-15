package assetregistry

import "sort"

// QuarantineExact atomically removes one exact artifact and every publication
// that transitively depends on handles owned by a quarantined publication. A
// stale artifact cannot quarantine a newer active publication.
func (r *Registry) QuarantineExact(artifact Artifact) (uint64, []Artifact, error) {
	if r == nil {
		return 0, []Artifact{}, ErrInvalid
	}
	normalized, err := normalizeArtifact(artifact)
	if err != nil {
		return r.load().revision, []Artifact{}, ErrInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	publication, found := current.publications[normalized.ExtensionID]
	if !found {
		return current.revision, []Artifact{}, nil
	}
	if publication.Artifact != normalized {
		return current.revision, []Artifact{}, ErrArtifactConflict
	}

	quarantinedOwners := map[string]bool{normalized.ExtensionID: true}
	quarantinedHandles := map[string]bool{}
	addOwnedHandles(publication, quarantinedHandles)
	ownerIDs := make([]string, 0, len(current.publications))
	for extensionID := range current.publications {
		ownerIDs = append(ownerIDs, extensionID)
	}
	sort.Strings(ownerIDs)

	// Publication-level cycles can exist even though the asset graph itself is a
	// DAG. Iterate to a fixed point so every transitive consumer is removed once.
	for changed := true; changed; {
		changed = false
		for _, extensionID := range ownerIDs {
			if quarantinedOwners[extensionID] {
				continue
			}
			candidate := current.publications[extensionID]
			if !publicationDependsOnAny(candidate, quarantinedHandles) {
				continue
			}
			quarantinedOwners[extensionID] = true
			addOwnedHandles(candidate, quarantinedHandles)
			changed = true
		}
	}

	remaining := clonePublicationMap(current.publications)
	quarantined := make([]Artifact, 0, len(quarantinedOwners))
	for extensionID := range quarantinedOwners {
		quarantined = append(quarantined, current.publications[extensionID].Artifact)
		delete(remaining, extensionID)
	}
	sort.Slice(quarantined, func(i, j int) bool {
		left, right := quarantined[i], quarantined[j]
		if left.ExtensionID != right.ExtensionID {
			return left.ExtensionID < right.ExtensionID
		}
		if left.ExtensionVersion != right.ExtensionVersion {
			return left.ExtensionVersion < right.ExtensionVersion
		}
		if left.PackageDigest != right.PackageDigest {
			return left.PackageDigest < right.PackageDigest
		}
		return left.ImpactDigest < right.ImpactDigest
	})
	next, err := buildState(current.revision+1, publicationValues(remaining))
	if err != nil {
		return current.revision, []Artifact{}, err
	}
	r.state.Store(next)
	return next.revision, quarantined, nil
}

func addOwnedHandles(publication Publication, handles map[string]bool) {
	for _, asset := range publication.Assets {
		handles[asset.Handle] = true
	}
}

func publicationDependsOnAny(publication Publication, handles map[string]bool) bool {
	for _, asset := range publication.Assets {
		for _, dependency := range asset.Dependencies {
			if handles[dependency] {
				return true
			}
		}
	}
	return false
}
