package identityregistry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
)

const durableRootPublicationDomain = "sforum.identity-registry.root-publication@1\x00"

type durableDesiredRootPublication struct {
	tip DurableRootPublicationTip
}

type durableRootPublication struct {
	tip         DurableRootPublicationTip
	publication Publication
}

func desiredDurableRootPublication(publication *Publication) (*durableDesiredRootPublication, error) {
	if publication == nil {
		return nil, nil
	}
	normalized, raw, digest, err := canonicalDurableRootPublication(*publication)
	if err != nil || normalized.Artifact.Core {
		return nil, ErrInvalid
	}
	return &durableDesiredRootPublication{
		tip: DurableRootPublicationTip{
			OwnerExtensionID: normalized.Artifact.ExtensionID,
			Revision:         1, RegistryState: RegistryStateActive,
			ExtensionVersionID: normalized.Artifact.VersionID,
			ExtensionVersion:   normalized.Artifact.ExtensionVersion,
			PackageDigest:      normalized.Artifact.PackageDigest,
			SchemaVersion:      SchemaVersion, PublicationDigest: digest,
			PublicationJSON: append(json.RawMessage(nil), raw...),
		},
	}, nil
}

// canonicalDurableRootPublication validates live publication material before
// removing the process-local runtime id from durable declarative evidence.
func canonicalDurableRootPublication(publication Publication) (Publication, []byte, string, error) {
	normalized, err := normalizePublication(publication)
	if err != nil {
		return Publication{}, nil, "", err
	}
	normalized.Artifact.RuntimeInstanceID = ""
	raw, err := json.Marshal(normalized)
	if err != nil {
		return Publication{}, nil, "", ErrInvalid
	}
	return normalized, raw, durableRootPublicationDigest(raw), nil
}

func decodeDurableRootPublication(raw json.RawMessage) (Publication, []byte, string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var publication Publication
	if err := decoder.Decode(&publication); err != nil {
		return Publication{}, nil, "", ErrInvalid
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Publication{}, nil, "", ErrInvalid
	}
	if publication.Artifact.RuntimeInstanceID != "" || publication.Artifact.Core {
		return Publication{}, nil, "", ErrInvalid
	}

	// Executable Identity validation normally requires a live runtime id. The
	// durable payload intentionally omits it, so use a local validation marker
	// and remove it again before canonical encoding.
	validation := publication
	if identityDeclarationRequiresRuntime(validation.Identity) {
		validation.Artifact.RuntimeInstanceID = "durable-publication-validation"
	}
	normalized, err := normalizePublication(validation)
	if err != nil {
		return Publication{}, nil, "", err
	}
	normalized.Artifact.RuntimeInstanceID = ""
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return Publication{}, nil, "", ErrInvalid
	}
	return normalized, canonical, durableRootPublicationDigest(canonical), nil
}

func durableRootPublicationDigest(raw []byte) string {
	sum := sha256.Sum256(append([]byte(durableRootPublicationDomain), raw...))
	return hex.EncodeToString(sum[:])
}

func identityDeclarationRequiresRuntime(identity *IdentityDeclaration) bool {
	if identity == nil {
		return false
	}
	return len(identity.Providers) > 0 || len(identity.RiskHooks) > 0 ||
		identity.SessionPolicy != "" && identity.SessionPolicy != "core.session.default"
}

func durableRootPublications(state DurableState) (map[string]durableRootPublication, error) {
	result := make(map[string]durableRootPublication, len(state.RootTips))
	for _, raw := range state.RootTips {
		tip := raw
		tip.OwnerExtensionID = strings.ToLower(strings.TrimSpace(tip.OwnerExtensionID))
		tip.RegistryState = strings.ToLower(strings.TrimSpace(tip.RegistryState))
		tip.ExtensionVersion = strings.TrimSpace(tip.ExtensionVersion)
		tip.PackageDigest = strings.ToLower(strings.TrimSpace(tip.PackageDigest))
		tip.SchemaVersion = strings.TrimSpace(tip.SchemaVersion)
		tip.PublicationDigest = strings.ToLower(strings.TrimSpace(tip.PublicationDigest))
		tip.PublicationJSON = append(json.RawMessage(nil), tip.PublicationJSON...)
		if !idPattern.MatchString(tip.OwnerExtensionID) || strings.HasPrefix(tip.OwnerExtensionID, "core.") ||
			tip.Revision <= 0 ||
			(tip.RegistryState != RegistryStateActive && tip.RegistryState != RegistryStateTombstone) ||
			tip.ExtensionVersionID <= 0 || !strictSemVer(tip.ExtensionVersion) ||
			!digestPattern.MatchString(tip.PackageDigest) || tip.SchemaVersion != SchemaVersion ||
			!digestPattern.MatchString(tip.PublicationDigest) || len(tip.PublicationJSON) == 0 ||
			tip.ActorUserID <= 0 || tip.AuditEventID <= 0 {
			return nil, ErrInvalid
		}
		publication, canonical, digest, err := decodeDurableRootPublication(tip.PublicationJSON)
		if err != nil || publication.Artifact.ExtensionID != tip.OwnerExtensionID ||
			publication.Artifact.VersionID != tip.ExtensionVersionID ||
			publication.Artifact.ExtensionVersion != tip.ExtensionVersion ||
			publication.Artifact.PackageDigest != tip.PackageDigest || digest != tip.PublicationDigest {
			return nil, ErrInvalid
		}
		tip.PublicationJSON = append(json.RawMessage(nil), canonical...)
		current, found := result[tip.OwnerExtensionID]
		if found && current.tip.Revision == tip.Revision {
			return nil, ErrInvalid
		}
		if !found || tip.Revision > current.tip.Revision {
			result[tip.OwnerExtensionID] = durableRootPublication{tip: tip, publication: publication}
		}
	}
	return result, nil
}

func validateDurableRootPublication(
	state DurableState,
	publication Publication,
) (Publication, error) {
	normalized, _, digest, err := canonicalDurableRootPublication(publication)
	if err != nil || normalized.Artifact.Core {
		return Publication{}, ErrInvalid
	}
	roots, err := durableRootPublications(state)
	if err != nil {
		return Publication{}, err
	}
	root, found := roots[normalized.Artifact.ExtensionID]
	if !found {
		// Missing root is not a shape error: startup saw an enabled identity
		// surface without any durable Host publication history for that owner.
		return Publication{}, ErrNotFound
	}
	if root.tip.RegistryState != RegistryStateActive {
		return Publication{}, ErrStale
	}
	if durableRootTipArtifactIdentity(root.tip) != durableArtifactIdentityOf(normalized.Artifact) ||
		root.tip.PublicationDigest != digest || !reflect.DeepEqual(root.publication, normalized) {
		return Publication{}, ErrArtifactConflict
	}
	return normalized, nil
}

// ValidateDurablePublicationSet proves that startup has exactly one expected
// enabled publication for every active durable root and no orphan active leaf.
func ValidateDurablePublicationSet(state DurableState, publications []Publication) error {
	if _, err := DurableStateToTombstones(state); err != nil {
		return err
	}
	roots, err := durableRootPublications(state)
	if err != nil {
		return err
	}
	desired := make(map[string]Publication, len(publications))
	for _, publication := range publications {
		normalized, validateErr := validateDurableRootPublication(state, publication)
		if validateErr != nil {
			return validateErr
		}
		if _, duplicate := desired[normalized.Artifact.ExtensionID]; duplicate {
			return ErrInvalid
		}
		desired[normalized.Artifact.ExtensionID] = normalized
		if err := validateDurableLeaves(state, normalized); err != nil {
			return err
		}
	}
	for owner, root := range roots {
		_, published := desired[owner]
		if root.tip.RegistryState == RegistryStateActive && !published {
			return ErrArtifactConflict
		}
		if root.tip.RegistryState == RegistryStateTombstone && published {
			return ErrStale
		}
	}
	for _, tip := range latestDurableDeclarationTips(state.Tips) {
		if tip.RegistryState == RegistryStateActive {
			if _, published := desired[tip.OwnerExtensionID]; !published {
				return ErrArtifactConflict
			}
		}
	}
	return nil
}

// ValidateDurableRetirement proves that one plugin has no active root or leaf
// before the process graph removes it.
func ValidateDurableRetirement(state DurableState, extensionID string) error {
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	if !idPattern.MatchString(extensionID) || strings.HasPrefix(extensionID, "core.") {
		return ErrInvalid
	}
	if _, err := DurableStateToTombstones(state); err != nil {
		return err
	}
	roots, err := durableRootPublications(state)
	if err != nil {
		return err
	}
	if root, found := roots[extensionID]; found && root.tip.RegistryState == RegistryStateActive {
		return ErrStale
	}
	for _, tip := range latestDurableDeclarationTips(state.Tips) {
		if tip.OwnerExtensionID == extensionID && tip.RegistryState == RegistryStateActive {
			return ErrStale
		}
	}
	return nil
}

func latestDurableDeclarationTips(input []DurableDeclarationTip) map[string]DurableDeclarationTip {
	result := make(map[string]DurableDeclarationTip, len(input))
	for _, tip := range input {
		key := ownershipKey(tip.IdentityKind, tip.StableID)
		if current, found := result[key]; !found || tip.Revision > current.Revision {
			result[key] = tip
		}
	}
	return result
}

func durableRootTipArtifactIdentity(tip DurableRootPublicationTip) durableArtifactIdentity {
	return durableArtifactIdentity{
		VersionID: tip.ExtensionVersionID, Version: tip.ExtensionVersion,
		PackageDigest: tip.PackageDigest,
	}
}
