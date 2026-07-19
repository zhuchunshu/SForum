package identityregistry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// testTrustImpactWire mirrors Models/Extensions.TrustImpact JSON field order so
// IdentityRegistry tests can recompute the production canonical digest without
// importing Models/Extensions (import cycle via Models/Identity).
//
// Nested objects use typed fields (not json.RawMessage) so PostgreSQL jsonb
// key-order normalization still round-trips to the production digest. Production
// runtime injects extensions.ValidateStoredTrustImpact per PostgresStore instance.
type testTrustImpactWire struct {
	SchemaVersion         string                  `json:"schemaVersion"`
	Action                string                  `json:"action"`
	ExtensionID           string                  `json:"extensionId"`
	ExtensionVersion      string                  `json:"extensionVersion"`
	ExtensionType         string                  `json:"extensionType"`
	Source                string                  `json:"source"`
	PackageDigest         string                  `json:"packageDigest"`
	ManifestContract      string                  `json:"manifestContract"`
	ArtifactDigests       map[string]string       `json:"artifactDigests"`
	Binaries              []testTrustArtifact     `json:"binaries"`
	Backend               testTrustBackend        `json:"backend"`
	Routes                []json.RawMessage       `json:"routes"`
	Guards                []testTrustGuard        `json:"guards"`
	GuardDeclarations     []json.RawMessage       `json:"guardDeclarations"`
	Hooks                 []json.RawMessage       `json:"hooks"`
	Events                []json.RawMessage       `json:"events"`
	Migrations            []testTrustMigration    `json:"migrations"`
	MigrationDeclarations []json.RawMessage       `json:"migrationDeclarations"`
	Providers             []json.RawMessage       `json:"providers"`
	Jobs                  []json.RawMessage       `json:"jobs"`
	Schedules             []json.RawMessage       `json:"schedules"`
	Components            []json.RawMessage       `json:"components"`
	RegistryComponents    []json.RawMessage       `json:"registryComponents"`
	Templates             []json.RawMessage       `json:"templates"`
	Assets                []json.RawMessage       `json:"assets"`
	Content               []json.RawMessage       `json:"content"`
	Database              json.RawMessage         `json:"database"`
	Cache                 []json.RawMessage       `json:"cache"`
	SEO                   []json.RawMessage       `json:"seo,omitempty"`
	Services              []json.RawMessage       `json:"services"`
	Commands              []json.RawMessage       `json:"commands"`
	AdminSurfaces         []json.RawMessage       `json:"adminSurfaces"`
	Queries               []json.RawMessage       `json:"queries"`
	Identity              *legacyManifestIdentity `json:"identity"`
	PermissionDefinitions []PermissionDefinition  `json:"permissionDefinitions"`
	Media                 []json.RawMessage       `json:"media"`
	Navigation            []json.RawMessage       `json:"navigation"`
	Regions               []json.RawMessage       `json:"regions"`
	Contributions         []json.RawMessage       `json:"contributions"`
	Capabilities          []json.RawMessage       `json:"capabilities"`
	Permissions           []string                `json:"permissions"`
	RequiredFeatures      []string                `json:"requiredFeatures"`
	Dependencies          []json.RawMessage       `json:"dependencies"`
	Lifecycle             json.RawMessage         `json:"lifecycle"`
	OpenAPI               []json.RawMessage       `json:"openapi"`
	PackageFiles          []json.RawMessage       `json:"packageFiles"`
	RequestedAuthority    testTrustAuthority      `json:"requestedAuthority"`
	Contracts             testTrustContracts      `json:"contracts"`
	Digest                string                  `json:"digest"`
}

type testTrustArtifact struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type testTrustMigration struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type testTrustGuard struct {
	Path       string   `json:"path"`
	Methods    []string `json:"methods"`
	Access     string   `json:"access"`
	Permission string   `json:"permission,omitempty"`
}

type testTrustBackend struct {
	Entry string `json:"entry"`
	RPC   string `json:"rpc"`
}

type testTrustAuthority struct {
	BackendExecution       bool     `json:"backendExecution"`
	AdminFrontendExecution bool     `json:"adminFrontendExecution"`
	RawRequest             bool     `json:"rawRequest"`
	RawCoreDatabase        bool     `json:"rawCoreDatabase"`
	OutboundNetwork        bool     `json:"outboundNetwork"`
	PackageFiles           []string `json:"packageFiles"`
	Secrets                []string `json:"secrets"`
}

type testTrustContracts struct {
	HostAPI     string `json:"hostApi"`
	FrontendAPI string `json:"frontendApi,omitempty"`
}

func testCanonicalTrustImpactDigest(wire testTrustImpactWire) (string, error) {
	wire.Digest = ""
	body, err := json.Marshal(wire)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func testValidateStoredTrustImpact(document []byte, expectedDigest string) error {
	expectedDigest = strings.ToLower(strings.TrimSpace(expectedDigest))
	if len(document) == 0 || !digestPattern.MatchString(expectedDigest) {
		return fmt.Errorf("invalid stored trust impact")
	}
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	var wire testTrustImpactWire
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing stored trust impact")
	}
	if wire.Digest != expectedDigest {
		return fmt.Errorf("document digest mismatch")
	}
	recomputed, err := testCanonicalTrustImpactDigest(wire)
	if err != nil {
		return err
	}
	if recomputed != expectedDigest {
		return fmt.Errorf("canonical digest mismatch")
	}
	return nil
}

func mustCanonicalTrustImpactDocument(
	t *testing.T,
	publication Publication,
	mutate func(*testTrustImpactWire),
) ([]byte, string) {
	t.Helper()
	wire := testTrustImpactWire{
		SchemaVersion:         trustImpactSchemaV2,
		Action:                trustGrantEnableAction,
		ExtensionID:           publication.Artifact.ExtensionID,
		ExtensionVersion:      publication.Artifact.ExtensionVersion,
		ExtensionType:         trustImpactExtensionTypePlugin,
		PackageDigest:         publication.Artifact.PackageDigest,
		Identity:              legacyManifestIdentityFromDeclaration(publication.Identity),
		PermissionDefinitions: append([]PermissionDefinition(nil), publication.Permissions...),
	}
	if mutate != nil {
		mutate(&wire)
	}
	digest, err := testCanonicalTrustImpactDigest(wire)
	if err != nil {
		t.Fatal(err)
	}
	wire.Digest = digest
	body, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if err := testValidateStoredTrustImpact(body, digest); err != nil {
		t.Fatalf("local impact document self-check: %v body=%s", err, string(body))
	}
	return body, digest
}

func mustCanonicalTrustImpactForPublication(
	t *testing.T,
	publication Publication,
	mutate func(*testTrustImpactWire),
) ([]byte, string) {
	t.Helper()
	return mustCanonicalTrustImpactDocument(t, publication, mutate)
}
