package identityregistry

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestTrustImpactAuthorizesDesiredPublicationExactSurface(t *testing.T) {
	publication := adminSurfaceReferencePermissionOnlyPublication(5866)
	impact, digest := mustCanonicalTrustImpactForPublication(t, publication, nil)
	if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, impact, digest, publication); err != nil {
		t.Fatalf("exact permission-only cover: %v", err)
	}

	// Extra grant permission that the process-local publication does not carry.
	extra := publication.Permissions[0]
	extra.Key = "sforum.admin-surface-reference.extra"
	extra.ContractVersion = "sforum.admin-surface-reference.permission.extra@1"
	bloatedPub := publication
	bloatedPub.Permissions = []PermissionDefinition{publication.Permissions[0], extra}
	bloated, bloatedDigest := mustCanonicalTrustImpactForPublication(t, bloatedPub, nil)
	if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, bloated, bloatedDigest, publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("extra impact permission error=%v", err)
	}

	// Label drift is not package-digest evidence.
	drifted := publication
	drifted.Permissions = []PermissionDefinition{publication.Permissions[0]}
	drifted.Permissions[0].Label = "changed"
	mismatched, mismatchedDigest := mustCanonicalTrustImpactForPublication(t, drifted, nil)
	if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, mismatched, mismatchedDigest, publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("impact permission drift error=%v", err)
	}

	if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, nil, digest, publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty impact error=%v", err)
	}
}

func TestTrustImpactAuthorizesDesiredPublicationIdentitySurface(t *testing.T) {
	publication := publicationStoreFixture(
		publicationStoreArtifact(101, "1.0.0", "a", "runtime-v1"), 1, []string{"member"},
	)
	impact, digest := mustCanonicalTrustImpactForPublication(t, publication, nil)
	if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, impact, digest, publication); err != nil {
		t.Fatalf("exact identity cover: %v", err)
	}

	// Missing identity while desired requires it fails closed.
	permissionOnly := publication
	permissionOnly.Identity = nil
	body, digestOnly := mustCanonicalTrustImpactForPublication(t, permissionOnly, nil)
	if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, body, digestOnly, publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing impact identity error=%v", err)
	}
}

func TestTrustImpactAuthorizesDesiredPublicationFullIntegrityMatrix(t *testing.T) {
	publication := adminSurfaceReferencePermissionOnlyPublication(5866)
	base, baseDigest := mustCanonicalTrustImpactDocument(t, publication, nil)

	cases := []struct {
		name        string
		mutate      func(wire *testTrustImpactWire)
		keepDigest  bool
		wrongColumn bool
	}{
		{name: "wrong action", mutate: func(w *testTrustImpactWire) { w.Action = "upgrade" }, keepDigest: true},
		{name: "wrong package", mutate: func(w *testTrustImpactWire) { w.PackageDigest = strings.Repeat("f", 64) }, keepDigest: true},
		{name: "wrong schema", mutate: func(w *testTrustImpactWire) { w.SchemaVersion = "sforum.trust-impact@1" }, keepDigest: true},
		{name: "wrong extension type", mutate: func(w *testTrustImpactWire) { w.ExtensionType = "theme" }, keepDigest: true},
		{name: "wrong extension id", mutate: func(w *testTrustImpactWire) { w.ExtensionID = "other.plugin" }, keepDigest: true},
		{name: "wrong version", mutate: func(w *testTrustImpactWire) { w.ExtensionVersion = "9.9.9" }, keepDigest: true},
		// Document claims a different digest than the grant column while body is unchanged.
		{name: "wrong document digest field", mutate: func(w *testTrustImpactWire) { w.Digest = strings.Repeat("0", 64) }, keepDigest: true},
		{name: "column digest mismatch", wrongColumn: true},
		// Body action forged; digest recomputed so document is self-consistent but wrong for desired.
		{name: "subtree correct wrong action recomputed", mutate: func(w *testTrustImpactWire) { w.Action = "upgrade" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wrongColumn {
				if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, base, strings.Repeat("c", 64), publication); !errors.Is(err, ErrInvalid) {
					t.Fatalf("error=%v", err)
				}
				return
			}
			var wire testTrustImpactWire
			if err := json.Unmarshal(base, &wire); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&wire)
			if !tc.keepDigest {
				digest, err := testCanonicalTrustImpactDigest(wire)
				if err != nil {
					t.Fatal(err)
				}
				wire.Digest = digest
				body, err := json.Marshal(wire)
				if err != nil {
					t.Fatal(err)
				}
				if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, body, baseDigest, publication); !errors.Is(err, ErrInvalid) {
					if err2 := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, body, digest, publication); !errors.Is(err2, ErrInvalid) {
						t.Fatalf("recomputed forgery accepted: columnErr=%v recomputedErr=%v", err, err2)
					}
				}
				return
			}
			// keepDigest: preserve the mutated document (including digest field)
			// and evaluate it against the original grant column digest.
			body, err := json.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, body, baseDigest, publication); !errors.Is(err, ErrInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestTrustImpactAuthorizesDesiredPublicationRequiresValidator(t *testing.T) {
	publication := adminSurfaceReferencePermissionOnlyPublication(5866)
	body, digest := mustCanonicalTrustImpactForPublication(t, publication, nil)
	if err := trustImpactAuthorizesDesiredPublication(nil, body, digest, publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil validator error=%v", err)
	}
}

func TestStoredTrustImpactValidatorsAreInstanceIsolated(t *testing.T) {
	publication := adminSurfaceReferencePermissionOnlyPublication(5866)
	body, digest := mustCanonicalTrustImpactForPublication(t, publication, nil)

	acceptCalls, rejectCalls := 0, 0
	accept := StoredTrustImpactValidator(func(document []byte, expectedDigest string) error {
		acceptCalls++
		return testValidateStoredTrustImpact(document, expectedDigest)
	})
	reject := StoredTrustImpactValidator(func(document []byte, expectedDigest string) error {
		rejectCalls++
		return errors.New("reject-all")
	})

	// Two independent validators do not share mutable package state.
	if err := trustImpactAuthorizesDesiredPublication(accept, body, digest, publication); err != nil {
		t.Fatalf("accept validator: %v", err)
	}
	if err := trustImpactAuthorizesDesiredPublication(reject, body, digest, publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("reject validator error=%v", err)
	}
	if acceptCalls != 1 || rejectCalls != 1 {
		t.Fatalf("calls accept=%d reject=%d", acceptCalls, rejectCalls)
	}

	// Store instances bind their own validator; ordinary NewPostgresStore fails closed.
	unconfigured := NewPostgresStore(nil)
	configuredAccept := NewPostgresStoreWithStoredTrustImpactValidator(nil, accept)
	configuredReject := NewPostgresStoreWithStoredTrustImpactValidator(nil, reject)
	if unconfigured.HasStoredTrustImpactValidator() {
		t.Fatal("NewPostgresStore must leave validator unset")
	}
	if !configuredAccept.HasStoredTrustImpactValidator() || !configuredReject.HasStoredTrustImpactValidator() {
		t.Fatal("with-validator constructor must bind instance validator")
	}
	// Mutating one instance must not affect the other.
	configuredAccept.trustImpactValidator = reject
	if !configuredReject.HasStoredTrustImpactValidator() {
		t.Fatal("peer store validator was cleared")
	}
	if err := trustImpactAuthorizesDesiredPublication(configuredReject.trustImpactValidator, body, digest, publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("reject store still rejects: %v", err)
	}
	configuredAccept.trustImpactValidator = accept
	if err := trustImpactAuthorizesDesiredPublication(configuredAccept.trustImpactValidator, body, digest, publication); err != nil {
		t.Fatalf("accept store after peer use: %v", err)
	}
}

func TestLocalTrustImpactWireMatchesProductionAdminDigest(t *testing.T) {
	// Pin against the production-computed digest for the admin-surface fixture so
	// the test wire algorithm cannot silently drift from Models/Extensions.
	const wantDigest = "4a30a962bb10763184dc5f014805012183661d4858ccc81d8bd1cd637dfdc07c"
	publication := adminSurfaceReferencePermissionOnlyPublication(5866)
	_, digest := mustCanonicalTrustImpactForPublication(t, publication, nil)
	if digest != wantDigest {
		t.Fatalf("local wire digest=%s want production %s", digest, wantDigest)
	}
}
