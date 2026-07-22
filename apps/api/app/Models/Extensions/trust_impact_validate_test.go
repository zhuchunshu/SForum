package extensions

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateStoredTrustImpactAcceptsCanonicalDocument(t *testing.T) {
	impact := minimalStoredTrustImpact("demo.validate", "1.0.0", strings.Repeat("a", 64))
	digest, err := canonicalTrustImpactDigest(impact)
	if err != nil {
		t.Fatal(err)
	}
	impact.Digest = digest
	body, err := json.Marshal(impact)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStoredTrustImpact(body, digest); err != nil {
		t.Fatalf("valid document: %v", err)
	}
}

func TestValidateStoredTrustImpactRejectsDigestTampering(t *testing.T) {
	impact := minimalStoredTrustImpact("demo.tamper", "1.0.0", strings.Repeat("b", 64))
	digest, err := canonicalTrustImpactDigest(impact)
	if err != nil {
		t.Fatal(err)
	}
	impact.Digest = digest
	body, err := json.Marshal(impact)
	if err != nil {
		t.Fatal(err)
	}

	// Column digest does not match document field.
	if err := ValidateStoredTrustImpact(body, strings.Repeat("c", 64)); !errors.Is(err, ErrStoredTrustImpactInvalid) {
		t.Fatalf("column mismatch error=%v", err)
	}

	// Document field rewritten without recomputing body digest.
	tampered := impact
	tampered.Digest = strings.Repeat("d", 64)
	tamperedBody, err := json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStoredTrustImpact(tamperedBody, digest); !errors.Is(err, ErrStoredTrustImpactInvalid) {
		t.Fatalf("document digest field mismatch error=%v", err)
	}

	// Action changed while keeping original digest claim (subtree-correct forgery).
	forged := impact
	forged.Action = "upgrade"
	forged.Digest = digest
	forgedBody, err := json.Marshal(forged)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStoredTrustImpact(forgedBody, digest); !errors.Is(err, ErrStoredTrustImpactInvalid) {
		t.Fatalf("action forgery error=%v", err)
	}

	if err := ValidateStoredTrustImpact(nil, digest); !errors.Is(err, ErrStoredTrustImpactInvalid) {
		t.Fatalf("empty document error=%v", err)
	}
	if err := ValidateStoredTrustImpact(body, "not-a-digest"); !errors.Is(err, ErrStoredTrustImpactInvalid) {
		t.Fatalf("bad expected digest error=%v", err)
	}
}

func TestValidateStoredTrustImpactRejectsUnknownTopLevelField(t *testing.T) {
	impact := minimalStoredTrustImpact("demo.unknown", "1.0.0", strings.Repeat("e", 64))
	digest, err := canonicalTrustImpactDigest(impact)
	if err != nil {
		t.Fatal(err)
	}
	impact.Digest = digest
	body, err := json.Marshal(impact)
	if err != nil {
		t.Fatal(err)
	}
	// Inject a top-level key outside TrustImpact so typed Unmarshal would ignore it.
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatal(err)
	}
	object["extraCapability"] = json.RawMessage(`"not-in-schema"`)
	forged, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStoredTrustImpact(forged, digest); !errors.Is(err, ErrStoredTrustImpactInvalid) {
		t.Fatalf("unknown top-level field error=%v", err)
	}
}

func TestValidateStoredTrustImpactRejectsTrailingJSONValue(t *testing.T) {
	impact := minimalStoredTrustImpact("demo.trailing", "1.0.0", strings.Repeat("f", 64))
	digest, err := canonicalTrustImpactDigest(impact)
	if err != nil {
		t.Fatal(err)
	}
	impact.Digest = digest
	body, err := json.Marshal(impact)
	if err != nil {
		t.Fatal(err)
	}
	// Second top-level JSON value after a valid document.
	trailing := append(append([]byte(nil), body...), []byte(`{"extra":true}`)...)
	if err := ValidateStoredTrustImpact(trailing, digest); !errors.Is(err, ErrStoredTrustImpactInvalid) {
		t.Fatalf("trailing JSON error=%v", err)
	}
}

func minimalStoredTrustImpact(extensionID, version, packageDigest string) TrustImpact {
	return TrustImpact{
		SchemaVersion:    TrustImpactSchemaV2,
		Action:           TrustActionEnable,
		ExtensionID:      extensionID,
		ExtensionVersion: version,
		ExtensionType:    TypePlugin,
		PackageDigest:    packageDigest,
		PermissionDefinitions: []ManifestPermissionDefinition{{
			Key: extensionID + ".manage", ContractVersion: extensionID + ".permission.manage@1",
			Label: LocalizedText{Default: "Manage"}, Description: LocalizedText{Default: "Manage"}, AssignmentPolicy: "host",
			RecommendedRoles: []string{"administrator"},
		}},
	}
}
