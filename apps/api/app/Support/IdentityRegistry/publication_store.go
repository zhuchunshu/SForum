package identityregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

type durableArtifactIdentity struct {
	VersionID     int64
	Version       string
	PackageDigest string
}

type normalizedReconcilePublicationInput struct {
	extensionID  string
	artifacts    []Artifact
	allowed      map[durableArtifactIdentity]struct{}
	desired      *Publication
	actorUserID  int64
	auditEventID int64
}

type durableDesiredDeclaration struct {
	kind            string
	stableID        string
	contractVersion string
	digest          string
	artifact        Artifact
	permission      *PermissionDefinition
}

// ValidateDurablePublication proves that one process-local publication is the
// exact active declaration graph authorized by durable Host history. Runtime
// instance identity is checked by the caller because durable tips intentionally
// bind package artifacts, not process lifetimes.
func ValidateDurablePublication(state DurableState, publication Publication) error {
	if _, err := DurableStateToTombstones(state); err != nil {
		return err
	}
	normalized, err := validateDurableRootPublication(state, publication)
	if err != nil {
		return err
	}
	return validateDurableLeaves(state, normalized)
}

func validateDurableLeaves(state DurableState, normalized Publication) error {
	desired, err := desiredDurableDeclarations(&normalized)
	if err != nil {
		return err
	}
	desiredByKey := make(map[string]durableDesiredDeclaration, len(desired))
	for _, declaration := range desired {
		desiredByKey[ownershipKey(declaration.kind, declaration.stableID)] = declaration
	}
	owners := make(map[string]string, len(state.Owners))
	for _, owner := range state.Owners {
		owners[ownershipKey(owner.IdentityKind, owner.StableID)] = owner.OwnerExtensionID
	}
	tips := make(map[string]DurableDeclarationTip, len(state.Tips))
	for _, tip := range state.Tips {
		key := ownershipKey(tip.IdentityKind, tip.StableID)
		if current, found := tips[key]; !found || tip.Revision > current.Revision {
			tips[key] = tip
		}
	}
	for key, declaration := range desiredByKey {
		if owner := owners[key]; owner == "" {
			return ErrInvalid
		} else if owner != normalized.Artifact.ExtensionID {
			return ErrConflict
		}
		tip, found := tips[key]
		if !found {
			return ErrInvalid
		}
		if tip.RegistryState != RegistryStateActive {
			return ErrStale
		}
		if tip.OwnerExtensionID != normalized.Artifact.ExtensionID ||
			durableTipArtifactIdentity(tip) != durableArtifactIdentityOf(declaration.artifact) ||
			tip.ContractVersion != declaration.contractVersion || tip.DeclarationDigest != declaration.digest {
			return ErrArtifactConflict
		}
	}
	for key, tip := range tips {
		if tip.OwnerExtensionID == normalized.Artifact.ExtensionID && tip.RegistryState == RegistryStateActive {
			if _, found := desiredByKey[key]; !found {
				return ErrArtifactConflict
			}
		}
	}
	return nil
}

func normalizeReconcilePublicationInput(input ReconcilePublicationInput) (normalizedReconcilePublicationInput, error) {
	extensionID := strings.ToLower(strings.TrimSpace(input.ExtensionID))
	if !idPattern.MatchString(extensionID) || strings.HasPrefix(extensionID, "core.") ||
		input.ActorUserID <= 0 || input.AuditEventID <= 0 {
		return normalizedReconcilePublicationInput{}, ErrInvalid
	}

	result := normalizedReconcilePublicationInput{
		extensionID:  extensionID,
		allowed:      make(map[durableArtifactIdentity]struct{}, 2),
		actorUserID:  input.ActorUserID,
		auditEventID: input.AuditEventID,
	}
	var target *Artifact
	for index, raw := range []*Artifact{input.AllowedSource, input.AllowedTarget} {
		if raw == nil {
			continue
		}
		artifact, err := normalizeArtifact(*raw)
		if err != nil || artifact.Core || artifact.ExtensionID != extensionID {
			return normalizedReconcilePublicationInput{}, ErrInvalid
		}
		identity := durableArtifactIdentityOf(artifact)
		if _, duplicate := result.allowed[identity]; !duplicate {
			result.allowed[identity] = struct{}{}
			result.artifacts = append(result.artifacts, artifact)
		}
		if index == 1 {
			target = &artifact
		}
	}
	if len(result.allowed) == 0 {
		return normalizedReconcilePublicationInput{}, ErrInvalid
	}
	sort.Slice(result.artifacts, func(i, j int) bool {
		left, right := durableArtifactIdentityOf(result.artifacts[i]), durableArtifactIdentityOf(result.artifacts[j])
		if left.VersionID != right.VersionID {
			return left.VersionID < right.VersionID
		}
		if left.Version != right.Version {
			return left.Version < right.Version
		}
		return left.PackageDigest < right.PackageDigest
	})

	if input.Desired != nil {
		publication, err := normalizePublication(*input.Desired)
		if err != nil || publication.Artifact.Core || publication.Artifact.ExtensionID != extensionID ||
			target == nil || publication.Artifact != *target {
			return normalizedReconcilePublicationInput{}, ErrInvalid
		}
		result.desired = &publication
	}
	return result, nil
}

func desiredDurableDeclarations(publication *Publication) ([]durableDesiredDeclaration, error) {
	if publication == nil {
		return nil, nil
	}
	result := make([]durableDesiredDeclaration, 0, len(publication.Permissions))
	for index := range publication.Permissions {
		permission := publication.Permissions[index]
		digest, err := durableDeclarationDigest(TombstoneKindPermission, permission)
		if err != nil {
			return nil, err
		}
		result = append(result, durableDesiredDeclaration{
			kind: TombstoneKindPermission, stableID: permission.Key,
			contractVersion: permission.ContractVersion, digest: digest,
			artifact: publication.Artifact, permission: &permission,
		})
	}
	if publication.Identity != nil {
		for _, field := range publication.Identity.UserFields {
			digest, err := durableDeclarationDigest(TombstoneKindUserField, field)
			if err != nil {
				return nil, err
			}
			result = append(result, durableDesiredDeclaration{
				kind: TombstoneKindUserField, stableID: field.ID,
				contractVersion: field.ContractVersion, digest: digest,
				artifact: publication.Artifact,
			})
		}
		for _, provider := range publication.Identity.Providers {
			digest, err := durableDeclarationDigest(TombstoneKindProvider, provider)
			if err != nil {
				return nil, err
			}
			result = append(result, durableDesiredDeclaration{
				kind: TombstoneKindProvider, stableID: provider.ID,
				contractVersion: provider.ContractVersion, digest: digest,
				artifact: publication.Artifact,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		leftOrder, rightOrder := durableKindOrder(result[i].kind), durableKindOrder(result[j].kind)
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		return result[i].stableID < result[j].stableID
	})
	return result, nil
}

func durableDeclarationDigest(kind string, declaration any) (string, error) {
	raw, err := json.Marshal(declaration)
	if err != nil {
		return "", ErrInvalid
	}
	sum := sha256.Sum256(append([]byte(kind+"\x00"), raw...))
	return hex.EncodeToString(sum[:]), nil
}

func durableArtifactIdentityOf(artifact Artifact) durableArtifactIdentity {
	return durableArtifactIdentity{
		VersionID: artifact.VersionID, Version: artifact.ExtensionVersion,
		PackageDigest: artifact.PackageDigest,
	}
}

func durableTipArtifactIdentity(tip DurableDeclarationTip) durableArtifactIdentity {
	return durableArtifactIdentity{
		VersionID: tip.ExtensionVersionID, Version: tip.ExtensionVersion,
		PackageDigest: tip.PackageDigest,
	}
}

func durableKindOrder(kind string) int {
	switch kind {
	case TombstoneKindPermission:
		return 0
	case TombstoneKindUserField:
		return 1
	case TombstoneKindProvider:
		return 2
	default:
		return 3
	}
}
