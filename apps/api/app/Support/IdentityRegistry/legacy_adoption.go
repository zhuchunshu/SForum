package identityregistry

import (
	"encoding/json"
	"reflect"
	"strings"
)

// trustGrantEnableAction is the only grant action admitted for legacy
// Identity Registry publication adoption. Upgrade grants are intentionally
// excluded: adoption only repairs pre-feature first-enable history.
const trustGrantEnableAction = "enable"

// trustGrantAuditAction is the immutable Host audit evidence for an exact
// executable trust grant. Prefer the original grant audit over enable audit.
const trustGrantAuditAction = "extension.trust_grant"

// trustImpactSchemaV2 is the only impact schema admitted for adoption. Kept as
// a string constant so IdentityRegistry does not import Models/Extensions
// (Models/Extensions → Models/Identity → IdentityRegistry would cycle).
const trustImpactSchemaV2 = "sforum.trust-impact@2"

// trustImpactExtensionTypePlugin is the only extension type that may own
// Identity Registry publications under Host restore.
const trustImpactExtensionTypePlugin = "plugin"

// StoredTrustImpactValidator recomputes the production canonical TrustImpact
// digest over a stored impact_document. Models/Extensions owns the typed
// TrustImpact schema; production wires extensions.ValidateStoredTrustImpact
// into PostgresStore via NewPostgresStoreWithStoredTrustImpactValidator.
// There is no package-global validator: each store instance holds its own.
type StoredTrustImpactValidator func(document []byte, expectedDigest string) error

// legacyTrustImpactAdoptionSurface is the decoded grant impact slice used after
// full canonical digest integrity succeeds. Top-level authority fields must
// still match the selected grant/desired artifact before identity comparison.
type legacyTrustImpactAdoptionSurface struct {
	SchemaVersion         string                 `json:"schemaVersion"`
	Action                string                 `json:"action"`
	ExtensionID           string                 `json:"extensionId"`
	ExtensionVersion      string                 `json:"extensionVersion"`
	ExtensionType         string                 `json:"extensionType"`
	PackageDigest         string                 `json:"packageDigest"`
	Digest                string                 `json:"digest"`
	Identity              *IdentityDeclaration   `json:"identity"`
	PermissionDefinitions []PermissionDefinition `json:"permissionDefinitions"`
}

// trustImpactAuthorizesDesiredPublication proves:
//  1. production canonical digest integrity over the full typed document;
//  2. top-level grant/desired artifact fields (schema/action/type/package/...);
//  3. identity + permissionDefinitions exact match after Host normalize.
//
// Package digest alone, or identity-subtree-only equality, is never sufficient.
// A nil validator fails closed so stores without an explicit integrity dependency
// cannot adopt.
func trustImpactAuthorizesDesiredPublication(
	validator StoredTrustImpactValidator,
	impactJSON []byte,
	impactDigest string,
	desired Publication,
) error {
	if validator == nil || len(impactJSON) == 0 || !digestPattern.MatchString(impactDigest) {
		return ErrInvalid
	}
	if err := validator(impactJSON, impactDigest); err != nil {
		return ErrInvalid
	}

	var surface legacyTrustImpactAdoptionSurface
	if err := json.Unmarshal(impactJSON, &surface); err != nil {
		return ErrInvalid
	}
	if surface.SchemaVersion != trustImpactSchemaV2 ||
		surface.Action != trustGrantEnableAction ||
		surface.ExtensionType != trustImpactExtensionTypePlugin ||
		surface.ExtensionID != desired.Artifact.ExtensionID ||
		surface.ExtensionVersion != desired.Artifact.ExtensionVersion ||
		surface.PackageDigest != desired.Artifact.PackageDigest ||
		surface.Digest != impactDigest {
		return ErrInvalid
	}

	// Rebuild a candidate under the desired artifact so normalizePublication
	// applies the same ownership / role / runtime rules as live reconcile.
	candidate := Publication{
		Artifact:    desired.Artifact,
		Identity:    surface.Identity,
		Permissions: append([]PermissionDefinition(nil), surface.PermissionDefinitions...),
	}
	normalizedCandidate, err := normalizePublication(candidate)
	if err != nil {
		return ErrInvalid
	}
	normalizedDesired, err := normalizePublication(desired)
	if err != nil {
		return ErrInvalid
	}
	if !reflect.DeepEqual(normalizedCandidate.Permissions, normalizedDesired.Permissions) {
		return ErrInvalid
	}
	if !reflect.DeepEqual(normalizedCandidate.Identity, normalizedDesired.Identity) {
		return ErrInvalid
	}
	return nil
}

func auditMetadataString(metadata map[string]any, key string) string {
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
